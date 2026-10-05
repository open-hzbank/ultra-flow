package control

import (
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/delegate"
)

const compensateStepDefaultPrefix = "补偿: "

// CompositeCompensableTaskStep 可根据当前执行状态进行适应性补偿的 复合编排步骤
type CompositeCompensableTaskStep struct {
	*delegate.DelegateTaskStep
	name                 string
	taskCtx              *core.TaskContext
	compensateAwareSteps []core.CompensateAwareTaskStep
}

// NewCompositeCompensableTaskStepFromAware 构建可补偿的复合编排步骤
func NewCompositeCompensableTaskStepFromAware(name string, taskCtx *core.TaskContext, taskSteps []core.CompensateAwareTaskStep) *CompositeCompensableTaskStep {
	normalSteps := make([]core.TaskStep, len(taskSteps))
	for i, ca := range taskSteps {
		normalSteps[i] = ca.OriginTaskStep()
	}
	compositeStep := NewCompositeTaskStep(name, taskCtx, normalSteps)
	return &CompositeCompensableTaskStep{
		DelegateTaskStep:     delegate.NewDelegateTaskStep(compositeStep),
		name:                 name,
		taskCtx:              taskCtx,
		compensateAwareSteps: taskSteps,
	}
}

// BuildCompositeCompensableTaskStep 构建可补偿的复合编排步骤 (taskSteps 需按照: 普通步骤, 补偿步骤 的顺序依次填充)
// 其中设置的补偿步骤会同时运用于已取消状态的补偿步骤和已成功状态的补偿步骤
//
//	原始步骤链路及状态: step1 (SUCCESS) -> step2 (RUNNING) -> step3 (CREATED)
//	则生成的补偿步骤为: step2 (RUNNING) -> compensate_step2 (created) -> compensate_step1 (created)
func BuildCompositeCompensableTaskStep(name string, taskCtx *core.TaskContext, taskSteps []core.TaskStep) *CompositeCompensableTaskStep {
	compensateAwareSteps := BuildCompensateAwareTaskSteps(taskSteps)
	return NewCompositeCompensableTaskStepFromAware(name, taskCtx, compensateAwareSteps)
}

func (s *CompositeCompensableTaskStep) OriginTaskStep() core.TaskStep {
	return s
}

func (s *CompositeCompensableTaskStep) ExecutedTaskStep() (core.TaskStep, bool) {
	executedSteps := GetExecutedSteps(s.compensateAwareSteps)
	if len(executedSteps) == 0 {
		return nil, false
	}
	return NewCompositeTaskStep(s.name, s.taskCtx, executedSteps), true
}

func (s *CompositeCompensableTaskStep) CompensateTaskStep() (core.TaskStep, bool) {
	compensateSteps := GetCompensateSteps(s.compensateAwareSteps)
	if len(compensateSteps) == 0 {
		return nil, false
	}
	return NewCompositeTaskStep(s.name, s.taskCtx, compensateSteps), true
}

func (s *CompositeCompensableTaskStep) GenerateCompensableTaskStep() core.TaskStep {
	// 提取已执行过的普通步骤
	executedSteps := GetExecutedSteps(s.compensateAwareSteps)
	// 根据原有步骤的执行状态生成适应性的补偿步骤
	compensateSteps := GetCompensateSteps(s.compensateAwareSteps)

	executedComposite := NewCompositeTaskStep(s.name, s.taskCtx, executedSteps)
	compensateComposite := NewCompositeTaskStep(compensateStepDefaultPrefix+s.name, s.taskCtx, compensateSteps)

	// 为了使用统一的接口 (CompensateAwareTaskStep) 以便于管理, composite compensable step 的补偿逻辑
	// 采用了由 executedStep + compensateStep 串联起的 sequentialTaskStep
	chain := []core.TaskStep{executedComposite, compensateComposite}
	return NewCompensatingSequentialTaskStep(s.name, s.taskCtx, chain)
}
