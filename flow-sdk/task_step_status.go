package flowsdk

// TaskStepStatus 步骤的状态枚举
type TaskStepStatus string

const (
	TaskStepStatusPending     TaskStepStatus = "PENDING"
	TaskStepStatusRunning     TaskStepStatus = "RUNNING"
	TaskStepStatusSuccess     TaskStepStatus = "SUCCESS"
	TaskStepStatusSkipped     TaskStepStatus = "SKIPPED"
	TaskStepStatusFailure     TaskStepStatus = "FAILURE"
	TaskStepStatusInterrupted TaskStepStatus = "INTERRUPTED"
)

var (
	executingStatuses = []TaskStepStatus{TaskStepStatusRunning, TaskStepStatusFailure}
	taskStepActiveStatuses    = []TaskStepStatus{TaskStepStatusPending, TaskStepStatusRunning, TaskStepStatusFailure}
	processedStatus   = []TaskStepStatus{TaskStepStatusSuccess, TaskStepStatusRunning, TaskStepStatusFailure, TaskStepStatusInterrupted, TaskStepStatusSkipped}
	normalEndedStatus = []TaskStepStatus{TaskStepStatusSuccess, TaskStepStatusSkipped}
	endedStatus       = []TaskStepStatus{TaskStepStatusSuccess, TaskStepStatusSkipped, TaskStepStatusInterrupted}
	notExecutedStatus = []TaskStepStatus{TaskStepStatusPending, TaskStepStatusSkipped}
)

func NeedCompensate(status TaskStepStatus) bool {
	return status == TaskStepStatusSuccess ||
		status == TaskStepStatusInterrupted ||
		status == TaskStepStatusFailure ||
		status == TaskStepStatusRunning
}

func (s TaskStepStatus) NotStarted() bool {
	return s == TaskStepStatusPending
}

func (s TaskStepStatus) NotExecuted() bool {
	for _, status := range notExecutedStatus {
		if s == status {
			return true
		}
	}
	return false
}

func (s TaskStepStatus) IsExecutingStatus() bool {
	for _, status := range executingStatuses {
		if s == status {
			return true
		}
	}
	return false
}

func (s TaskStepStatus) IsProcessedStatus() bool {
	for _, status := range processedStatus {
		if s == status {
			return true
		}
	}
	return false
}

func (s TaskStepStatus) IsCanceled() bool {
	return s == TaskStepStatusInterrupted
}

func (s TaskStepStatus) IsFail() bool {
	return s == TaskStepStatusFailure
}

func (s TaskStepStatus) IsNormalRunning() bool {
	return s == TaskStepStatusRunning
}

func (s TaskStepStatus) IsActive() bool {
	for _, status := range taskStepActiveStatuses {
		if s == status {
			return true
		}
	}
	return false
}

func (s TaskStepStatus) NormalEnded() bool {
	for _, status := range normalEndedStatus {
		if s == status {
			return true
		}
	}
	return false
}

func (s TaskStepStatus) Ended() bool {
	for _, status := range endedStatus {
		if s == status {
			return true
		}
	}
	return false
}

func (s TaskStepStatus) Skipped() bool {
	return s == TaskStepStatusSkipped
}

func ActiveStatuses() []TaskStepStatus {
	return taskStepActiveStatuses
}
