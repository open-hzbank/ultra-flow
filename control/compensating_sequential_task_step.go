package control

import (
	"github.com/open-hzbank/ultra-flow/core"
)

// CompensatingSequentialTaskStep 补偿用的串行步骤
type CompensatingSequentialTaskStep struct {
	*SequentialTaskStep
}

// NewCompensatingSequentialTaskStep 任务取消时, 允许 stepChain 上的所有步骤自动执行
func NewCompensatingSequentialTaskStep(name string, taskCtx *core.TaskContext,
	compensatingStepChain []core.TaskStep) *CompensatingSequentialTaskStep {
	autoConfigs := make([]bool, len(compensatingStepChain))
	for i := range autoConfigs {
		autoConfigs[i] = true
	}
	inner := NewSequentialTaskStep(name, taskCtx, compensatingStepChain, autoConfigs)
	inner.calcEntireStepStatus = compensatingCalcEntireStepStatus
	return &CompensatingSequentialTaskStep{SequentialTaskStep: inner}
}

// compensatingCalcEntireStepStatus 补偿链中跳过 INTERRUPTED 以继续执行后续的补偿步骤
func compensatingCalcEntireStepStatus(subStepResult core.TaskStepResult) *core.TaskStepResult {
	switch subStepResult.Status {
	case core.StepPending, core.StepRunning:
		r := core.NewStepResult(core.StepRunning)
		return &r
	case core.StepFailure:
		r := core.NewStepResultWithMessage(core.StepFailure, subStepResult.Message)
		return &r
	case core.StepInterrupted, core.StepSkipped, core.StepSuccess:
		return nil
	default:
		r := core.NewStepResult(core.StepFailure)
		return &r
	}
}
