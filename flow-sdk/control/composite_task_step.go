package control

import (
	"sync"

	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
)

// CompositeTaskStep 并行任务步骤
type CompositeTaskStep struct {
	*flowsdk.AbstractTaskStep
	taskSteps []flowsdk.TaskStep
}

// NewCompositeTaskStep 创建并行任务步骤
func NewCompositeTaskStep(name string, taskContext *flowsdk.TaskContext, taskSteps []flowsdk.TaskStep) *CompositeTaskStep {
	taskStep := &CompositeTaskStep{
		AbstractTaskStep: flowsdk.NewAbstractTaskStep(name, taskContext, true),
		taskSteps:        taskSteps,
	}
	return taskStep
}

// GetType 获取任务步骤类型
func (s *CompositeTaskStep) GetType() string {
	return "composite"
}

// DoExecute 执行具体逻辑
func (s *CompositeTaskStep) DoExecute() flowsdk.TaskStepResult {
	if len(s.taskSteps) == 0 {
		return flowsdk.NewTaskStepResult(flowsdk.TaskStepStatusSuccess)
	}

	var wg sync.WaitGroup
	results := make([]flowsdk.TaskStepResult, len(s.taskSteps))
	errChan := make(chan error, len(s.taskSteps))
	runningChan := make(chan bool, len(s.taskSteps))

	for i, taskStep := range s.taskSteps {
		wg.Add(1)
		go func(idx int, step flowsdk.TaskStep) {
			defer wg.Done()
			result := step.Execute()
			results[idx] = result
			switch result.Status {
			case flowsdk.TaskStepStatusFailure:
				errChan <- result.Error
			case flowsdk.TaskStepStatusInterrupted:
				errChan <- nil
			case flowsdk.TaskStepStatusRunning:
				runningChan <- true
			}
		}(i, taskStep)
	}

	wg.Wait()
	close(errChan)
	close(runningChan)

	// 检查是否有失败或中断
	for err := range errChan {
		if err != nil {
			return flowsdk.NewTaskStepResultWithError(flowsdk.TaskStepStatusFailure, err, "并行步骤执行失败")
		}
		return flowsdk.NewTaskStepResult(flowsdk.TaskStepStatusInterrupted)
	}

	// 检查是否有运行中的步骤
	for range runningChan {
		return flowsdk.NewTaskStepResult(flowsdk.TaskStepStatusRunning)
	}

	return flowsdk.NewTaskStepResult(flowsdk.TaskStepStatusSuccess)
}

// OnSuccess 成功回调
func (s *CompositeTaskStep) OnSuccess() {
	for _, taskStep := range s.taskSteps {
		taskStep.OnSuccess()
	}
}

// OnFailure 失败回调
func (s *CompositeTaskStep) OnFailure(err error) {
	for _, taskStep := range s.taskSteps {
		taskStep.OnFailure(err)
	}
}

// OnInterrupt 中断回调
func (s *CompositeTaskStep) OnInterrupt() {
	for _, taskStep := range s.taskSteps {
		taskStep.OnInterrupt()
	}
	s.AbstractTaskStep.OnInterrupt()
}
