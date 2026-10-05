package compensate

import (
	"github.com/open-hzbank/ultra-flow/core"
)

const isCancelKey = "isCancel"

// ExecuteOrCancelTaskStep 用于复杂的取消流程, 用于取消当前节点, 并执行复杂的取消流程步骤, 然后把整个任务变为取消
// 仅会取消当前运行中的节点, 已完结节点不会取消
type ExecuteOrCancelTaskStep struct {
	state       *core.StepState
	executeStep core.TaskStep
	cancelStep  core.TaskStep
}

func NewExecuteOrCancelTaskStep(name string, taskCtx *core.TaskContext, executeStep, cancelStep core.TaskStep) *ExecuteOrCancelTaskStep {
	s := &ExecuteOrCancelTaskStep{
		executeStep: executeStep,
		cancelStep:  cancelStep,
	}
	s.state = core.NewStepStateWithTransfer(name, taskCtx, s, false)
	if transfer := taskCtx.GetTaskStepTransfer(); transfer != nil {
		transfer.Transfer(s)
	}
	return s
}

func (s *ExecuteOrCancelTaskStep) GetName() string     { return s.state.Name }
func (s *ExecuteOrCancelTaskStep) GetType() string     { return "executeOrCancel" }
func (s *ExecuteOrCancelTaskStep) OnSuccess()          {}
func (s *ExecuteOrCancelTaskStep) OnFailure(err error) {}

func (s *ExecuteOrCancelTaskStep) OnInterrupt() {
	s.state.DefaultInterrupt()
}

func (s *ExecuteOrCancelTaskStep) Execute() core.TaskStepResult {
	return s.state.ExecuteStep(s, s.doExecute)
}

func (s *ExecuteOrCancelTaskStep) doExecute() core.TaskStepResult {
	if s.isCancel() {
		type statusAware interface{ GetStatus() core.TaskStepStatus }
		if sa, ok := s.executeStep.(statusAware); ok {
			if sa.GetStatus().IsExecuting() {
				s.executeStep.OnInterrupt()
			}
		}
		stepResult := s.cancelStep.Execute()
		// 取消任务成功, 整个流程才算取消
		if stepResult.Status == core.StepSuccess {
			return core.NewStepResult(core.StepInterrupted)
		}
		return stepResult
	}
	return s.executeStep.Execute()
}

func (s *ExecuteOrCancelTaskStep) isCancel() bool {
	if v := s.state.TaskCtx.Get(isCancelKey); v != nil {
		if str, ok := v.(string); ok && str == "true" {
			return true
		}
	}
	if s.state.StepCtx != nil {
		if v := s.state.StepCtx.Get(isCancelKey); v != nil {
			if str, ok := v.(string); ok && str == "true" {
				return true
			}
		}
	}
	return false
}

func (s *ExecuteOrCancelTaskStep) Describe() core.StepDescription {
	return core.NewStepDescription("分支判断: 执行步骤或取消步骤", nil)
}

func (s *ExecuteOrCancelTaskStep) GetStatus() core.TaskStepStatus {
	return s.state.GetStatus()
}

func (s *ExecuteOrCancelTaskStep) SetStatus(st core.TaskStepStatus) {
	s.state.SetStatus(st)
}

func (s *ExecuteOrCancelTaskStep) GetTaskStepContext() *core.TaskStepContext {
	return s.state.StepCtx
}

func (s *ExecuteOrCancelTaskStep) GetTaskContext() *core.TaskContext {
	return s.state.TaskCtx
}
