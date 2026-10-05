package unified

import (
	"fmt"
	"sort"
	"time"

	"github.com/open-hzbank/ultra-flow/arrange"
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/ops"
	"github.com/open-hzbank/ultra-flow/stateful"
	"github.com/open-hzbank/ultra-flow/support"
	flowsync "github.com/open-hzbank/ultra-flow/sync"
)

const (
	// PublishTaskName 发布任务名称
	PublishTaskName = "FlowPublishTask"
	// PublishTaskType 发布任务类型
	PublishTaskType = "FLOW_UNIFIED_ARRANGE"

	// SubjectPublishConfigNames 发布配置名称
	SubjectPublishConfigNames = "configNames"
	// SubjectPublishBiz 发布业务域
	SubjectPublishBiz = "biz"
	// SubjectTaskType 任务类型
	SubjectTaskType = "taskType"

	// MatchKeyPublishTask 匹配键: 发布任务
	MatchKeyPublishTask = "publishTask"
	// MatchKeyPublishBiz 匹配键: 发布业务域
	MatchKeyPublishBiz = "publishBiz"
	// MatchKeyPublishEnv 匹配键: 发布环境
	MatchKeyPublishEnv = "publishEnv"
)

const (
	publishTaskNotExist = "未找到指定流控规则发布任务"
	lockWaitTime        = 1000 * time.Millisecond
)

// FlowPublishService 与统一流量发布任务相关的基础服务
type FlowPublishService struct {
	taskSnapshotRepository     stateful.TaskSnapshotRepository
	taskStepSnapshotRepository stateful.TaskStepSnapshotRepository
	opsTaskRunnerFactory       *ops.OpsTaskRunnerFactory
	lockService                flowsync.LockService
	taskDefConfigService       *arrange.TaskDefConfigService
	publishTaskBuilder         *FlowPublishTaskBuilder
}

// NewFlowPublishService 创建流量发布服务
func NewFlowPublishService(
	taskSnapshotRepository stateful.TaskSnapshotRepository,
	taskStepSnapshotRepository stateful.TaskStepSnapshotRepository,
	opsTaskRunnerFactory *ops.OpsTaskRunnerFactory,
	lockService flowsync.LockService,
	taskDefConfigService *arrange.TaskDefConfigService,
	publishTaskBuilder *FlowPublishTaskBuilder,
) *FlowPublishService {
	return &FlowPublishService{
		taskSnapshotRepository:     taskSnapshotRepository,
		taskStepSnapshotRepository: taskStepSnapshotRepository,
		opsTaskRunnerFactory:       opsTaskRunnerFactory,
		lockService:                lockService,
		taskDefConfigService:       taskDefConfigService,
		publishTaskBuilder:         publishTaskBuilder,
	}
}

// region create task

// CreateTask 创建流控策略发布任务
//
// 特性:
//  1. 使用全局互斥锁保证并发安全, 保证同一个流控策略同一时刻只能有一个未达终态的任务
//  2. 使用幂等 id 做幂等判断, 重复提交幂等返回
func (s *FlowPublishService) CreateTask(publishData []FlowConfig,
	publishTypes map[string]core.PublishType,
	biz, creator string,
	publishEnv core.Env,
	idempotentID, publishReason string,
	emergencyPublish bool,
	taskType core.TaskType,
) support.Result[string] {
	return s.doInLockedResources(publishData, func() support.Result[string] {
		// 幂等判断
		idempotentResult := s.checkSubmitted(idempotentID)
		if idempotentResult.IsSuccess() {
			return idempotentResult
		}
		// 资源发布冲突检测
		configNames := make([]string, 0, len(publishData))
		for _, fc := range publishData {
			configNames = append(configNames, fc.Name)
		}
		conflictResult := s.checkConflict(configNames, publishEnv)
		if conflictResult.IsSuccess() {
			return conflictResult
		}
		// todo 实时性校验: 检查是否是最新版本

		// 任务创建
		ctx := &FlowPublishContext{
			FlowConfigs:      publishData,
			PublishTypes:     publishTypes,
			Creator:          creator,
			IdempotentID:     idempotentID,
			PublishEnv:       publishEnv,
			Biz:              biz,
			PublishReason:    publishReason,
			TaskType:         taskType,
			EmergencyPublish: taskType == core.TaskTypeRollback || emergencyPublish,
		}
		taskID := s.doCreateTask(ctx)
		return support.SuccessResultWithData(taskID)
	})
}

