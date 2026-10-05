package delegate

import (
	"hzbank.com.cn/ultra-flow/core"
)

// DelegateTask 代理任务: 可继承该类, 针对指定方法实现自定义的能力
type DelegateTask struct {
	// 被代理的任务
	Delegated core.Task
}

func NewDelegateTask(delegated core.Task) *DelegateTask {
	return &DelegateTask{Delegated: delegated}
}

func (t *DelegateTask) GetTaskID() string      { return t.Delegated.GetTaskID() }
func (t *DelegateTask) GetName() string         { return t.Delegated.GetName() }
func (t *DelegateTask) GetType() string         { return t.Delegated.GetType() }
func (t *DelegateTask) GetSubject() any         { return t.Delegated.GetSubject() }
func (t *DelegateTask) GetCreator() string      { return t.Delegated.GetCreator() }
func (t *DelegateTask) GetPublishEnv() core.Env { return t.Delegated.GetPublishEnv() }
func (t *DelegateTask) Execute() core.TaskResult { return t.Delegated.Execute() }
func (t *DelegateTask) Cancel()                 { t.Delegated.Cancel() }
func (t *DelegateTask) Interrupt()              { t.Delegated.Interrupt() }
