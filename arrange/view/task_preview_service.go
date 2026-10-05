package view

import (
	"sort"
	"time"

	"hzbank.com.cn/ultra-flow/auto"
	"hzbank.com.cn/ultra-flow/control"
	"hzbank.com.cn/ultra-flow/core"
	"hzbank.com.cn/ultra-flow/ops"
	"hzbank.com.cn/ultra-flow/stateful"
)

// TaskPreviewService 任务预览服务
type TaskPreviewService struct {
	taskRepo stateful.TaskSnapshotRepository
	stepRepo stateful.TaskStepSnapshotRepository
}

func NewTaskPreviewService(taskRepo stateful.TaskSnapshotRepository, stepRepo stateful.TaskStepSnapshotRepository) *TaskPreviewService {
	return &TaskPreviewService{taskRepo: taskRepo, stepRepo: stepRepo}
}

// StepViewBuilder 步骤视图构建器 (由 StepDefinition 实现)
type StepViewBuilder interface {
	BuildStepView(allStepSnapshots map[string]*stateful.TaskStepSnapshot, isCompensateStage, isTaskCancelling bool) (*TaskStepViewResult, bool)
	IsPending(allStepSnapshots map[string]*stateful.TaskStepSnapshot) bool
	GetStage() Stage
	IsCompensateStageDef() bool
}

// TaskStepViewResult 步骤视图构建结果
type TaskStepViewResult struct {
	View TaskStepView
}

// Stage 阶段标识
type Stage struct {
	// 阶段的顺序
	StageOrder int
	// 阶段名称
	StageName string
}

// TaskPreview 查询 taskId 对应的任务概要, 构建任务预览
func (s *TaskPreviewService) TaskPreview(firstStageDef StageNavigator, taskDataSerDeser core.TaskDataSerDeser, taskID string) *TaskPreview {
	taskSnapshot := s.taskRepo.Get(taskID)

	var gmtCreate, gmtModified *int64
	if !taskSnapshot.GmtCreate.IsZero() {
		t := taskSnapshot.GmtCreate.UnixMilli()
		gmtCreate = &t
	}
	if !taskSnapshot.GmtModified.IsZero() {
		t := taskSnapshot.GmtModified.UnixMilli()
		gmtModified = &t
	}

	taskView := &TaskView{
		TaskID:      taskID,
		Creator:     NewUserViewFromEmpID(taskSnapshot.Creator),
		Title:       taskSnapshot.Description.Title,
		Reason:      taskSnapshot.Description.Reason,
		PublishEnv:  taskSnapshot.PublishEnv,
		Status:      taskSnapshot.Status,
		GmtCreate:   gmtCreate,
		GmtModified: gmtModified,
		Detail:      ops.GetGlobalBusDataFromContext(taskSnapshot.Context, taskDataSerDeser),
	}

	taskSteps := s.stepRepo.GetTaskSteps(taskID)
	allStepSnapshots := make(map[string]*stateful.TaskStepSnapshot)
	for _, step := range taskSteps {
		// 过滤掉非业务自动执行步骤
		if step.Type == auto.AutoExecuteTaskStepType {
			continue
		}
		allStepSnapshots[step.Name] = step
	}

	isTaskCancelling := taskSnapshot.Status.IsInCancel()
	taskStepViews := s.buildTaskStepViews(allStepSnapshots, firstStageDef, isTaskCancelling)

	var stagedViews []*StageView
	for stage, views := range taskStepViews {
		stagedViews = append(stagedViews, NewStageView(stage.StageOrder, stage.StageName, views))
	}
	sort.Slice(stagedViews, func(i, j int) bool {
		return stagedViews[i].StageOrder < stagedViews[j].StageOrder
	})

	return NewTaskPreview(taskView, stagedViews)
}

// StageNavigator 阶段导航接口
type StageNavigator interface {
	GetDefinitions() []StepViewBuilder
	Next() StageNavigator
	Compensate() StageNavigator
	IsCompensateStageDef() bool
}

func (s *TaskPreviewService) buildTaskStepViews(
	allStepSnapshots map[string]*stateful.TaskStepSnapshot,
	firstStageDef StageNavigator,
	isTaskCancelling bool,
) map[Stage][]TaskStepView {
	result := make(map[Stage][]TaskStepView)

	current := firstStageDef
	for current != nil {
		defs := current.GetDefinitions()
		if len(defs) == 0 {
			current = current.Next()
			continue
		}

		isCompensateStage := current.IsCompensateStageDef()
		var stageStatuses []core.TaskStepStatus

		// 收集 taskStep preview 信息, 以及 taskStep 的状态
		for _, def := range defs {
			if viewResult, ok := def.BuildStepView(allStepSnapshots, isCompensateStage, isTaskCancelling); ok {
				stage := def.GetStage()
				result[stage] = append(result[stage], viewResult.View)
				stageStatuses = append(stageStatuses, viewResult.View.GetStatus())
			}
		}

		// 根据当前状态计算下一个要渲染的步骤
		compositeStatus := control.CalcCompositeStatus(stageStatuses)
		current = s.calcNextStageDef(allStepSnapshots, compositeStatus, current, isTaskCancelling)
	}

	return result
}

func (s *TaskPreviewService) calcNextStageDef(
	allStepSnapshots map[string]*stateful.TaskStepSnapshot,
	stageStatus core.TaskStepStatus,
	current StageNavigator,
	isTaskCancelling bool,
) StageNavigator {
	if stageStatus.NotStarted() || stageStatus.Skipped() {
		return current.Next()
	}
	if current.IsCompensateStageDef() {
		return current.Next()
	}
	if stageStatus.IsCanceled() {
		return current.Compensate()
	}
	if isTaskCancelling {
		if stageStatus.IsActive() {
			return current.Compensate()
		}
		if s.isNextStepPending(current, allStepSnapshots) && stageStatus.NormalEnded() {
			return current.Compensate()
		}
		return current.Next()
	}
	return current.Next()
}

func (s *TaskPreviewService) isNextStepPending(current StageNavigator, allStepSnapshots map[string]*stateful.TaskStepSnapshot) bool {
	next := current.Next()
	if next == nil {
		return false
	}
	for _, def := range next.GetDefinitions() {
		if !def.IsPending(allStepSnapshots) {
			return false
		}
	}
	return true
}

// TimeToMillisPtr 辅助函数
func TimeToMillisPtr(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	ms := t.UnixMilli()
	return &ms
}
