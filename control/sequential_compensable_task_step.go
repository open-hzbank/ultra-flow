package control

import (
	"hzbank.com.cn/ultra-flow/core"
	"hzbank.com.cn/ultra-flow/delegate"
)

// SequentialCompensableTaskStep 可根据当前执行状态进行适应性补偿的 串行编排步骤
type SequentialCompensableTaskStep struct {
	*delegate.DelegateTaskStep
	name                string
	taskCtx             *core.TaskContext
	compensateAwareSteps []core.CompensateAwareTaskStep
}

// BuildSequentialCompensableTaskStep 构建可补偿的串行编排步骤 (taskSteps 需按照: 普通步骤, 补偿步骤 的顺序依次填充)
// 其中设置的补偿步骤会同时运用于已中断状态的补偿步骤和已成功状态的补偿步骤
//
//	原始步骤链路及状态: step1 (SUCCESS) -> step2 (RUNNING) -> step3 (CREATED)
//	则生成的补偿步骤为: step2 (RUNNING) -> compensate_step2 (created) -> compensate_step1 (created)
func BuildSequentialCompensableTaskStep(name string, taskCtx *core.TaskContext,
	autoExecuteConfigs []bool, taskSteps []core.TaskStep) *SequentialCompensableTaskStep {

	compensateAwareSteps := BuildCompensateAwareTaskSteps(taskSteps)
	if len(autoExecuteConfigs) != len(compensateAwareSteps) {
		panic("步骤数量与 autoExecute 配置数量不匹配")
	}

	normalSteps := make([]core.TaskStep, len(compensateAwareSteps))
	for i, ca := range compensateAwareSteps {
		normalSteps[i] = ca.OriginTaskStep()
	}

	sequentialStep := NewSequentialTaskStep(name, taskCtx, normalSteps, autoExecuteConfigs)
	return &SequentialCompensableTaskStep{
		DelegateTaskStep:   delegate.NewDelegateTaskStep(sequentialStep),
		name:               name,
		taskCtx:            taskCtx,
		compensateAwareSteps: compensateAwareSteps,
	}
}

func (s *SequentialCompensableTaskStep) OriginTaskStep() core.TaskStep {
	return s
}

func (s *SequentialCompensableTaskStep) ExecutedTaskStep() (core.TaskStep, bool) {
	executedSteps := GetExecutedSteps(s.compensateAwareSteps)
	if len(executedSteps) == 0 {
		return nil, false
	}
	return NewCompensatingSequentialTaskStep(s.name, s.taskCtx, executedSteps), true
}

func (s *SequentialCompensableTaskStep) CompensateTaskStep() (core.TaskStep, bool) {
	compensateSteps := GetCompensateSteps(s.compensateAwareSteps)
	reversed := reverseSteps(compensateSteps)
	if len(reversed) == 0 {
		return nil, false
	}
	return NewCompensatingSequentialTaskStep(s.name, s.taskCtx, reversed), true
}

func (s *SequentialCompensableTaskStep) GenerateCompensableTaskStep() core.TaskStep {
	// 提取已执行过的普通步骤
	executedSteps := GetExecutedSteps(s.compensateAwareSteps)
	// 根据原有步骤的执行状态生成适应性的补偿步骤
	compensateSteps := GetCompensateSteps(s.compensateAwareSteps)
	reversed := reverseSteps(compensateSteps)

	chain := make([]core.TaskStep, 0, len(executedSteps)+len(reversed))
	chain = append(chain, executedSteps...)
	chain = append(chain, reversed...)
	return NewCompensatingSequentialTaskStep(s.name, s.taskCtx, chain)
}

func reverseSteps(steps []core.TaskStep) []core.TaskStep {
	n := len(steps)
	result := make([]core.TaskStep, n)
	for i := 0; i < n; i++ {
		result[i] = steps[n-1-i]
	}
	return result
}
