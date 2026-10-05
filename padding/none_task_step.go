package padding

import (
	"github.com/open-hzbank/ultra-flow/core"
)

const NoneStepName = "占位空步骤"

// NoneTaskStep 占位空步骤, 执行时直接返回 SUCCESS
type NoneTaskStep struct {
	state *core.StepState
}

func NewNoneTaskStep(name string, taskCtx *core.TaskContext) *NoneTaskStep {
	s := &NoneTaskStep{}
	s.state = core.NewStepStateWithTransfer(name, taskCtx, s, false)
	return s
}

func (s *NoneTaskStep) GetName() string     { return s.state.Name }
func (s *NoneTaskStep) GetType() string     { return "none" }
func (s *NoneTaskStep) OnSuccess()          {}
func (s *NoneTaskStep) OnFailure(err error) {}
func (s *NoneTaskStep) OnInterrupt()        {}

func (s *NoneTaskStep) Execute() core.TaskStepResult {
	return s.state.ExecuteStep(s, func() core.TaskStepResult {
		return core.NewStepResult(core.StepSuccess)
	})
}

func (s *NoneTaskStep) GetStatus() core.TaskStepStatus {
	return s.state.GetStatus()
}

func (s *NoneTaskStep) SetStatus(st core.TaskStepStatus) {
	s.state.SetStatus(st)
}

func (s *NoneTaskStep) GetTaskStepContext() *core.TaskStepContext {
	return s.state.StepCtx
}

func (s *NoneTaskStep) GetTaskContext() *core.TaskContext {
	return s.state.TaskCtx
}

func (s *NoneTaskStep) Describe() core.StepDescription {
	return core.NewStepDescription(NoneStepName, nil)
}

// NoneTaskStepBuilder NoneTaskStep 的构建器
type NoneTaskStepBuilder struct{}

func (b *NoneTaskStepBuilder) Build(name string, taskCtx *core.TaskContext, deps any) core.TaskStep {
	return NewNoneTaskStep(name, taskCtx)
}

func (b *NoneTaskStepBuilder) TaskStepTitle() string {
	return NoneStepName
}
