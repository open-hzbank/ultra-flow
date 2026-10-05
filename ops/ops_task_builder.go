package ops

import "github.com/open-hzbank/ultra-flow/core"

// OpsTaskBuilder 运维任务构建器接口
type OpsTaskBuilder interface {
	GetOpsTaskName() string
	GetOpsTaskType() string

	ResumeBuild(taskID string) core.Task

	// ResumeBuildWithStepConfig 将用户给定的 key-value 传给指定的 step
	ResumeBuildWithStepConfig(taskID, stepName, key string, value any) core.Task

	ResumeBuildWithStepConfigs(taskID string, stepConfigs []core.StepExecuteConfig) core.Task

	// ResumeBuildWithStepContext 将用户给定的一组 key-value 传给指定的 step
	ResumeBuildWithStepContext(taskID, stepName string, stepContext map[string]any) core.Task

	// ResumeBuildWithTaskConfig 将用户给定的 key-value 传给指定的 task
	ResumeBuildWithTaskConfig(taskID, key string, value any) core.Task

	// ResumeBuildWithTaskContext 将用户给定的一组 key-value 传给指定的 task
	ResumeBuildWithTaskContext(taskID string, taskContext map[string]any) core.Task

	// region 系统执行，无用户上下文进行权限校验场景，用于如回调、自动执行等场景

	InternalResumeBuild(taskID string) core.Task

	// endregion
}

// BaseOpsTaskBuilder 提供 OpsTaskBuilder 默认实现的基类
type BaseOpsTaskBuilder struct {
	Name string
	Type string
}

func (b *BaseOpsTaskBuilder) GetOpsTaskName() string { return b.Name }
func (b *BaseOpsTaskBuilder) GetOpsTaskType() string { return b.Type }

func (b *BaseOpsTaskBuilder) ResumeBuild(string) core.Task {
	panic("not implemented")
}

// ResumeBuildWithStepConfig 将用户给定的 key-value 传给指定的 step
func (b *BaseOpsTaskBuilder) ResumeBuildWithStepConfig(string, string, string, any) core.Task {
	panic("not implemented")
}

func (b *BaseOpsTaskBuilder) ResumeBuildWithStepConfigs(string, []core.StepExecuteConfig) core.Task {
	panic("not implemented")
}

// ResumeBuildWithStepContext 将用户给定的一组 key-value 传给指定的 step
func (b *BaseOpsTaskBuilder) ResumeBuildWithStepContext(string, string, map[string]any) core.Task {
	panic("not implemented")
}

// ResumeBuildWithTaskConfig 将用户给定的 key-value 传给指定的 task
func (b *BaseOpsTaskBuilder) ResumeBuildWithTaskConfig(string, string, any) core.Task {
	panic("not implemented")
}

// ResumeBuildWithTaskContext 将用户给定的一组 key-value 传给指定的 task
func (b *BaseOpsTaskBuilder) ResumeBuildWithTaskContext(string, map[string]any) core.Task {
	panic("not implemented")
}

// InternalResumeBuild 系统执行，无用户上下文进行权限校验场景，用于如回调、自动执行等场景
func (b *BaseOpsTaskBuilder) InternalResumeBuild(taskID string) core.Task {
	return b.ResumeBuild(taskID)
}
