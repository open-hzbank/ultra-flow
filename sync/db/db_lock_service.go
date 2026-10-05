package db

import (
	"time"
	"hzbank.com.cn/ultra-flow/sync"

	"gorm.io/gorm"
)

// DbLockService 基于数据库的锁服务
type DbLockService struct {
	db                *gorm.DB
	dbLockRepository  DbLockRepository
	lockConfigService *sync.LockConfigService
}

// NewDbLockService 创建基于数据库的锁服务
func NewDbLockService(db *gorm.DB, dbLockRepository DbLockRepository, lockConfigService *sync.LockConfigService) *DbLockService {
	return &DbLockService{
		db:                db,
		dbLockRepository:  dbLockRepository,
		lockConfigService: lockConfigService,
	}
}

// BuildLock 构建指定名称的锁
func (s *DbLockService) BuildLock(lockName string) sync.BriefLock {
	reentrantLock := NewDBReentrantLock(
		lockName,
		s.db,
		s.dbLockRepository,
		s.lockConfigService.GetRetryIntervalMillis(),
	)
	timeout := time.Duration(s.lockConfigService.GetLockTimeoutSeconds()) * time.Second
	return sync.NewForceTimeoutDelegateLock(timeout, reentrantLock, lockName)
}

// 确保接口实现
var _ sync.LockService = (*DbLockService)(nil)
