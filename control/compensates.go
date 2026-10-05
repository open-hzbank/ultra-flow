package control

import (
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/padding"
)

// Compensates 补偿工具方法
type Compensates struct{}

// BuildCompensateAwareTaskSteps 构建 CompensateAwareTaskStep 列表
// taskSteps 需按照: 普通步骤, 补偿步骤 的顺序依次填充
func BuildCompensateAwareTaskSteps(taskSteps []core.TaskStep) []core.CompensateAwareTaskStep {
	const stepTypesCount = 2
	if len(taskSteps)%stepTypesCount != 0 {
		panic("taskStep 数量不符合要求, 应该为 2 的倍数")
	}
	result := make([]core.CompensateAwareTaskStep, 0, len(taskSteps)/stepTypesCount)
	for i := 0; i < len(taskSteps)/stepTypesCount; i++ {
		normalStep := taskSteps[stepTypesCount*i]
		compensateStep := taskSteps[stepTypesCount*i+1]
		result = append(result, buildCompensateAwareStep(normalStep, compensateStep))
	}
	return result
}

func buildCompensateAwareStep(normalStep, compensateStep core.TaskStep) core.CompensateAwareTaskStep {
	if _, ok := normalStep.(*padding.NoneTaskStep); ok {
		panic("buildCompensateAwareStep 异常: normalStep 没有内容")
	}

	normalComposite, normalIsComposite := normalStep.(*CompositeTaskStep)
	compensateComposite, compensateIsComposite := compensateStep.(*CompositeTaskStep)
	if normalIsComposite && compensateIsComposite {
		return zipTaskSteps(normalComposite, compensateComposite)
	}

	normalCA, normalIsCA := normalStep.(core.CompensateAwareTaskStep)
	_, compensateIsNone := compensateStep.(*padding.NoneTaskStep)
	if normalIsCA && compensateIsNone {
		return normalCA
	}

	compensateCA, compensateIsCA := compensateStep.(core.CompensateAwareTaskStep)
	if normalIsCA && compensateIsCA {
		panic("buildCompensateAwareStep 异常, normalStep 和 compensateStep 类型无法组合")
	}
	_ = normalCA
	_ = compensateCA

	return &atomicCompensateAwareStep{
		normalStep:     normalStep,
		compensateStep: compensateStep,
	}
}

func zipTaskSteps(normalStep, compensateStep *CompositeTaskStep) core.CompensateAwareTaskStep {
	minLen := len(normalStep.TaskSteps)
	if len(compensateStep.TaskSteps) < minLen {
		minLen = len(compensateStep.TaskSteps)
	}
	zipped := make([]core.CompensateAwareTaskStep, 0, minLen)
	for i := 0; i < minLen; i++ {
		zipped = append(zipped, buildCompensateAwareStep(normalStep.TaskSteps[i], compensateStep.TaskSteps[i]))
	}
	return NewCompositeCompensableTaskStepFromAware(
		"CompositeCompensableStep",
		normalStep.GetTaskContext(),
		zipped,
	)
}

// GetExecutedSteps 提取已执行的步骤
func GetExecutedSteps(steps []core.CompensateAwareTaskStep) []core.TaskStep {
	var result []core.TaskStep
	for _, s := range steps {
		if executed, ok := s.ExecutedTaskStep(); ok {
			result = append(result, executed)
		}
	}
	return result
}

// GetCompensateSteps 提取补偿步骤
func GetCompensateSteps(steps []core.CompensateAwareTaskStep) []core.TaskStep {
	var result []core.TaskStep
	for _, s := range steps {
		if comp, ok := s.CompensateTaskStep(); ok {
			result = append(result, comp)
		}
	}
	return result
}

// atomicCompensateAwareStep 原子级别的补偿感知步骤
type atomicCompensateAwareStep struct {
	normalStep     core.TaskStep
	compensateStep core.TaskStep
}

func (s *atomicCompensateAwareStep) OriginTaskStep() core.TaskStep {
	return s.normalStep
}

func (s *atomicCompensateAwareStep) ExecutedTaskStep() (core.TaskStep, bool) {
	type statusAware interface{ GetStatus() core.TaskStepStatus }
	sa, ok := s.normalStep.(statusAware)
	if !ok {
		return nil, false
	}
	if sa.GetStatus().IsProcessed() {
		return s.normalStep, true
	}
	return nil, false
}

func (s *atomicCompensateAwareStep) CompensateTaskStep() (core.TaskStep, bool) {
	type statusAware interface{ GetStatus() core.TaskStepStatus }
	sa, ok := s.normalStep.(statusAware)
	if !ok {
		return nil, false
	}
	if !core.NeedCompensate(sa.GetStatus()) {
		return nil, false
	}
	if _, isNone := s.compensateStep.(*padding.NoneTaskStep); isNone {
		return nil, false
	}
	return s.compensateStep, true
}
