package stateful

import (
	"github.com/open-hzbank/ultra-flow/core"
	"time"
)

// TaskState flow 模块内部使用的任务信息传递结构
// TaskSnapshot 公开对外透露, 部分字段的存取有结构性限制
// 本类仅用于内部信息传递, 所有字段透传
type TaskState struct {
	ID           int64
	GmtCreate    time.Time
	GmtModified  time.Time
	TaskID       string
	IdempotentID string
	Name         string
	Type         string
	Subject      any
	PublishEnv   core.Env
	Context      map[string]any
	Creator      string
	Status       core.TaskStatus
	Description  core.Description
}

func TaskStateFromSnapshot(snapshot *TaskSnapshot) *TaskState {
	if snapshot == nil {
		return nil
	}
	return &TaskState{
		ID:           snapshot.ID,
		GmtCreate:    snapshot.GmtCreate,
		GmtModified:  snapshot.GmtModified,
		TaskID:       snapshot.TaskID,
		IdempotentID: snapshot.IdempotentID,
		Name:         snapshot.Name,
		Type:         snapshot.Type,
		Subject:      snapshot.Subject,
		PublishEnv:   snapshot.PublishEnv,
		Context:      snapshot.Context,
		Creator:      snapshot.Creator,
		Status:       snapshot.Status,
		Description:  snapshot.Description,
	}
}
