package stateful

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
)

// StatefulTask 有状态的任务
type StatefulTask struct {
	*flowsdk.AbstractTask
	taskPersistence TaskPersistence
}

// NewStatefulTask 创建有状态的任务
func NewStatefulTask(taskId, name, type_, creator string, subject interface{}, publishEnv flowsdk.Env, idempotentId string, description flowsdk.Description, taskPersistence TaskPersistence) *StatefulTask {
	task := &StatefulTask{
		AbstractTask:     flowsdk.NewAbstractTask(taskId, name, type_, creator, subject, publishEnv, idempotentId, description),
		taskPersistence:  taskPersistence,
	}

	// 恢复任务状态
	task.recoverTask()

	return task
}

// Execute 执行任务
func (t *StatefulTask) Execute() flowsdk.TaskResult {
	defer t.saveTask()

	return t.AbstractTask.Execute()
}

// recoverTask 恢复任务状态
func (t *StatefulTask) recoverTask() {
	taskSnapshot := t.taskPersistence.RecoverTask(t.GetTaskId())
	if taskSnapshot != nil {
		// 恢复状态
		t.SetStatus(taskSnapshot.Status)
	}
}

// saveTask 保存任务状态
func (t *StatefulTask) saveTask() {
	taskSnapshot := &TaskSnapshot{
		TaskId:   t.GetTaskId(),
		Status:   t.GetStatus(),
		Context:  t.GetTaskContext(),
	}

	t.taskPersistence.SaveTask(taskSnapshot)
}

// TaskSnapshot 任务快照
type TaskSnapshot struct {
	TaskId  string
	Status  flowsdk.TaskStatus
	Context *flowsdk.TaskContext
}

// DefaultTaskPersistence 默认任务持久化实现
type DefaultTaskPersistence struct {}

// NewDefaultTaskPersistence 创建默认任务持久化实现
func NewDefaultTaskPersistence() *DefaultTaskPersistence {
	return &DefaultTaskPersistence{}
}

// SaveTask 保存任务快照
func (p *DefaultTaskPersistence) SaveTask(snapshot *TaskSnapshot) {
	// 默认实现，子类可以覆盖
}

// RecoverTask 恢复任务快照
func (p *DefaultTaskPersistence) RecoverTask(taskId string) *TaskSnapshot {
	// 默认实现，子类可以覆盖
	return nil
}

// SaveTaskStep 保存任务步骤快照
func (p *DefaultTaskPersistence) SaveTaskStep(snapshot *TaskStepSnapshot) {
	// 默认实现，子类可以覆盖
}

// RecoverTaskStep 恢复任务步骤快照
func (p *DefaultTaskPersistence) RecoverTaskStep(taskId, stepName string) *TaskStepSnapshot {
	// 默认实现，子类可以覆盖
	return nil
}
