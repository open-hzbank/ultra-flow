package compensate

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/control"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/delegate"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/padding"
)

// SequentialCompensableTaskStep 串行可补偿任务步骤
type SequentialCompensableTaskStep struct {
	*delegate.DelegateTaskStep
	compensateAwareSteps []CompensateAwareTaskStep
	taskContext         *flowsdk.TaskContext
}

// NewSequentialCompensableTaskStep 创建串行可补偿任务步骤
func NewSequentialCompensableTaskStep(name string, taskContext *flowsdk.TaskContext, compensateAwareSteps []CompensateAwareTaskStep) *SequentialCompensableTaskStep {
	// 构建正常的串行步骤
	normalSteps := make([]flowsdk.TaskStep, len(compensateAwareSteps))
	for i, awareStep := range compensateAwareSteps {
		normalSteps[i] = awareStep.OriginTaskStep()
	}
	sequentialStep := control.NewSequentialTaskStep(name, taskContext, normalSteps)

	taskStep := &SequentialCompensableTaskStep{
		DelegateTaskStep:    delegate.NewDelegateTaskStep(name, taskContext, sequentialStep),
		compensateAwareSteps: compensateAwareSteps,
		taskContext:         taskContext,
	}

	return taskStep
}

// GetType 获取任务步骤类型
func (s *SequentialCompensableTaskStep) GetType() string {
	return "sequentialCompensable"
}

// GenerateCompensableTaskStep 生成可补偿的任务步骤
func (s *SequentialCompensableTaskStep) GenerateCompensableTaskStep() flowsdk.TaskStep {
	executedSteps := s.getExecutedSteps()
	compensateSteps := s.getCompensateSteps()

	// 构建补偿步骤链
	compensatingSteps := make([]flowsdk.TaskStep, 0, len(executedSteps)+len(compensateSteps))
	compensatingSteps = append(compensatingSteps, executedSteps...)
	
	// 补偿步骤需要反向
	for i := len(compensateSteps) - 1; i >= 0; i-- {
		compensatingSteps = append(compensatingSteps, compensateSteps[i])
	}

	if len(compensatingSteps) == 0 {
		return padding.NewNoneTaskStep("emptyCompensate", s.taskContext)
	}

	return control.NewSequentialTaskStep("compensating_"+s.GetName(), s.taskContext, compensatingSteps)
}

// ExecutedTaskStep 获取已执行的步骤
func (s *SequentialCompensableTaskStep) ExecutedTaskStep() flowsdk.TaskStep {
	executedSteps := s.getExecutedSteps()
	if len(executedSteps) == 0 {
		return nil
	}
	return control.NewSequentialTaskStep("executed_"+s.GetName(), s.taskContext, executedSteps)
}

// CompensateTaskStep 获取补偿步骤
func (s *SequentialCompensableTaskStep) CompensateTaskStep() flowsdk.TaskStep {
	compensateSteps := s.getCompensateSteps()
	if len(compensateSteps) == 0 {
		return nil
	}
	
	// 补偿步骤需要反向
	reversedSteps := make([]flowsdk.TaskStep, len(compensateSteps))
	for i := len(compensateSteps) - 1; i >= 0; i-- {
		reversedSteps[len(compensateSteps)-1-i] = compensateSteps[i]
	}

	return control.NewSequentialTaskStep("compensate_"+s.GetName(), s.taskContext, reversedSteps)
}

// OriginTaskStep 获取原始步骤
func (s *SequentialCompensableTaskStep) OriginTaskStep() flowsdk.TaskStep {
	return s.DelegateTaskStep
}

// getExecutedSteps 获取已执行的步骤
func (s *SequentialCompensableTaskStep) getExecutedSteps() []flowsdk.TaskStep {
	executedSteps := make([]flowsdk.TaskStep, 0)
	for _, awareStep := range s.compensateAwareSteps {
		if step := awareStep.ExecutedTaskStep(); step != nil {
			executedSteps = append(executedSteps, step)
		}
	}
	return executedSteps
}

// getCompensateSteps 获取补偿步骤
func (s *SequentialCompensableTaskStep) getCompensateSteps() []flowsdk.TaskStep {
	compensateSteps := make([]flowsdk.TaskStep, 0)
	for _, awareStep := range s.compensateAwareSteps {
		if step := awareStep.CompensateTaskStep(); step != nil {
			compensateSteps = append(compensateSteps, step)
		}
	}
	return compensateSteps
}
