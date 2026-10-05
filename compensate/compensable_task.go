package compensate

import (
	"fmt"
	"log"

	"hzbank.com.cn/ultra-flow/core"
)

const TaskCancelMark = "taskCancelled"

// CompensableTask 可对已执行的普通步骤执行补偿步骤的任务
type CompensableTask struct {
	*core.AbstractTask
	// 可执行补偿回退的 taskStep
	compensableStep core.CompensableTaskStep
}

type CompensableTaskConfig struct {
	TaskID       string
	Name         string
	Type         string
	Subject      any
	Creator      string
	PublishEnv   core.Env
	IdempotentID string
	Status       core.TaskStatus
	Description  core.Description
	Context      map[string]any
	Transfer     core.TaskStepTransfer
}

// NewCompensableTask 恢复任务调用: 不传入 stepSupplier, compensableStep 需通过 InitCompensableTaskStep 设置
func NewCompensableTask(cfg CompensableTaskConfig) *CompensableTask {
	if cfg.Status == 0 {
		cfg.Status = core.TaskCreated
	}
	base := core.NewAbstractTask(core.AbstractTaskConfig{
		TaskID:       cfg.TaskID,
		Name:         cfg.Name,
		Type:         cfg.Type,
		Subject:      cfg.Subject,
		Creator:      cfg.Creator,
		PublishEnv:   cfg.PublishEnv,
		IdempotentID: cfg.IdempotentID,
		Status:       cfg.Status,
		Description:  cfg.Description,
		Context:      cfg.Context,
		Transfer:     cfg.Transfer,
	})
	return &CompensableTask{AbstractTask: base}
}

// NewCompensableTaskWithSupplier 初始化任务调用: 通过 stepSupplier 构建 compensableStep
func NewCompensableTaskWithSupplier(cfg CompensableTaskConfig, stepSupplier func(*core.TaskContext) core.CompensableTaskStep) *CompensableTask {
	t := NewCompensableTask(cfg)
	t.InitCompensableTaskStep(stepSupplier)
	return t
}

// InitCompensableTaskStep 由子类或构造函数调用, 通过 supplier 初始化补偿步骤
func (t *CompensableTask) InitCompensableTaskStep(stepSupplier func(*core.TaskContext) core.CompensableTaskStep) {
	t.compensableStep = stepSupplier(t.GetTaskContext())
	t.SetTaskStep(t.compensableStep)
}

func (t *CompensableTask) Execute() core.TaskResult {
	result := t.ExecuteTask()
	t.SetStatus(result.Status)
	return result
}

func (t *CompensableTask) ExecuteTask() core.TaskResult {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("execute compensate task panic, taskId: %s, error: %v", t.GetTaskID(), r)
		}
	}()

	if t.isCancelling() {
		return t.executeCompensatedStep()
	}
	if t.CanExecute() {
		return t.executeNormalStep()
	}
	return core.NewTaskResult(t.GetTaskID(), t.GetStatus())
}

func (t *CompensableTask) isCancelling() bool {
	return t.GetStatus() == core.TaskCancelling
}

func (t *CompensableTask) executeCompensatedStep() core.TaskResult {
	compensateStep := t.compensableStep.GenerateCompensableTaskStep()
	stepResult := compensateStep.Execute()

	switch stepResult.Status {
	case core.StepPending, core.StepRunning:
		return core.NewTaskResult(t.GetTaskID(), core.TaskCancelling)
	case core.StepSuccess, core.StepSkipped:
		return core.NewTaskResult(t.GetTaskID(), core.TaskCanceled)
	case core.StepFailure:
		compensateStep.OnFailure(stepResult.Error)
		return core.NewTaskResultWithError(t.GetTaskID(), core.TaskCancelling, stepResult.Error, stepResult.Message)
	default:
		panic(fmt.Sprintf("unexpected step status: %v", stepResult.Status))
	}
}

func (t *CompensableTask) executeNormalStep() core.TaskResult {
	stepResult := t.compensableStep.Execute()

	switch stepResult.Status {
	case core.StepPending, core.StepRunning:
		return core.NewTaskResult(t.GetTaskID(), core.TaskRunning)
	case core.StepSuccess, core.StepSkipped:
		return core.NewTaskResult(t.GetTaskID(), core.TaskSuccess)
	case core.StepFailure:
		t.compensableStep.OnFailure(stepResult.Error)
		return core.NewTaskResultWithError(t.GetTaskID(), core.TaskFailure, stepResult.Error, stepResult.Message)
	case core.StepInterrupted:
		// 业务步骤主动返回 INTERRUPTED, CompensableTask 将试图对其做可补偿的取消
		t.Cancel()
		return core.NewTaskResult(t.GetTaskID(), t.GetStatus())
	default:
		panic(fmt.Sprintf("unexpected step status: %v", stepResult.Status))
	}
}

func (t *CompensableTask) Cancel() {
	// 幂等返回
	if t.GetStatus() == core.TaskCanceled || t.GetStatus() == core.TaskCancelling {
		return
	}
	// 终态不允许取消
	if t.GetStatus() == core.TaskSuccess {
		panic("已执行成功的任务无法取消")
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Cancel task error, taskId: %s, error: %v", t.GetTaskID(), r)
			t.SetStatus(core.TaskFailure)
			t.compensableStep.OnFailure(fmt.Errorf("%v", r))
		}
	}()

	t.compensableStep.OnInterrupt()
	t.GetTaskContext().Put(TaskCancelMark, true)
	result := t.executeCompensatedStep()
	t.SetStatus(result.Status)
}
