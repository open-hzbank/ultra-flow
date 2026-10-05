package arrange

import (
	"hzbank.com.cn/ultra-flow/arrange/view"
	"hzbank.com.cn/ultra-flow/auto"
	"hzbank.com.cn/ultra-flow/core"
	"hzbank.com.cn/ultra-flow/padding"
	"hzbank.com.cn/ultra-flow/stateful"
)

// StepDefinition 步骤的定义
type StepDefinition interface {
	GetType() StepDefinitionType
	GetStage() view.Stage
	// BuildTaskStep 基于 StepDefinition 构建步骤执行实例
	BuildTaskStep(taskPersistence stateful.TaskPersistence, dependencies any, taskCtx *core.TaskContext) core.TaskStep
	// BuildStepView 基于 StepDefinition 构建步骤展示视图
	// isCompensateStage: 当前 definition 是否处于一个补偿阶段
	// isTaskCancelling: 当前任务是否已被取消或正在取消中
	BuildStepView(allStepSnapshots map[string]*stateful.TaskStepSnapshot, isCompensateStage, isTaskCancelling bool) (*view.TaskStepViewResult, bool)
	// IsPending 判断当前 StepDefinition 定义的步骤是否已执行
	IsPending(allStepSnapshots map[string]*stateful.TaskStepSnapshot) bool
}

// StepDefinitionType 步骤定义类型
type StepDefinitionType int

const (
	// StepDefAtomic 原子步骤
	StepDefAtomic StepDefinitionType = iota
	// StepDefNested 嵌套复合步骤
	StepDefNested
)

// AtomicStepDefinition 原子步骤定义
type AtomicStepDefinition struct {
	StageInfo                view.Stage
	Name                     string
	TaskStepBuilder          core.TaskStepBuilder
	// 步骤是否需要持有状态
	Stateful                 bool
	// 步骤是否需要流程引擎持续的状态追踪并自动更新
	AutoUpdate               bool
	AutoUpdateIntervalSecs   int
	AutoUpdateErrorRetry     int
	AutoUpdateMaxTimes       int
	// 步骤是否是整个流程的卡点
	Blockable                bool
}

func (d *AtomicStepDefinition) GetType() StepDefinitionType { return StepDefAtomic }
func (d *AtomicStepDefinition) GetStage() view.Stage        { return d.StageInfo }

func (d *AtomicStepDefinition) BuildTaskStep(taskPersistence stateful.TaskPersistence, dependencies any, taskCtx *core.TaskContext) core.TaskStep {
	var taskStep core.TaskStep
	// 有状态 特性包装
	if d.Stateful {
		taskStep = stateful.WrapWithSupplier(
			func(ctx *core.TaskContext) core.TaskStep {
				return d.TaskStepBuilder.Build(d.Name, ctx, dependencies)
			},
			taskCtx, taskPersistence, d.Name,
		)
	} else {
		taskStep = d.TaskStepBuilder.Build(d.Name, taskCtx, dependencies)
	}

	// 自动更新 特性包装
	if d.AutoUpdate {
		taskStep = auto.Wrap(taskStep, taskPersistence,
			d.AutoUpdateIntervalSecs, d.AutoUpdateErrorRetry, d.AutoUpdateMaxTimes)
	}
	return taskStep
}

func (d *AtomicStepDefinition) BuildStepView(allStepSnapshots map[string]*stateful.TaskStepSnapshot, isCompensateStage, isTaskCancelling bool) (*view.TaskStepViewResult, bool) {
	stepSnapshot := allStepSnapshots[d.Name]

	// 应当丢弃, 无需展示
	if d.shouldDrop(allStepSnapshots, isCompensateStage, isTaskCancelling, stepSnapshot) {
		return nil, false
	}

	// 不能丢弃, 填充展示信息
	if stepSnapshot == nil {
		// 没有执行, PENDING
		return &view.TaskStepViewResult{
			View: &view.AtomicStepView{
				Name:      d.Name,
				Title:     d.TaskStepBuilder.TaskStepTitle(),
				Blockable: d.Blockable,
				Status:    core.StepPending,
			},
		}, true
	}

	// 已执行, 从 snapshot 中恢复

	var gmtCreate, gmtModified *int64
	if !stepSnapshot.GmtCreate.IsZero() {
		t := stepSnapshot.GmtCreate.UnixMilli()
		gmtCreate = &t
	}
	if !stepSnapshot.GmtModified.IsZero() {
		t := stepSnapshot.GmtModified.UnixMilli()
		gmtModified = &t
	}

	var errMsg string
	if stepSnapshot.TaskStepContext != nil {
		if tips, ok := stepSnapshot.TaskStepContext[ops_TaskErrorTips]; ok {
			if s, ok := tips.(string); ok {
				errMsg = s
			}
		}
	}

	return &view.TaskStepViewResult{
		View: &view.AtomicStepView{
			Name:         d.Name,
			Title:        d.TaskStepBuilder.TaskStepTitle(),
			Blockable:    d.Blockable,
			Status:       stepSnapshot.Status,
			Detail:       stepSnapshot.Description.Detail,
			ErrorMessage: errMsg,
			GmtCreate:    gmtCreate,
			GmtModified:  gmtModified,
		},
	}, true
}

