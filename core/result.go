package core

import "fmt"

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID  string     `json:"taskId"`
	Status  TaskStatus `json:"status"`
	Error   error      `json:"-"`
	Message string     `json:"message,omitempty"`
}

func NewTaskResult(taskID string, status TaskStatus) TaskResult {
	return TaskResult{TaskID: taskID, Status: status}
}

func NewTaskResultWithError(taskID string, status TaskStatus, err error, message string) TaskResult {
	return TaskResult{TaskID: taskID, Status: status, Error: err, Message: message}
}

// TaskStepResult 任务步骤执行结果
type TaskStepResult struct {
	Status  TaskStepStatus
	Error   error
	Message string
}

func NewStepResult(status TaskStepStatus) TaskStepResult {
	return TaskStepResult{Status: status}
}

func NewStepResultWithMessage(status TaskStepStatus, message string) TaskStepResult {
	return TaskStepResult{Status: status, Message: message}
}

func NewStepResultWithError(status TaskStepStatus, err error, message string) TaskStepResult {
	return TaskStepResult{Status: status, Error: err, Message: message}
}

// Description 任务/步骤的描述
type Description struct {
	// 任务的展示性标题
	Title string
	// 发布原因
	Reason string
}

func NewDescription(title, reason string) Description {
	return Description{Title: title, Reason: reason}
}

// StepDescription 步骤描述
type StepDescription struct {
	Title  string
	Detail any
}

func NewStepDescription(title string, detail any) StepDescription {
	return StepDescription{Title: title, Detail: detail}
}

// LockFailureError 锁获取失败错误
type LockFailureError struct {
	Message string
}

func (e *LockFailureError) Error() string {
	return fmt.Sprintf("未成功获取到锁: %s", e.Message)
}

func NewLockFailureError(msg string) *LockFailureError {
	return &LockFailureError{Message: msg}
}
