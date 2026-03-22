package stateful

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/delegate"
)

// StatefulTaskStep 有状态的任务步骤
type StatefulTaskStep struct {
	*delegate.DelegateTaskStep
	taskPersistence TaskPersistence
	taskId          string
}

// NewStatefulTaskStep 创建有状态的任务步骤
func NewStatefulTaskStep(name string, taskContext *flowsdk.TaskContext, delegatedTaskStep flowsdk.TaskStep, taskPersistence TaskPersistence) *StatefulTaskStep {
	taskStep := &StatefulTaskStep{
		DelegateTaskStep: delegate.NewDelegateTaskStep(name, taskContext, delegatedTaskStep),
		taskPersistence:  taskPersistence,
		taskId:           taskContext.GetTask().GetTaskId(),
	}

	// 恢复步骤状态
	taskStep.recoverTaskStep()

	return taskStep
}

// GetType 获取任务步骤类型
func (s *StatefulTaskStep) GetType() string {
	return "stateful"
}

// DoExecute 执行具体逻辑
func (s *StatefulTaskStep) DoExecute() flowsdk.TaskStepResult {
	defer s.saveTaskStep()

	return s.DelegateTaskStep.DoExecute()
}

// recoverTaskStep 恢复任务步骤状态
func (s *StatefulTaskStep) recoverTaskStep() {
	taskStepSnapshot := s.taskPersistence.RecoverTaskStep(s.taskId, s.GetName())
	if taskStepSnapshot != nil {
		// 恢复上下文
		if taskStepSnapshot.TaskStepContext != nil {
			// 这里需要实现上下文的恢复逻辑
		}
		// 恢复状态
		s.SetStatus(taskStepSnapshot.Status)
	}
}

// saveTaskStep 保存任务步骤状态
func (s *StatefulTaskStep) saveTaskStep() {
	status := s.GetStatus()

	taskStepSnapshot := &TaskStepSnapshot{
		TaskId:          s.taskId,
		Name:            s.GetName(),
		Type:            s.GetType(),
		Status:          status,
		TaskStepContext: nil, // 这里需要实现上下文的保存逻辑
	}

	s.taskPersistence.SaveTaskStep(taskStepSnapshot)
}

// TaskPersistence 任务持久化接口
type TaskPersistence interface {
	// SaveTaskStep 保存任务步骤快照
	SaveTaskStep(snapshot *TaskStepSnapshot)
	// RecoverTaskStep 恢复任务步骤快照
	RecoverTaskStep(taskId, stepName string) *TaskStepSnapshot
	// SaveTask 保存任务快照
	SaveTask(snapshot *TaskSnapshot)
	// RecoverTask 恢复任务快照
	RecoverTask(taskId string) *TaskSnapshot
}

// TaskStepSnapshot 任务步骤快照
type TaskStepSnapshot struct {
	TaskId          string
	Name            string
	Type            string
	Status          flowsdk.TaskStepStatus
	TaskStepContext interface{}
}
