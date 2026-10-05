// TaskArranges 任务 (阶段、步骤) 编排构建工具
package arrange

import (
	"github.com/open-hzbank/ultra-flow/arrange/view"
	"github.com/open-hzbank/ultra-flow/control"
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/padding"
	"github.com/open-hzbank/ultra-flow/stateful"
)

// TaskStageDefinition 任务的阶段编排定义
// 在用户配置层与该类对标的概念是 Stage: StageConfig
// DepType 为 TaskStepBuilder 用于构建 taskStep 所依赖的数据类型
type TaskStageDefinition struct {
	// 同 stage 下多个并行步骤的定义描述集合
	Definitions []StepDefinition
	// 当阶段内步骤全部执行成功后下一个要执行的阶段
	NextDef *TaskStageDefinition
	// 当任务被取消后, 当前阶段对应的补偿阶段
	compensate *CompensateStageDefinition
	// 是否需要在上一个阶段完成时, 自动执行当前阶段内的步骤
	AutoExecute bool
}

// CompensateStageDefinition 补偿阶段定义
type CompensateStageDefinition struct {
	TaskStageDefinition
}

// BuildCompensableDefinition 执行步骤 & 补偿步骤 捆绑定义
func BuildCompensableDefinition(autoExecute bool, definitions []StepDefinition) *TaskStageDefinition {
	const defTypesCount = 2
	if len(definitions)%defTypesCount != 0 {
		panic("definition 数量不符合要求, 应该为 2 的倍数")
	}
	var normalDefs, compensateDefs []StepDefinition
	for i := 0; i < len(definitions)/defTypesCount; i++ {
		normalDefs = append(normalDefs, definitions[defTypesCount*i])
		compensateDefs = append(compensateDefs, definitions[defTypesCount*i+1])
	}
	return &TaskStageDefinition{
		Definitions: normalDefs,
		compensate:  &CompensateStageDefinition{TaskStageDefinition: TaskStageDefinition{Definitions: compensateDefs}},
		AutoExecute: autoExecute,
	}
}

func (d *TaskStageDefinition) Next() *TaskStageDefinition             { return d.NextDef }
func (d *TaskStageDefinition) Compensate() *CompensateStageDefinition { return d.compensate }

// SetNext 设置下一阶段, 同时建立补偿链
func (d *TaskStageDefinition) SetNext(next *TaskStageDefinition) {
	d.NextDef = next
	next.compensate.TaskStageDefinition.NextDef = &d.compensate.TaskStageDefinition
}

func (c *CompensateStageDefinition) IsCompensateStage() bool { return true }

// BuildTaskStep 基于阶段定义构建步骤实例
func BuildTaskStep(taskCtx *core.TaskContext, dependencies any, firstStageDef *TaskStageDefinition, taskPersistence stateful.TaskPersistence) core.CompensableTaskStep {
	var taskSteps []core.TaskStep
	var autoExecuteSignals []bool

	current := firstStageDef
	for current != nil {
		taskSteps = append(taskSteps, doBuildTaskStep(current, dependencies, taskCtx, taskPersistence))
		taskSteps = append(taskSteps, doBuildTaskStep(&current.compensate.TaskStageDefinition, dependencies, taskCtx, taskPersistence))
		autoExecuteSignals = append(autoExecuteSignals, current.AutoExecute)
		current = current.NextDef
	}

	if len(taskSteps) == 2 {
		// 只有一对 normal/compensate, 用复合步骤编排
		awareSteps := control.BuildCompensateAwareTaskSteps(taskSteps)
		return control.NewCompositeCompensableTaskStepFromAware(
			"CompositeCompensableStep", taskCtx, awareSteps)
	}
	// 有多对 normal/compensate, 用串行步骤编排
	return control.BuildSequentialCompensableTaskStep(
		"SequentialCompensableStep", taskCtx, autoExecuteSignals, taskSteps)
}

