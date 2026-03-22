package flowsdk

import (
	"sync"
)

// TaskContext 任务上下文
type TaskContext struct {
	task             Task
	values           map[string]interface{}
	mutex            sync.RWMutex
	taskStepTransfer TaskStepTransfer
	path             []TaskStep
}

// NewTaskContext 创建任务上下文
func NewTaskContext(task Task) *TaskContext {
	return &TaskContext{
		task:   task,
		values: make(map[string]interface{}),
		path:   make([]TaskStep, 0),
	}
}

// GetTask 获取任务
func (c *TaskContext) GetTask() Task {
	return c.task
}

// Put 设置上下文值
func (c *TaskContext) Put(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.values[key] = value
}

// Get 获取上下文值
func (c *TaskContext) Get(key string) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.values[key]
}

// PutAll 批量设置上下文值
func (c *TaskContext) PutAll(values map[string]interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for k, v := range values {
		c.values[k] = v
	}
}

// GetTaskStepTransfer 获取任务步骤转换器
func (c *TaskContext) GetTaskStepTransfer() TaskStepTransfer {
	return c.taskStepTransfer
}

// SetTaskStepTransfer 设置任务步骤转换器
func (c *TaskContext) SetTaskStepTransfer(transfer TaskStepTransfer) {
	c.taskStepTransfer = transfer
}

// AddPath 添加执行路径
func (c *TaskContext) AddPath(step TaskStep) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.path = append(c.path, step)
}

// GetPath 获取执行路径
func (c *TaskContext) GetPath() []TaskStep {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.path
}

// TaskStepContext 任务步骤上下文
type TaskStepContext struct {
	task        Task
	taskStep    TaskStep
	values      map[string]interface{}
	mutex       sync.RWMutex
}

// NewTaskStepContext 创建任务步骤上下文
func NewTaskStepContext(task Task, taskStep TaskStep) *TaskStepContext {
	return &TaskStepContext{
		task:     task,
		taskStep: taskStep,
		values:   make(map[string]interface{}),
	}
}

// GetTask 获取任务
func (c *TaskStepContext) GetTask() Task {
	return c.task
}

// GetTaskStep 获取任务步骤
func (c *TaskStepContext) GetTaskStep() TaskStep {
	return c.taskStep
}

// Put 设置上下文值
func (c *TaskStepContext) Put(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.values[key] = value
}

// Get 获取上下文值
func (c *TaskStepContext) Get(key string) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.values[key]
}

// PutAll 批量设置上下文值
func (c *TaskStepContext) PutAll(values map[string]interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for k, v := range values {
		c.values[k] = v
	}
}

// TaskStepTransfer 任务步骤转换器
type TaskStepTransfer interface {
	Transfer(step TaskStep)
}
