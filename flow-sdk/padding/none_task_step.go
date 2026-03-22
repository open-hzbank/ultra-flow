package padding

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
)

// NoneTaskStep 空任务步骤
type NoneTaskStep struct {
	*flowsdk.AbstractTaskStep
}

// NewNoneTaskStep 创建空任务步骤
func NewNoneTaskStep(name string, taskContext *flowsdk.TaskContext) *NoneTaskStep {
	taskStep := &NoneTaskStep{
		AbstractTaskStep: flowsdk.NewAbstractTaskStep(name, taskContext, true),
	}
	return taskStep
}

// GetType 获取任务步骤类型
func (s *NoneTaskStep) GetType() string {
	return "none"
}

// DoExecute 执行具体逻辑
func (s *NoneTaskStep) DoExecute() flowsdk.TaskStepResult {
	return flowsdk.NewTaskStepResult(flowsdk.TaskStepStatusSuccess)
}
