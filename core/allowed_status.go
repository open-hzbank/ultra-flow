package core

// AllowedStatus 状态转换规则
type AllowedStatus struct{}

// key为目标态，value为原状态，是否可以从当前原状态变更到目标态
var taskStepStatusMapping = map[TaskStepStatus][]TaskStepStatus{
	StepRunning:     {StepPending, StepRunning, StepFailure},
	StepFailure:     {StepPending, StepRunning, StepFailure},
	StepSuccess:     {StepPending, StepRunning, StepFailure},
	StepInterrupted: {StepPending, StepRunning, StepFailure},
	StepPending:     {StepPending},
	StepSkipped:     {StepSkipped, StepPending},
}

var taskStatusMapping = map[TaskStatus][]TaskStatus{
	TaskRunning:    {TaskCreated, TaskRunning, TaskFailure},
	TaskFailure:    {TaskCreated, TaskRunning, TaskFailure},
	TaskSuccess:    {TaskCreated, TaskRunning, TaskFailure},
	TaskCanceled:   {TaskCreated, TaskRunning, TaskFailure, TaskCancelling},
	TaskCreated:    {TaskCreated},
	TaskCancelling: {TaskCreated, TaskRunning, TaskFailure},
}

func GetAllowedStepStatuses(toStatus TaskStepStatus) []string {
	allowed := taskStepStatusMapping[toStatus]
	result := make([]string, len(allowed))
	for i, s := range allowed {
		result[i] = s.String()
	}
	return result
}

func GetAllowedTaskStatuses(toStatus TaskStatus) []string {
	allowed := taskStatusMapping[toStatus]
	result := make([]string, len(allowed))
	for i, s := range allowed {
		result[i] = s.String()
	}
	return result
}
