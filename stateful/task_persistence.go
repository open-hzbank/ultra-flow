package stateful

// TaskPersistence 任务数据持久化接口
type TaskPersistence interface {
	SaveTask(snapshot *TaskSnapshot)
	RecoverTask(taskID string) *TaskSnapshot
	SaveTaskStep(snapshot *TaskStepSnapshot)
	RecoverTaskStep(taskID, name string) *TaskStepSnapshot
}
