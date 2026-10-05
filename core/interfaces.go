package core

// Task 任务主体，任务下面会包含任务步骤，主体一般不包含业务逻辑
type Task interface {
	// GetTaskID 一般由外部构建，用于任务串联
	GetTaskID() string
	// GetName 任务名称，一个任务类型可以有多个任务名称实例
	GetName() string
	// GetType 任务类型，通常内置，当前任务本身类型，非任务实例级别
	GetType() string
	// GetSubject 任务目标主体, 各类型的任务需要自己实现获取主体的逻辑
	GetSubject() any
	// GetPublishEnv 任务发布目标环境
	GetPublishEnv() Env
	// GetCreator 任务创建者
	GetCreator() string
	// Execute 任务执行入口，返回任务执行结果，如状态、异常传递
	Execute() TaskResult
	// Cancel 任务取消
	Cancel()
	// Interrupt 任务中断，会中断执行中的步骤并立即结束当前任务, 和 Cancel() 在语义上的区别:
	// - 本方法强调任务状态的强制终止, 不对数据状态的一致性做承诺, 只要持久化能力正常, 本方法就应当保证任务状态一定转为 INTERRUPTED;
	// - Cancel() 强调回撤, 以回退到发布前的状态为目标, 在实现时必须考虑数据状态的最终一致性;
	//
	// 注意事项:
	// 调用此接口可能会导致数据状态不可控: 最终既不是发布前的状态, 也不是预期的发布成功的状态
	// 仅适用于任务流执行异常, 若无人工介入则永远无法自主结束的场景
	// 建议该接口不要直接对用户暴露, 若调用必须告知用户重新覆盖发布以实现数据状态的修正
	Interrupt()
}

// TaskStep 任务步骤: 一个任务主体会分成很多个任务步骤
type TaskStep interface {
	// GetName 任务名称, 一个任务步骤可以有多个名称
	GetName() string
	// GetType 任务步骤类型, 任务步骤本身类型, 一个任务步骤可以定义出多个名称, 复用在不同的任务定义中
	GetType() string
	// Execute 任务步骤执行入口, 通常仅任务和任务步骤调用, 外部一般不允许调用
	Execute() TaskStepResult
	// OnSuccess 任务成功回调, 当任务整体调用成功结束后, 会调用此方法
	// 任务步骤状态流转需要具体实现自行处理
	OnSuccess()
	// OnFailure 步骤失败回调, 当步骤有异常或失败时会调用此方法
	// 任务步骤状态流转需要具体实现自行处理
	OnFailure(err error)
	// OnInterrupt 任务被取消时, 任务会调用步骤的 OnInterrupt 方法作为对中断信号的响应
	// 步骤可以自行实现对中断信号的处理, 默认是不做任何逻辑处理, 仅针对 RUNNING, FAILURE 状态的步骤, 将其状态置为 INTERRUPTED
	//
	// 注意: 当任务取消时, 如果步骤的补偿逻辑无法在 OnInterrupt 方法中同步完成, 则应当将补偿逻辑全都放置到独立的补偿步骤中,
	// 因为流程引擎不支持步骤存在 "中断进行中" 的中间状态
	OnInterrupt()
}

// TaskStepBuilder 步骤构建器, DepType 为构建 taskStep 所依赖的组件类型集合
type TaskStepBuilder interface {
	Build(name string, taskCtx *TaskContext, deps any) TaskStep
	TaskStepTitle() string
}

// TaskStepBuilderFunc 函数式步骤构建器
type TaskStepBuilderFunc struct {
	BuildFn  func(name string, taskCtx *TaskContext, deps any) TaskStep
	TitleFn  func() string
}

func (b *TaskStepBuilderFunc) Build(name string, taskCtx *TaskContext, deps any) TaskStep {
	return b.BuildFn(name, taskCtx, deps)
}

func (b *TaskStepBuilderFunc) TaskStepTitle() string {
	return b.TitleFn()
}

// TaskStepTransfer 用于传递对任务步骤的修改, 用于任务回调等场景
type TaskStepTransfer interface {
	Transfer(step TaskStep)
	Merge(other TaskStepTransfer)
}

// StepExecuteConfig 步骤执行配置
type StepExecuteConfig struct {
	// 步骤名称
	StepName string
	// 上下文 key
	Key      string
	// 上下文 value
	Value    any
}

// CompensableTaskStep 可对已执行步骤做补偿回退的 taskStep
type CompensableTaskStep interface {
	TaskStep
	GenerateCompensableTaskStep() TaskStep
}

// CompensateAwareTaskStep 能感知补偿相关业务逻辑的步骤
type CompensateAwareTaskStep interface {
	OriginTaskStep() TaskStep
	ExecutedTaskStep() (TaskStep, bool)
	CompensateTaskStep() (TaskStep, bool)
}
