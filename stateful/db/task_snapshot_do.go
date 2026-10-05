package db

import (
	"time"
)

// TaskSnapshotDO 任务快照数据库对象
type TaskSnapshotDO struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement"`
	GmtCreate   time.Time `gorm:"column:gmt_create"`
	GmtModified time.Time `gorm:"column:gmt_modified"`

	TaskID       string `gorm:"column:task_id"`
	IdempotentID string `gorm:"column:idempotent_id"` // 请求源提供的幂等 id
	Name         string `gorm:"column:name"`
	Type         string `gorm:"column:type"`
	PublishEnv   string `gorm:"column:publish_env"`
	Subject      string `gorm:"column:subject"`  // 任务目标主体
	Status       string `gorm:"column:status"`
	Context      string `gorm:"column:context"`
	Creator      string `gorm:"column:creator"` // 任务创建人
	Description  string `gorm:"column:description"`
}

func (TaskSnapshotDO) TableName() string {
	return "hzb_flow_task_snapshot"
}
