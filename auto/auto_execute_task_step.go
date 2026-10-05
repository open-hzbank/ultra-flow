package auto

import (
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/delegate"
	"github.com/open-hzbank/ultra-flow/stateful"
	"time"
)

const AutoExecuteTaskStepType = "autoExecute"

const (
	AllowAutoExecute  = "allowAutoExecute"
	NextExecuteTime   = "nextExecuteTime"
	ExecuteErrorCount = "executeErrorCount"
	ExecuteTotalCount = "executeTotalCount"
)

// AutoExecuteTaskStep 定时任务自动调度的任务步骤
// 多用于异步任务: 需要流程引擎持续的状态追踪并自动更新
// 通过组合 StatefulTaskStep 实现状态持久化
type AutoExecuteTaskStep struct {
	*delegate.DelegateTaskStep
}

// Wrap 包装步骤为自动执行步骤
func Wrap(taskStep core.TaskStep, taskPersistence stateful.TaskPersistence,
	intervalSeconds, errorRetryTimes, maxTimes int) *AutoExecuteTaskStep {
	taskName := "AutoExecute:" + taskStep.GetName()
	doExeStep := newDoAutoExecuteStep(taskName, taskStep, intervalSeconds, errorRetryTimes, maxTimes)
	statefulTaskStep := stateful.Wrap(doExeStep, taskPersistence)
	return &AutoExecuteTaskStep{
		DelegateTaskStep: delegate.NewDelegateTaskStepWithName(taskName, statefulTaskStep),
	}
}

func incrementContext(ctx *core.TaskStepContext, key string) int {
	val := ctx.Get(key)
	var curr int
	if val != nil {
		switch v := val.(type) {
		case int:
			curr = v + 1
		case int64:
			curr = int(v) + 1
		default:
			curr = 1
		}
	} else {
		curr = 1
	}
	ctx.Put(key, curr)
	return curr
}

// doAutoExecuteStep 内部步骤, 负责实际执行与自动执行上下文维护
type doAutoExecuteStep struct {
	*delegate.DelegateTaskStep
	intervalSeconds int
	errorRetryTimes int
	maxTimes        int
	status          core.TaskStepStatus
}

func newDoAutoExecuteStep(name string, taskStep core.TaskStep, intervalSeconds, errorRetryTimes, maxTimes int) *doAutoExecuteStep {
	return &doAutoExecuteStep{
		DelegateTaskStep: delegate.NewDelegateTaskStepWithName(name, taskStep),
		intervalSeconds:  intervalSeconds,
		errorRetryTimes:  errorRetryTimes,
		maxTimes:         maxTimes,
		status:           core.StepPending,
	}
}

func (s *doAutoExecuteStep) GetType() string {
	return AutoExecuteTaskStepType
}

func (s *doAutoExecuteStep) Execute() core.TaskStepResult {
	result := s.Delegated.Execute()

	if stepCtx := s.GetTaskStepContext(); stepCtx != nil {
		currTotalCount := incrementContext(stepCtx, ExecuteTotalCount)

		var allowAutoExecute bool
		if result.Status.IsFail() {
			currErrorCount := incrementContext(stepCtx, ExecuteErrorCount)
			allowAutoExecute = currErrorCount < s.errorRetryTimes && currTotalCount < s.maxTimes
		} else {
			allowAutoExecute = currTotalCount < s.maxTimes
		}

		stepCtx.Put(NextExecuteTime, time.Now().Add(time.Duration(s.intervalSeconds)*time.Second))
		stepCtx.Put(AllowAutoExecute, allowAutoExecute)
	}

	return result
}

func (s *doAutoExecuteStep) OnInterrupt() {
	s.Delegated.OnInterrupt()
	s.status = core.StepInterrupted
}

func (s *doAutoExecuteStep) GetStatus() core.TaskStepStatus {
	return s.status
}

func (s *doAutoExecuteStep) SetStatus(status core.TaskStepStatus) {
	s.status = status
}

func (s *doAutoExecuteStep) Describe() core.StepDescription {
	type describable interface {
		Describe() core.StepDescription
	}
	if d, ok := s.Delegated.(describable); ok {
		desc := d.Describe()
		return core.NewStepDescription("自动执行: "+desc.Title, desc.Detail)
	}
	return core.NewStepDescription("自动执行: "+s.Delegated.GetName(), nil)
}
