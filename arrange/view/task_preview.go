package view

// TaskPreview 任务展示详情
type TaskPreview struct {
	Task *TaskView `json:"task"`
	// 不同阶段下的步骤详情
	TaskSteps []*StageView `json:"taskSteps"`
}

func NewTaskPreview(task *TaskView, taskSteps []*StageView) *TaskPreview {
	return &TaskPreview{Task: task, TaskSteps: taskSteps}
}
