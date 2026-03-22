package auto

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/delegate"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/stateful"
)

// AutoExecuteTaskStep 自动执行任务步骤
type AutoExecuteTaskStep struct {
	*delegate.DelegateTaskStep
}

// NewAutoExecuteTaskStep 创建自动执行任务步骤
func NewAutoExecuteTaskStep(name string, taskContext *flowsdk.TaskContext, scheduledTaskStep flowsdk.TaskStep, taskPersistence stateful.TaskPersistence) *AutoExecuteTaskStep {
	// 包装为有状态步骤
	statefulStep := stateful.NewStatefulTaskStep(name, taskContext, scheduledTaskStep, taskPersistence)

	// 创建自动执行步骤
	autoStep := &AutoExecuteTaskStep{
		DelegateTaskStep: delegate.NewDelegateTaskStep(name, taskContext, statefulStep),
	}

	return autoStep
}

// GetType 获取任务步骤类型
func (s *AutoExecuteTaskStep) GetType() string {
	return "autoExecute"
}

// AutoExecuteScheduler 自动执行调度器
type AutoExecuteScheduler interface {
	// Start 启动调度器
	Start()
	// Stop 停止调度器
	Stop()
}
