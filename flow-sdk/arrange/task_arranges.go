package arrange

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/compensate"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/padding"
)

// TaskArranges 任务编排工具
type TaskArranges struct {}

// NewTaskArranges 创建任务编排工具
func NewTaskArranges() *TaskArranges {
	return &TaskArranges{}
}

// BuildSequentialCompensableTaskStep 构建串行可补偿任务步骤
func (a *TaskArranges) BuildSequentialCompensableTaskStep(name string, taskContext *flowsdk.TaskContext, compensateAwareSteps []compensate.CompensateAwareTaskStep) compensate.CompensableTaskStep {
	return compensate.NewSequentialCompensableTaskStep(name, taskContext, compensateAwareSteps)
}

// BuildCompositeCompensableTaskStep 构建并行可补偿任务步骤
func (a *TaskArranges) BuildCompositeCompensableTaskStep(name string, taskContext *flowsdk.TaskContext, compensateAwareSteps []compensate.CompensateAwareTaskStep) compensate.CompensableTaskStep {
	return compensate.NewCompositeCompensableTaskStep(name, taskContext, compensateAwareSteps)
}

// BuildAtomicCompensateAwareStep 构建原子补偿感知步骤
func (a *TaskArranges) BuildAtomicCompensateAwareStep(normalStep, compensateStep flowsdk.TaskStep) compensate.CompensateAwareTaskStep {
	// 如果补偿步骤为空，使用 NoneTaskStep
	if compensateStep == nil {
		compensateStep = padding.NewNoneTaskStep("noneCompensate", normalStep.(interface{ GetTaskContext() *flowsdk.TaskContext }).GetTaskContext())
	}
	return compensate.NewAtomicCompensateAwareStep(normalStep, compensateStep)
}

// TaskStageDefinition 任务阶段定义
type TaskStageDefinition struct {
	Name  string
	Order int
	Steps []TaskStepDefinition
}

// TaskStepDefinition 任务步骤定义
type TaskStepDefinition struct {
	Name       string
	Type       string
	Configs    map[string]interface{}
	Stages     []TaskStageDefinition
}

// TaskDefConfigService 任务定义配置服务
type TaskDefConfigService interface {
	// LoadTaskDefinition 加载任务定义
	LoadTaskDefinition(defId string) (*TaskStageDefinition, error)
	// SaveTaskDefinition 保存任务定义
	SaveTaskDefinition(defId string, definition *TaskStageDefinition) error
}
