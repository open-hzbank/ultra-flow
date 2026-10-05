package core

import (
	"fmt"
	"log"
	"sync/atomic"
)

// StepState 步骤共享状态管理
// 各具体步骤类型通过组合 StepState 来复用公共的状态管理逻辑
type StepState struct {
	Name            string
	StepCtx         *TaskStepContext
	TaskCtx         *TaskContext
	status          atomic.Value // TaskStepStatus
}

func NewStepState(name string, taskCtx *TaskContext, self TaskStep) *StepState {
	return NewStepStateWithTransfer(name, taskCtx, self, true)
}

func NewStepStateWithTransfer(name string, taskCtx *TaskContext, self TaskStep, needTransfer bool) *StepState {
	s := &StepState{
		Name:    name,
		TaskCtx: taskCtx,
	}
	s.status.Store(StepPending)
	s.StepCtx = NewTaskStepContext(taskCtx.GetTask(), self)
	if needTransfer {
		transfer := taskCtx.GetTaskStepTransfer()
		if transfer != nil {
			transfer.Transfer(self)
		}
	}
	return s
}

func (s *StepState) GetStatus() TaskStepStatus {
	return s.status.Load().(TaskStepStatus)
}

func (s *StepState) SetStatus(st TaskStepStatus) {
	s.status.Store(st)
}

// ExecuteStep 公共的步骤执行逻辑
// self 参数用于 addPath 和接口回调
func (s *StepState) ExecuteStep(self TaskStep, doExecute func() TaskStepResult) TaskStepResult {
	s.TaskCtx.AddPath(self)
	result := s.executeStepInner(self, doExecute)
	s.SetStatus(result.Status)
	return result
}

func (s *StepState) executeStepInner(self TaskStep, doExecute func() TaskStepResult) TaskStepResult {
	if !s.CanExecute() {
		return NewStepResult(s.GetStatus())
	}

	var result TaskStepResult
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Execute task step error, taskId: %s, stepName: %s, error: %v",
					s.TaskCtx.GetTask().GetTaskID(), s.Name, r)
				self.OnFailure(fmt.Errorf("%v", r))
				result = NewStepResultWithError(StepFailure, fmt.Errorf("%v", r), fmt.Sprintf("%v", r))
			}
		}()
		result = doExecute()
	}()

	switch result.Status {
	case StepPending, StepRunning:
		return NewStepResult(StepRunning)
	case StepSuccess:
		self.OnSuccess()
		return NewStepResult(StepSuccess)
	case StepFailure:
		self.OnFailure(result.Error)
		return NewStepResultWithMessage(StepFailure, result.Message)
	case StepInterrupted:
		self.OnInterrupt()
		return NewStepResult(StepInterrupted)
	case StepSkipped:
		return NewStepResult(StepSkipped)
	default:
		return NewStepResult(StepFailure)
	}
}

func (s *StepState) CanExecute() bool {
	st := s.GetStatus()
	return st == StepPending || st == StepRunning || st == StepFailure
}

func (s *StepState) CanInterrupt() bool {
	return s.GetStatus().IsExecuting()
}

// DefaultInterrupt 默认中断处理
func (s *StepState) DefaultInterrupt() {
	if !s.CanInterrupt() {
		return
	}
	s.SetStatus(StepInterrupted)
}
