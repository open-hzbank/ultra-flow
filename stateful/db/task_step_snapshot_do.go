package db

import (
	"time"
)

// TaskStepSnapshotDO 任务步骤快照数据库对象
type TaskStepSnapshotDO struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement"`
	GmtCreate   time.Time `gorm:"column:gmt_create"`
	GmtModified time.Time `gorm:"column:gmt_modified"`

	TaskID          string `gorm:"column:task_id"`
	Name            string `gorm:"column:name"`
	Type            string `gorm:"column:type"`
	Status          string `gorm:"column:status"`
	TaskStepContext string `gorm:"column:task_step_context"`
	Description     string `gorm:"column:description"`
}

func (TaskStepSnapshotDO) TableName() string {
	return "hzb_flow_task_step_snapshot"
}
