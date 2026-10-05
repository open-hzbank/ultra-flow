package unified

import (
	"github.com/open-hzbank/ultra-flow/arrange"
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/ops"
	"github.com/open-hzbank/ultra-flow/stateful"
	"github.com/open-hzbank/ultra-flow/support"
	"github.com/open-hzbank/ultra-flow/transfer"
)

// FlowPublishTask 统一流量编排任务
type FlowPublishTask struct {
	*ops.CompensableOpsTask
}

// NewFlowPublishTask 创建新的流量发布任务
func NewFlowPublishTask(ctx *FlowPublishContext,
	taskPersistence stateful.TaskPersistence,
	transactionable stateful.Transactionable,
	stepSupplier func(*core.TaskContext, *FlowPublishContext) core.CompensableTaskStep,
) *FlowPublishTask {
	task := ops.NewCompensableOpsTask(
		ctx,
		support.BuildFlowID(),
		PublishTaskName,
		PublishTaskType,
		buildSubject(ctx),
		ctx.Creator,
		ctx.IdempotentID,
		ctx.PublishEnv,
		buildDescription(ctx.PublishReason),
		transfer.NewDefaultTaskStepTransfer(),
		func(taskCtx *core.TaskContext, data any) core.CompensableTaskStep {
			return stepSupplier(taskCtx, data.(*FlowPublishContext))
		},
		taskPersistence,
		transactionable,
		core.NewJSONTaskDataSerDeser(&FlowPublishContext{}),
	)
	return &FlowPublishTask{CompensableOpsTask: task}
}

// ResumeFlowPublishTask 恢复流量发布任务
func ResumeFlowPublishTask(taskID string,
	appendContext map[string]any,
	taskStepTransfer core.TaskStepTransfer,
	taskPersistence stateful.TaskPersistence,
	transactionable stateful.Transactionable,
	stepSupplier func(*core.TaskContext, *FlowPublishContext) core.CompensableTaskStep,
) *FlowPublishTask {
	task := ops.ResumeCompensableOpsTask(
		taskID,
		appendContext,
		taskStepTransfer,
		func(taskCtx *core.TaskContext, data any) core.CompensableTaskStep {
			ctx, _ := data.(*FlowPublishContext)
			return stepSupplier(taskCtx, ctx)
		},
		taskPersistence,
		transactionable,
		core.NewJSONTaskDataSerDeser(&FlowPublishContext{}),
	)
	return &FlowPublishTask{CompensableOpsTask: task}
}

func buildSubject(ctx *FlowPublishContext) map[string]any {
	configNames := make([]string, 0, len(ctx.FlowConfigs))
	for _, fc := range ctx.FlowConfigs {
		configNames = append(configNames, fc.Name)
	}
	return map[string]any{
		SubjectPublishConfigNames: support.ToKeyMap(configNames),
		SubjectPublishBiz:         ctx.Biz,
		SubjectTaskType:           support.TaskTypeToString(ctx.TaskType),
	}
}

func buildDescription(publishReason string) core.Description {
	return core.NewDescription("统一流量编排发布流程", publishReason)
}

// FlowPublishTaskBuilder 统一流量编排任务构建器
type FlowPublishTaskBuilder struct {
	ops.BaseOpsTaskBuilder
	transactionable      stateful.Transactionable
	taskPersistence      stateful.TaskPersistence
	taskDefConfigService *arrange.TaskDefConfigService
}

// NewFlowPublishTaskBuilder 创建任务构建器
func NewFlowPublishTaskBuilder(
	transactionable stateful.Transactionable,
	taskPersistence stateful.TaskPersistence,
	taskDefConfigService *arrange.TaskDefConfigService,
) *FlowPublishTaskBuilder {
	return &FlowPublishTaskBuilder{
		BaseOpsTaskBuilder: ops.BaseOpsTaskBuilder{
			Name: PublishTaskName,
			Type: PublishTaskType,
		},
		transactionable:      transactionable,
		taskPersistence:      taskPersistence,
		taskDefConfigService: taskDefConfigService,
	}
}

// Register 注册任务类型
func (b *FlowPublishTaskBuilder) Register() {
	def := ops.NewOpsTaskDefinition(b)
	ops.Register(def)
}

// Destroy 取消注册
func (b *FlowPublishTaskBuilder) Destroy() {
	ops.Unregister(PublishTaskName, PublishTaskType)
}

// Build 构建新的流量发布任务
func (b *FlowPublishTaskBuilder) Build(ctx *FlowPublishContext) *FlowPublishTask {
	return NewFlowPublishTask(ctx, b.taskPersistence, b.transactionable, b.buildCompensableTaskStep)
}

// ResumeBuild 实现 OpsTaskBuilder 接口
func (b *FlowPublishTaskBuilder) ResumeBuild(taskID string) core.Task {
	return b.internalResumeBuild(taskID, nil, nil)
}

