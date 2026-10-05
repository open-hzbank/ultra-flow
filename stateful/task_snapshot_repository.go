package stateful

import (
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/support"
	"time"
)

// TaskSnapshotRepository 任务快照仓库接口
type TaskSnapshotRepository interface {
	SaveSnapshot(snapshot *TaskSnapshot)
	Get(taskID string) *TaskSnapshot
	BatchGet(taskIDs []string, limit int) []*TaskSnapshot
	FindByIdempotentID(idempotentID, taskName, taskType string) *TaskSnapshot
	FindBySubject(publishEnv core.Env, subjectQueryParams map[string]any, taskName, taskType string, activeStatuses []core.TaskStatus) *TaskSnapshot
	FindRunningTasks(taskName, taskType string) []*TaskSnapshot
	SearchLatestOne(taskName, taskType string, publishEnv core.Env, tenant string, statuses []core.TaskStatus, subjectQueryParams map[string]any) *TaskSnapshot
	Search(creator string, publishEnvs []core.Env, order support.Order, subjectQueryParams [][2]string, targetStatuses []core.TaskStatus, pageNum, pageSize int, fromTime, toTime time.Time, taskName, taskType string) support.PageResult[TaskSnapshot]
	SelectForUpdate(taskID string) *TaskSnapshot
}
