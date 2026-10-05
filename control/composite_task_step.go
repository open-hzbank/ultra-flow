package control

import (
	"strings"
	"hzbank.com.cn/ultra-flow/core"
)

// statusChangePriority 以下 status 的优先级由高到低排序 (index 越小优先级越高)
var statusChangePriority = map[core.TaskStepStatus]int{
	core.StepInterrupted: 0,
	core.StepFailure:     1,
	core.StepRunning:     2,
	core.StepPending:     3,
	core.StepSuccess:     4,
	core.StepSkipped:     5,
}

// CompositeTaskStep 复合编排步骤: 依次执行给定的多个子步骤, 整体执行状态按照状态优先级取各子步骤中优先级最高的
// 可用于组合需要并行执行的步骤 (虽非严格意义上的物理并行, 但从任务执行轮次的粒度看, 相比于 sequential, composite 已等效于并行)
type CompositeTaskStep struct {
	state    *core.StepState
	TaskSteps []core.TaskStep
}

func NewCompositeTaskStep(name string, taskCtx *core.TaskContext, taskSteps []core.TaskStep) *CompositeTaskStep {
	s := &CompositeTaskStep{TaskSteps: taskSteps}
	s.state = core.NewStepStateWithTransfer(name, taskCtx, s, false)
	return s
}

func (s *CompositeTaskStep) GetName() string    { return s.state.Name }
func (s *CompositeTaskStep) GetType() string    { return "composite" }
func (s *CompositeTaskStep) OnSuccess()         { for _, step := range s.TaskSteps { step.OnSuccess() } }
func (s *CompositeTaskStep) OnFailure(err error) { for _, step := range s.TaskSteps { step.OnFailure(err) } }
// OnInterrupt 中断回调由下游自身来保证
func (s *CompositeTaskStep) OnInterrupt() {
	for _, step := range s.TaskSteps {
		step.OnInterrupt()
	}
	s.state.DefaultInterrupt()
}

func (s *CompositeTaskStep) Execute() core.TaskStepResult {
	return s.state.ExecuteStep(s, func() core.TaskStepResult {
		compositeStatus := core.StepSkipped
		var msgBuilder strings.Builder
		for _, step := range s.TaskSteps {
			subResult := step.Execute()
			compositeStatus = calcChangeStatus(compositeStatus, subResult.Status)
			if subResult.Message != "" {
				msgBuilder.WriteString(subResult.Message)
				msgBuilder.WriteString("\n")
			}
		}
		return core.TaskStepResult{Status: compositeStatus, Message: msgBuilder.String()}
	})
}

func (s *CompositeTaskStep) GetStatus() core.TaskStepStatus {
	statuses := make([]core.TaskStepStatus, 0, len(s.TaskSteps))
	for _, step := range s.TaskSteps {
		type statusAware interface{ GetStatus() core.TaskStepStatus }
		if sa, ok := step.(statusAware); ok {
			statuses = append(statuses, sa.GetStatus())
		}
	}
	return CalcCompositeStatus(statuses)
}

func (s *CompositeTaskStep) GetTaskStepContext() *core.TaskStepContext { return s.state.StepCtx }
func (s *CompositeTaskStep) GetTaskContext() *core.TaskContext          { return s.state.TaskCtx }

func (s *CompositeTaskStep) Describe() core.StepDescription {
	return core.NewStepDescription("复合步骤", nil)
}

// CalcCompositeStatus 计算多个状态的综合状态
func CalcCompositeStatus(statuses []core.TaskStepStatus) core.TaskStepStatus {
	composite := core.StepSkipped
	for _, st := range statuses {
		composite = calcChangeStatus(composite, st)
	}
	return composite
}

func calcChangeStatus(origin, changed core.TaskStepStatus) core.TaskStepStatus {
	originIdx, ok1 := statusChangePriority[origin]
	changedIdx, ok2 := statusChangePriority[changed]
	if !ok1 {
		originIdx = 6
	}
	if !ok2 {
		changedIdx = 6
	}
	if changedIdx < originIdx {
		return changed
	}
	return origin
}