const ops_TaskErrorTips = "taskErrorTips"

// shouldDrop 是否不应该展示指定的步骤
func (d *AtomicStepDefinition) shouldDrop(allStepSnapshots map[string]*stateful.TaskStepSnapshot, isCompensateStage, isTaskCancelling bool, stepSnapshot *stateful.TaskStepSnapshot) bool {
	// 如果 step 是用于占位的 NoneTaskStep, 直接无条件丢弃
	if _, ok := d.TaskStepBuilder.(*padding.NoneTaskStepBuilder); ok {
		return true
	}
	// 任务取消时如果 normalStep 未被执行, 则丢弃
	if (stepSnapshot == nil || stepSnapshot.Status == core.StepPending) && isTaskCancelling && !isCompensateStage {
		return true
	}
	return false
}

func (d *AtomicStepDefinition) IsPending(allStepSnapshots map[string]*stateful.TaskStepSnapshot) bool {
	return allStepSnapshots[d.Name] == nil
}

// CompensateAtomicStepDefinition 用于补偿的原子步骤定义
type CompensateAtomicStepDefinition struct {
	AtomicStepDefinition
	// 关联原始的步骤定义
	OriginDefinition *AtomicStepDefinition
}

func (d *CompensateAtomicStepDefinition) BuildStepView(allStepSnapshots map[string]*stateful.TaskStepSnapshot, isCompensateStage, isTaskCancelling bool) (*view.TaskStepViewResult, bool) {
	// 任务取消时如果 normalStep 未实际执行, 则对应的 compensateStep 也需要被丢弃
	if isTaskCancelling && isCompensateStage {
		// 检查 normalStep 的状态是否执行
		originView, originOk := d.OriginDefinition.BuildStepView(allStepSnapshots, false, true)
		if !originOk {
			return nil, false
		}
		if originView.View.GetStatus().NotExecuted() {
			return nil, false
		}
	}
	return d.AtomicStepDefinition.BuildStepView(allStepSnapshots, isCompensateStage, isTaskCancelling)
}

// NestedStepDefinition 嵌套步骤定义
type NestedStepDefinition struct {
	StageInfo        view.Stage
	NestedDefinition *TaskStageDefinition
}

func (d *NestedStepDefinition) GetType() StepDefinitionType { return StepDefNested }
func (d *NestedStepDefinition) GetStage() view.Stage        { return d.StageInfo }

func (d *NestedStepDefinition) BuildTaskStep(taskPersistence stateful.TaskPersistence, dependencies any, taskCtx *core.TaskContext) core.TaskStep {
	return BuildTaskStep(taskCtx, dependencies, d.NestedDefinition, taskPersistence)
}

func (d *NestedStepDefinition) BuildStepView(allStepSnapshots map[string]*stateful.TaskStepSnapshot, isCompensateStage, isTaskCancelling bool) (*view.TaskStepViewResult, bool) {
	var stages []*view.StageView
	current := d.NestedDefinition
	order := 0
	for current != nil {
		var steps []view.TaskStepView
		for _, subDef := range current.Definitions {
			if viewResult, ok := subDef.BuildStepView(allStepSnapshots, isCompensateStage, isTaskCancelling); ok {
				steps = append(steps, viewResult.View)
			}
		}
		stageName := calcStageName(current)
		stages = append(stages, view.NewStageView(order, stageName, steps))
		current = current.NextDef
		order++
	}
	return &view.TaskStepViewResult{
		View: view.NewNestedStepView(stages),
	}, true
}

func (d *NestedStepDefinition) IsPending(allStepSnapshots map[string]*stateful.TaskStepSnapshot) bool {
	// StageDefinition 链上的步骤是串行执行的, 所以只需检查第一个 definition 是否执行即可
	for _, subDef := range d.NestedDefinition.Definitions {
		if !subDef.IsPending(allStepSnapshots) {
			return false
		}
	}
	return true
}

func calcStageName(def *TaskStageDefinition) string {
	if len(def.Definitions) == 0 {
		return ""
	}
	return def.Definitions[0].GetStage().StageName
}
