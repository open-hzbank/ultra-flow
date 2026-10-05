package transfer

import (
	"github.com/open-hzbank/ultra-flow/core"
)

// DefaultTaskStepTransfer TaskStepTransfer 的默认实现
// TaskStepTransfer 用于传递对任务步骤的修改, 用于任务回调等场景
type DefaultTaskStepTransfer struct {
	contexts map[string]map[string]any
}

func NewDefaultTaskStepTransfer() *DefaultTaskStepTransfer {
	return &DefaultTaskStepTransfer{
		contexts: make(map[string]map[string]any),
	}
}

func NewDefaultTaskStepTransferWithEntry(stepName, key string, value any) *DefaultTaskStepTransfer {
	ctx := make(map[string]map[string]any)
	stepCtx := make(map[string]any)
	stepCtx[key] = value
	ctx[stepName] = stepCtx
	return &DefaultTaskStepTransfer{contexts: ctx}
}

func NewDefaultTaskStepTransferFromConfigs(configs []core.StepExecuteConfig) *DefaultTaskStepTransfer {
	ctx := make(map[string]map[string]any)
	for _, cfg := range configs {
		stepCtx, ok := ctx[cfg.StepName]
		if !ok {
			stepCtx = make(map[string]any)
			ctx[cfg.StepName] = stepCtx
		}
		stepCtx[cfg.Key] = cfg.Value
	}
	return &DefaultTaskStepTransfer{contexts: ctx}
}

func NewDefaultTaskStepTransferFromMap(contexts map[string]map[string]any) *DefaultTaskStepTransfer {
	cp := make(map[string]map[string]any, len(contexts))
	for k, v := range contexts {
		stepCtx := make(map[string]any, len(v))
		for sk, sv := range v {
			stepCtx[sk] = sv
		}
		cp[k] = stepCtx
	}
	return &DefaultTaskStepTransfer{contexts: cp}
}

func (t *DefaultTaskStepTransfer) Transfer(step core.TaskStep) {
	stepCtx := t.contexts[step.GetName()]
	if len(stepCtx) == 0 {
		return
	}
	type stepContextAware interface {
		GetTaskStepContext() *core.TaskStepContext
	}
	if aware, ok := step.(stepContextAware); ok {
		ctx := aware.GetTaskStepContext()
		if ctx != nil {
			ctx.PutAll(stepCtx)
		}
	}
}

// Merge 合并另一个 transfer
// newerTransfer 的优先级更高
// originTransfer 和 newerTransfer 有相同的 key, 优先使用 newerTransfer 的 value
func (t *DefaultTaskStepTransfer) Merge(newerTransfer core.TaskStepTransfer) {
	other, ok := newerTransfer.(*DefaultTaskStepTransfer)
	if !ok {
		panic("暂不支持合并 DefaultTaskStepTransfer 外的类型")
	}
	for stepName, newContext := range other.contexts {
		originCtx, exists := t.contexts[stepName]
		if !exists {
			originCtx = make(map[string]any, len(newContext))
			t.contexts[stepName] = originCtx
		}
		for k, v := range newContext {
			originCtx[k] = v
		}
	}
}
