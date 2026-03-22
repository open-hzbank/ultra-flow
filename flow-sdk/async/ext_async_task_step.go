package async

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/delegate"
)

// ExtAsyncTaskStep 外部异步任务步骤
type ExtAsyncTaskStep struct {
	*delegate.DelegateTaskStep
	statusManager MultiTaskStatusManager
}

// NewExtAsyncTaskStep 创建外部异步任务步骤
func NewExtAsyncTaskStep(name string, taskContext *flowsdk.TaskContext, delegatedTaskStep flowsdk.TaskStep, statusManager MultiTaskStatusManager) *ExtAsyncTaskStep {
	taskStep := &ExtAsyncTaskStep{
		DelegateTaskStep: delegate.NewDelegateTaskStep(name, taskContext, delegatedTaskStep),
		statusManager:    statusManager,
	}

	return taskStep
}

// GetType 获取任务步骤类型
func (s *ExtAsyncTaskStep) GetType() string {
	return "extAsync"
}

// MultiTaskStatusManager 多任务状态管理器
type MultiTaskStatusManager interface {
	// RegisterTask 注册任务
	RegisterTask(taskId, stepName string)
	// UpdateTaskStatus 更新任务状态
	UpdateTaskStatus(taskId, stepName string, status flowsdk.TaskStepStatus)
	// GetTaskStatus 获取任务状态
	GetTaskStatus(taskId, stepName string) flowsdk.TaskStepStatus
}
