package view

import "github.com/open-hzbank/ultra-flow/core"

// TaskOverView 任务总览视图
type TaskOverView struct {
	Operation   OperationType `json:"operation"`
	TaskType    core.TaskType `json:"taskType"`
	Task        *TaskView     `json:"task"`
	TaskSteps   []*StageView  `json:"taskSteps"`
	CanRollback bool          `json:"canRollback"`
}

func NewTaskOverView(operation OperationType, taskType core.TaskType, task *TaskView, taskSteps []*StageView, canRollback bool) *TaskOverView {
	return &TaskOverView{
		Operation:   operation,
		TaskType:    taskType,
		Task:        task,
		TaskSteps:   taskSteps,
		CanRollback: canRollback,
	}
}
