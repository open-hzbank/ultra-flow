package db

import "time"

// LockStatus 锁状态
type LockStatus int

const (
	LockStatusLocked   LockStatus = iota // LOCK: 当前处于锁定状态
	LockStatusUnlocked                   // UNLOCK: 当前处于非锁定状态
)

// DbLock 数据库锁领域模型
type DbLock struct {
	// LockKey 序列名
	LockKey string
	// GmtCreate 创建时间
	GmtCreate time.Time
	// GmtModified 修改时间
	GmtModified time.Time
	// LockBy 用于表示当前获得该锁的角色，需要全局唯一
	LockBy string
	// LockTime 获取该锁的时间
	LockTime time.Time
	// LockStatus 锁状态
	LockStatus LockStatus
}
