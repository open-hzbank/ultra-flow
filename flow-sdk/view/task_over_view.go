package view

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
)

// TaskOverView 任务概要视图
type TaskOverView struct {
	TaskId        string
	Name          string
	Type          string
	Status        flowsdk.TaskStatus
	Creator       string
	PublishEnv    flowsdk.Env
	CreateTime    int64
	UpdateTime    int64
	StepOverViews []StepOverView
}

// StepOverView 步骤概要视图
type StepOverView struct {
	Name        string
	Type        string
	Status      flowsdk.TaskStepStatus
	StartTime   int64
	EndTime     int64
	ErrorMessage string
}

// OperationType 操作类型
type OperationType string

const (
	OperationTypeCreate  OperationType = "CREATE"
	OperationTypeUpdate  OperationType = "UPDATE"
	OperationTypeDelete  OperationType = "DELETE"
	OperationTypeExecute OperationType = "EXECUTE"
	OperationTypeCancel  OperationType = "CANCEL"
)

// TaskRollbackRequest 任务回滚请求
type TaskRollbackRequest struct {
	TaskId    string
	Reason    string
	Operator  string
	Operation OperationType
}
