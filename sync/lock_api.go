package sync

import (
	"fmt"
	stdsync "sync"
	"time"
)

// BriefLock 带名称的简短锁接口
// 仅支持 tryLock 和 unlock 操作, 不支持阻塞式锁定
type BriefLock interface {
	// TryLock 尝试获取锁, 立即返回
	TryLock() bool
	// TryLockWithTimeout 尝试在指定超时时间内获取锁
	TryLockWithTimeout(timeout time.Duration) bool
	// Unlock 释放锁
	Unlock()
}

// LockService 锁服务接口
type LockService interface {
	// BuildLock 构建指定名称的锁
	BuildLock(lockName string) BriefLock
}

// LockFailureException 获取锁失败时抛出的异常
type LockFailureException struct {
	Message string
}

func NewLockFailureException(message string) *LockFailureException {
	return &LockFailureException{Message: message}
}

func (e *LockFailureException) Error() string {
	return fmt.Sprintf("获取锁失败: %s", e.Message)
}

// UnLockFailureException 释放锁失败时抛出的异常
type UnLockFailureException struct {
	Message string
	Cause   error
}

func NewUnLockFailureException(message string, cause ...error) *UnLockFailureException {
	e := &UnLockFailureException{Message: message}
	if len(cause) > 0 {
		e.Cause = cause[0]
	}
	return e
}

func (e *UnLockFailureException) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("释放锁失败: %s, 原因: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("释放锁失败: %s", e.Message)
}

// LockConfigService 锁配置服务
type LockConfigService struct {
	// LockTimeoutSeconds 最长的占用锁时间 (默认 24 秒)
	LockTimeoutSeconds int
	// RetryIntervalMillis 重新尝试获取锁的间隔时间 (默认 200 毫秒)
	RetryIntervalMillis int64
	mu                  stdsync.RWMutex
}

func NewLockConfigService() *LockConfigService {
	return &LockConfigService{
		LockTimeoutSeconds:  24,
		RetryIntervalMillis: 200,
	}
}

// LoadConfig 从 properties 格式的配置字符串加载配置
func (s *LockConfigService) LoadConfig(config string) {
	props := parseProperties(config)
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := props["lock.timeout.seconds"]; ok {
		if n, err := parseInt(v, 24); err == nil {
			s.LockTimeoutSeconds = n
		}
	}
	if v, ok := props["lock.retry.millis"]; ok {
		if n, err := parseInt(v, 200); err == nil {
			s.RetryIntervalMillis = int64(n)
		}
	}
}

func (s *LockConfigService) GetLockTimeoutSeconds() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LockTimeoutSeconds
}

func (s *LockConfigService) GetRetryIntervalMillis() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RetryIntervalMillis
}

// ForceTimeoutDelegateLock 强制超时委托锁
// 包装一个底层锁, 强制使用超时方式获取
type ForceTimeoutDelegateLock struct {
	// defaultTryLockWaitTime 默认的尝试获取锁的等待时间
	defaultTryLockWaitTime time.Duration
	// delegatedLock 被代理的锁实例
	delegatedLock BriefLock
	// lockTarget 锁的作用目标
	lockTarget string
}

func NewForceTimeoutDelegateLock(defaultTryLockWaitTime time.Duration, delegatedLock BriefLock, lockTarget string) *ForceTimeoutDelegateLock {
	return &ForceTimeoutDelegateLock{
		defaultTryLockWaitTime: defaultTryLockWaitTime,
		delegatedLock:          delegatedLock,
		lockTarget:             lockTarget,
	}
}

func (l *ForceTimeoutDelegateLock) TryLock() bool {
	return l.delegatedLock.TryLockWithTimeout(l.defaultTryLockWaitTime)
}

func (l *ForceTimeoutDelegateLock) TryLockWithTimeout(timeout time.Duration) bool {
	return l.delegatedLock.TryLockWithTimeout(timeout)
}

func (l *ForceTimeoutDelegateLock) Unlock() {
	l.delegatedLock.Unlock()
}

// 辅助函数: 解析 properties 格式的配置
func parseProperties(config string) map[string]string {
	result := make(map[string]string)
	lines := splitLines(config)
	for _, line := range lines {
		line = trimSpace(line)
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		idx := indexOf(line, '=')
		if idx < 0 {
			idx = indexOf(line, ':')
		}
		if idx >= 0 {
			key := trimSpace(line[:idx])
			value := trimSpace(line[idx+1:])
			result[key] = value
		}
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func parseInt(s string, defaultVal int) (int, error) {
	if s == "" {
		return defaultVal, nil
	}
	n := 0
	sign := 1
	start := 0
	if s[0] == '-' {
		sign = -1
		start = 1
	} else if s[0] == '+' {
		start = 1
	}
	for i := start; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return defaultVal, fmt.Errorf("invalid number: %s", s)
		}
		n = n*10 + int(s[i]-'0')
	}
	return sign * n, nil
}
