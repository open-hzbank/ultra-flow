package mem

import (
	"strings"
	"sync"
	"time"

	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/stateful"
	"github.com/open-hzbank/ultra-flow/support"
)

// MemoryMockedTransaction 内存 mock 事务实现: 无事务
type MemoryMockedTransaction struct{}

func NewMemoryMockedTransaction() *MemoryMockedTransaction {
	return &MemoryMockedTransaction{}
}

func (t *MemoryMockedTransaction) Execute(action stateful.TransactionAction) any {
	return action(nil)
}

// MemoryTaskSnapshotRepository 基于内存的任务快照仓库
type MemoryTaskSnapshotRepository struct {
	mu    sync.RWMutex
	tasks []*stateful.TaskSnapshot
}

func NewMemoryTaskSnapshotRepository() *MemoryTaskSnapshotRepository {
	return &MemoryTaskSnapshotRepository{
		tasks: make([]*stateful.TaskSnapshot, 0),
	}
}

func (r *MemoryTaskSnapshotRepository) SaveSnapshot(snapshot *stateful.TaskSnapshot) {
	newSnapshot := r.buildSnapshotWithTime(snapshot)
	r.mu.Lock()
	defer r.mu.Unlock()
	// 移除相同 taskId 的旧记录
	for i, t := range r.tasks {
		if t.TaskID == newSnapshot.TaskID {
			r.tasks = append(r.tasks[:i], r.tasks[i+1:]...)
			break
		}
	}
	r.tasks = append(r.tasks, newSnapshot)
}

func (r *MemoryTaskSnapshotRepository) buildSnapshotWithTime(snapshot *stateful.TaskSnapshot) *stateful.TaskSnapshot {
	now := time.Now()
	return &stateful.TaskSnapshot{
		ID:           snapshot.ID,
		GmtCreate:    now,
		GmtModified:  now,
		TaskID:       snapshot.TaskID,
		IdempotentID: snapshot.IdempotentID,
		Name:         snapshot.Name,
		Type:         snapshot.Type,
		Subject:      snapshot.Subject,
		PublishEnv:   snapshot.PublishEnv,
		Context:      snapshot.Context,
		Creator:      snapshot.Creator,
		Status:       snapshot.Status,
		Description:  snapshot.Description,
	}
}

func (r *MemoryTaskSnapshotRepository) Get(taskID string) *stateful.TaskSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.tasks {
		if t.TaskID == taskID {
			return t
		}
	}
	return nil
}

func (r *MemoryTaskSnapshotRepository) BatchGet(taskIDs []string, limit int) []*stateful.TaskSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	idSet := make(map[string]bool, len(taskIDs))
	for _, id := range taskIDs {
		idSet[id] = true
	}
	var result []*stateful.TaskSnapshot
	for _, t := range r.tasks {
		if idSet[t.TaskID] {
			result = append(result, t)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result
}

func (r *MemoryTaskSnapshotRepository) FindByIdempotentID(idempotentID, taskName, taskType string) *stateful.TaskSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.tasks {
		if t.IdempotentID == idempotentID && t.Name == taskName && t.Type == taskType {
			return t
		}
	}
	return nil
}

func (r *MemoryTaskSnapshotRepository) FindBySubject(publishEnv core.Env, subjectQueryParams map[string]any, taskName, taskType string, activeStatuses []core.TaskStatus) *stateful.TaskSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	statusSet := make(map[core.TaskStatus]bool, len(activeStatuses))
	for _, s := range activeStatuses {
		statusSet[s] = true
	}
	for _, t := range r.tasks {
		if t.PublishEnv == publishEnv && t.Name == taskName && statusSet[t.Status] {
			return t
		}
	}
	return nil
}

func (r *MemoryTaskSnapshotRepository) FindRunningTasks(taskName, taskType string) []*stateful.TaskSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*stateful.TaskSnapshot
	for _, t := range r.tasks {
		if t.Name == taskName && t.Type == taskType && t.Status.IsActive() {
			result = append(result, t)
		}
	}
	return result
}

func (r *MemoryTaskSnapshotRepository) SearchLatestOne(taskName, taskType string, publishEnv core.Env, tenant string, statuses []core.TaskStatus, subjectQueryParams map[string]any) *stateful.TaskSnapshot {
	return nil
}

func (r *MemoryTaskSnapshotRepository) Search(creator string, publishEnvs []core.Env, order support.Order, subjectQueryParams [][2]string, targetStatuses []core.TaskStatus, pageNum, pageSize int, fromTime, toTime time.Time, taskName, taskType string) support.PageResult[stateful.TaskSnapshot] {
	return support.PageResult[stateful.TaskSnapshot]{}
}

func (r *MemoryTaskSnapshotRepository) SelectForUpdate(taskID string) *stateful.TaskSnapshot {
	return r.Get(taskID)
}

// MemoryTaskStepSnapshotRepository 基于内存的步骤快照仓库
type MemoryTaskStepSnapshotRepository struct {
	mu    sync.RWMutex
	steps []*stateful.TaskStepSnapshot
}

func NewMemoryTaskStepSnapshotRepository() *MemoryTaskStepSnapshotRepository {
	return &MemoryTaskStepSnapshotRepository{
		steps: make([]*stateful.TaskStepSnapshot, 0),
	}
}

func (r *MemoryTaskStepSnapshotRepository) Save(snapshot *stateful.TaskStepSnapshot) {
	newSnapshot := r.buildSnapshotWithTime(snapshot)
	r.mu.Lock()
	defer r.mu.Unlock()
	// 移除相同 (taskId, name) 的旧记录
	for i, s := range r.steps {
		if s.TaskID == newSnapshot.TaskID && s.Name == newSnapshot.Name {
			r.steps = append(r.steps[:i], r.steps[i+1:]...)
			break
		}
	}
	r.steps = append(r.steps, newSnapshot)
}

func (r *MemoryTaskStepSnapshotRepository) buildSnapshotWithTime(snapshot *stateful.TaskStepSnapshot) *stateful.TaskStepSnapshot {
	now := time.Now()
	return &stateful.TaskStepSnapshot{
		ID:              snapshot.ID,
		GmtCreate:       now,
		GmtModified:     now,
		TaskID:          snapshot.TaskID,
		Name:            snapshot.Name,
		Type:            snapshot.Type,
		TaskStepContext: snapshot.TaskStepContext,
		Status:          snapshot.Status,
		Description:     snapshot.Description,
	}
}

func (r *MemoryTaskStepSnapshotRepository) GetTaskStep(taskID, name string) *stateful.TaskStepSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.steps {
		if s.TaskID == taskID && s.Name == name {
			return s
		}
	}
	return nil
}

func (r *MemoryTaskStepSnapshotRepository) GetTaskSteps(taskID string) []*stateful.TaskStepSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*stateful.TaskStepSnapshot
	for _, s := range r.steps {
		if s.TaskID == taskID {
			result = append(result, s)
		}
	}
	return result
}

func (r *MemoryTaskStepSnapshotRepository) GetByStepTypes(stepType string, statuses []core.TaskStepStatus, contextQueryParams map[string]any) []*stateful.TaskStepSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	statusSet := make(map[core.TaskStepStatus]bool, len(statuses))
	for _, s := range statuses {
		statusSet[s] = true
	}
	var result []*stateful.TaskStepSnapshot
	for _, s := range r.steps {
		if s.Type == stepType && statusSet[s.Status] {
			result = append(result, s)
		}
	}
	return result
}

// 辅助函数: 字符串相等判断 (忽略空值)
func stringEquals(a, b string) bool {
	return strings.EqualFold(a, b)
}
