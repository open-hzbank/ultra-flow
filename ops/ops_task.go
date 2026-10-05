package ops

import (
	"hzbank.com.cn/ultra-flow/core"
	"hzbank.com.cn/ultra-flow/stateful"
	"hzbank.com.cn/ultra-flow/transfer"
)

const (
	TaskErrorTips           = "taskErrorTips"
	OpsTaskGlobalBusDataKey = "ops.task.bus.data"
)

// OpsTask 通用的业务运维任务
type OpsTask struct {
	statefulTask *stateful.StatefulTask
	// 业务自定义的任务级总线数据: 任务下辖各步骤间全局共享
	globalBusData any
	serDeser      core.TaskDataSerDeser
}

// NewOpsTask 初始化构造任务
func NewOpsTask(
	globalBusData any,
	taskID, name, opsType string,
	subject any,
	publishEnv core.Env,
	creator, idempotentID string,
	description core.Description,
	taskStepTransfer core.TaskStepTransfer,
	taskStepSupplier func(*core.TaskContext, any) core.TaskStep,
	taskPersistence stateful.TaskPersistence,
	transactionable stateful.Transactionable,
	serDeser core.TaskDataSerDeser,
) *OpsTask {

	delegated := newDelegatedOpsTask(delegatedOpsInitConfig{
		taskID:       taskID,
		name:         name,
		taskType:     opsType,
		subject:      subject,
		creator:      creator,
		publishEnv:   publishEnv,
		idempotentID: idempotentID,
		description:  description,
		transfer:     taskStepTransfer,
		stepSupplier: func(ctx *core.TaskContext) core.TaskStep {
			return taskStepSupplier(ctx, globalBusData)
		},
	})

	sf := stateful.NewStatefulTaskFromTask(delegated, taskPersistence, transactionable)
	t := &OpsTask{
		statefulTask:  sf,
		globalBusData: globalBusData,
		serDeser:      serDeser,
	}
	t.refreshGlobalBusDataToContext()
	return t
}

// ResumeOpsTask 恢复任务
func ResumeOpsTask(
	taskID string,
	appendContext map[string]any,
	taskStepTransfer core.TaskStepTransfer,
	taskStepSupplier func(*core.TaskContext, any) core.TaskStep,
	taskPersistence stateful.TaskPersistence,
	transactionable stateful.Transactionable,
	serDeser core.TaskDataSerDeser,
) *OpsTask {

	sf := stateful.NewStatefulTask(
		func(taskState *stateful.TaskState) core.Task {
			return newDelegatedOpsTaskFromState(taskState, appendContext, taskStepTransfer)
		},
		taskPersistence, transactionable, taskID,
	)

	t := &OpsTask{
		statefulTask: sf,
		serDeser:     serDeser,
	}
	t.globalBusData = t.recoverGlobalBusData()

	delegated := sf.Delegated.(*delegatedOpsTask)
	delegated.initTaskStep(func(ctx *core.TaskContext) core.TaskStep {
		return taskStepSupplier(ctx, t.globalBusData)
	})
	return t
}

func (t *OpsTask) Execute() core.TaskResult {
	return t.statefulTask.ExecuteWithProcessors(func() {}, t.refreshGlobalBusDataToContext)
}

func (t *OpsTask) Cancel() {
	t.statefulTask.Cancel()
}

func (t *OpsTask) Interrupt() {
	t.statefulTask.Interrupt()
}

func (t *OpsTask) GetTaskID() string       { return t.statefulTask.GetTaskID() }
func (t *OpsTask) GetName() string         { return t.statefulTask.GetName() }
func (t *OpsTask) GetType() string         { return t.statefulTask.GetType() }
func (t *OpsTask) GetSubject() any         { return t.statefulTask.GetSubject() }
func (t *OpsTask) GetCreator() string      { return t.statefulTask.GetCreator() }
func (t *OpsTask) GetPublishEnv() core.Env { return t.statefulTask.GetPublishEnv() }

func (t *OpsTask) GetGlobalBusData() any { return t.globalBusData }

func (t *OpsTask) GetTaskContext() *core.TaskContext {
	type taskContextAware interface {
		GetTaskContext() *core.TaskContext
	}
	if aware, ok := t.statefulTask.Delegated.(taskContextAware); ok {
		return aware.GetTaskContext()
	}
	return nil
}

