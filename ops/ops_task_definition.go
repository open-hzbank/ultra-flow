package ops

// OpsTaskDefinition 运维任务的定义
type OpsTaskDefinition struct {
	Name          string
	Type          string
	OpsTaskBuilder OpsTaskBuilder
}

func NewOpsTaskDefinition(builder OpsTaskBuilder) *OpsTaskDefinition {
	return &OpsTaskDefinition{
		Name:          builder.GetOpsTaskName(),
		Type:          builder.GetOpsTaskType(),
		OpsTaskBuilder: builder,
	}
}