// ResumeBuildWithStepConfig 实现 OpsTaskBuilder 接口
func (b *FlowPublishTaskBuilder) ResumeBuildWithStepConfig(taskID, stepName, key string, value any) core.Task {
	t := NewTransferFromStepConfig(stepName, key, value)
	return b.internalResumeBuild(taskID, nil, t)
}

// ResumeBuildWithStepConfigs 实现 OpsTaskBuilder 接口
func (b *FlowPublishTaskBuilder) ResumeBuildWithStepConfigs(taskID string, stepConfigs []core.StepExecuteConfig) core.Task {
	t := NewTransferFromConfigs(stepConfigs)
	return b.internalResumeBuild(taskID, nil, t)
}

// ResumeBuildWithStepContext 实现 OpsTaskBuilder 接口
func (b *FlowPublishTaskBuilder) ResumeBuildWithStepContext(taskID, stepName string, stepContext map[string]any) core.Task {
	t := NewTransferFromMap(map[string]map[string]any{stepName: stepContext})
	return b.internalResumeBuild(taskID, nil, t)
}

// ResumeBuildWithTaskConfig 实现 OpsTaskBuilder 接口
func (b *FlowPublishTaskBuilder) ResumeBuildWithTaskConfig(taskID, key string, value any) core.Task {
	return b.internalResumeBuild(taskID, map[string]any{key: value}, nil)
}

// ResumeBuildWithTaskContext 实现 OpsTaskBuilder 接口
func (b *FlowPublishTaskBuilder) ResumeBuildWithTaskContext(taskID string, taskContext map[string]any) core.Task {
	return b.internalResumeBuild(taskID, taskContext, nil)
}

// InternalResumeBuild 实现 OpsTaskBuilder 接口
func (b *FlowPublishTaskBuilder) InternalResumeBuild(taskID string) core.Task {
	return b.internalResumeBuild(taskID, nil, nil)
}

// ResumeFlowPublishBuild 恢复任务并返回 FlowPublishTask
func (b *FlowPublishTaskBuilder) ResumeFlowPublishBuild(taskID string) *FlowPublishTask {
	return b.internalResumeFlowPublishBuild(taskID, nil, nil)
}

func (b *FlowPublishTaskBuilder) internalResumeBuild(taskID string, appendContext map[string]any, stepTransfer core.TaskStepTransfer) core.Task {
	if appendContext == nil {
		appendContext = map[string]any{}
	}
	if stepTransfer == nil {
		stepTransfer = transfer.NewDefaultTaskStepTransfer()
	}
	return ResumeFlowPublishTask(taskID, appendContext, stepTransfer, b.taskPersistence, b.transactionable, b.buildCompensableTaskStep)
}

func (b *FlowPublishTaskBuilder) internalResumeFlowPublishBuild(taskID string, appendContext map[string]any, stepTransfer core.TaskStepTransfer) *FlowPublishTask {
	if appendContext == nil {
		appendContext = map[string]any{}
	}
	if stepTransfer == nil {
		stepTransfer = transfer.NewDefaultTaskStepTransfer()
	}
	return ResumeFlowPublishTask(taskID, appendContext, stepTransfer, b.taskPersistence, b.transactionable, b.buildCompensableTaskStep)
}

func (b *FlowPublishTaskBuilder) buildCompensableTaskStep(taskCtx *core.TaskContext, publishCtx *FlowPublishContext) core.CompensableTaskStep {
	deps := &FlowTaskDependencies{
		PublishContext: publishCtx,
	}
	stageDef := b.getTaskStageDefinition(publishCtx)
	return arrange.BuildTaskStep(taskCtx, deps, stageDef, b.taskPersistence)
}

func (b *FlowPublishTaskBuilder) getTaskStageDefinition(publishCtx *FlowPublishContext) *arrange.TaskStageDefinition {
	matchCondition := map[string]string{
		MatchKeyPublishTask: PublishTaskName,
		MatchKeyPublishBiz:  publishCtx.Biz,
		MatchKeyPublishEnv:  string(publishCtx.PublishEnv),
	}
	stageDef, ok := b.taskDefConfigService.GetFlowDefinition(matchCondition)
	if !ok {
		panic("找不到合适的发布流程, matchCondition = " + mapToString(matchCondition))
	}
	return stageDef
}

// 辅助函数: 从步骤配置创建 TaskStepTransfer
func NewTransferFromStepConfig(stepName, key string, value any) core.TaskStepTransfer {
	return ops.NewTransferFromStepConfig(stepName, key, value)
}

// 辅助函数: 从配置列表创建 TaskStepTransfer
func NewTransferFromConfigs(configs []core.StepExecuteConfig) core.TaskStepTransfer {
	return ops.NewTransferFromConfigs(configs)
}

// 辅助函数: 从 map 创建 TaskStepTransfer
func NewTransferFromMap(contexts map[string]map[string]any) core.TaskStepTransfer {
	return ops.NewTransferFromMap(contexts)
}

// 辅助函数: map 转字符串 (用于错误消息)
func mapToString(m map[string]string) string {
	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ", "
		}
		result += k + "=" + v
		first = false
	}
	result += "}"
	return result
}
