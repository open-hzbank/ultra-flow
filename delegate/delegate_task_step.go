package delegate

import (
	"hzbank.com.cn/ultra-flow/core"
)

const DelegateTaskStepType = "delegate"

// DelegateTaskStep 代理步骤: 将所有操作透明委托给被代理的步骤
type DelegateTaskStep struct {
	Delegated core.TaskStep
	name      string
	taskCtx   *core.TaskContext
}

func NewDelegateTaskStep(delegated core.TaskStep) *DelegateTaskStep {
	type namedStep interface {
		GetTaskContext() *core.TaskContext
	}
	var taskCtx *core.TaskContext
	if ns, ok := delegated.(namedStep); ok {
		taskCtx = ns.GetTaskContext()
	}
	return &DelegateTaskStep{
		Delegated: delegated,
		name:      delegated.GetName(),
		taskCtx:   taskCtx,
	}
}

func NewDelegateTaskStepWithName(name string, delegated core.TaskStep) *DelegateTaskStep {
	type namedStep interface {
		GetTaskContext() *core.TaskContext
	}
	var taskCtx *core.TaskContext
	if ns, ok := delegated.(namedStep); ok {
		taskCtx = ns.GetTaskContext()
	}
	return &DelegateTaskStep{
		Delegated: delegated,
		name:      name,
		taskCtx:   taskCtx,
	}
}

func (s *DelegateTaskStep) GetName() string { return s.name }
func (s *DelegateTaskStep) GetType() string { return DelegateTaskStepType }

// 在父类中 doExecute 被 execute 调用,
// 而本类中两个方法被各自重写为代理调用, 不再具备上下游调用关系,
// 为防止子类只重写 doExecute, 导致无法被 execute 调用, 将本方法设置为 final: 即只允许子类使用实现透明代理, 但不允许重写
func (s *DelegateTaskStep) Execute() core.TaskStepResult {
	return s.Delegated.Execute()
}

func (s *DelegateTaskStep) OnSuccess() {
	s.Delegated.OnSuccess()
}

func (s *DelegateTaskStep) OnFailure(err error) {
	s.Delegated.OnFailure(err)
}

func (s *DelegateTaskStep) OnInterrupt() {
	s.Delegated.OnInterrupt()
}

func (s *DelegateTaskStep) GetTaskContext() *core.TaskContext {
	return s.taskCtx
}

type statusAware interface {
	GetStatus() core.TaskStepStatus
}

func (s *DelegateTaskStep) GetStatus() core.TaskStepStatus {
	if sa, ok := s.Delegated.(statusAware); ok {
		return sa.GetStatus()
	}
	return core.StepPending
}

type stepContextAware interface {
	GetTaskStepContext() *core.TaskStepContext
}

func (s *DelegateTaskStep) GetTaskStepContext() *core.TaskStepContext {
	if sca, ok := s.Delegated.(stepContextAware); ok {
		return sca.GetTaskStepContext()
	}
	return nil
}

type describable interface {
	Describe() core.StepDescription
}

func (s *DelegateTaskStep) Describe() core.StepDescription {
	if d, ok := s.Delegated.(describable); ok {
		return d.Describe()
	}
	return core.NewStepDescription(s.name, nil)
}
