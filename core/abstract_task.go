package core

import (
	"fmt"
	"log"
	"sync/atomic"
)

// AbstractTask 任务为多例模式，每次任务调用会创建任务实例，任务实例维持自有的上下文和状态
type AbstractTask struct {
	taskID       string
	name         string
	taskType     string
	subject      any
	creator      string
	publishEnv   Env
	idempotentID string
	description  Description
	taskContext  *TaskContext
	status       atomic.Value // TaskStatus
	taskStep     TaskStep
}

type AbstractTaskConfig struct {
	TaskID       string
	Name         string
	Type         string
	Subject      any
	Creator      string
	PublishEnv   Env
	IdempotentID string
	Status       TaskStatus
	Description  Description
	Context      map[string]any
	Transfer     TaskStepTransfer
}

func NewAbstractTask(cfg AbstractTaskConfig) *AbstractTask {
	t := &AbstractTask{
		taskID:       cfg.TaskID,
		name:         cfg.Name,
		taskType:     cfg.Type,
		subject:      cfg.Subject,
		creator:      cfg.Creator,
		publishEnv:   cfg.PublishEnv,
		idempotentID: cfg.IdempotentID,
		description:  cfg.Description,
	}
	t.status.Store(cfg.Status)
	t.taskContext = NewTaskContextWithTransfer(t, cfg.Context, cfg.Transfer)
	return t
}

// NewAbstractTaskWithStep 创建带步骤的任务
func NewAbstractTaskWithStep(cfg AbstractTaskConfig, stepSupplier func(*TaskContext) TaskStep) *AbstractTask {
	if cfg.Status == 0 {
		cfg.Status = TaskCreated
	}
	t := NewAbstractTask(cfg)
	t.taskStep = stepSupplier(t.taskContext)
	return t
}

// CopyFromDelegated 从代理任务复制属性
func (t *AbstractTask) CopyFromDelegated(d *AbstractTask) {
	t.taskID = d.taskID
	t.name = d.name
	t.taskType = d.taskType
	t.subject = d.subject
	t.creator = d.creator
	t.publishEnv = d.publishEnv
	t.idempotentID = d.idempotentID
	t.description = d.description
	t.status.Store(d.GetStatus())
	t.taskContext = d.taskContext
	t.taskStep = d.taskStep
}

func (t *AbstractTask) GetTaskID() string       { return t.taskID }
func (t *AbstractTask) GetName() string          { return t.name }
func (t *AbstractTask) GetType() string          { return t.taskType }
func (t *AbstractTask) GetSubject() any          { return t.subject }
func (t *AbstractTask) GetCreator() string       { return t.creator }
func (t *AbstractTask) GetPublishEnv() Env       { return t.publishEnv }
func (t *AbstractTask) GetDescription() Description { return t.description }
func (t *AbstractTask) GetIdempotentID() string  { return t.idempotentID }
func (t *AbstractTask) GetTaskContext() *TaskContext { return t.taskContext }

func (t *AbstractTask) GetStatus() TaskStatus {
	return t.status.Load().(TaskStatus)
}

func (t *AbstractTask) SetStatus(s TaskStatus) {
	t.status.Store(s)
}

func (t *AbstractTask) SetTaskStep(step TaskStep) {
	t.taskStep = step
}

func (t *AbstractTask) GetTaskStep() TaskStep {
	return t.taskStep
}

func (t *AbstractTask) InitTaskStep(supplier func(*TaskContext) TaskStep) {
	t.taskStep = supplier(t.taskContext)
}

func (t *AbstractTask) Execute() TaskResult {
	result := t.ExecuteTask()
	t.SetStatus(result.Status)
	return result
}

// ExecuteWithProcessors 带前置/后置处理的执行
func (t *AbstractTask) ExecuteWithProcessors(before, after func()) TaskResult {
	before()
	result := t.Execute()
	after()
	return result
}

// ExecuteTask 任务执行核心逻辑 (可被子类覆盖)
func (t *AbstractTask) ExecuteTask() TaskResult {
	if !t.CanExecute() {
		return NewTaskResult(t.taskID, t.GetStatus())
	}
	if t.taskStep == nil {
		return NewTaskResultWithError(t.taskID, TaskFailure,
			fmt.Errorf("taskStep is nil"), "任务步骤未初始化")
	}

	stepResult := t.taskStep.Execute()
	switch stepResult.Status {
	case StepPending, StepRunning:
		return NewTaskResult(t.taskID, TaskRunning)
	case StepSkipped, StepSuccess:
		return NewTaskResult(t.taskID, TaskSuccess)
	case StepFailure:
		t.taskStep.OnFailure(stepResult.Error)
		return NewTaskResultWithError(t.taskID, TaskFailure, stepResult.Error, stepResult.Message)
	case StepInterrupted:
		// 业务步骤主动返回 INTERRUPTED, 触发任务整体取消
		t.taskStep.OnInterrupt()
		return NewTaskResult(t.taskID, TaskCanceled)
	default:
		return NewTaskResultWithError(t.taskID, TaskFailure,
			fmt.Errorf("unexpected step status: %v", stepResult.Status), "未知步骤状态")
	}
}

// CanExecute 允许执行的状态，可覆盖，用于重试等策略
func (t *AbstractTask) CanExecute() bool {
	s := t.GetStatus()
	return s == TaskCreated || s == TaskRunning || s == TaskFailure
}

func (t *AbstractTask) Cancel() {
	// 幂等返回
	if t.GetStatus() == TaskCanceled {
		return
	}
	// 终态不允许取消
	if t.GetStatus().IsEnd() {
		panic("已到达终态的任务无法取消")
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Cancel task error, taskId: %s, error: %v", t.taskID, r)
			t.SetStatus(TaskFailure)
			t.taskStep.OnFailure(fmt.Errorf("%v", r))
		}
	}()
	t.taskStep.OnInterrupt()
	t.SetStatus(TaskCanceled)
}

func (t *AbstractTask) Interrupt() {
	// 幂等返回
	if t.GetStatus() == TaskInterrupted {
		return
	}
	// 终态不允许中断
	if t.GetStatus().IsEnd() {
		panic("已到达终态的任务无法中断")
	}
	t.taskStep.OnInterrupt()
	t.SetStatus(TaskInterrupted)
}

// IdentityProcessor 前/后置 空处理实现, 外部可直接使用
var IdentityProcessor = func() {}
