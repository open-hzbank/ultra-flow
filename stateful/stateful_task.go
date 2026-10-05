package stateful

import (
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/delegate"
	"log"
)

// StatefulTask 有状态能力的包装: delegatedTask 执行结束后会持久化其所有的状态数据, 执行开始前会从持久化数据中恢复状态
type StatefulTask struct {
	*delegate.DelegateTask
	taskPersistence     TaskPersistence
	transactionable     Transactionable
	beforeExecuteStatus core.TaskStatus
}

// NewStatefulTask 从 supplier 创建有状态任务
// 从持久化数据恢复任务上下文
func NewStatefulTask(delegatedTaskSupplier func(*TaskState) core.Task, taskPersistence TaskPersistence,
	transactionable Transactionable, taskID string) *StatefulTask {
	snapshot := loadTaskSnapshot(taskPersistence, taskID)
	delegated := delegatedTaskSupplier(snapshot)
	t := &StatefulTask{
		DelegateTask:    delegate.NewDelegateTask(delegated),
		taskPersistence: taskPersistence,
		transactionable: transactionable,
	}
	type statusAware interface {
		GetStatus() core.TaskStatus
	}
	if sa, ok := delegated.(statusAware); ok {
		t.beforeExecuteStatus = sa.GetStatus()
	}
	return t
}

// NewStatefulTaskFromTask 包装已有任务为有状态任务
func NewStatefulTaskFromTask(delegatedTask core.Task, taskPersistence TaskPersistence,
	transactionable Transactionable) *StatefulTask {
	t := &StatefulTask{
		DelegateTask:    delegate.NewDelegateTask(delegatedTask),
		taskPersistence: taskPersistence,
		transactionable: transactionable,
	}
	type statusAware interface {
		GetStatus() core.TaskStatus
	}
	if sa, ok := delegatedTask.(statusAware); ok {
		t.beforeExecuteStatus = sa.GetStatus()
	}
	return t
}

// loadTaskSnapshot 从持久化数据加载任务快照, 不允许为空, 否则主动抛异常
func loadTaskSnapshot(taskPersistence TaskPersistence, taskID string) *TaskState {
	snapshot := taskPersistence.RecoverTask(taskID)
	return TaskStateFromSnapshot(snapshot)
}

func (t *StatefulTask) Execute() core.TaskResult {
	return t.ExecuteWithProcessors(func() {}, func() {})
}

// ExecuteWithProcessors 带前置/后置处理的执行
// 以默认的 REQUIRED 事务执行, StatefulTask 内的所有 StatefulTaskStep 都在事务范围内
func (t *StatefulTask) ExecuteWithProcessors(before, after func()) core.TaskResult {
	if nested, ok := t.Delegated.(*StatefulTask); ok {
		return nested.ExecuteWithProcessors(before, after)
	}
	result := t.transactionable.Execute(func(status TxStatus) any {
		defer t.saveTask()
		before()
		r := t.DelegateTask.Execute()
		after()
		return r
	})
	return result.(core.TaskResult)
}

func (t *StatefulTask) Cancel() {
	t.transactionable.Execute(func(status TxStatus) any {
		defer t.saveTask()
		t.DelegateTask.Cancel()
		return nil
	})
}

func (t *StatefulTask) Interrupt() {
	t.transactionable.Execute(func(status TxStatus) any {
		defer t.saveTask()
		t.DelegateTask.Interrupt()
		return nil
	})
}

// saveTask 保存任务快照, 终态不会重复保存
func (t *StatefulTask) saveTask() {
	if (t.beforeExecuteStatus == core.TaskSuccess || t.beforeExecuteStatus == core.TaskCanceled) &&
		t.GetDelegatedStatus() != t.beforeExecuteStatus {
		return
	}

	type fullTaskAware interface {
		GetIdempotentID() string
		GetDescription() core.Description
		GetTaskContext() *core.TaskContext
	}
	aware, ok := t.Delegated.(fullTaskAware)
	if !ok {
		log.Printf("StatefulTask: delegated task does not implement fullTaskAware, skip save")
		return
	}

	snapshot := &TaskSnapshot{
		TaskID:       t.GetTaskID(),
		IdempotentID: aware.GetIdempotentID(),
		Name:         t.GetName(),
		Type:         t.GetType(),
		Subject:      t.GetSubject(),
		PublishEnv:   t.GetPublishEnv(),
		Creator:      t.GetCreator(),
		Status:       t.GetDelegatedStatus(),
		Description:  aware.GetDescription(),
	}
	taskCtx := aware.GetTaskContext()
	if taskCtx != nil {
		snapshot.Context = taskCtx.GetContext()
	}
	t.taskPersistence.SaveTask(snapshot)
}

func (t *StatefulTask) GetDelegatedStatus() core.TaskStatus {
	type statusAware interface {
		GetStatus() core.TaskStatus
	}
	if sa, ok := t.Delegated.(statusAware); ok {
		return sa.GetStatus()
	}
	return core.TaskCreated
}
