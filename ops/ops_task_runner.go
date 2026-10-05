package ops

import "hzbank.com.cn/ultra-flow/core"

// OpsTaskRunner 运维任务运行器接口
type OpsTaskRunner interface {
	Execute(builder OpsTaskBuilder, taskSupplier func(OpsTaskBuilder) core.Task) core.TaskResult
	Cancel(builder OpsTaskBuilder, taskSupplier func(OpsTaskBuilder) core.Task)
	Interrupt(builder OpsTaskBuilder, taskSupplier func(OpsTaskBuilder) core.Task)
}

// DefaultOpsTaskRunner 默认实现: 直接执行, 无额外处理
type DefaultOpsTaskRunner struct{}

func (r *DefaultOpsTaskRunner) Execute(builder OpsTaskBuilder, taskSupplier func(OpsTaskBuilder) core.Task) core.TaskResult {
	task := taskSupplier(builder)
	return task.Execute()
}

func (r *DefaultOpsTaskRunner) Cancel(builder OpsTaskBuilder, taskSupplier func(OpsTaskBuilder) core.Task) {
	task := taskSupplier(builder)
	task.Cancel()
}

func (r *DefaultOpsTaskRunner) Interrupt(builder OpsTaskBuilder, taskSupplier func(OpsTaskBuilder) core.Task) {
	task := taskSupplier(builder)
	task.Interrupt()
}
