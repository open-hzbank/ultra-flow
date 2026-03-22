package flowsdk

import (
	"errors"
	"sync"
)

// AbstractTaskStep 任务步骤的抽象实现
type AbstractTaskStep struct {
	name           string
	taskStepContext *TaskStepContext
	taskContext     *TaskContext
	status          TaskStepStatus
	statusMutex     sync.RWMutex
}

// TaskStepDescription 任务步骤描述
type TaskStepDescription struct {
	Title  string
	Detail interface{}
}

// NewTaskStepDescription 创建任务步骤描述
func NewTaskStepDescription(title string, detail interface{}) TaskStepDescription {
	return TaskStepDescription{
		Title:  title,
		Detail: detail,
	}
}

// NewAbstractTaskStep 创建抽象任务步骤
func NewAbstractTaskStep(name string, taskContext *TaskContext, needTransferStep bool) *AbstractTaskStep {
	taskStep := &AbstractTaskStep{
		name:       name,
		taskContext: taskContext,
		status:     TaskStepStatusPending,
	}
	taskStep.taskStepContext = NewTaskStepContext(taskContext.GetTask(), taskStep)

	if needTransferStep {
		transfer := taskContext.GetTaskStepTransfer()
		if transfer != nil {
			transfer.Transfer(taskStep)
		}
	}

	return taskStep
}

// GetName 获取任务步骤名称
func (s *AbstractTaskStep) GetName() string {
	return s.name
}

// GetType 获取任务步骤类型
func (s *AbstractTaskStep) GetType() string {
	return "abstract"
}

// GetStatus 获取任务步骤状态
func (s *AbstractTaskStep) GetStatus() TaskStepStatus {
	s.statusMutex.RLock()
	defer s.statusMutex.RUnlock()
	return s.status
}

// SetStatus 设置任务步骤状态
func (s *AbstractTaskStep) SetStatus(status TaskStepStatus) {
	s.statusMutex.Lock()
	defer s.statusMutex.Unlock()
	s.status = status
}

// GetTaskStepContext 获取任务步骤上下文
func (s *AbstractTaskStep) GetTaskStepContext() *TaskStepContext {
	return s.taskStepContext
}

// GetTaskContext 获取任务上下文
func (s *AbstractTaskStep) GetTaskContext() *TaskContext {
	return s.taskContext
}

// Execute 执行任务步骤
func (s *AbstractTaskStep) Execute() TaskStepResult {
	s.taskContext.AddPath(s)
	taskStepResult := s.ExecuteStep()
	s.SetStatus(taskStepResult.Status)
	return taskStepResult
}

// ExecuteStep 执行步骤的核心逻辑
func (s *AbstractTaskStep) ExecuteStep() TaskStepResult {
	if !s.CanExecute() {
		return NewTaskStepResult(s.GetStatus())
	}

	taskStepResult := s.DoExecute()
	switch taskStepResult.Status {
	case TaskStepStatusPending, TaskStepStatusRunning:
		return NewTaskStepResult(TaskStepStatusRunning)
	case TaskStepStatusSuccess:
		s.OnSuccess()
		return NewTaskStepResult(TaskStepStatusSuccess)
	case TaskStepStatusFailure:
		s.OnFailure(taskStepResult.Error)
		return NewTaskStepResultWithError(TaskStepStatusFailure, taskStepResult.Error, taskStepResult.Message)
	case TaskStepStatusInterrupted:
		s.OnInterrupt()
		return NewTaskStepResult(TaskStepStatusInterrupted)
	case TaskStepStatusSkipped:
		return NewTaskStepResult(TaskStepStatusSkipped)
	default:
		return NewTaskStepResultWithError(TaskStepStatusFailure, errors.New("unknown task step status"), "未知任务步骤状态")
	}
}

// CanExecute 检查是否可以执行
func (s *AbstractTaskStep) CanExecute() bool {
	status := s.GetStatus()
	return status == TaskStepStatusPending || status == TaskStepStatusRunning || status == TaskStepStatusFailure
}

// DoExecute 执行具体逻辑，子类需要实现
func (s *AbstractTaskStep) DoExecute() TaskStepResult {
	return NewTaskStepResult(TaskStepStatusSuccess)
}

// OnSuccess 成功回调
func (s *AbstractTaskStep) OnSuccess() {
}

// OnFailure 失败回调
func (s *AbstractTaskStep) OnFailure(err error) {
}

// OnInterrupt 中断回调
func (s *AbstractTaskStep) OnInterrupt() {
	if !s.CanInterrupt() {
		return
	}
	interruptedStatus := s.DoInterrupt()
	s.SetStatus(interruptedStatus)
}

// CanInterrupt 检查是否可以中断
func (s *AbstractTaskStep) CanInterrupt() bool {
	return s.GetStatus().IsExecutingStatus()
}

// DoInterrupt 执行中断逻辑
func (s *AbstractTaskStep) DoInterrupt() TaskStepStatus {
	return TaskStepStatusInterrupted
}

// Describe 描述任务步骤
func (s *AbstractTaskStep) Describe() TaskStepDescription {
	return NewTaskStepDescription(s.name, nil)
}
