package control

import (
	"github.com/open-hzbank/ultra-flow/auto"
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/delegate"
	"github.com/open-hzbank/ultra-flow/stateful"
	"github.com/open-hzbank/ultra-flow/support"
)

// FireNextBatchSignal 手动触发下一个批次的信号
const FireNextBatchSignal = "fireNextBatch"

// BatchableTaskStep 具有分批能力的步骤
type BatchableTaskStep struct {
	state     *core.StepState
	TaskSteps []core.TaskStep
	batchCalc batchCalcFuncs
}

type batchCalcFuncs struct {
	batchNumbers   func(taskCtx *core.TaskContext, deps any) int
	buildBatchName func(batchID int) string
	buildBatchStep func(batchID int, taskCtx *core.TaskContext, deps any) core.TaskStep
	calcBatchID    func(step core.TaskStep) int
	canExecute     func(currentBatchID int) bool
	needSkip       func() bool
	preProcess     func() support.Result[any]
	postProcess    func() support.Result[any]
}

// BatchableConfig 分批步骤配置
type BatchableConfig struct {
	BatchNumbers   func(taskCtx *core.TaskContext, deps any) int
	BuildBatchName func(batchID int) string
	BuildBatchStep func(batchID int, taskCtx *core.TaskContext, deps any) core.TaskStep
	CalcBatchID    func(step core.TaskStep) int
	CanExecute     func(currentBatchID int) bool
	NeedSkip       func() bool
	PreProcess     func() support.Result[any]
	PostProcess    func() support.Result[any]
}

// NewBatchableTaskStep 创建分批步骤
//
//	stateful:                  是否需要具备有状态能力
//	autoUpdate:                是否需要追踪状态并自动更新
//	autoUpdateIntervalSeconds: 自动更新间隔时间 (autoUpdate == true 时有效)
//	autoUpdateErrorRetryTimes: 自动更新错误重试次数阈值 (autoUpdate == true 时有效)
func NewBatchableTaskStep(name string, taskCtx *core.TaskContext, deps any,
	taskPersistence stateful.TaskPersistence, cfg BatchableConfig,
	statefulEnabled, autoUpdate bool, autoUpdateIntervalSeconds, autoUpdateErrorRetryTimes, autoUpdateMaxTimes int) *BatchableTaskStep {

	s := &BatchableTaskStep{}
	s.state = core.NewStepStateWithTransfer(name, taskCtx, s, false)
	s.batchCalc = batchCalcFuncs{
		batchNumbers:   cfg.BatchNumbers,
		buildBatchName: cfg.BuildBatchName,
		buildBatchStep: cfg.BuildBatchStep,
		calcBatchID:    cfg.CalcBatchID,
		canExecute:     cfg.CanExecute,
		needSkip:       cfg.NeedSkip,
		preProcess:     cfg.PreProcess,
		postProcess:    cfg.PostProcess,
	}
	if s.batchCalc.needSkip == nil {
		s.batchCalc.needSkip = func() bool { return false }
	}
	if s.batchCalc.preProcess == nil {
		s.batchCalc.preProcess = func() support.Result[any] { return support.SuccessResult[any]() }
	}
	if s.batchCalc.postProcess == nil {
		s.batchCalc.postProcess = func() support.Result[any] { return support.SuccessResult[any]() }
	}

	s.TaskSteps = s.buildBatchSteps(taskCtx, deps, taskPersistence,
		statefulEnabled, autoUpdate, autoUpdateIntervalSeconds, autoUpdateErrorRetryTimes, autoUpdateMaxTimes)
	return s
}

func (s *BatchableTaskStep) buildBatchSteps(taskCtx *core.TaskContext, deps any,
	taskPersistence stateful.TaskPersistence,
	statefulEnabled, autoUpdate bool,
	autoUpdateIntervalSeconds, autoUpdateErrorRetryTimes, autoUpdateMaxTimes int) []core.TaskStep {

	batchNum := s.batchCalc.batchNumbers(taskCtx, deps)
	steps := make([]core.TaskStep, 0, batchNum)
	for batchID := 1; batchID <= batchNum; batchID++ {
		step := s.buildOneBatchStep(batchID, taskCtx, deps, taskPersistence,
			statefulEnabled, autoUpdate, autoUpdateIntervalSeconds, autoUpdateErrorRetryTimes, autoUpdateMaxTimes)
		steps = append(steps, step)
	}
	return steps
}