func doBuildTaskStep(stageDef *TaskStageDefinition, dependencies any, taskCtx *core.TaskContext, taskPersistence stateful.TaskPersistence) core.TaskStep {
	definitions := stageDef.Definitions
	if len(definitions) == 0 {
		return padding.NewNoneTaskStep("PaddingStep", taskCtx)
	}

	var taskSteps []core.TaskStep
	for _, def := range definitions {
		taskStep := def.BuildTaskStep(taskPersistence, dependencies, taskCtx)
		taskSteps = append(taskSteps, taskStep)
	}

	if len(taskSteps) == 1 {
		return taskSteps[0]
	}
	return buildCompositeTaskStep(taskSteps, taskCtx)
}

func buildCompositeTaskStep(taskSteps []core.TaskStep, taskCtx *core.TaskContext) core.TaskStep {
	allCA := true
	for _, step := range taskSteps {
		if _, ok := step.(core.CompensateAwareTaskStep); !ok {
			allCA = false
			break
		}
	}
	if allCA {
		var caSteps []core.CompensateAwareTaskStep
		for _, step := range taskSteps {
			caSteps = append(caSteps, step.(core.CompensateAwareTaskStep))
		}
		return control.NewCompositeCompensableTaskStepFromAware(
			"CompositeCompensableStep", taskCtx, caSteps)
	}
	return control.NewCompositeTaskStep("CompositeStep", taskCtx, taskSteps)
}

// StageNavigatorAdapter 适配 TaskStageDefinition 到 view.StageNavigator
type StageNavigatorAdapter struct {
	Def *TaskStageDefinition
}

func (a *StageNavigatorAdapter) GetDefinitions() []view.StepViewBuilder {
	var result []view.StepViewBuilder
	for _, def := range a.Def.Definitions {
		result = append(result, &stepViewBuilderAdapter{def: def})
	}
	return result
}

func (a *StageNavigatorAdapter) Next() view.StageNavigator {
	if a.Def.NextDef == nil {
		return nil
	}
	return &StageNavigatorAdapter{Def: a.Def.NextDef}
}

func (a *StageNavigatorAdapter) Compensate() view.StageNavigator {
	if a.Def.compensate == nil {
		return nil
	}
	return &CompensateStageNavigatorAdapter{Def: &a.Def.compensate.TaskStageDefinition}
}

func (a *StageNavigatorAdapter) IsCompensateStageDef() bool { return false }

// CompensateStageNavigatorAdapter 补偿阶段的导航适配器
type CompensateStageNavigatorAdapter struct {
	Def *TaskStageDefinition
}

func (a *CompensateStageNavigatorAdapter) GetDefinitions() []view.StepViewBuilder {
	var result []view.StepViewBuilder
	for _, def := range a.Def.Definitions {
		result = append(result, &stepViewBuilderAdapter{def: def})
	}
	return result
}

func (a *CompensateStageNavigatorAdapter) Next() view.StageNavigator {
	if a.Def.NextDef == nil {
		return nil
	}
	if a.Def.NextDef.compensate != nil {
		return &CompensateStageNavigatorAdapter{Def: &a.Def.NextDef.compensate.TaskStageDefinition}
	}
	return &StageNavigatorAdapter{Def: a.Def.NextDef}
}

func (a *CompensateStageNavigatorAdapter) Compensate() view.StageNavigator {
	return nil
}

func (a *CompensateStageNavigatorAdapter) IsCompensateStageDef() bool { return true }

// stepViewBuilderAdapter 适配 StepDefinition 到 view.StepViewBuilder
type stepViewBuilderAdapter struct {
	def StepDefinition
}

func (a *stepViewBuilderAdapter) BuildStepView(allStepSnapshots map[string]*stateful.TaskStepSnapshot, isCompensateStage, isTaskCancelling bool) (*view.TaskStepViewResult, bool) {
	return a.def.BuildStepView(allStepSnapshots, isCompensateStage, isTaskCancelling)
}

func (a *stepViewBuilderAdapter) IsPending(allStepSnapshots map[string]*stateful.TaskStepSnapshot) bool {
	return a.def.IsPending(allStepSnapshots)
}

func (a *stepViewBuilderAdapter) GetStage() view.Stage {
	return a.def.GetStage()
}

func (a *stepViewBuilderAdapter) IsCompensateStageDef() bool {
	_, ok := a.def.(*CompensateAtomicStepDefinition)
	return ok
}
