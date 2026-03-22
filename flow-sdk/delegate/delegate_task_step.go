package delegate

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
)

// DelegateTaskStep 委托代理任务步骤
type DelegateTaskStep struct {
	*flowsdk.AbstractTaskStep
	delegatedTaskStep flowsdk.TaskStep
}

// NewDelegateTaskStep 创建委托代理任务步骤
func NewDelegateTaskStep(name string, taskContext *flowsdk.TaskContext, delegatedTaskStep flowsdk.TaskStep) *DelegateTaskStep {
	taskStep := &DelegateTaskStep{
		AbstractTaskStep:   flowsdk.NewAbstractTaskStep(name, taskContext, true),
		delegatedTaskStep: delegatedTaskStep,
	}
	return taskStep
}

// GetType 获取任务步骤类型
func (s *DelegateTaskStep) GetType() string {
	return s.delegatedTaskStep.GetType()
}

// DoExecute 执行具体逻辑
func (s *DelegateTaskStep) DoExecute() flowsdk.TaskStepResult {
	return s.delegatedTaskStep.Execute()
}

// OnSuccess 成功回调
func (s *DelegateTaskStep) OnSuccess() {
	s.delegatedTaskStep.OnSuccess()
}

// OnFailure 失败回调
func (s *DelegateTaskStep) OnFailure(err error) {
	s.delegatedTaskStep.OnFailure(err)
}

// OnInterrupt 中断回调
func (s *DelegateTaskStep) OnInterrupt() {
	s.delegatedTaskStep.OnInterrupt()
	s.AbstractTaskStep.OnInterrupt()
}

// Describe 描述任务步骤
func (s *DelegateTaskStep) Describe() flowsdk.TaskStepDescription {
	if delegated, ok := s.delegatedTaskStep.(interface{ Describe() flowsdk.TaskStepDescription }); ok {
		return delegated.Describe()
	}
	return s.AbstractTaskStep.Describe()
}
