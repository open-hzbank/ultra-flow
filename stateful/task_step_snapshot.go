package stateful

import (
	"github.com/open-hzbank/ultra-flow/core"
	"time"
)

// TaskStepSnapshot 任务步骤快照
type TaskStepSnapshot struct {
	ID          int64
	GmtCreate   time.Time
	GmtModified time.Time

	// TaskID root task snapshot id
	TaskID string
	// Name 任务步骤名称，当前任务惟一
	Name string
	// Type 步骤类型
	Type string
	// TaskStepContext 当前步骤执行上下文
	TaskStepContext map[string]any
	// Status CREATED、RUNNING、SUCCESS、FAILURE、CANCEL、SKIPPED
	Status core.TaskStepStatus
	// Description 当前任务步骤描述
	Description core.StepDescription
}
