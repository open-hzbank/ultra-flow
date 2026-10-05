package view

import (
	"github.com/open-hzbank/ultra-flow/control"
	"github.com/open-hzbank/ultra-flow/core"
)

// TaskStepView 步骤展示视图
type TaskStepView interface {
	GetStatus() core.TaskStepStatus
}

// AtomicStepView 原子步骤视图
type AtomicStepView struct {
	Name      string              `json:"name"`
	Title     string              `json:"title"`
	Blockable bool                `json:"blockable"`
	Status    core.TaskStepStatus `json:"status"`
	// 以下四个字段: 当步骤未执行时皆为 nil
	Detail       any    `json:"detail"`
	ErrorMessage string `json:"errorMessage"`
	GmtCreate    *int64 `json:"gmtCreate"`
	GmtModified  *int64 `json:"gmtModified"`
}

func (v *AtomicStepView) GetStatus() core.TaskStepStatus { return v.Status }

// NestedStepView 嵌套步骤视图
type NestedStepView struct {
	Stages []*StageView `json:"stages"`
}

func NewNestedStepView(stages []*StageView) *NestedStepView {
	return &NestedStepView{Stages: stages}
}

func (v *NestedStepView) GetStatus() core.TaskStepStatus {
	for _, stage := range v.Stages {
		switch stage.StageStatus() {
		case core.StepPending, core.StepRunning:
			return core.StepRunning
		case core.StepFailure:
			return core.StepFailure
		case core.StepInterrupted:
			return core.StepInterrupted
		case core.StepSkipped, core.StepSuccess:
			continue
		default:
			return core.StepFailure
		}
	}
	return core.StepSuccess
}

// StageView TaskStep 的阶段:
// 1. 同一 stage 内的 taskStep 并行执行, 无顺序先后;
// 2. 不同 stage 的 taskStep 按照 StageOrder 顺序从小到大依次执行
type StageView struct {
	StageOrder int            `json:"stageOrder"`
	StageName  string         `json:"stageName"`
	Steps      []TaskStepView `json:"steps"`
}

func NewStageView(stageOrder int, stageName string, steps []TaskStepView) *StageView {
	return &StageView{
		StageOrder: stageOrder,
		StageName:  stageName,
		Steps:      steps,
	}
}

func (v *StageView) StageStatus() core.TaskStepStatus {
	statuses := make([]core.TaskStepStatus, 0, len(v.Steps))
	for _, step := range v.Steps {
		statuses = append(statuses, step.GetStatus())
	}
	return control.CalcCompositeStatus(statuses)
}
