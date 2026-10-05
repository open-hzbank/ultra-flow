package core

import (
	"encoding/json"
	"fmt"
)

// TaskStatus 任务的状态枚举
type TaskStatus int

const (
	TaskCreated     TaskStatus = iota // 已创建
	TaskRunning                       // 执行中
	TaskSuccess                       // 执行成功
	TaskFailure                       // 执行失败
	TaskCancelling                    // 取消中 (用于状态补偿)
	TaskCanceled                      // 已取消
	TaskInterrupted                   // 被强制中断
)

var taskStatusNames = map[TaskStatus]string{
	TaskCreated:     "CREATED",
	TaskRunning:     "RUNNING",
	TaskSuccess:     "SUCCESS",
	TaskFailure:     "FAILURE",
	TaskCancelling:  "CANCELLING",
	TaskCanceled:    "CANCELED",
	TaskInterrupted: "INTERRUPTED",
}

var taskStatusValues = map[string]TaskStatus{
	"CREATED":     TaskCreated,
	"RUNNING":     TaskRunning,
	"SUCCESS":     TaskSuccess,
	"FAILURE":     TaskFailure,
	"CANCELLING":  TaskCancelling,
	"CANCELED":    TaskCanceled,
	"INTERRUPTED": TaskInterrupted,
}

func (s TaskStatus) String() string {
	if name, ok := taskStatusNames[s]; ok {
		return name
	}
	return "UNKNOWN"
}

func (s TaskStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *TaskStatus) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	if v, ok := taskStatusValues[name]; ok {
		*s = v
		return nil
	}
	return fmt.Errorf("unknown TaskStatus: %s", name)
}

var activeTaskStatuses = map[TaskStatus]bool{
	TaskCreated: true, TaskRunning: true, TaskFailure: true, TaskCancelling: true,
}
var cancelTaskStatuses = map[TaskStatus]bool{TaskCancelling: true, TaskCanceled: true}
var canCancelStatuses = map[TaskStatus]bool{TaskCreated: true, TaskRunning: true, TaskFailure: true}
var endStatuses = map[TaskStatus]bool{TaskSuccess: true, TaskCanceled: true, TaskInterrupted: true}

func (s TaskStatus) IsActive() bool    { return activeTaskStatuses[s] }
// IsInCancel 正在取消中或已被取消
func (s TaskStatus) IsInCancel() bool  { return cancelTaskStatuses[s] }
// CanCancel 可以被取消的状态
func (s TaskStatus) CanCancel() bool   { return canCancelStatuses[s] }
func (s TaskStatus) UnStarted() bool   { return s == TaskCreated }
func (s TaskStatus) IsFail() bool      { return s == TaskFailure }
func (s TaskStatus) IsSuccess() bool   { return s == TaskSuccess }
func (s TaskStatus) IsEnd() bool       { return endStatuses[s] }

func GetActiveStatuses() []TaskStatus {
	return []TaskStatus{TaskCreated, TaskRunning, TaskFailure, TaskCancelling}
}

func GetEffectiveStatuses() []TaskStatus {
	return []TaskStatus{TaskCreated, TaskRunning, TaskSuccess, TaskFailure, TaskCancelling}
}
