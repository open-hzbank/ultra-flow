package compensate

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/padding"
)

// CompensableTaskStep 可补偿的任务步骤
type CompensableTaskStep interface {
	flowsdk.TaskStep
	// GenerateCompensableTaskStep 基于原有 taskStep 的当前状态, 针对性地生成具备补偿原有 taskStep 能力的新的步骤链路
	GenerateCompensableTaskStep() flowsdk.TaskStep
}

// CompensateAwareTaskStep 对补偿有感知能力的步骤
type CompensateAwareTaskStep interface {
	// ExecutedTaskStep 当所在任务被取消时, 提取 originStep 已经被执行的 "逻辑步骤"
	ExecutedTaskStep() flowsdk.TaskStep
	// CompensateTaskStep 当所在任务被取消时, 基于 originStep 的状态生成对应的补偿 "逻辑步骤"
	CompensateTaskStep() flowsdk.TaskStep
	// OriginTaskStep 原始的 taskStep
	OriginTaskStep() flowsdk.TaskStep
}

// AtomicCompensateAwareStep 原子补偿感知步骤
type AtomicCompensateAwareStep struct {
	normalStep     flowsdk.TaskStep
	compensateStep flowsdk.TaskStep
}

// NewAtomicCompensateAwareStep 创建原子补偿感知步骤
func NewAtomicCompensateAwareStep(normalStep, compensateStep flowsdk.TaskStep) *AtomicCompensateAwareStep {
	return &AtomicCompensateAwareStep{
		normalStep:     normalStep,
		compensateStep: compensateStep,
	}
}

// ExecutedTaskStep 获取已执行的步骤
func (s *AtomicCompensateAwareStep) ExecutedTaskStep() flowsdk.TaskStep {
	if abstractStep, ok := s.normalStep.(interface{ GetStatus() flowsdk.TaskStepStatus }); ok {
		if abstractStep.GetStatus().IsProcessedStatus() {
			return s.normalStep
		}
	}
	return nil
}

// CompensateTaskStep 获取补偿步骤
func (s *AtomicCompensateAwareStep) CompensateTaskStep() flowsdk.TaskStep {
	if abstractStep, ok := s.normalStep.(interface{ GetStatus() flowsdk.TaskStepStatus }); ok {
		if flowsdk.NeedCompensate(abstractStep.GetStatus()) {
			// 检查补偿步骤是否为 NoneTaskStep
			if _, isNone := s.compensateStep.(*padding.NoneTaskStep); !isNone {
				return s.compensateStep
			}
		}
	}
	return nil
}

// OriginTaskStep 获取原始步骤
func (s *AtomicCompensateAwareStep) OriginTaskStep() flowsdk.TaskStep {
	return s.normalStep
}
