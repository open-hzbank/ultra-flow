package stateful

import (
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/delegate"
	"github.com/open-hzbank/ultra-flow/transfer"
)

const StatefulTaskStepType = "stateful"

// StatefulTaskStep 有状态的任务步骤:
// 任务步骤开始执行时如果已执行过会从历史恢复状态, 执行结束后会保存当前执行状态快照, 用于下次执行上下文恢复
type StatefulTaskStep struct {
	*delegate.DelegateTaskStep
	taskPersistence     TaskPersistence
	beforeExecuteStatus core.TaskStepStatus
}

// Wrap 包装已有步骤为有状态步骤
// 该方法仅在 taskStep 实例初始化之后加载状态
// 而在 taskStep 初始化时无法获取状态
func Wrap(taskStep core.TaskStep, taskPersistence TaskPersistence) *StatefulTaskStep {
	s := &StatefulTaskStep{
		DelegateTaskStep: delegate.NewDelegateTaskStep(taskStep),
		taskPersistence:  taskPersistence,
	}
	s.recoverTaskStep()
	return s
}

// WrapWithSupplier 使用 supplier 在初始化时即加载状态
// 该方法在使用 taskStepSupplier 初始化步骤实例时就加载状态
// 在目标 taskStep 构造方法中即可获取状态并使用
func WrapWithSupplier(taskStepSupplier func(*core.TaskContext) core.TaskStep, taskCtx *core.TaskContext,
	taskPersistence TaskPersistence, stepName string) core.TaskStep {
	snapshot := taskPersistence.RecoverTaskStep(taskCtx.GetTask().GetTaskID(), stepName)

	if snapshot == nil {
		// 没有历史状态
		taskStep := taskStepSupplier(taskCtx)
		type statusSetter interface {
			SetStatus(core.TaskStepStatus)
		}
		if ss, ok := taskStep.(statusSetter); ok {
			ss.SetStatus(core.StepPending)
		}
		return &StatefulTaskStep{
			DelegateTaskStep:    delegate.NewDelegateTaskStep(taskStep),
			taskPersistence:     taskPersistence,
			beforeExecuteStatus: core.StepPending,
		}
	}

	stepCtxSnapshot := snapshot.TaskStepContext
	if len(stepCtxSnapshot) > 0 {
		// 加载状态
		snapshotTransfer := transfer.NewDefaultTaskStepTransferWithEntry(stepName, "", nil)
		for k, v := range stepCtxSnapshot {
			snapshotTransfer = transfer.NewDefaultTaskStepTransferWithEntry(stepName, k, v)
			_ = v
		}
		snapshotTransfer = buildTransferFromMap(stepName, stepCtxSnapshot)
		loadSnapshotToTaskContext(taskCtx, snapshotTransfer)
	}

	// 构建 state aware taskStep
	taskStep := taskStepSupplier(taskCtx)
	type statusSetter interface {
		SetStatus(core.TaskStepStatus)
	}
	if ss, ok := taskStep.(statusSetter); ok {
		ss.SetStatus(snapshot.Status)
	}

	return &StatefulTaskStep{
		DelegateTaskStep:    delegate.NewDelegateTaskStep(taskStep),
		taskPersistence:     taskPersistence,
		beforeExecuteStatus: snapshot.Status,
	}
}

func buildTransferFromMap(stepName string, ctx map[string]any) *transfer.DefaultTaskStepTransfer {
	result := transfer.NewDefaultTaskStepTransfer()
	for k, v := range ctx {
		result.Merge(transfer.NewDefaultTaskStepTransferWithEntry(stepName, k, v))
	}
	return result
}

func (s *StatefulTaskStep) GetType() string {
	return StatefulTaskStepType
}

