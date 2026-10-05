package async

import (
	"encoding/json"
	"fmt"

	"hzbank.com.cn/ultra-flow/core"
)

const stepBasedSnapshotsKey = "taskSnapshots"

// TaskStatus 多任务管理中的任务状态
type TaskStatus int

const (
	TaskStatusPending  TaskStatus = iota // 尚未提交
	TaskStatusSkipped                    // 已跳过
	TaskStatusRunning                    // 运行中
	TaskStatusSuccess                    // 成功
	TaskStatusFailure                    // 失败
	TaskStatusCanceled                   // 已取消
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "PENDING"
	case TaskStatusSkipped:
		return "SKIPPED"
	case TaskStatusRunning:
		return "RUNNING"
	case TaskStatusSuccess:
		return "SUCCESS"
	case TaskStatusFailure:
		return "FAILURE"
	case TaskStatusCanceled:
		return "CANCELED"
	default:
		return "UNKNOWN"
	}
}

func (s TaskStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *TaskStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	switch str {
	case "PENDING":
		*s = TaskStatusPending
	case "SKIPPED":
		*s = TaskStatusSkipped
	case "RUNNING":
		*s = TaskStatusRunning
	case "SUCCESS":
		*s = TaskStatusSuccess
	case "FAILURE":
		*s = TaskStatusFailure
	case "CANCELED":
		*s = TaskStatusCanceled
	default:
		return fmt.Errorf("unknown TaskStatus: %s", str)
	}
	return nil
}

func IsNormalTaskStatus(status TaskStatus) bool {
	return status == TaskStatusRunning || status == TaskStatusSuccess
}

func (s TaskStatus) IsRunning() bool {
	return s != TaskStatusSuccess && s != TaskStatusCanceled && s != TaskStatusSkipped
}

func (s TaskStatus) IsSuccessOrSkipped() bool {
	return s == TaskStatusSuccess || s == TaskStatusSkipped
}

func (s TaskStatus) IsSkipped() bool {
	return s == TaskStatusSkipped
}

func (s TaskStatus) Started() bool {
	return s != TaskStatusPending
}

// MapToTaskStepStatus 将 TaskStatus 映射为 TaskStepStatus
func MapToTaskStepStatus(status TaskStatus) core.TaskStepStatus {
	switch status {
	case TaskStatusRunning:
		return core.StepRunning
	case TaskStatusSuccess:
		return core.StepSuccess
	case TaskStatusFailure:
		return core.StepFailure
	case TaskStatusCanceled:
		return core.StepInterrupted
	case TaskStatusSkipped:
		return core.StepSkipped
	case TaskStatusPending:
		return core.StepPending
	default:
		return core.StepPending
	}
}

// MultiTaskSnapshot 单个任务的快照
type MultiTaskSnapshot struct {
	// 外部发布系统的任务 id (工单号), 如果为空则说明任务未提交成功
	TaskID      string     `json:"taskId,omitempty"`
	// 任务执行状态
	Status      TaskStatus `json:"status"`
	// 错误信息 (提交报错或任务执行报错)
	ErrorReason string     `json:"errorReason,omitempty"`
}

func NewNormalSnapshot(taskID string, status TaskStatus, errorReason string) *MultiTaskSnapshot {
	return &MultiTaskSnapshot{TaskID: taskID, Status: status, ErrorReason: errorReason}
}

func NewFailSnapshot(errorReason string) *MultiTaskSnapshot {
	return &MultiTaskSnapshot{TaskID: "", Status: TaskStatusFailure, ErrorReason: errorReason}
}

func NewPendingSnapshot() *MultiTaskSnapshot {
	return &MultiTaskSnapshot{Status: TaskStatusPending}
}

// TaskSubmitResult 任务提交结果
type TaskSubmitResult struct {
	// 是否提交成功
	Successful  bool
	// 成功时的任务 id
	TaskID      string
	// 失败时的错误原因
	ErrorReason string
}

func SubmitSuccess(taskID string) *TaskSubmitResult {
	return &TaskSubmitResult{Successful: true, TaskID: taskID}
}

func SubmitFail(errorReason string) *TaskSubmitResult {
	return &TaskSubmitResult{Successful: false, ErrorReason: errorReason}
}

