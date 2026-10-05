package padding

import (
	"fmt"
	"hzbank.com.cn/ultra-flow/core"
)

const FixedStatusStepName = "固定状态步骤"

// FixedStatusTaskStep 返回固定 status 占位的步骤 (可用于单元测试组合控制各步骤的状态)
type FixedStatusTaskStep struct {
	state  *core.StepState
	status core.TaskStepStatus
}

func NewFixedStatusTaskStep(name string, taskCtx *core.TaskContext, fixedStatus core.TaskStepStatus) *FixedStatusTaskStep {
	s := &FixedStatusTaskStep{status: fixedStatus}
	s.state = core.NewStepStateWithTransfer(name, taskCtx, s, false)
	return s
}

func (s *FixedStatusTaskStep) GetName() string    { return s.state.Name }
func (s *FixedStatusTaskStep) GetType() string    { return "fixedStatus" }
func (s *FixedStatusTaskStep) OnSuccess()         {}
func (s *FixedStatusTaskStep) OnFailure(err error) {}
func (s *FixedStatusTaskStep) OnInterrupt()       {}

func (s *FixedStatusTaskStep) Execute() core.TaskStepResult {
	return s.state.ExecuteStep(s, func() core.TaskStepResult {
		return core.NewStepResult(s.status)
	})
}

func (s *FixedStatusTaskStep) GetStatus() core.TaskStepStatus {
	return s.state.GetStatus()
}

func (s *FixedStatusTaskStep) SetStatus(st core.TaskStepStatus) {
	s.state.SetStatus(st)
}

func (s *FixedStatusTaskStep) GetTaskStepContext() *core.TaskStepContext {
	return s.state.StepCtx
}

func (s *FixedStatusTaskStep) GetTaskContext() *core.TaskContext {
	return s.state.TaskCtx
}

func (s *FixedStatusTaskStep) Describe() core.StepDescription {
	return core.NewStepDescription(fmt.Sprintf("%s: %s", FixedStatusStepName, s.status), nil)
}

// FixedStatusAware 固定状态感知接口
type FixedStatusAware interface {
	SetFixedStepStatus(statusByStepName map[string]core.TaskStepStatus)
	GetFixedStepStatus(stepName string) core.TaskStepStatus
}

// FixedStatusTaskBuilder FixedStatusTaskStep 的构建器
type FixedStatusTaskBuilder struct{}

func (b *FixedStatusTaskBuilder) Build(name string, taskCtx *core.TaskContext, deps any) core.TaskStep {
	aware := deps.(FixedStatusAware)
	return NewFixedStatusTaskStep(name, taskCtx, aware.GetFixedStepStatus(name))
}

func (b *FixedStatusTaskBuilder) TaskStepTitle() string {
	return FixedStatusStepName
}
