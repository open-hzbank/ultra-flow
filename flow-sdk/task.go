package flowsdk



// Task 任务主体，任务下面会包含任务步骤，主体一般不包含业务逻辑
type Task interface {
	// GetTaskId 一般由外部构建，用于任务串联
	GetTaskId() string
	// GetName 任务名称，一个任务类型可以有多个任务名称实例
	GetName() string
	// GetType 任务类型，通常内置，当前任务本身类型，非任务实例级别
	GetType() string
	// GetSubject 任务目标主体, 各类型的任务需要自己实现获取主体的逻辑
	GetSubject() interface{}
	// GetPublishEnv 任务发布目标环境
	GetPublishEnv() Env
	// GetCreator 任务创建者
	GetCreator() string
	// Execute 任务执行入口
	Execute() TaskResult
	// Cancel 任务取消
	Cancel() error
	// Interrupt 任务中断，会中断执行中的步骤并立即结束当前任务
	Interrupt() error
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskId    string
	Status    TaskStatus
	Error     error
	Message   string
}

// NewTaskResult 创建任务执行结果
func NewTaskResult(taskId string, status TaskStatus) TaskResult {
	return TaskResult{
		TaskId: taskId,
		Status: status,
	}
}

// NewTaskResultWithError 创建带错误的任务执行结果
func NewTaskResultWithError(taskId string, status TaskStatus, err error, message string) TaskResult {
	return TaskResult{
		TaskId:  taskId,
		Status:  status,
		Error:   err,
		Message: message,
	}
}