func (t *OpsTask) refreshGlobalBusDataToContext() {
	ctx := t.GetTaskContext()
	if ctx != nil {
		ctx.Put(OpsTaskGlobalBusDataKey, t.serDeser.Serialize(t.globalBusData))
	}
}

func (t *OpsTask) recoverGlobalBusData() any {
	ctx := t.GetTaskContext()
	if ctx == nil {
		return nil
	}
	value := ctx.Get(OpsTaskGlobalBusDataKey)
	if value == nil {
		return nil
	}
	if str, ok := value.(string); ok {
		return t.serDeser.Deserialize(str)
	}
	return value
}

// GetGlobalBusDataFromContext 从上下文中获取全局总线数据
func GetGlobalBusDataFromContext(context map[string]any, serDeser core.TaskDataSerDeser) any {
	value := context[OpsTaskGlobalBusDataKey]
	if value == nil {
		return nil
	}
	if str, ok := value.(string); ok {
		return serDeser.Deserialize(str)
	}
	return value
}

func mergeContext(highPriority, lowPriority map[string]any) map[string]any {
	if len(highPriority) == 0 {
		if lowPriority == nil {
			return map[string]any{}
		}
		return lowPriority
	}
	if len(lowPriority) == 0 {
		return highPriority
	}
	result := make(map[string]any, len(highPriority)+len(lowPriority))
	for k, v := range highPriority {
		result[k] = v
	}
	for k, v := range lowPriority {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}
	return result
}

type delegatedOpsInitConfig struct {
	taskID       string
	name         string
	taskType     string
	subject      any
	creator      string
	publishEnv   core.Env
	idempotentID string
	description  core.Description
	transfer     core.TaskStepTransfer
	stepSupplier func(*core.TaskContext) core.TaskStep
}

// delegatedOpsTask OpsTask 默认使用的代理任务
type delegatedOpsTask struct {
	*core.AbstractTask
}

func newDelegatedOpsTask(cfg delegatedOpsInitConfig) *delegatedOpsTask {
	base := core.NewAbstractTaskWithStep(core.AbstractTaskConfig{
		TaskID:       cfg.taskID,
		Name:         cfg.name,
		Type:         cfg.taskType,
		Subject:      cfg.subject,
		Creator:      cfg.creator,
		PublishEnv:   cfg.publishEnv,
		IdempotentID: cfg.idempotentID,
		Description:  cfg.description,
		Transfer:     cfg.transfer,
	}, cfg.stepSupplier)
	return &delegatedOpsTask{AbstractTask: base}
}

func newDelegatedOpsTaskFromState(taskState *stateful.TaskState, appendContext map[string]any, taskStepTransfer core.TaskStepTransfer) *delegatedOpsTask {
	mergedCtx := mergeContext(appendContext, taskState.Context)
	base := core.NewAbstractTask(core.AbstractTaskConfig{
		TaskID:       taskState.TaskID,
		Name:         taskState.Name,
		Type:         taskState.Type,
		Subject:      taskState.Subject,
		Creator:      taskState.Creator,
		PublishEnv:   taskState.PublishEnv,
		IdempotentID: taskState.IdempotentID,
		Status:       taskState.Status,
		Description:  taskState.Description,
		Context:      mergedCtx,
		Transfer:     taskStepTransfer,
	})
	return &delegatedOpsTask{AbstractTask: base}
}

func (t *delegatedOpsTask) initTaskStep(supplier func(*core.TaskContext) core.TaskStep) {
	t.AbstractTask.InitTaskStep(supplier)
}

// NewTransferFromStepConfig 从步骤配置创建 TaskStepTransfer
func NewTransferFromStepConfig(stepName, key string, value any) core.TaskStepTransfer {
	return transfer.NewDefaultTaskStepTransferWithEntry(stepName, key, value)
}

// NewTransferFromConfigs 从配置列表创建 TaskStepTransfer
func NewTransferFromConfigs(configs []core.StepExecuteConfig) core.TaskStepTransfer {
	return transfer.NewDefaultTaskStepTransferFromConfigs(configs)
}

// NewTransferFromMap 从 map 创建 TaskStepTransfer
func NewTransferFromMap(contexts map[string]map[string]any) core.TaskStepTransfer {
	return transfer.NewDefaultTaskStepTransferFromMap(contexts)
}
