package db

import (
	"github.com/open-hzbank/ultra-flow/sync"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DefaultDbLockRepository 默认数据库锁仓库实现
type DefaultDbLockRepository struct {
	lockConfigService *sync.LockConfigService
	db                *gorm.DB
}

// NewDefaultDbLockRepository 创建默认数据库锁仓库
func NewDefaultDbLockRepository(db *gorm.DB, lockConfigService *sync.LockConfigService) *DefaultDbLockRepository {
	return &DefaultDbLockRepository{db: db, lockConfigService: lockConfigService}
}

// CreateLock 创建锁记录
func (r *DefaultDbLockRepository) CreateLock(lock *DbLock) error {
	do := dbLockToDO(lock)
	now := time.Now()
	do.GmtCreate = now
	do.GmtModified = now
	do.LockState = 0

	err := r.db.Create(do).Error
	if err != nil {
		// 判断是否唯一键冲突
		if isDuplicateKeyError(err) {
			return NewDuplicateLockException("锁已存在: "+lock.LockKey, err)
		}
		return err
	}
	return nil
}

// SelectLock 查询锁记录
func (r *DefaultDbLockRepository) SelectLock(lockKey string) *DbLock {
	var do DbLockDO
	if err := r.db.Where("lock_key = ?", lockKey).First(&do).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("查询锁记录失败: %v", err)
		}
		return nil
	}
	return dbLockFromDO(&do)
}

// SelectLockForUpdate 悲观锁查询
func (r *DefaultDbLockRepository) SelectLockForUpdate(lockKey string) *DbLock {
	var do DbLockDO
	err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("lock_key = ?", lockKey).
		First(&do).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("悲观锁查询锁记录失败: %v", err)
		}
		return nil
	}
	return dbLockFromDO(&do)
}

// TryLock 尝试获取锁
func (r *DefaultDbLockRepository) TryLock(lockKey, lockBy string) bool {
	timeoutSeconds := r.lockConfigService.GetLockTimeoutSeconds()
	result := r.db.Exec(
		`UPDATE hzb_flow_db_lock
		 SET lock_state = CASE
		   WHEN UNIX_TIMESTAMP(lock_time) < UNIX_TIMESTAMP(NOW()) - ? THEN 1
		   ELSE lock_state + 1
		 END,
		 lock_by = ?,
		 lock_time = NOW()
		 WHERE lock_key = ?
		   AND (lock_state <= 0
		        OR UNIX_TIMESTAMP(lock_time) < UNIX_TIMESTAMP(NOW()) - ?
		        OR lock_by = ?)`,
		timeoutSeconds, lockBy, lockKey, timeoutSeconds, lockBy,
	)
	return result.RowsAffected > 0
}

// UnLock 释放锁
func (r *DefaultDbLockRepository) UnLock(lockKey, lockBy string) bool {
	result := r.db.Exec(
		`UPDATE hzb_flow_db_lock SET lock_state = lock_state - 1 WHERE lock_key = ? AND lock_by = ?`,
		lockKey, lockBy,
	)
	return result.RowsAffected > 0
}

// RemoveLock 删除锁记录 (仅在未锁定时删除)
func (r *DefaultDbLockRepository) RemoveLock(lockKey string) bool {
	result := r.db.Where("lock_key = ? AND lock_state <= 0", lockKey).Delete(&DbLockDO{})
	return result.RowsAffected > 0
}

// 转换器: 领域对象 -> DO
func dbLockToDO(lock *DbLock) *DbLockDO {
	do := &DbLockDO{
		LockKey:     lock.LockKey,
		GmtCreate:   lock.GmtCreate,
		GmtModified: lock.GmtModified,
		LockBy:      lock.LockBy,
		LockTime:    lock.LockTime,
	}
	if lock.LockStatus == LockStatusLocked {
		do.LockState = 1
	} else {
		do.LockState = 0
	}
	return do
}

// 转换器: DO -> 领域对象
func dbLockFromDO(do *DbLockDO) *DbLock {
	if do == nil {
		return nil
	}
	lock := &DbLock{
		LockKey:     do.LockKey,
		GmtCreate:   do.GmtCreate,
		GmtModified: do.GmtModified,
		LockBy:      do.LockBy,
		LockTime:    do.LockTime,
	}
	if do.LockState > 0 {
		lock.LockStatus = LockStatusLocked
	} else {
		lock.LockStatus = LockStatusUnlocked
	}
	return lock
}

// isDuplicateKeyError 判断是否为唯一键冲突错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// MySQL 唯一键冲突错误码: 1062
	return contains(msg, "1062") || contains(msg, "Duplicate entry")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// 确保接口实现
var _ DbLockRepository = (*DefaultDbLockRepository)(nil)
