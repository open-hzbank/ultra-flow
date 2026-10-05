package ops

import (
	"fmt"
	"time"

	"github.com/open-hzbank/ultra-flow/core"
	flowsync "github.com/open-hzbank/ultra-flow/sync"
)

const lockWaitTime = 1000 * time.Millisecond

// OpsTaskRunnerFactory 运维任务运行器工厂
type OpsTaskRunnerFactory struct {
	lockService flowsync.LockService
}

func NewOpsTaskRunnerFactory(lockService flowsync.LockService) *OpsTaskRunnerFactory {
	return &OpsTaskRunnerFactory{lockService: lockService}
}

// DefaultTaskRunner 直接执行
func (f *OpsTaskRunnerFactory) DefaultTaskRunner() OpsTaskRunner {
	return &DefaultOpsTaskRunner{}
}

// SynchronizedTaskRunner 线程安全、全局互斥执行
func (f *OpsTaskRunnerFactory) SynchronizedTaskRunner(taskID string) OpsTaskRunner {
	lock := f.lockService.BuildLock(taskID)
	return NewSynchronizedOpsTaskRunner(lock)
}

// SynchronizedOpsTaskRunner 线程安全、全局互斥的运维任务运行器
type SynchronizedOpsTaskRunner struct {
	// 任务同步执行器不能使用 transaction + select for update 的方式实现互斥锁:
	// 因为当两个事务并发执行, 在数据库事务默认的 Repeatable-Read 隔离级别下, 第一个事务中对某条数据记录的更新, 在第二个事务中是不可见的
	// 如果两次并发的任务调度 (比如定时调度和用户点击取消 同时发生) 需要操纵同一条数据记录, 就可能导致数据可见性问题
	lock flowsync.BriefLock
}

func NewSynchronizedOpsTaskRunner(lock flowsync.BriefLock) *SynchronizedOpsTaskRunner {
	return &SynchronizedOpsTaskRunner{lock: lock}
}

func (r *SynchronizedOpsTaskRunner) Execute(builder OpsTaskBuilder, taskSupplier func(OpsTaskBuilder) core.Task) core.TaskResult {
	r.acquireLock()
	defer r.lock.Unlock()
	task := taskSupplier(builder)
	return task.Execute()
}

func (r *SynchronizedOpsTaskRunner) Cancel(builder OpsTaskBuilder, taskSupplier func(OpsTaskBuilder) core.Task) {
	r.acquireLock()
	defer r.lock.Unlock()
	task := taskSupplier(builder)
	task.Cancel()
}

func (r *SynchronizedOpsTaskRunner) Interrupt(builder OpsTaskBuilder, taskSupplier func(OpsTaskBuilder) core.Task) {
	r.acquireLock()
	defer r.lock.Unlock()
	task := taskSupplier(builder)
	task.Interrupt()
}

func (r *SynchronizedOpsTaskRunner) acquireLock() {
	if !r.lock.TryLockWithTimeout(lockWaitTime) {
		panic(&core.LockFailureError{Message: fmt.Sprintf("未成功获取到锁: %v", r.lock)})
	}
}
