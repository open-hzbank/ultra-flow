package access

import (
	"hzbank.com.cn/ultra-flow/arrange/view"
	"hzbank.com.cn/ultra-flow/core"
)

// TaskBrief 任务简要信息 (列表展示用)
type TaskBrief struct {
	GmtCreate   int64            `json:"gmtCreate"`
	GmtModified int64            `json:"gmtModified"`
	TaskID      string           `json:"taskId"`
	PublishEnv  core.Env         `json:"publishEnv"`
	Creator     *view.UserView   `json:"creator"`
	Reason      string           `json:"reason"`
	Biz         string           `json:"biz"`
	TaskType    core.TaskType    `json:"taskType"`
	Status      core.TaskStatus  `json:"status"`
	CanRollback bool             `json:"canRollback"`
}

func NewTaskBrief(gmtCreate, gmtModified int64, taskID string, publishEnv core.Env,
	creator *view.UserView, reason, biz string, taskType core.TaskType,
	status core.TaskStatus, canRollback bool) *TaskBrief {
	return &TaskBrief{
		GmtCreate:   gmtCreate,
		GmtModified: gmtModified,
		TaskID:      taskID,
		PublishEnv:  publishEnv,
		Creator:     creator,
		Reason:      reason,
		Biz:         biz,
		TaskType:    taskType,
		Status:      status,
		CanRollback: canRollback,
	}
}
