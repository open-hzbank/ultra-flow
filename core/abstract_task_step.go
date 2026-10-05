package core

import (
	"fmt"
	"log"
	"sync/atomic"
)

// AbstractTaskStep 步骤基础实现
type AbstractTaskStep struct {
	// name 任务名称，全局惟一
	name            string
	// 任务步骤的上下文，用于存储当前任务步骤的上下文变量，不允许任务步骤之间传递
	taskStepContext *TaskStepContext
	// 任务上下文
	taskContext *TaskContext
	// 任务步骤的状态
	status          atomic.Value // TaskStepStatus
	DoExecuteFunc   func() TaskStepResult // 子类设置的具体执行逻辑
}

func NewAbstractTaskStep(name string, taskCtx *TaskContext) *AbstractTaskStep {
	return NewAbstractTaskStepWithTransfer(name, taskCtx, true)
}

func NewAbstractTaskStepWithTransfer(name string, taskCtx *TaskContext, needTransfer bool) *AbstractTaskStep {
	s := &AbstractTaskStep{
		name:        name,
		taskContext: taskCtx,
	}
	s.status.Store(StepPending)
	s.taskStepContext = NewTaskStepContext(taskCtx.GetTask(), s)
	if needTransfer {
		// 外部传递设置任务步骤
		transfer := taskCtx.GetTaskStepTransfer()
		if transfer != nil {
			transfer.Transfer(s)
		}
	}
	return s
}

func (s *AbstractTaskStep) GetName() string { return s.name }

func (s *AbstractTaskStep) GetStatus() TaskStepStatus {
	return s.status.Load().(TaskStepStatus)
}

func (s *AbstractTaskStep) SetStatus(status TaskStepStatus) {
	s.status.Store(status)
}

func (s *AbstractTaskStep) GetTaskStepContext() *TaskStepContext { return s.taskStepContext }
func (s *AbstractTaskStep) GetTaskContext() *TaskContext          { return s.taskContext }

// Execute 步骤执行入口
func (s *AbstractTaskStep) Execute() TaskStepResult {
	s.taskContext.AddPath(s)
	result := s.ExecuteStep()
	s.SetStatus(result.Status)
	return result
}

// ExecuteStep 内部执行逻辑
func (s *AbstractTaskStep) ExecuteStep() TaskStepResult {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Execute task step error, taskId: %s, stepName: %s, error: %v",
				s.taskContext.GetTask().GetTaskID(), s.name, r)
		}
	}()

	if !s.CanExecute() {
		return NewStepResult(s.GetStatus())
	}

	result := s.DoExecute()
	switch result.Status {
	case StepPending, StepRunning:
		return NewStepResult(StepRunning)
	case StepSuccess:
		s.OnSuccess()
		return NewStepResult(StepSuccess)
	case StepFailure:
		s.OnFailure(result.Error)
		return NewStepResultWithMessage(StepFailure, result.Message)
	case StepInterrupted:
		s.OnInterrupt()
		return NewStepResult(StepInterrupted)
	case StepSkipped:
		return NewStepResult(StepSkipped)
	default:
		return NewStepResultWithError(StepFailure,
			fmt.Errorf("unexpected status: %v", result.Status), "未知步骤状态")
	}
}

// CanExecute 任务可执行的状态，可覆盖，如失败是否继续重试等处理策略
func (s *AbstractTaskStep) CanExecute() bool {
	st := s.GetStatus()
	return st == StepPending || st == StepRunning || st == StepFailure
}

// DoExecute 子类实现的具体执行逻辑
func (s *AbstractTaskStep) DoExecute() TaskStepResult {
	if s.DoExecuteFunc != nil {
		return s.DoExecuteFunc()
	}
	return NewStepResult(StepPending)
}

// DoExecute 子类实现的具体执行逻辑 (通过接口注入)
// 这个字段在构造时由子类设置
var _ TaskStep = (*AbstractTaskStep)(nil)

func (s *AbstractTaskStep) OnSuccess()              {}
func (s *AbstractTaskStep) OnFailure(err error)     {}
// OnInterrupt todo 所有步骤类型都不应该直接重写 OnInterrupt 方法, 而是重写 DoInterrupt 方法
// 后续重构需要优化掉
func (s *AbstractTaskStep) OnInterrupt() {
	// 非运行态不能中断
	if !s.CanInterrupt() {
		return
	}
	s.SetStatus(s.DoInterrupt())
}

func (s *AbstractTaskStep) CanInterrupt() bool {
	return s.GetStatus().IsExecuting()
}

func (s *AbstractTaskStep) DoInterrupt() TaskStepStatus {
	return StepInterrupted
}

// GetType 子类需覆盖
func (s *AbstractTaskStep) GetType() string { return "abstract" }

// Describe 子类需覆盖
func (s *AbstractTaskStep) Describe() StepDescription {
	return NewStepDescription(s.name, nil)
}
