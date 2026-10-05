package core

import (
	"sync"
)

// TaskContext 任务上下文，当前执行过程惟一，不替换此实例
type TaskContext struct {
	// 当前任务引用
	task            Task
	context         map[string]any
	mu              sync.RWMutex
	// 用于外部传递任务步骤的修改，如对于回调的处理
	stepTransfer    TaskStepTransfer
	// 运行时注入的上下文，不会持久化
	runtimeContext  map[string]any
	runtimeMu       sync.RWMutex
	stepPaths       []TaskStep
	pathsMu         sync.Mutex
}

func NewTaskContext(task Task, context map[string]any) *TaskContext {
	return NewTaskContextWithTransfer(task, context, nil)
}

func NewTaskContextWithTransfer(task Task, context map[string]any, transfer TaskStepTransfer) *TaskContext {
	ctx := make(map[string]any)
	for k, v := range context {
		ctx[k] = v
	}
	return &TaskContext{
		task:           task,
		context:        ctx,
		stepTransfer:   transfer,
		runtimeContext: make(map[string]any),
	}
}

func (c *TaskContext) Put(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.context[key] = value
}

func (c *TaskContext) PutAll(m map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range m {
		c.context[k] = v
	}
}

func (c *TaskContext) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.context, key)
}

func (c *TaskContext) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.context[key]
}

func (c *TaskContext) GetContext() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cp := make(map[string]any, len(c.context))
	for k, v := range c.context {
		cp[k] = v
	}
	return cp
}

func (c *TaskContext) GetTask() Task { return c.task }

func (c *TaskContext) GetTaskStepTransfer() TaskStepTransfer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stepTransfer
}

func (c *TaskContext) SetTaskStepTransfer(transfer TaskStepTransfer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stepTransfer = transfer
}

func (c *TaskContext) AddPath(step TaskStep) {
	c.pathsMu.Lock()
	defer c.pathsMu.Unlock()
	c.stepPaths = append(c.stepPaths, step)
}

func (c *TaskContext) GetTaskStepPaths() []TaskStep {
	c.pathsMu.Lock()
	defer c.pathsMu.Unlock()
	cp := make([]TaskStep, len(c.stepPaths))
	copy(cp, c.stepPaths)
	return cp
}

func (c *TaskContext) GetRuntimeContext() map[string]any {
	return c.runtimeContext
}

func (c *TaskContext) PutRuntime(key string, value any) {
	c.runtimeMu.Lock()
	defer c.runtimeMu.Unlock()
	c.runtimeContext[key] = value
}

func (c *TaskContext) GetRuntime(key string) any {
	c.runtimeMu.RLock()
	defer c.runtimeMu.RUnlock()
	return c.runtimeContext[key]
}
