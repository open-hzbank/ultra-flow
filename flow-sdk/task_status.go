package flowsdk

// TaskStatus 任务的状态枚举
type TaskStatus string

const (
	TaskStatusCreated    TaskStatus = "CREATED"
	TaskStatusRunning    TaskStatus = "RUNNING"
	TaskStatusSuccess    TaskStatus = "SUCCESS"
	TaskStatusFailure    TaskStatus = "FAILURE"
	TaskStatusCancelling TaskStatus = "CANCELLING"
	TaskStatusCanceled   TaskStatus = "CANCELED"
	TaskStatusInterrupted TaskStatus = "INTERRUPTED"
)

var (
	activeStatuses   = []TaskStatus{TaskStatusCreated, TaskStatusRunning, TaskStatusFailure, TaskStatusCancelling}
	effectiveStatuses = []TaskStatus{TaskStatusCreated, TaskStatusRunning, TaskStatusSuccess, TaskStatusFailure, TaskStatusCancelling}
	canCancelStatuses = []TaskStatus{TaskStatusCreated, TaskStatusRunning, TaskStatusFailure}
	cancelStatus     = []TaskStatus{TaskStatusCancelling, TaskStatusCanceled}
	endStatus        = []TaskStatus{TaskStatusSuccess, TaskStatusCanceled, TaskStatusInterrupted}
)

func (s TaskStatus) IsActive() bool {
	for _, status := range activeStatuses {
		if s == status {
			return true
		}
	}
	return false
}

func (s TaskStatus) IsInCancel() bool {
	for _, status := range cancelStatus {
		if s == status {
			return true
		}
	}
	return false
}

func (s TaskStatus) CanCancel() bool {
	for _, status := range canCancelStatuses {
		if s == status {
			return true
		}
	}
	return false
}

func (s TaskStatus) UnStarted() bool {
	return s == TaskStatusCreated
}

func (s TaskStatus) IsFail() bool {
	return s == TaskStatusFailure
}

func (s TaskStatus) IsSuccess() bool {
	return s == TaskStatusSuccess
}

func (s TaskStatus) IsEnd() bool {
	for _, status := range endStatus {
		if s == status {
			return true
		}
	}
	return false
}
