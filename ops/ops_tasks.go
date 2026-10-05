package ops

import (
	"fmt"
	"sync"

	"github.com/open-hzbank/ultra-flow/stateful"
)

type taskKey struct {
	name string
	typ  string
}

type taskStepKey struct {
	name     string
	typ      string
	stepType string
}

var (
	opsTaskBuilders   = make(map[taskKey]*OpsTaskDefinition)
	opsTaskBuildersMu sync.RWMutex

	// 各任务类型下具有分批能力的步骤类型集合
	batchableTaskSteps   = make(map[taskKey][]string)
	batchableTaskStepsMu sync.RWMutex

	// 可分批步骤的精细化运行状态探测逻辑
	batchableStepProbes   = make(map[taskStepKey]func(map[string]any) bool)
	batchableStepProbesMu sync.RWMutex
)

// Register 注册一个 OpsTask 的定义
func Register(def *OpsTaskDefinition) {
	opsTaskBuildersMu.Lock()
	defer opsTaskBuildersMu.Unlock()
	opsTaskBuilders[taskKey{name: def.Name, typ: def.Type}] = def
}

// Unregister 取消注册
func Unregister(taskName, taskType string) {
	opsTaskBuildersMu.Lock()
	defer opsTaskBuildersMu.Unlock()
	delete(opsTaskBuilders, taskKey{name: taskName, typ: taskType})
}

// GetOpsTaskBuilder 获取已注册的任务构建器
func GetOpsTaskBuilder(taskName, taskType string) OpsTaskBuilder {
	opsTaskBuildersMu.RLock()
	defer opsTaskBuildersMu.RUnlock()
	def, ok := opsTaskBuilders[taskKey{name: taskName, typ: taskType}]
	if !ok {
		panic(fmt.Sprintf("task: %s, taskType: %s not defined", taskName, taskType))
	}
	return def.OpsTaskBuilder
}

// RegisterBatchableStep 注册可分批步骤
func RegisterBatchableStep(def *OpsTaskDefinition, stepType string, runningProbe func(map[string]any) bool) {
	key := taskKey{name: def.Name, typ: def.Type}

	batchableTaskStepsMu.Lock()
	steps := batchableTaskSteps[key]
	found := false
	for _, s := range steps {
		if s == stepType {
			found = true
			break
		}
	}
	if !found {
		batchableTaskSteps[key] = append(steps, stepType)
	}
	batchableTaskStepsMu.Unlock()

	batchableStepProbesMu.Lock()
	defer batchableStepProbesMu.Unlock()
	sKey := taskStepKey{name: def.Name, typ: def.Type, stepType: stepType}
	if _, exists := batchableStepProbes[sKey]; !exists {
		batchableStepProbes[sKey] = runningProbe
	}
}

// GetBatchableStepsOfTask 获取指定任务类型下的可分批步骤
func GetBatchableStepsOfTask(taskName, taskType string) []string {
	batchableTaskStepsMu.RLock()
	defer batchableTaskStepsMu.RUnlock()
	steps := batchableTaskSteps[taskKey{name: taskName, typ: taskType}]
	if len(steps) == 0 {
		return nil
	}
	result := make([]string, len(steps))
	copy(result, steps)
	return result
}

// HasRunningBatchInStep 步骤中是否存在运行中的批次
func HasRunningBatchInStep(task *stateful.TaskSnapshot, step *stateful.TaskStepSnapshot) bool {
	batchableStepProbesMu.RLock()
	defer batchableStepProbesMu.RUnlock()
	probe, ok := batchableStepProbes[taskStepKey{name: task.Name, typ: task.Type, stepType: step.Type}]
	if !ok {
		return false
	}
	return probe(task.Context)
}
