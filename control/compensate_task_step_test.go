package control

import (
	"testing"

	"hzbank.com.cn/ultra-flow/core"
	"hzbank.com.cn/ultra-flow/padding"
)

func TestCompensableSequentialTaskStepBuild(t *testing.T) {
	taskCtx := core.NewTaskContext(nil, make(map[string]any))

	exeStep := padding.NewFixedStatusTaskStep("execute", taskCtx, core.StepSuccess)
	exeStep.SetStatus(core.StepSuccess)

	compensateStep := padding.NewFixedStatusTaskStep("compensate", taskCtx, core.StepSuccess)
	noneStep := padding.NewNoneTaskStep("none", taskCtx)

	taskSteps := []core.TaskStep{
		exeStep, noneStep,
		exeStep, noneStep,
		exeStep, compensateStep,
	}

	seqCompensable := BuildSequentialCompensableTaskStep(
		"mockStep", taskCtx, []bool{true, true, true}, taskSteps)

	compensateAwareSteps := seqCompensable.compensateAwareSteps
	if len(compensateAwareSteps) != 3 {
		t.Fatalf("expected compensateAwareSteps size 3, got %d", len(compensateAwareSteps))
	}

	// pair 0: executed=yes, compensate=no (compensate step is NoneTaskStep)
	executed0, ok0 := compensateAwareSteps[0].ExecutedTaskStep()
	if !ok0 || executed0 != exeStep {
		t.Errorf("[0] expected executedTaskStep=exeStep, got ok=%v step=%v", ok0, executed0)
	}
	if _, ok := compensateAwareSteps[0].CompensateTaskStep(); ok {
		t.Error("[0] expected compensateTaskStep to be absent")
	}

	// pair 1: executed=yes, compensate=no
	executed1, ok1 := compensateAwareSteps[1].ExecutedTaskStep()
	if !ok1 || executed1 != exeStep {
		t.Errorf("[1] expected executedTaskStep=exeStep, got ok=%v step=%v", ok1, executed1)
	}
	if _, ok := compensateAwareSteps[1].CompensateTaskStep(); ok {
		t.Error("[1] expected compensateTaskStep to be absent")
	}

	// pair 2: executed=yes, compensate=yes
	executed2, ok2 := compensateAwareSteps[2].ExecutedTaskStep()
	if !ok2 || executed2 != exeStep {
		t.Errorf("[2] expected executedTaskStep=exeStep, got ok=%v step=%v", ok2, executed2)
	}
	comp2, ok2c := compensateAwareSteps[2].CompensateTaskStep()
	if !ok2c || comp2 != compensateStep {
		t.Errorf("[2] expected compensateTaskStep=compensateStep, got ok=%v step=%v", ok2c, comp2)
	}

	// generateCompensableTaskStep → SequentialTaskStep with 4 taskSteps
	genStep := seqCompensable.GenerateCompensableTaskStep()
	compStep, ok := genStep.(*CompensatingSequentialTaskStep)
	if !ok {
		t.Fatalf("expected *CompensatingSequentialTaskStep, got %T", genStep)
	}

	if len(compStep.TaskSteps) != 4 {
		t.Fatalf("expected 4 taskSteps, got %d", len(compStep.TaskSteps))
	}

	// first 3 are executed normal steps, last is compensate step
	if compStep.TaskSteps[0].Delegated != exeStep {
		t.Errorf("[0] expected delegated=exeStep, got %v", compStep.TaskSteps[0].Delegated)
	}
	if compStep.TaskSteps[1].Delegated != exeStep {
		t.Errorf("[1] expected delegated=exeStep, got %v", compStep.TaskSteps[1].Delegated)
	}
	if compStep.TaskSteps[2].Delegated != exeStep {
		t.Errorf("[2] expected delegated=exeStep, got %v", compStep.TaskSteps[2].Delegated)
	}
	if compStep.TaskSteps[3].Delegated != compensateStep {
		t.Errorf("[3] expected delegated=compensateStep, got %v", compStep.TaskSteps[3].Delegated)
	}
}