// MultiTaskStatusManager 批量请求提交场景下的多任务状态统一管理
type MultiTaskStatusManager struct {
	taskSnapshotsMap    map[string]*MultiTaskSnapshot
	taskStepContext     *core.TaskStepContext
	taskContext         *core.TaskContext
	taskBasedSnapshotKey string
}

// NewStepBasedManager 基于步骤上下文的构造
func NewStepBasedManager(stepCtx *core.TaskStepContext) *MultiTaskStatusManager {
	m := &MultiTaskStatusManager{
		taskStepContext: stepCtx,
		taskSnapshotsMap: make(map[string]*MultiTaskSnapshot),
	}
	raw := stepCtx.Get(stepBasedSnapshotsKey)
	if raw != nil {
		if str, ok := raw.(string); ok && str != "" {
			_ = json.Unmarshal([]byte(str), &m.taskSnapshotsMap)
		}
	}
	return m
}

// NewTaskBasedManager 基于任务上下文的构造
func NewTaskBasedManager(taskCtx *core.TaskContext, snapshotKey string, totalTaskNames []string) *MultiTaskStatusManager {
	m := &MultiTaskStatusManager{
		taskContext:         taskCtx,
		taskBasedSnapshotKey: snapshotKey,
		taskSnapshotsMap:    make(map[string]*MultiTaskSnapshot),
	}
	raw := taskCtx.Get(snapshotKey)
	if raw != nil {
		if str, ok := raw.(string); ok && str != "" {
			_ = json.Unmarshal([]byte(str), &m.taskSnapshotsMap)
		}
	} else {
		for _, name := range totalTaskNames {
			m.taskSnapshotsMap[name] = NewPendingSnapshot()
		}
	}
	return m
}

// SaveTaskSnapshots 序列化并持久化快照
func (m *MultiTaskStatusManager) SaveTaskSnapshots() {
	data, _ := json.Marshal(m.taskSnapshotsMap)
	serialized := string(data)
	if m.taskStepContext != nil {
		m.taskStepContext.Put(stepBasedSnapshotsKey, serialized)
	} else if m.taskContext != nil {
		m.taskContext.Put(m.taskBasedSnapshotKey, serialized)
	}
}

func (m *MultiTaskStatusManager) GetTaskSnapshots() map[string]*MultiTaskSnapshot {
	cp := make(map[string]*MultiTaskSnapshot, len(m.taskSnapshotsMap))
	for k, v := range m.taskSnapshotsMap {
		cp[k] = v
	}
	return cp
}

func (m *MultiTaskStatusManager) GetTaskSnapshot(key string) *MultiTaskSnapshot {
	return m.taskSnapshotsMap[key]
}

// AllTasksSubmitted 是否所有任务都已提交
func (m *MultiTaskStatusManager) AllTasksSubmitted() bool {
	for key := range m.taskSnapshotsMap {
		if !m.isTaskSubmitted(key) {
			return false
		}
	}
	return true
}

func (m *MultiTaskStatusManager) AllTasksSubmittedOrSkippedOrSuccess() bool {
	for key := range m.taskSnapshotsMap {
		if !m.isTaskSubmitted(key) && !m.isTaskSkipped(key) && !m.isTaskSuccess(key) {
			return false
		}
	}
	return true
}

func (m *MultiTaskStatusManager) isTaskSuccess(key string) bool {
	snap := m.taskSnapshotsMap[key]
	return snap != nil && snap.Status == TaskStatusSuccess
}

func (m *MultiTaskStatusManager) isTaskSkipped(key string) bool {
	snap := m.taskSnapshotsMap[key]
	return snap != nil && snap.Status == TaskStatusSkipped
}

func (m *MultiTaskStatusManager) isTaskSubmitted(key string) bool {
	snap := m.taskSnapshotsMap[key]
	// 有任务 id 即已提交
	return snap != nil && snap.TaskID != ""
}

func (m *MultiTaskStatusManager) AllSkippedTasks() []string {
	var result []string
	for key := range m.taskSnapshotsMap {
		if m.isTaskSkipped(key) {
			result = append(result, key)
		}
	}
	return result
}

