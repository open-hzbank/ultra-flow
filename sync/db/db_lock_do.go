package db

import "time"

// DbLockDO 数据库锁数据对象
type DbLockDO struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement"`
	GmtCreate   time.Time `gorm:"column:gmt_create"`
	GmtModified time.Time `gorm:"column:gmt_modified"`
	LockKey     string    `gorm:"column:lock_key"`
	LockBy      string    `gorm:"column:lock_by"`
	LockTime    time.Time `gorm:"column:lock_time"`
	LockState   int64     `gorm:"column:lock_state"`
}

func (DbLockDO) TableName() string {
	return "hzb_flow_db_lock"
}
