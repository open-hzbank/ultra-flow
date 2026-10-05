package db

import (
	"fmt"
	"log"
	"time"
	"hzbank.com.cn/ultra-flow/sync"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DBReentrantLock 基于数据库的可重入锁实现
//
// 锁获取逻辑:
//   - lock_state <= 0: 可以获取; lock_state = lock_state + 1
//   - lock_state > 0 且 lock_owner 匹配: 可以获取 (重入); lock_state = lock_state + 1
//   - lock_state > 0 且 lock_time 超时: 可以获取 (强制); lock_state = 1
type DBReentrantLock struct {
	lockKey             string
	retryIntervalMillis int64
	lockByHolder        string
	dbLockRepository    DbLockRepository
	db                  *gorm.DB
}

// NewDBReentrantLock 创建数据库可重入锁
func NewDBReentrantLock(lockKey string, db *gorm.DB, dbLockRepository DbLockRepository, retryIntervalMillis int64) *DBReentrantLock {
	lock := &DBReentrantLock{
		lockKey:             lockKey,
		retryIntervalMillis: retryIntervalMillis,
		dbLockRepository:    dbLockRepository,
		db:                  db,
	}
	lock.initDbLockIfNotPresent(lockKey)
	return lock
}

// initDbLockIfNotPresent 初始化锁记录 (如果不存在)
func (l *DBReentrantLock) initDbLockIfNotPresent(lockKey string) {
	err := l.db.Transaction(func(tx *gorm.DB) error {
		var do DbLockDO
		err := tx.Where("lock_key = ?", lockKey).First(&do).Error
		if err == gorm.ErrRecordNotFound {
			newLock := &DbLock{LockKey: lockKey}
			if createErr := l.dbLockRepository.CreateLock(newLock); createErr != nil {
				if _, ok := createErr.(*DuplicateLockException); ok {
					return nil
				}
				return createErr
			}
			return nil
		}
		return err
	})
	if err != nil {
		log.Printf("初始化锁记录失败: %v", err)
	}
}

// TryLock 尝试获取锁, 立即返回
func (l *DBReentrantLock) TryLock() bool {
	var success bool
	err := l.db.Transaction(func(tx *gorm.DB) error {
		lockBy := l.lockBy()
		l.initDbLockIfNotPresent(l.lockKey)
		success = l.dbLockRepository.TryLock(l.lockKey, lockBy)
		if !success {
			l.cleanLockBy(true)
		}
		return nil
	})
	if err != nil {
		log.Printf("尝试获取锁失败: %v", err)
		return false
	}
	return success
}

// TryLockWithTimeout 尝试在指定超时时间内获取锁
func (l *DBReentrantLock) TryLockWithTimeout(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if l.TryLock() {
			return true
		}
		time.Sleep(time.Duration(l.retryIntervalMillis) * time.Millisecond)
	}
	return false
}

// Unlock 释放锁
func (l *DBReentrantLock) Unlock() {
	var unlockErr error
	err := l.db.Transaction(func(tx *gorm.DB) error {
		lock := l.dbLockRepository.SelectLockForUpdate(l.lockKey)
		if lock == nil {
			return nil
		}
		success := l.dbLockRepository.UnLock(l.lockKey, l.lockByHolder)
		if !success {
			unlockErr = sync.NewUnLockFailureException(fmt.Sprintf("解锁失败: lockKey=%s", l.lockKey))
			return nil
		}
		if lock.LockStatus == LockStatusUnlocked {
			l.dbLockRepository.RemoveLock(l.lockKey)
		}
		return nil
	})
	if err != nil {
		log.Printf("解锁事务失败: %v", err)
	}
	if unlockErr != nil {
		panic(unlockErr)
	}
	l.cleanLockBy(false)
}

// lockBy 获取或创建当前锁持有者标识
func (l *DBReentrantLock) lockBy() string {
	if l.lockByHolder == "" {
		l.lockByHolder = GetLocalHostIp() + "_" + uuid.New().String()
	}
	return l.lockByHolder
}

// cleanLockBy 清理锁持有者标识
func (l *DBReentrantLock) cleanLockBy(forceClean bool) {
	if forceClean {
		l.lockByHolder = ""
		return
	}
	lock := l.dbLockRepository.SelectLock(l.lockKey)
	if lock == nil || lock.LockStatus == LockStatusUnlocked || lock.LockBy != l.lockByHolder {
		l.lockByHolder = ""
	}
}

func (l *DBReentrantLock) String() string {
	return fmt.Sprintf("DBReentrantLock {lockKey = '%s'}", l.lockKey)
}

// 确保接口实现
var _ sync.BriefLock = (*DBReentrantLock)(nil)