func (s *BatchableTaskStep) buildOneBatchStep(batchID int, taskCtx *core.TaskContext, deps any,
	taskPersistence stateful.TaskPersistence,
	statefulEnabled, autoUpdate bool,
	autoUpdateIntervalSeconds, autoUpdateErrorRetryTimes, autoUpdateMaxTimes int) core.TaskStep {

	var targetStep core.TaskStep
	if statefulEnabled {
		targetStep = stateful.WrapWithSupplier(
			func(ctx *core.TaskContext) core.TaskStep {
				return s.batchCalc.buildBatchStep(batchID, ctx, deps)
			},
			taskCtx, taskPersistence, s.batchCalc.buildBatchName(batchID),
		)
	} else {
		targetStep = s.batchCalc.buildBatchStep(batchID, taskCtx, deps)
	}

	if autoUpdate {
		targetStep = auto.Wrap(targetStep, taskPersistence,
			autoUpdateIntervalSeconds, autoUpdateErrorRetryTimes, autoUpdateMaxTimes)
	}
	return targetStep
}

func (s *BatchableTaskStep) GetName() string                           { return s.state.Name }
func (s *BatchableTaskStep) GetType() string                           { return "batchable" }
func (s *BatchableTaskStep) GetTaskStepContext() *core.TaskStepContext { return s.state.StepCtx }
func (s *BatchableTaskStep) GetTaskContext() *core.TaskContext         { return s.state.TaskCtx }

func (s *BatchableTaskStep) Execute() core.TaskStepResult {
	return s.state.ExecuteStep(s, s.doExecute)
}

func (s *BatchableTaskStep) doExecute() core.TaskStepResult {
	// 是否需要跳过步骤执行
	if s.batchCalc.needSkip() {
		return core.NewStepResult(core.StepSkipped)
	}

	// 前置处理
	allNotStarted := true
	for _, step := range s.TaskSteps {
		type statusAware interface{ GetStatus() core.TaskStepStatus }
		if sa, ok := step.(statusAware); ok {
			if !sa.GetStatus().NotStarted() {
				allNotStarted = false
				break
			}
		}
	}
	if allNotStarted {
		preResult := s.batchCalc.preProcess()
		if !preResult.IsSuccess() {
			return core.NewStepResultWithMessage(core.StepFailure, preResult.GetErrorMessage())
		}
	}

	// 当前所处的活跃批次步骤
	var unfinishedStep core.TaskStep
	for _, step := range s.TaskSteps {
		type statusAware interface{ GetStatus() core.TaskStepStatus }
		if sa, ok := step.(statusAware); ok {
			if !sa.GetStatus().NormalEnded() {
				unfinishedStep = step
				break
			}
		}
	}

	// 分批步骤都已执行完, 后置处理
	if unfinishedStep == nil {
		postResult := s.batchCalc.postProcess()
		if postResult.IsSuccess() {
			return core.NewStepResult(core.StepSuccess)
		}
		return core.NewStepResultWithMessage(core.StepFailure, postResult.GetErrorMessage())
	}

	type statusAware interface{ GetStatus() core.TaskStepStatus }
	sa := unfinishedStep.(statusAware)
	currentStatus := sa.GetStatus()

	if currentStatus.IsExecuting() {
		return s.doExecuteStep(unfinishedStep)
	}
	// 批次被中断取消
	if currentStatus.IsCanceled() {
		return core.NewStepResult(core.StepInterrupted)
	}

	// 存在未执行完的批次, 继续执行
	// 当前所处的批次号
	currentBatchID := s.batchCalc.calcBatchID(unwrapToBatchStep(unfinishedStep))
	if s.batchCalc.canExecute(currentBatchID) {
		// 执行下一个新的批次
		return s.doExecuteStep(unfinishedStep)
	}
	// 不能继续执行, 返回当前状态
	return core.NewStepResult(s.state.GetStatus())
}

func (s *BatchableTaskStep) doExecuteStep(step core.TaskStep) core.TaskStepResult {
	result := step.Execute()
	if result.Status.IsFail() {
		return result
	}
	return core.NewStepResult(core.StepRunning)
}

func (s *BatchableTaskStep) OnSuccess() {
	for _, step := range s.TaskSteps {
		step.OnSuccess()
	}
}

func (s *BatchableTaskStep) OnFailure(err error) {
	for _, step := range s.TaskSteps {
		step.OnFailure(err)
	}
}

func (s *BatchableTaskStep) OnInterrupt() {
	for _, step := range s.TaskSteps {
		step.OnInterrupt()
	}
	s.state.DefaultInterrupt()
}

func (s *BatchableTaskStep) Describe() core.StepDescription {
	return core.NewStepDescription("分批执行步骤", nil)
}

// unwrapToBatchStep 分批子步骤类型可能被 stateful 和 autoUpdate 特性嵌套包装
// 本方法将给定的分批子步骤直接实例, 实施可能的解包装过程, 最终得到原始的分批子步骤实例
func unwrapToBatchStep(step core.TaskStep) core.TaskStep {
	for {
		if d, ok := step.(*delegate.DelegateTaskStep); ok {
			step = d.Delegated
		} else {
			return step
		}
	}
}