func (m *MultiTaskStatusManager) AllTaskSnapshotKeys() []string {
	keys := make([]string, 0, len(m.taskSnapshotsMap))
	for k := range m.taskSnapshotsMap {
		keys = append(keys, k)
	}
	return keys
}

func (m *MultiTaskStatusManager) UnSubmittedTaskSnapshotKeys() []string {
	var result []string
	for key := range m.taskSnapshotsMap {
		if !m.isTaskSubmitted(key) && !m.isTaskSkipped(key) && !m.isTaskSuccess(key) {
			result = append(result, key)
		}
	}
	return result
}

func (m *MultiTaskStatusManager) SubmittedTaskSnapshotKeys() []string {
	var result []string
	for key := range m.taskSnapshotsMap {
		if m.isTaskSubmitted(key) {
			result = append(result, key)
		}
	}
	return result
}

// CalcMultiTaskStatus 根据当前管理的多任务状态计算整体状态
func (m *MultiTaskStatusManager) CalcMultiTaskStatus() TaskStatus {
	snapshots := m.GetTaskSnapshots()
	// 没有任务提交, 直接跳过
	if len(snapshots) == 0 {
		return TaskStatusSkipped
	}

	statuses := make([]TaskStatus, 0, len(snapshots))
	for _, snap := range snapshots {
		statuses = append(statuses, snap.Status)
	}

	// PENDING 状态的判定
	allPending := true
	for _, st := range statuses {
		if st.Started() {
			allPending = false
			break
		}
	}
	if allPending {
		return TaskStatusPending
	}

	// 失败状态的判定
	anyFailure := false
	for _, st := range statuses {
		if st == TaskStatusFailure {
			anyFailure = true
			break
		}
	}
	if anyFailure {
		return TaskStatusFailure
	}

	// 跳过状态的判定
	allSkipped := true
	for _, st := range statuses {
		if !st.IsSkipped() {
			allSkipped = false
			break
		}
	}
	if allSkipped {
		return TaskStatusSkipped
	}

	// 成功状态的判定
	allSuccessOrSkipped := true
	for _, st := range statuses {
		if !st.IsSuccessOrSkipped() {
			allSuccessOrSkipped = false
			break
		}
	}
	if allSuccessOrSkipped {
		return TaskStatusSuccess
	}

	// 取消状态的判定: 至少有一个 task 是 CANCELED, 除了 CANCELED task, 其余 task 必须是 SUCCESS
	anyCanceled := false
	allSuccessOrCanceled := true
	for _, st := range statuses {
		if st == TaskStatusCanceled {
			anyCanceled = true
		}
		if st != TaskStatusCanceled && st != TaskStatusSuccess {
			allSuccessOrCanceled = false
		}
	}
	if anyCanceled && allSuccessOrCanceled {
		return TaskStatusCanceled
	}

	// 其余状态为运行中
	return TaskStatusRunning
}

// BuildErrorReasonMap 收集所有非空的错误信息
func (m *MultiTaskStatusManager) BuildErrorReasonMap() map[string]string {
	result := make(map[string]string)
	for k, v := range m.taskSnapshotsMap {
		if v.ErrorReason != "" {
			result[k] = v.ErrorReason
		}
	}
	return result
}

// RecordSubmitResult 记录任务提交结果
func (m *MultiTaskStatusManager) RecordSubmitResult(key string, submitResult *TaskSubmitResult, status ...TaskStatus) {
	actualStatus := TaskStatusRunning
	if len(status) > 0 {
		actualStatus = status[0]
	}
	if submitResult.Successful {
		m.taskSnapshotsMap[key] = NewNormalSnapshot(submitResult.TaskID, actualStatus, "")
	} else {
		m.taskSnapshotsMap[key] = NewFailSnapshot(submitResult.ErrorReason)
	}
}

// RefreshTaskStatus 刷新任务状态
func (m *MultiTaskStatusManager) RefreshTaskStatus(key string, statusOrError any) {
	snap, ok := m.taskSnapshotsMap[key]
	if !ok {
		return
	}
	switch v := statusOrError.(type) {
	case TaskStatus:
		m.taskSnapshotsMap[key] = NewNormalSnapshot(snap.TaskID, v, "")
	case string:
		m.taskSnapshotsMap[key] = NewNormalSnapshot(snap.TaskID, TaskStatusFailure, v)
	}
}
