package db

// DbLockRepository 数据库锁仓库接口
type DbLockRepository interface {
	// CreateLock 创建锁记录
	CreateLock(lock *DbLock) error
	// SelectLock 查询锁记录
	SelectLock(lockKey string) *DbLock
	// SelectLockForUpdate 悲观锁查询
	SelectLockForUpdate(lockKey string) *DbLock
	// TryLock 尝试获取锁
	TryLock(lockKey, lockBy string) bool
	// UnLock 释放锁
	UnLock(lockKey, lockBy string) bool
	// RemoveLock 删除锁记录
	RemoveLock(lockKey string) bool
}
