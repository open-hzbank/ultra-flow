package control

import (
	"github.com/open-hzbank/ultra-flow/core"
)

// SelectorTaskStep 选择器步骤: 选择一个子步骤执行
type SelectorTaskStep struct {
	state *core.StepState
	// selectFunc 注意不要依赖变量，否则流程回放时路径会不一致
	selectFunc func() core.TaskStep
}

func NewSelectorTaskStep(name string, taskCtx *core.TaskContext, selectFunc func() core.TaskStep) *SelectorTaskStep {
	s := &SelectorTaskStep{selectFunc: selectFunc}
	s.state = core.NewStepStateWithTransfer(name, taskCtx, s, false)
	return s
}

func (s *SelectorTaskStep) GetName() string                           { return s.state.Name }
func (s *SelectorTaskStep) GetType() string                           { return "select" }
func (s *SelectorTaskStep) GetTaskStepContext() *core.TaskStepContext { return s.state.StepCtx }
func (s *SelectorTaskStep) GetTaskContext() *core.TaskContext         { return s.state.TaskCtx }

func (s *SelectorTaskStep) Execute() core.TaskStepResult {
	return s.state.ExecuteStep(s, func() core.TaskStepResult {
		step := s.selectFunc()
		result := step.Execute()
		switch result.Status {
		case core.StepPending, core.StepRunning:
			return core.NewStepResult(core.StepRunning)
		case core.StepSuccess:
			return core.NewStepResult(core.StepSuccess)
		case core.StepFailure:
			msg := ""
			if result.Error != nil {
				msg = result.Error.Error()
			}
			return core.NewStepResultWithMessage(core.StepFailure, msg)
		case core.StepInterrupted:
			return core.NewStepResult(core.StepInterrupted)
		default:
			return core.NewStepResult(core.StepFailure)
		}
	})
}

func (s *SelectorTaskStep) OnSuccess() {
	s.selectFunc().OnSuccess()
}

func (s *SelectorTaskStep) OnFailure(err error) {
	s.selectFunc().OnFailure(err)
}

// OnInterrupt 中断回调由下游自身来保证
func (s *SelectorTaskStep) OnInterrupt() {
	s.selectFunc().OnInterrupt()
}

func (s *SelectorTaskStep) Describe() core.StepDescription {
	return core.NewStepDescription("选择器步骤", nil)
}