func (s *FlowPublishService) doInLockedResources(publishData []FlowConfig, procedure func() support.Result[string]) support.Result[string] {
	// 按名称排序
	sortedConfigs := make([]FlowConfig, len(publishData))
	copy(sortedConfigs, publishData)
	sort.Slice(sortedConfigs, func(i, j int) bool {
		return sortedConfigs[i].Name < sortedConfigs[j].Name
	})

	// 按序获取所有资源的锁
	locks := make([]flowsync.BriefLock, 0, len(sortedConfigs))
	for _, config := range sortedConfigs {
		lock := s.lockService.BuildLock(config.Name)
		if !lock.TryLockWithTimeout(lockWaitTime) {
			// 释放已获取的锁
			for _, l := range locks {
				l.Unlock()
			}
			panic(&core.LockFailureError{Message: "未成功获取到锁"})
		}
		locks = append(locks, lock)
	}

	// 成功获取所有锁后，执行目标逻辑
	defer func() {
		for _, lock := range locks {
			lock.Unlock()
		}
	}()
	return procedure()
}

// doCreateTask 创建任务
func (s *FlowPublishService) doCreateTask(ctx *FlowPublishContext) string {
	task := s.publishTaskBuilder.Build(ctx)
	// 保存任务快照
	snapshot := &stateful.TaskSnapshot{
		TaskID:       task.GetTaskID(),
		IdempotentID: ctx.IdempotentID,
		Name:         task.GetName(),
		Type:         task.GetType(),
		Subject:      task.GetSubject(),
		PublishEnv:   task.GetPublishEnv(),
		Creator:      task.GetCreator(),
		Status:       core.TaskCreated,
		Description:  buildDescription(ctx.PublishReason),
	}
	s.taskSnapshotRepository.SaveSnapshot(snapshot)
	return task.GetTaskID()
}

// endregion

// region execute task

// ExecuteTask 任务执行 (驱动)
func (s *FlowPublishService) ExecuteTask(taskID string) core.TaskResult {
	builder := ops.GetOpsTaskBuilder(PublishTaskName, PublishTaskType)
	return s.opsTaskRunnerFactory.SynchronizedTaskRunner(taskID).Execute(builder,
		func(b ops.OpsTaskBuilder) core.Task {
			return b.InternalResumeBuild(taskID)
		})
}

// ExecuteTaskWithStepConfig 针对指定步骤携带指定性参数的任务执行 (驱动)
func (s *FlowPublishService) ExecuteTaskWithStepConfig(taskID, stepName, key string, value any) core.TaskResult {
	builder := ops.GetOpsTaskBuilder(PublishTaskName, PublishTaskType)
	return s.opsTaskRunnerFactory.SynchronizedTaskRunner(taskID).Execute(builder,
		func(b ops.OpsTaskBuilder) core.Task {
			return b.ResumeBuildWithStepConfig(taskID, stepName, key, value)
		})
}

// ExecuteTaskWithStepConfigs 携带步骤配置列表的任务执行 (驱动)
func (s *FlowPublishService) ExecuteTaskWithStepConfigs(taskID string, stepConfigs []core.StepExecuteConfig) core.TaskResult {
	builder := ops.GetOpsTaskBuilder(PublishTaskName, PublishTaskType)
	return s.opsTaskRunnerFactory.SynchronizedTaskRunner(taskID).Execute(builder,
		func(b ops.OpsTaskBuilder) core.Task {
			return b.ResumeBuildWithStepConfigs(taskID, stepConfigs)
		})
}

// ExecuteTaskWithTaskContext 批量携带任务全局参数的任务执行 (驱动)
func (s *FlowPublishService) ExecuteTaskWithTaskContext(taskID string, taskContext map[string]any) core.TaskResult {
	builder := ops.GetOpsTaskBuilder(PublishTaskName, PublishTaskType)
	return s.opsTaskRunnerFactory.SynchronizedTaskRunner(taskID).Execute(builder,
		func(b ops.OpsTaskBuilder) core.Task {
			return b.ResumeBuildWithTaskContext(taskID, taskContext)
		})
}

// endregion

// region query task

// GetTask 获取指定任务
func (s *FlowPublishService) GetTask(taskID string) support.Result[*stateful.TaskSnapshot] {
	snapshot := s.taskSnapshotRepository.Get(taskID)
	if snapshot == nil {
		return support.FailResult[*stateful.TaskSnapshot](publishTaskNotExist)
	}
	return support.SuccessResultWithData(snapshot)
}

// GetTaskSteps 获取任务步骤
func (s *FlowPublishService) GetTaskSteps(taskID string) support.Result[[]*stateful.TaskStepSnapshot] {
	steps := s.taskStepSnapshotRepository.GetTaskSteps(taskID)
	if steps == nil {
		return support.FailResult[[]*stateful.TaskStepSnapshot](publishTaskNotExist)
	}
	return support.SuccessResultWithData(steps)
}

// IsStepRunning 判断步骤是否正在运行
func (s *FlowPublishService) IsStepRunning(taskID, taskStepType string) bool {
	result := s.GetTaskSteps(taskID)
	if !result.IsSuccess() {
		return false
	}
	steps := result.GetData()
	for _, step := range steps {
		if step.Type == taskStepType && step.Status.IsExecuting() {
			return true
		}
	}
	return false
}

