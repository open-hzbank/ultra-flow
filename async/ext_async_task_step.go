package async

import (
	"github.com/open-hzbank/ultra-flow/core"
)

// ExtAsyncTaskStep 异步提交、异步响应的扩展任务步骤
type ExtAsyncTaskStep struct {
	state *core.StepState

	// 判断任务是否已提交
	IsExtTaskSubmitted func() bool
	// 获取任务的最新状态 (执行结果)
	GetExtTaskResult func() core.TaskStepResult
	// 提交任务
	SubmitExtTask func() core.TaskStepResult
}

func NewExtAsyncTaskStep(
	name string,
	taskCtx *core.TaskContext,
	isExtTaskSubmitted func() bool,
	getExtTaskResult func() core.TaskStepResult,
	submitExtTask func() core.TaskStepResult,
) *ExtAsyncTaskStep {
	s := &ExtAsyncTaskStep{
		IsExtTaskSubmitted: isExtTaskSubmitted,
		GetExtTaskResult:   getExtTaskResult,
		SubmitExtTask:      submitExtTask,
	}
	s.state = core.NewStepStateWithTransfer(name, taskCtx, s, false)
	if transfer := taskCtx.GetTaskStepTransfer(); transfer != nil {
		transfer.Transfer(s)
	}
	return s
}

func (s *ExtAsyncTaskStep) GetName() string     { return s.state.Name }
func (s *ExtAsyncTaskStep) GetType() string     { return "extAsync" }
func (s *ExtAsyncTaskStep) OnSuccess()          {}
func (s *ExtAsyncTaskStep) OnFailure(err error) {}

func (s *ExtAsyncTaskStep) Execute() core.TaskStepResult {
	return s.state.ExecuteStep(s, func() core.TaskStepResult {
		if s.IsExtTaskSubmitted() {
			return s.GetExtTaskResult()
		}
		return s.SubmitExtTask()
	})
}

func (s *ExtAsyncTaskStep) OnInterrupt() {
	// 非运行态不中断
	if !s.canInterrupt() {
		return
	}
	// 提交失败: 不能中断而应该跳过, 因为任务没有实际提交, 无需补偿
	if !s.IsExtTaskSubmitted() {
		s.SetStatus(core.StepSkipped)
		return
	}
	// 因为异步任务状态可能被异步更新过, 因此需要做一次状态同步
	latestResult := s.GetExtTaskResult()
	// 再次检查: 当还是运行状态时则需要中断任务
	if s.canInterrupt() {
		latestResult = core.TaskStepResult{Status: s.doInterrupt()}
	}
	s.SetStatus(latestResult.Status)
}

func (s *ExtAsyncTaskStep) canInterrupt() bool {
	return s.GetStatus().IsExecuting()
}

func (s *ExtAsyncTaskStep) doInterrupt() core.TaskStepStatus {
	return core.StepInterrupted
}

func (s *ExtAsyncTaskStep) GetStatus() core.TaskStepStatus {
	return s.state.GetStatus()
}

func (s *ExtAsyncTaskStep) SetStatus(status core.TaskStepStatus) {
	s.state.SetStatus(status)
}

func (s *ExtAsyncTaskStep) GetTaskStepContext() *core.TaskStepContext {
	return s.state.StepCtx
}

func (s *ExtAsyncTaskStep) GetTaskContext() *core.TaskContext {
	return s.state.TaskCtx
}

func (s *ExtAsyncTaskStep) Describe() core.StepDescription {
	return core.NewStepDescription(s.GetName(), nil)
}
