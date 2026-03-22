package control

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
)

// SequentialTaskStep 串行任务步骤
type SequentialTaskStep struct {
	*flowsdk.AbstractTaskStep
	taskSteps []flowsdk.TaskStep
}

// NewSequentialTaskStep 创建串行任务步骤
func NewSequentialTaskStep(name string, taskContext *flowsdk.TaskContext, taskSteps []flowsdk.TaskStep) *SequentialTaskStep {
	taskStep := &SequentialTaskStep{
		AbstractTaskStep: flowsdk.NewAbstractTaskStep(name, taskContext, true),
		taskSteps:        taskSteps,
	}
	return taskStep
}

// GetType 获取任务步骤类型
func (s *SequentialTaskStep) GetType() string {
	return "sequential"
}

// DoExecute 执行具体逻辑
func (s *SequentialTaskStep) DoExecute() flowsdk.TaskStepResult {
	for _, taskStep := range s.taskSteps {
		result := taskStep.Execute()
		switch result.Status {
		case flowsdk.TaskStepStatusFailure, flowsdk.TaskStepStatusInterrupted:
			return result
		case flowsdk.TaskStepStatusRunning:
			return flowsdk.NewTaskStepResult(flowsdk.TaskStepStatusRunning)
		}
	}
	return flowsdk.NewTaskStepResult(flowsdk.TaskStepStatusSuccess)
}

// OnSuccess 成功回调
func (s *SequentialTaskStep) OnSuccess() {
	for _, taskStep := range s.taskSteps {
		taskStep.OnSuccess()
	}
}

// OnFailure 失败回调
func (s *SequentialTaskStep) OnFailure(err error) {
	for _, taskStep := range s.taskSteps {
		taskStep.OnFailure(err)
	}
}

// OnInterrupt 中断回调
func (s *SequentialTaskStep) OnInterrupt() {
	for _, taskStep := range s.taskSteps {
		taskStep.OnInterrupt()
	}
	s.AbstractTaskStep.OnInterrupt()
}
