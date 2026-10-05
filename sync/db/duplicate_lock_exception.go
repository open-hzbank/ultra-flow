package db

import "fmt"

// DuplicateLockException 重复创建锁异常
type DuplicateLockException struct {
	Message string
	Cause   error
}

func NewDuplicateLockException(message string, cause error) *DuplicateLockException {
	return &DuplicateLockException{Message: message, Cause: cause}
}

func (e *DuplicateLockException) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("重复锁异常: %s, 原因: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("重复锁异常: %s", e.Message)
}

func (e *DuplicateLockException) Unwrap() error {
	return e.Cause
}
