package stateful

// DefaultTaskPersistence 任务数据持久化统一入口
// 请根据自己的项目情况使用合适的 TaskSnapshotRepository 和 TaskStepSnapshotRepository 作为仓库 & 数据源
type DefaultTaskPersistence struct {
	taskRepo      TaskSnapshotRepository
	taskStepRepo  TaskStepSnapshotRepository
}

func NewDefaultTaskPersistence(taskRepo TaskSnapshotRepository, taskStepRepo TaskStepSnapshotRepository) *DefaultTaskPersistence {
	return &DefaultTaskPersistence{
		taskRepo:     taskRepo,
		taskStepRepo: taskStepRepo,
	}
}

func (p *DefaultTaskPersistence) SaveTask(snapshot *TaskSnapshot) {
	p.taskRepo.SaveSnapshot(snapshot)
}

func (p *DefaultTaskPersistence) RecoverTask(taskID string) *TaskSnapshot {
	return p.taskRepo.Get(taskID)
}

func (p *DefaultTaskPersistence) SaveTaskStep(snapshot *TaskStepSnapshot) {
	p.taskStepRepo.Save(snapshot)
}

func (p *DefaultTaskPersistence) RecoverTaskStep(taskID, name string) *TaskStepSnapshot {
	return p.taskStepRepo.GetTaskStep(taskID, name)
}
