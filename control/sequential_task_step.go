package control

import (
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/delegate"
)

// ManualFireNextStepSignal 手动触发下一个步骤的信号
const ManualFireNextStepSignal = "manualFireNextStep"

// AutoExeConfigTaskStep 带自动执行配置的步骤包装
type AutoExeConfigTaskStep struct {
	*delegate.DelegateTaskStep
	// AutoExecute 对应的步骤是否可以在串行序列的上一个步骤执行完成后自动执行
	AutoExecute bool
}

func newAutoExeConfigTaskStep(step core.TaskStep, autoExecute bool) *AutoExeConfigTaskStep {
	return &AutoExeConfigTaskStep{
		DelegateTaskStep: delegate.NewDelegateTaskStep(step),
		AutoExecute:      autoExecute,
	}
}

// SequentialTaskStep 串行编排步骤: 按序依次执行给定的多个子步骤, 直到全部步骤都返回 SUCCESS 或被中断
type SequentialTaskStep struct {
	state     *core.StepState
	TaskSteps []*AutoExeConfigTaskStep
	// calcEntireStepStatus 子步骤结果 → 整体步骤状态的计算策略
	// nil 时使用默认实现 defaultCalcEntireStepStatus
	// CompensatingSequentialTaskStep 会注入自定义实现以跳过 INTERRUPTED
	calcEntireStepStatus func(core.TaskStepResult) *core.TaskStepResult
}

func NewSequentialTaskStep(name string, taskCtx *core.TaskContext,
	taskSteps []core.TaskStep, autoExecuteConfigs []bool) *SequentialTaskStep {

	autoExeSteps := make([]*AutoExeConfigTaskStep, len(taskSteps))
	for i, step := range taskSteps {
		autoExe := false
		if i < len(autoExecuteConfigs) {
			autoExe = autoExecuteConfigs[i]
		}
		autoExeSteps[i] = newAutoExeConfigTaskStep(step, autoExe)
	}

	s := &SequentialTaskStep{TaskSteps: autoExeSteps}
	s.state = core.NewStepStateWithTransfer(name, taskCtx, s, false)
	return s
}

func (s *SequentialTaskStep) GetName() string { return s.state.Name }
func (s *SequentialTaskStep) GetType() string { return "sequentialExecute" }

func (s *SequentialTaskStep) Execute() core.TaskStepResult {
	return s.state.ExecuteStep(s, s.doExecute)
}

func (s *SequentialTaskStep) doExecute() core.TaskStepResult {
	// 若当前有执行中的步骤, 则从该步骤继续执行
	if result := s.autoSequentialExecute(s.findCurrentStep); result.found {
		return result.result
	}
	// 若有待执行的下一个步骤, 则在条件允许下执行下一步骤
	if result := s.autoSequentialExecute(s.findNextExecutableStep); result.found {
		return result.result
	}
	// 以上条件均不满足:
	if !s.findNotEndedStep() {
		// 所有步骤完成, 整体执行成功
		return core.NewStepResult(core.StepSuccess)
	}
	// 有未完成的步骤, 整体保持未完成
	return core.NewStepResult(core.StepRunning)
}

type optionalResult struct {
	found  bool
	result core.TaskStepResult
}

// autoSequentialExecute 从给定的子步骤开始线性依次执行, 直到全部子步骤执行完成 或 不满足继续执行的条件而中断
// 若给定的首个子步骤不存在, 则直接中断
func (s *SequentialTaskStep) autoSequentialExecute(firstStepSupplier func() (*AutoExeConfigTaskStep, bool)) optionalResult {
	autoExeStep, found := firstStepSupplier()
	for found {
		subStepResult := autoExeStep.Execute()
		stepResult := s.resolveCalcEntireStepStatus(subStepResult)
		if stepResult != nil {
			return optionalResult{found: true, result: *stepResult}
		}
		autoExeStep, found = s.findNextExecutableStep()
	}
	return optionalResult{found: false}
}