// SearchTasks 按条件查询任务
func (s *FlowPublishService) SearchTasks(
	biz string,
	configNames []string,
	creator string,
	publishEnvs []core.Env,
	targetStatuses []core.TaskStatus,
	order support.Order,
	fromTime, toTime time.Time,
	pageNum, pageSize int,
) support.PageResult[stateful.TaskSnapshot] {
	var subjectQueryParams [][2]string
	if biz != "" {
		subjectQueryParams = append(subjectQueryParams, [2]string{
			"subject -> '$." + SubjectPublishConfigNames + "' = {0}",
			biz,
		})
	}
	if len(configNames) > 0 {
		for _, configName := range configNames {
			subjectQueryParams = append(subjectQueryParams, [2]string{
				"JSON_CONTAINS_PATH(subject -> '$." + SubjectPublishConfigNames + "', 'one', {0})",
				"$.\"" + configName + "\"",
			})
		}
	}
	return s.taskSnapshotRepository.Search(
		creator, publishEnvs, order,
		subjectQueryParams,
		targetStatuses,
		pageNum, pageSize,
		fromTime, toTime,
		PublishTaskName,
		PublishTaskType,
	)
}

// endregion

// CancelTask 取消任务
//
// 由 SynchronizedOpsTaskRunner.Execute 方法保证全局互斥
func (s *FlowPublishService) CancelTask(taskID string) {
	builder := ops.GetOpsTaskBuilder(PublishTaskName, PublishTaskType)
	s.opsTaskRunnerFactory.SynchronizedTaskRunner(taskID).Cancel(builder,
		func(b ops.OpsTaskBuilder) core.Task {
			return b.ResumeBuild(taskID)
		})
}

// InterruptTask 强制中断任务
func (s *FlowPublishService) InterruptTask(taskID string) {
	builder := ops.GetOpsTaskBuilder(PublishTaskName, PublishTaskType)
	s.opsTaskRunnerFactory.SynchronizedTaskRunner(taskID).Interrupt(builder,
		func(b ops.OpsTaskBuilder) core.Task {
			return b.ResumeBuild(taskID)
		})
}

// GetTaskStageDefinition 获取任务阶段定义
func (s *FlowPublishService) GetTaskStageDefinition(biz string, publishEnv core.Env) *arrange.TaskStageDefinition {
	matchCondition := map[string]string{
		MatchKeyPublishTask: PublishTaskName,
		MatchKeyPublishBiz:  biz,
		MatchKeyPublishEnv:  string(publishEnv),
	}
	stageDef, ok := s.taskDefConfigService.GetFlowDefinition(matchCondition)
	if !ok {
		panic("找不到合适的发布流程, matchCondition = " + mapToString(matchCondition))
	}
	return stageDef
}

// region 发布任务提交检测

// checkSubmitted 幂等判断是否已有相同请求提交
func (s *FlowPublishService) checkSubmitted(idempotentID string) support.Result[string] {
	if idempotentID == "" {
		return support.FailResult[string]("")
	}
	snapshot := s.taskSnapshotRepository.FindByIdempotentID(idempotentID, PublishTaskName, PublishTaskType)
	if snapshot == nil {
		return support.FailResult[string]("")
	}
	return support.SuccessResultWithData(snapshot.TaskID)
}

// checkConflict 资源发布冲突检测
func (s *FlowPublishService) checkConflict(publishContentNames []string, publishEnv core.Env) support.Result[string] {
	for _, name := range publishContentNames {
		taskID := s.getActivePublishTaskID(name, publishEnv)
		if taskID != "" {
			return support.SuccessResultWithData(taskID)
		}
	}
	return support.FailResult[string]("")
}

// filterPublishingContent 查询指定发布目标中是否有正在发布中的任务
func (s *FlowPublishService) filterPublishingContent(publishContentNames []string, publishEnv core.Env) map[string]string {
	result := make(map[string]string)
	for _, name := range publishContentNames {
		taskID := s.getActivePublishTaskID(name, publishEnv)
		if taskID != "" {
			result[name] = taskID
		}
	}
	return result
}

// getActivePublishTaskID 查询指定发布目标是否有正在发布中的任务
func (s *FlowPublishService) getActivePublishTaskID(publishContentName string, publishEnv core.Env) string {
	subjectQueryParams := map[string]any{
		fmt.Sprintf("JSON_CONTAINS_PATH( subject -> '$.%s', 'one', {0})", SubjectPublishConfigNames): fmt.Sprintf("$.\"%s\"", publishContentName),
	}
	snapshot := s.taskSnapshotRepository.FindBySubject(publishEnv, subjectQueryParams, PublishTaskName, PublishTaskType, core.GetActiveStatuses())
	if snapshot == nil {
		return ""
	}
	return snapshot.TaskID
}

// endregion
