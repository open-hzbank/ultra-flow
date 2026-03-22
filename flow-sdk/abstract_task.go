package flowsdk

import (
	"errors"
	"sync"
)

// AbstractTask 任务的抽象实现
type AbstractTask struct {
	taskId            string
	name              string
	type_             string
	subject           interface{}
	creator           string
	publishEnv        Env
	description       Description
	idempotentId      string
	taskContext       *TaskContext
	status            TaskStatus
	taskStep          TaskStep
	statusMutex       sync.RWMutex
}

// Description 任务描述
type Description struct {
	Title  string
	Reason string
}

// NewDescription 创建任务描述
func NewDescription(title, reason string) Description {
	return Description{
		Title:  title,
		Reason: reason,
	}
}

// NewAbstractTask 创建抽象任务
func NewAbstractTask(taskId, name, type_, creator string, subject interface{}, publishEnv Env, idempotentId string, description Description) *AbstractTask {
	task := &AbstractTask{
		taskId:         taskId,
		name:           name,
		type_:          type_,
		subject:        subject,
		creator:        creator,
		publishEnv:     publishEnv,
		description:    description,
		idempotentId:   idempotentId,
		status:         TaskStatusCreated,
	}
	task.taskContext = NewTaskContext(task)
	return task
}

// GetTaskId 获取任务ID
func (t *AbstractTask) GetTaskId() string {
	return t.taskId
}

// GetName 获取任务名称
func (t *AbstractTask) GetName() string {
	return t.name
}

// GetType 获取任务类型
func (t *AbstractTask) GetType() string {
	return t.type_
}

// GetSubject 获取任务主体
func (t *AbstractTask) GetSubject() interface{} {
	return t.subject
}

// GetPublishEnv 获取发布环境
func (t *AbstractTask) GetPublishEnv() Env {
	return t.publishEnv
}

// GetCreator 获取创建者
func (t *AbstractTask) GetCreator() string {
	return t.creator
}

// GetDescription 获取任务描述
func (t *AbstractTask) GetDescription() Description {
	return t.description
}

// GetIdempotentId 获取幂等ID
func (t *AbstractTask) GetIdempotentId() string {
	return t.idempotentId
}

// GetTaskContext 获取任务上下文
func (t *AbstractTask) GetTaskContext() *TaskContext {
	return t.taskContext
}

// GetStatus 获取任务状态
func (t *AbstractTask) GetStatus() TaskStatus {
	t.statusMutex.RLock()
	defer t.statusMutex.RUnlock()
	return t.status
}

// SetStatus 设置任务状态
func (t *AbstractTask) SetStatus(status TaskStatus) {
	t.statusMutex.Lock()
	defer t.statusMutex.Unlock()
	t.status = status
}

// SetTaskStep 设置任务步骤
func (t *AbstractTask) SetTaskStep(taskStep TaskStep) {
	t.taskStep = taskStep
}

// Execute 执行任务
func (t *AbstractTask) Execute() TaskResult {
	taskResult := t.ExecuteTask()
	t.SetStatus(taskResult.Status)
	return taskResult
}

// ExecuteWithProcessors 带前后处理器的执行
func (t *AbstractTask) ExecuteWithProcessors(beforeProcessor, afterProcessor func()) TaskResult {
	beforeProcessor()
	taskResult := t.Execute()
	afterProcessor()
	return taskResult
}

// ExecuteTask 执行任务的核心逻辑
func (t *AbstractTask) ExecuteTask() TaskResult {
	if !t.CanExecute() {
		return NewTaskResult(t.taskId, t.GetStatus())
	}

	if t.taskStep == nil {
		return NewTaskResultWithError(t.taskId, TaskStatusFailure, errors.New("task step is nil"), "任务步骤为空")
	}

	taskStepResult := t.taskStep.Execute()
	switch taskStepResult.Status {
	case TaskStepStatusPending, TaskStepStatusRunning:
		return NewTaskResult(t.taskId, TaskStatusRunning)
	case TaskStepStatusSkipped, TaskStepStatusSuccess:
		return NewTaskResult(t.taskId, TaskStatusSuccess)
	case TaskStepStatusFailure:
		t.taskStep.OnFailure(taskStepResult.Error)
		return NewTaskResultWithError(t.taskId, TaskStatusFailure, taskStepResult.Error, taskStepResult.Message)
	case TaskStepStatusInterrupted:
		t.taskStep.OnInterrupt()
		return NewTaskResult(t.taskId, TaskStatusCanceled)
	default:
		return NewTaskResultWithError(t.taskId, TaskStatusFailure, errors.New("unknown task step status"), "未知任务步骤状态")
	}
}

// CanExecute 检查是否可以执行
func (t *AbstractTask) CanExecute() bool {
	status := t.GetStatus()
	return status == TaskStatusCreated || status == TaskStatusRunning || status == TaskStatusFailure
}

// Cancel 取消任务
func (t *AbstractTask) Cancel() error {
	status := t.GetStatus()
	if status == TaskStatusCanceled {
		return nil
	}
	if status.IsEnd() {
		return errors.New("已到达终态的任务无法取消")
	}

	if t.taskStep != nil {
		t.taskStep.OnInterrupt()
	}
	t.SetStatus(TaskStatusCanceled)
	return nil
}

// Interrupt 中断任务
func (t *AbstractTask) Interrupt() error {
	status := t.GetStatus()
	if status == TaskStatusInterrupted {
		return nil
	}
	if status.IsEnd() {
		return errors.New("已到达终态的任务无法中断")
	}

	if t.taskStep != nil {
		t.taskStep.OnInterrupt()
	}
	t.SetStatus(TaskStatusInterrupted)
	return nil
}

// IdentityProcessor 空处理器
func IdentityProcessor() func() {
	return func() {}
}