func (s *SequentialTaskStep) resolveCalcEntireStepStatus(subStepResult core.TaskStepResult) *core.TaskStepResult {
	if s.calcEntireStepStatus != nil {
		return s.calcEntireStepStatus(subStepResult)
	}
	return defaultCalcEntireStepStatus(subStepResult)
}

// findNextExecutableStep 获取下一个满足可执行条件的步骤
// 执行条件: 配置了自动执行 || 手动触发
func (s *SequentialTaskStep) findNextExecutableStep() (*AutoExeConfigTaskStep, bool) {
	nextStep, found := s.findNextStep()
	if !found {
		return nil, false
	}
	manualExecute := s.fireNextBatchSignal()
	canAutoExecute := nextStep.AutoExecute
	if !manualExecute && !canAutoExecute {
		return nil, false
	}
	return nextStep, true
}

func (s *SequentialTaskStep) fireNextBatchSignal() bool {
	signal := s.state.TaskCtx.Get(ManualFireNextStepSignal)
	s.state.TaskCtx.Put(ManualFireNextStepSignal, false)
	if b, ok := signal.(bool); ok {
		return b
	}
	return false
}

// defaultCalcEntireStepStatus 默认的 SequentialTaskStep 整体状态计算
func defaultCalcEntireStepStatus(subStepResult core.TaskStepResult) *core.TaskStepResult {
	switch subStepResult.Status {
	case core.StepPending, core.StepRunning:
		r := core.NewStepResult(core.StepRunning)
		return &r
	case core.StepFailure:
		r := core.NewStepResultWithMessage(core.StepFailure, subStepResult.Message)
		return &r
	case core.StepInterrupted:
		r := core.NewStepResult(core.StepInterrupted)
		return &r
	case core.StepSkipped, core.StepSuccess:
		return nil
	default:
		r := core.NewStepResult(core.StepFailure)
		return &r
	}
}

func (s *SequentialTaskStep) findCurrentStep() (*AutoExeConfigTaskStep, bool) {
	for _, step := range s.TaskSteps {
		if step.GetStatus().IsExecuting() {
			return step, true
		}
	}
	return nil, false
}

func (s *SequentialTaskStep) findNextStep() (*AutoExeConfigTaskStep, bool) {
	for _, step := range s.TaskSteps {
		if step.GetStatus().NotStarted() {
			return step, true
		}
	}
	return nil, false
}

func (s *SequentialTaskStep) findNotEndedStep() bool {
	for _, step := range s.TaskSteps {
		if !step.GetStatus().Ended() {
			return true
		}
	}
	return false
}

func (s *SequentialTaskStep) GetStatus() core.TaskStepStatus {
	for i, step := range s.TaskSteps {
		st := step.GetStatus()
		if st.NotStarted() && i == 0 {
			return core.StepPending
		}
		if st.NotStarted() && i > 0 {
			return core.StepRunning
		}
		if st.IsFail() {
			return core.StepFailure
		}
		if st.IsNormalRunning() {
			return core.StepRunning
		}
	}
	return core.StepSuccess
}

func (s *SequentialTaskStep) OnSuccess() {
	for _, step := range s.TaskSteps {
		step.OnSuccess()
	}
}

func (s *SequentialTaskStep) OnFailure(err error) {
	for _, step := range s.TaskSteps {
		step.OnFailure(err)
	}
}

// OnInterrupt 中断回调由下游自身来处理, 应反向调用各步骤段的中断逻辑
func (s *SequentialTaskStep) OnInterrupt() {
	for i := len(s.TaskSteps) - 1; i >= 0; i-- {
		s.TaskSteps[i].OnInterrupt()
	}
}

func (s *SequentialTaskStep) GetTaskStepContext() *core.TaskStepContext { return s.state.StepCtx }
func (s *SequentialTaskStep) GetTaskContext() *core.TaskContext         { return s.state.TaskCtx }

func (s *SequentialTaskStep) Describe() core.StepDescription {
	return core.NewStepDescription("串行编排执行步骤", nil)
}