func (s *StatefulTaskStep) Execute() core.TaskStepResult {
	defer s.saveTaskStep()
	type taskCtxAware interface {
		GetTaskContext() *core.TaskContext
	}
	if tca, ok := s.Delegated.(taskCtxAware); ok {
		taskCtx := tca.GetTaskContext()
		if taskCtx != nil {
			taskCtx.AddPath(s)
		}
	}
	result := s.Delegated.Execute()
	type statusSetter interface {
		SetStatus(core.TaskStepStatus)
	}
	if ss, ok := s.Delegated.(statusSetter); ok {
		ss.SetStatus(result.Status)
	}
	return result
}

func (s *StatefulTaskStep) OnInterrupt() {
	defer s.saveTaskStep()
	s.Delegated.OnInterrupt()
}

// saveTaskStep 保存任务步骤快照
// 未执行则不会保存, 已达终态则不会重复保存
func (s *StatefulTaskStep) saveTaskStep() {
	type statusAware interface {
		GetStatus() core.TaskStepStatus
	}
	var delegatedStatus core.TaskStepStatus
	if sa, ok := s.Delegated.(statusAware); ok {
		delegatedStatus = sa.GetStatus()
	}

	if delegatedStatus == core.StepPending ||
		s.beforeExecuteStatus == core.StepSuccess ||
		s.beforeExecuteStatus == core.StepInterrupted {
		return
	}

	type fullStepAware interface {
		GetTaskContext() *core.TaskContext
		GetTaskStepContext() *core.TaskStepContext
		Describe() core.StepDescription
	}
	aware, ok := s.Delegated.(fullStepAware)
	if !ok {
		return
	}

	taskCtx := aware.GetTaskContext()
	if taskCtx == nil {
		return
	}
	task := taskCtx.GetTask()

	var stepCtx map[string]any
	if stepContext := aware.GetTaskStepContext(); stepContext != nil {
		stepCtx = stepContext.GetContext()
	}

	snapshot := &TaskStepSnapshot{
		TaskID:          task.GetTaskID(),
		Name:            s.Delegated.GetName(),
		Type:            s.Delegated.GetType(),
		TaskStepContext: stepCtx,
		Status:          delegatedStatus,
		Description:     aware.Describe(),
	}
	s.taskPersistence.SaveTaskStep(snapshot)
}

// recoverTaskStep 从持久化数据恢复任务步骤状态
func (s *StatefulTaskStep) recoverTaskStep() {
	type taskCtxAware interface {
		GetTaskContext() *core.TaskContext
	}
	tca, ok := s.Delegated.(taskCtxAware)
	if !ok {
		return
	}
	taskCtx := tca.GetTaskContext()
	if taskCtx == nil {
		return
	}

	snapshot := s.taskPersistence.RecoverTaskStep(taskCtx.GetTask().GetTaskID(), s.GetName())
	if snapshot == nil {
		return
	}

	type stepContextAware interface {
		GetTaskStepContext() *core.TaskStepContext
	}
	if sca, ok := s.Delegated.(stepContextAware); ok {
		ctx := sca.GetTaskStepContext()
		if ctx != nil {
			ctx.PutIfAbsent(snapshot.TaskStepContext)
		}
	}

	type statusSetter interface {
		SetStatus(core.TaskStepStatus)
	}
	if ss, ok := s.Delegated.(statusSetter); ok {
		ss.SetStatus(snapshot.Status)
	}
	s.beforeExecuteStatus = snapshot.Status
}

// loadSnapshotToTaskContext 将快照信息载入 taskContext
func loadSnapshotToTaskContext(taskCtx *core.TaskContext, snapshotTransfer core.TaskStepTransfer) {
	latestTransfer := taskCtx.GetTaskStepTransfer()
	// taskContext 没有设置 latestTransfer, 直接使用原有的 snapshotTransfer
	if latestTransfer == nil {
		taskCtx.SetTaskStepTransfer(snapshotTransfer)
		return
	}
	// taskContext 设置了 latestTransfer, 则用原有的 snapshotTransfer 合并新的 latestTransfer
	snapshotTransfer.Merge(latestTransfer)
	taskCtx.SetTaskStepTransfer(snapshotTransfer)
}
