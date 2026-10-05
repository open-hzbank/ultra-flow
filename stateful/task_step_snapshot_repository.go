package stateful

import "github.com/open-hzbank/ultra-flow/core"

// TaskStepSnapshotRepository 步骤快照仓库接口
type TaskStepSnapshotRepository interface {
	Save(snapshot *TaskStepSnapshot)
	GetTaskStep(taskID, name string) *TaskStepSnapshot
	GetTaskSteps(taskID string) []*TaskStepSnapshot
	GetByStepTypes(stepType string, statuses []core.TaskStepStatus, contextQueryParams map[string]any) []*TaskStepSnapshot
}
