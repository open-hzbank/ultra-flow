package sync

import (
	stdsync "sync"
	"time"
)

// NamedLockManager 命名锁管理器
type NamedLockManager struct {
	mu    stdsync.Mutex
	locks map[string]*stdsync.Mutex
}

func NewNamedLockManager() *NamedLockManager {
	return &NamedLockManager{
		locks: make(map[string]*stdsync.Mutex),
	}
}

func (m *NamedLockManager) GetLock(name string) *stdsync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	lock, ok := m.locks[name]
	if !ok {
		lock = &stdsync.Mutex{}
		m.locks[name] = lock
	}
	return lock
}

// MemoryLockService 基于内存的锁服务实现
type MemoryLockService struct {
	lockManager *NamedLockManager
}

func NewMemoryLockService() *MemoryLockService {
	return &MemoryLockService{
		lockManager: NewNamedLockManager(),
	}
}

func (s *MemoryLockService) BuildLock(lockName string) BriefLock {
	mu := s.lockManager.GetLock(lockName)
	return NewMemoryBriefLock(lockName, mu)
}

// MemoryBriefLock 基于内存的 BriefLock 实现
// 允许指定目标资源名的可重入锁实现
type MemoryBriefLock struct {
	name string
	mu   *stdsync.Mutex
}

func NewMemoryBriefLock(name string, mu *stdsync.Mutex) *MemoryBriefLock {
	return &MemoryBriefLock{name: name, mu: mu}
}

func (l *MemoryBriefLock) TryLock() bool {
	return l.mu.TryLock()
}

func (l *MemoryBriefLock) TryLockWithTimeout(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if l.mu.TryLock() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (l *MemoryBriefLock) Unlock() {
	l.mu.Unlock()
}

func (l *MemoryBriefLock) String() string {
	return "MemoryBriefLock{" + l.name + "}"
}
