package ops

import (
	"github.com/open-hzbank/ultra-flow/compensate"
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/stateful"
)

// CompensableOpsTask 可补偿的业务运维任务
type CompensableOpsTask struct {
	statefulTask  *stateful.StatefulTask
	globalBusData any
	serDeser      core.TaskDataSerDeser
}

// NewCompensableOpsTask 初始化可补偿的任务
func NewCompensableOpsTask(
	globalBusData any,
	taskID, name, opsType string,
	subject any,
	creator, idempotentID string,
	publishEnv core.Env,
	description core.Description,
	taskStepTransfer core.TaskStepTransfer,
	taskStepSupplier func(*core.TaskContext, any) core.CompensableTaskStep,
	taskPersistence stateful.TaskPersistence,
	transactionable stateful.Transactionable,
	serDeser core.TaskDataSerDeser,
) *CompensableOpsTask {

	delegated := newDelegatedCompensableOpsTask(delegatedCompensableConfig{
		taskID:       taskID,
		name:         name,
		taskType:     opsType,
		subject:      subject,
		creator:      creator,
		publishEnv:   publishEnv,
		idempotentID: idempotentID,
		description:  description,
		transfer:     taskStepTransfer,
		stepSupplier: func(ctx *core.TaskContext) core.CompensableTaskStep {
			return taskStepSupplier(ctx, globalBusData)
		},
	})

	sf := stateful.NewStatefulTaskFromTask(delegated, taskPersistence, transactionable)
	t := &CompensableOpsTask{
		statefulTask:  sf,
		globalBusData: globalBusData,
		serDeser:      serDeser,
	}
	t.refreshGlobalBusDataToContext()
	return t
}

// ResumeCompensableOpsTask 恢复可补偿的任务
func ResumeCompensableOpsTask(
	taskID string,
	appendContext map[string]any,
	taskStepTransfer core.TaskStepTransfer,
	taskStepSupplier func(*core.TaskContext, any) core.CompensableTaskStep,
	taskPersistence stateful.TaskPersistence,
	transactionable stateful.Transactionable,
	serDeser core.TaskDataSerDeser,
) *CompensableOpsTask {

	sf := stateful.NewStatefulTask(
		func(taskState *stateful.TaskState) core.Task {
			return newDelegatedCompensableOpsTaskFromState(taskState, appendContext, taskStepTransfer)
		},
		taskPersistence, transactionable, taskID,
	)

	t := &CompensableOpsTask{
		statefulTask: sf,
		serDeser:     serDeser,
	}
	t.globalBusData = t.recoverGlobalBusData()

	delegated := sf.Delegated.(*delegatedCompensableOpsTask)
	delegated.initTaskStep(func(ctx *core.TaskContext) core.CompensableTaskStep {
		return taskStepSupplier(ctx, t.globalBusData)
	})
	return t
}

func (t *CompensableOpsTask) Execute() core.TaskResult {
	return t.statefulTask.ExecuteWithProcessors(func() {}, t.refreshGlobalBusDataToContext)
}

func (t *CompensableOpsTask) Cancel() {
	t.statefulTask.Cancel()
}

func (t *CompensableOpsTask) Interrupt() {
	t.statefulTask.Interrupt()
}

func (t *CompensableOpsTask) GetTaskID() string       { return t.statefulTask.GetTaskID() }
func (t *CompensableOpsTask) GetName() string         { return t.statefulTask.GetName() }
func (t *CompensableOpsTask) GetType() string         { return t.statefulTask.GetType() }
func (t *CompensableOpsTask) GetSubject() any         { return t.statefulTask.GetSubject() }
func (t *CompensableOpsTask) GetCreator() string      { return t.statefulTask.GetCreator() }
func (t *CompensableOpsTask) GetPublishEnv() core.Env { return t.statefulTask.GetPublishEnv() }

func (t *CompensableOpsTask) GetGlobalBusData() any { return t.globalBusData }

func (t *CompensableOpsTask) GetTaskContext() *core.TaskContext {
	type taskContextAware interface {
		GetTaskContext() *core.TaskContext
	}
	if aware, ok := t.statefulTask.Delegated.(taskContextAware); ok {
		return aware.GetTaskContext()
	}
	return nil
}

func (t *CompensableOpsTask) refreshGlobalBusDataToContext() {
	ctx := t.GetTaskContext()
	if ctx != nil {
		ctx.Put(OpsTaskGlobalBusDataKey, t.serDeser.Serialize(t.globalBusData))
	}
}

func (t *CompensableOpsTask) recoverGlobalBusData() any {
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

type delegatedCompensableConfig struct {
	taskID       string
	name         string
	taskType     string
	subject      any
	creator      string
	publishEnv   core.Env
	idempotentID string
	description  core.Description
	transfer     core.TaskStepTransfer
	stepSupplier func(*core.TaskContext) core.CompensableTaskStep
}

// delegatedCompensableOpsTask CompensableOpsTask 默认使用的代理任务
type delegatedCompensableOpsTask struct {
	*compensate.CompensableTask
}

func newDelegatedCompensableOpsTask(cfg delegatedCompensableConfig) *delegatedCompensableOpsTask {
	ct := compensate.NewCompensableTaskWithSupplier(compensate.CompensableTaskConfig{
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
	return &delegatedCompensableOpsTask{CompensableTask: ct}
}

func newDelegatedCompensableOpsTaskFromState(taskState *stateful.TaskState, appendContext map[string]any,
	taskStepTransfer core.TaskStepTransfer) *delegatedCompensableOpsTask {
	mergedCtx := mergeContext(appendContext, taskState.Context)
	ct := compensate.NewCompensableTask(compensate.CompensableTaskConfig{
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
	return &delegatedCompensableOpsTask{CompensableTask: ct}
}

func (t *delegatedCompensableOpsTask) initTaskStep(supplier func(*core.TaskContext) core.CompensableTaskStep) {
	ct := compensate.NewCompensableTaskWithSupplier(compensate.CompensableTaskConfig{
		TaskID:       t.GetTaskID(),
		Name:         t.GetName(),
		Type:         t.GetType(),
		Subject:      t.GetSubject(),
		Creator:      t.GetCreator(),
		PublishEnv:   t.GetPublishEnv(),
		IdempotentID: t.GetIdempotentID(),
		Context:      t.GetTaskContext().GetContext(),
		Status:       t.GetStatus(),
		Description:  t.GetDescription(),
		Transfer:     nil,
	}, supplier)
	t.CompensableTask = ct
}
