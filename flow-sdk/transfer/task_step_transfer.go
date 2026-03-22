package transfer

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
)

// TaskStepTransfer 任务步骤转换器
type TaskStepTransfer interface {
	// Transfer 转换任务步骤
	Transfer(step flowsdk.TaskStep)
}

// DefaultTaskStepTransfer 默认任务步骤转换器
type DefaultTaskStepTransfer struct {}

// NewDefaultTaskStepTransfer 创建默认任务步骤转换器
func NewDefaultTaskStepTransfer() *DefaultTaskStepTransfer {
	return &DefaultTaskStepTransfer{}
}

// Transfer 转换任务步骤
func (t *DefaultTaskStepTransfer) Transfer(step flowsdk.TaskStep) {
	// 默认实现，不做任何转换
}
