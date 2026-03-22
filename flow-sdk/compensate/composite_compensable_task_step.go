package compensate

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/control"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/delegate"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/padding"
)

// CompositeCompensableTaskStep 并行可补偿任务步骤
type CompositeCompensableTaskStep struct {
	*delegate.DelegateTaskStep
	compensateAwareSteps []CompensateAwareTaskStep
	taskContext         *flowsdk.TaskContext
}

// NewCompositeCompensableTaskStep 创建并行可补偿任务步骤
func NewCompositeCompensableTaskStep(name string, taskContext *flowsdk.TaskContext, compensateAwareSteps []CompensateAwareTaskStep) *CompositeCompensableTaskStep {
	// 构建正常的并行步骤
	normalSteps := make([]flowsdk.TaskStep, len(compensateAwareSteps))
	for i, awareStep := range compensateAwareSteps {
		normalSteps[i] = awareStep.OriginTaskStep()
	}
	compositeStep := control.NewCompositeTaskStep(name, taskContext, normalSteps)

	taskStep := &CompositeCompensableTaskStep{
		DelegateTaskStep:    delegate.NewDelegateTaskStep(name, taskContext, compositeStep),
		compensateAwareSteps: compensateAwareSteps,
		taskContext:         taskContext,
	}

	return taskStep
}

// GetType 获取任务步骤类型
func (s *CompositeCompensableTaskStep) GetType() string {
	return "compositeCompensable"
}

// GenerateCompensableTaskStep 生成可补偿的任务步骤
func (s *CompositeCompensableTaskStep) GenerateCompensableTaskStep() flowsdk.TaskStep {
	executedSteps := s.getExecutedSteps()
	compensateSteps := s.getCompensateSteps()

	if len(executedSteps) == 0 && len(compensateSteps) == 0 {
		return padding.NewNoneTaskStep("emptyCompensate", s.taskContext)
	}

	var compensatingSteps []flowsdk.TaskStep

	// 构建已执行的并行步骤
	if len(executedSteps) > 0 {
		executedCompositeStep := control.NewCompositeTaskStep("executed_"+s.GetName(), s.taskContext, executedSteps)
		compensatingSteps = append(compensatingSteps, executedCompositeStep)
	}

	// 构建补偿的并行步骤
	if len(compensateSteps) > 0 {
		compensateCompositeStep := control.NewCompositeTaskStep("compensate_"+s.GetName(), s.taskContext, compensateSteps)
		compensatingSteps = append(compensatingSteps, compensateCompositeStep)
	}

	// 并行补偿的链路整合采用串行执行
	return control.NewSequentialTaskStep("compensating_"+s.GetName(), s.taskContext, compensatingSteps)
}

// ExecutedTaskStep 获取已执行的步骤
func (s *CompositeCompensableTaskStep) ExecutedTaskStep() flowsdk.TaskStep {
	executedSteps := s.getExecutedSteps()
	if len(executedSteps) == 0 {
		return nil
	}
	return control.NewCompositeTaskStep("executed_"+s.GetName(), s.taskContext, executedSteps)
}

// CompensateTaskStep 获取补偿步骤
func (s *CompositeCompensableTaskStep) CompensateTaskStep() flowsdk.TaskStep {
	compensateSteps := s.getCompensateSteps()
	if len(compensateSteps) == 0 {
		return nil
	}
	return control.NewCompositeTaskStep("compensate_"+s.GetName(), s.taskContext, compensateSteps)
}

// OriginTaskStep 获取原始步骤
func (s *CompositeCompensableTaskStep) OriginTaskStep() flowsdk.TaskStep {
	return s.DelegateTaskStep
}

// getExecutedSteps 获取已执行的步骤
func (s *CompositeCompensableTaskStep) getExecutedSteps() []flowsdk.TaskStep {
	executedSteps := make([]flowsdk.TaskStep, 0)
	for _, awareStep := range s.compensateAwareSteps {
		if step := awareStep.ExecutedTaskStep(); step != nil {
			executedSteps = append(executedSteps, step)
		}
	}
	return executedSteps
}

// getCompensateSteps 获取补偿步骤
func (s *CompositeCompensableTaskStep) getCompensateSteps() []flowsdk.TaskStep {
	compensateSteps := make([]flowsdk.TaskStep, 0)
	for _, awareStep := range s.compensateAwareSteps {
		if step := awareStep.CompensateTaskStep(); step != nil {
			compensateSteps = append(compensateSteps, step)
		}
	}
	return compensateSteps
}
