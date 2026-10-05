package core

import "sync"

// TaskStepContext 步骤级隔离上下文
type TaskStepContext struct {
	// 当前步骤所属的任务引用
	task    Task
	// 当前所属步骤引用
	step    TaskStep
	context map[string]any
	mu      sync.RWMutex
}

func NewTaskStepContext(task Task, step TaskStep) *TaskStepContext {
	return &TaskStepContext{
		task:    task,
		step:    step,
		context: make(map[string]any),
	}
}

func (c *TaskStepContext) Put(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.context[key] = value
}

func (c *TaskStepContext) PutAll(m map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range m {
		c.context[k] = v
	}
}

func (c *TaskStepContext) PutIfAbsent(m map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range m {
		if _, ok := c.context[k]; !ok {
			c.context[k] = v
		}
	}
}

func (c *TaskStepContext) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.context[key]
}

func (c *TaskStepContext) GetContext() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cp := make(map[string]any, len(c.context))
	for k, v := range c.context {
		cp[k] = v
	}
	return cp
}

func (c *TaskStepContext) GetTask() Task     { return c.task }
func (c *TaskStepContext) GetTaskStep() TaskStep { return c.step }
