package view

import (
	"github.com/open-hzbank/ultra-flow/core"
)

// TaskView 任务展示视图
type TaskView struct {
	TaskID  string    `json:"taskId"`
	Title   string    `json:"title"`
	Creator *UserView `json:"creator"`
	// 发布原因
	Reason     string          `json:"reason"`
	PublishEnv core.Env        `json:"publishEnv"`
	Status     core.TaskStatus `json:"status"`
	// 发布内容详情
	Detail      any    `json:"detail"`
	GmtCreate   *int64 `json:"gmtCreate"`
	GmtModified *int64 `json:"gmtModified"`
}
