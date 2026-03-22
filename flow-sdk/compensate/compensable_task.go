package compensate

import (
	"errors"

	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
)

// CompensableTask 可补偿的任务
type CompensableTask struct {
	*flowsdk.AbstractTask
	compensableTaskStep CompensableTaskStep
}

// NewCompensableTask 创建可补偿的任务
func NewCompensableTask(taskId, name, type_, creator string, subject interface{}, publishEnv flowsdk.Env, idempotentId string, description flowsdk.Description) *CompensableTask {
	task := &CompensableTask{
		AbstractTask: flowsdk.NewAbstractTask(taskId, name, type_, creator, subject, publishEnv, idempotentId, description),
	}
	return task
}

// SetCompensableTaskStep 设置可补偿的任务步骤
func (t *CompensableTask) SetCompensableTaskStep(taskStep CompensableTaskStep) {
	t.compensableTaskStep = taskStep
	t.SetTaskStep(taskStep)
}

// ExecuteTask 执行任务的核心逻辑
func (t *CompensableTask) ExecuteTask() flowsdk.TaskResult {
	if t.compensableTaskStep == nil {
		return flowsdk.NewTaskResultWithError(t.GetTaskId(), flowsdk.TaskStatusFailure, errors.New("compensable task step is nil"), "可补偿任务步骤为空")
	}

	status := t.GetStatus()
	if status == flowsdk.TaskStatusCancelling {
		return t.executeCompensatedStep()
	}

	if !t.CanExecute() {
		return flowsdk.NewTaskResult(t.GetTaskId(), status)
	}

	return t.executeNormalStep()
}

// Cancel 取消任务
func (t *CompensableTask) Cancel() error {
	status := t.GetStatus()
	if status == flowsdk.TaskStatusCanceled || status == flowsdk.TaskStatusCancelling {
		return nil
	}
	if status.IsEnd() {
		return errors.New("已执行成功的任务无法取消")
	}

	t.SetStatus(flowsdk.TaskStatusCancelling)
	if t.compensableTaskStep != nil {
		t.compensableTaskStep.OnInterrupt()
	}

	compensateResult := t.executeCompensatedStep()
	t.SetStatus(compensateResult.Status)
	return nil
}

// executeNormalStep 执行正常步骤
func (t *CompensableTask) executeNormalStep() flowsdk.TaskResult {
	taskStepResult := t.compensableTaskStep.Execute()
	switch taskStepResult.Status {
	case flowsdk.TaskStepStatusPending, flowsdk.TaskStepStatusRunning:
		return flowsdk.NewTaskResult(t.GetTaskId(), flowsdk.TaskStatusRunning)
	case flowsdk.TaskStepStatusSkipped, flowsdk.TaskStepStatusSuccess:
		return flowsdk.NewTaskResult(t.GetTaskId(), flowsdk.TaskStatusSuccess)
	case flowsdk.TaskStepStatusFailure:
		t.compensableTaskStep.OnFailure(taskStepResult.Error)
		return flowsdk.NewTaskResultWithError(t.GetTaskId(), flowsdk.TaskStatusFailure, taskStepResult.Error, taskStepResult.Message)
	case flowsdk.TaskStepStatusInterrupted:
		t.compensableTaskStep.OnInterrupt()
		return flowsdk.NewTaskResult(t.GetTaskId(), flowsdk.TaskStatusCanceled)
	default:
		return flowsdk.NewTaskResultWithError(t.GetTaskId(), flowsdk.TaskStatusFailure, errors.New("unknown task step status"), "未知任务步骤状态")
	}
}

// executeCompensatedStep 执行补偿步骤
func (t *CompensableTask) executeCompensatedStep() flowsdk.TaskResult {
	compensableStep := t.compensableTaskStep.GenerateCompensableTaskStep()
	taskStepResult := compensableStep.Execute()
	switch taskStepResult.Status {
	case flowsdk.TaskStepStatusPending, flowsdk.TaskStepStatusRunning:
		return flowsdk.NewTaskResult(t.GetTaskId(), flowsdk.TaskStatusCancelling)
	case flowsdk.TaskStepStatusSkipped, flowsdk.TaskStepStatusSuccess:
		return flowsdk.NewTaskResult(t.GetTaskId(), flowsdk.TaskStatusCanceled)
	case flowsdk.TaskStepStatusFailure:
		compensableStep.OnFailure(taskStepResult.Error)
		return flowsdk.NewTaskResultWithError(t.GetTaskId(), flowsdk.TaskStatusFailure, taskStepResult.Error, taskStepResult.Message)
	case flowsdk.TaskStepStatusInterrupted:
		compensableStep.OnInterrupt()
		return flowsdk.NewTaskResult(t.GetTaskId(), flowsdk.TaskStatusCanceled)
	default:
		return flowsdk.NewTaskResultWithError(t.GetTaskId(), flowsdk.TaskStatusFailure, errors.New("unknown task step status"), "未知任务步骤状态")
	}
}
