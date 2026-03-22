package flowsdk

// TaskStep 任务步骤: 一个任务主体会分成很多个任务步骤
type TaskStep interface {
	// GetName 任务名称, 一个任务步骤可以有多个名称
	GetName() string
	// GetType 任务步骤类型, 任务步骤本身类型, 一个任务步骤可以定义出多个名称, 复用在不同的任务定义中
	GetType() string
	// Execute 任务步骤执行入口, 通常仅任务和任务步骤调用, 外部一般不允许调用
	Execute() TaskStepResult
	// OnSuccess 任务成功回调, 当任务整体调用成功结束后, 会调用此方法
	OnSuccess()
	// OnFailure 步骤失败回调, 当步骤有异常或失败时会调用此方法
	OnFailure(err error)
	// OnInterrupt 任务被取消时, 任务会调用步骤的 OnInterrupt 方法作为对中断信号的响应
	OnInterrupt()
}

// TaskStepResult 任务步骤执行结果
type TaskStepResult struct {
	Status  TaskStepStatus
	Error   error
	Message string
}

// NewTaskStepResult 创建任务步骤执行结果
func NewTaskStepResult(status TaskStepStatus) TaskStepResult {
	return TaskStepResult{
		Status: status,
	}
}

// NewTaskStepResultWithError 创建带错误的任务步骤执行结果
func NewTaskStepResultWithError(status TaskStepStatus, err error, message string) TaskStepResult {
	return TaskStepResult{
		Status:  status,
		Error:   err,
		Message: message,
	}
}
