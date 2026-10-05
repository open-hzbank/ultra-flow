package core

import (
	"encoding/json"
	"fmt"
)

// TaskStepStatus 步骤的状态枚举
type TaskStepStatus int

const (
	StepPending     TaskStepStatus = iota // 步骤尚未开始
	StepRunning                           // 步骤正在执行
	StepSuccess                           // 步骤执行成功
	StepSkipped                           // 步骤被跳过
	StepFailure                           // 步骤执行失败
	StepInterrupted                       // 步骤被中断 (比如任务被取消)
)

func (s TaskStepStatus) String() string {
	switch s {
	case StepPending:
		return "PENDING"
	case StepRunning:
		return "RUNNING"
	case StepSuccess:
		return "SUCCESS"
	case StepSkipped:
		return "SKIPPED"
	case StepFailure:
		return "FAILURE"
	case StepInterrupted:
		return "INTERRUPTED"
	default:
		return "UNKNOWN"
	}
}

var stepStatusValues = map[string]TaskStepStatus{
	"PENDING":     StepPending,
	"RUNNING":     StepRunning,
	"SUCCESS":     StepSuccess,
	"SKIPPED":     StepSkipped,
	"FAILURE":     StepFailure,
	"INTERRUPTED": StepInterrupted,
}

func (s TaskStepStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *TaskStepStatus) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	if v, ok := stepStatusValues[name]; ok {
		*s = v
		return nil
	}
	return fmt.Errorf("unknown TaskStepStatus: %s", name)
}

var executingStatuses = map[TaskStepStatus]bool{StepRunning: true, StepFailure: true}
var activeStatuses = map[TaskStepStatus]bool{StepPending: true, StepRunning: true, StepFailure: true}
var processedStatuses = map[TaskStepStatus]bool{
	StepSuccess: true, StepRunning: true, StepFailure: true, StepInterrupted: true, StepSkipped: true,
}
var normalEndedStatuses = map[TaskStepStatus]bool{StepSuccess: true, StepSkipped: true}
var endedStatuses = map[TaskStepStatus]bool{StepSuccess: true, StepSkipped: true, StepInterrupted: true}
var notExecutedStatuses = map[TaskStepStatus]bool{StepPending: true, StepSkipped: true}

// NeedCompensate 状态是否需要被补偿
func NeedCompensate(status TaskStepStatus) bool {
	return status == StepSuccess || status == StepInterrupted ||
		status == StepFailure ||
		// 有些步骤不支持 cancel, 只能等待其执行成功或手动去第三方系统取消
		status == StepRunning
}

func (s TaskStepStatus) NotStarted() bool { return s == StepPending }

// NotExecuted 没有被实际执行过的状态
func (s TaskStepStatus) NotExecuted() bool { return notExecutedStatuses[s] }

// IsExecuting 是否是正在执行中的状态
func (s TaskStepStatus) IsExecuting() bool { return executingStatuses[s] }

// IsProcessed 是否是已被受理的状态
func (s TaskStepStatus) IsProcessed() bool     { return processedStatuses[s] }
func (s TaskStepStatus) IsCanceled() bool      { return s == StepInterrupted }
func (s TaskStepStatus) IsFail() bool          { return s == StepFailure }
func (s TaskStepStatus) IsNormalRunning() bool { return s == StepRunning }
func (s TaskStepStatus) IsActive() bool        { return activeStatuses[s] }
func (s TaskStepStatus) NormalEnded() bool     { return normalEndedStatuses[s] }

// Ended 已结束的状态, 包括正常结束及非正常结束
func (s TaskStepStatus) Ended() bool   { return endedStatuses[s] }
func (s TaskStepStatus) Skipped() bool { return s == StepSkipped }

// ActiveStepStatuses 未到达终态的活跃状态
func ActiveStepStatuses() []TaskStepStatus {
	return []TaskStepStatus{StepPending, StepRunning, StepFailure}
}
