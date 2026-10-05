package db

import (
	"encoding/json"
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/stateful"
	"log"

	"gorm.io/gorm"
)

// DbBasedTaskStepSnapshotRepository 基于数据库的任务步骤快照仓库实现
type DbBasedTaskStepSnapshotRepository struct {
	db *gorm.DB
}

// NewDbBasedTaskStepSnapshotRepository 创建基于数据库的任务步骤快照仓库
func NewDbBasedTaskStepSnapshotRepository(db *gorm.DB) *DbBasedTaskStepSnapshotRepository {
	return &DbBasedTaskStepSnapshotRepository{db: db}
}

// Save 保存任务步骤快照
func (r *DbBasedTaskStepSnapshotRepository) Save(snapshot *stateful.TaskStepSnapshot) {
	do := taskStepSnapshotToDO(snapshot)

	var existing TaskStepSnapshotDO
	err := r.db.Where("task_id = ? AND name = ?", snapshot.TaskID, snapshot.Name).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if err := r.db.Create(do).Error; err != nil {
			log.Printf("创建任务步骤快照失败: %v", err)
		}
		return
	}
	if err != nil {
		log.Printf("查询任务步骤快照失败: %v", err)
		return
	}

	do.ID = existing.ID
	if err := r.db.Save(do).Error; err != nil {
		log.Printf("更新任务步骤快照失败: %v", err)
	}
}

// GetTaskStep 根据任务ID和步骤名称获取步骤快照
func (r *DbBasedTaskStepSnapshotRepository) GetTaskStep(taskID, name string) *stateful.TaskStepSnapshot {
	var do TaskStepSnapshotDO
	if err := r.db.Where("task_id = ? AND name = ?", taskID, name).First(&do).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("查询任务步骤快照失败: %v", err)
		}
		return nil
	}
	return taskStepSnapshotFromDO(&do)
}

// GetTaskSteps 获取任务下的所有步骤快照
func (r *DbBasedTaskStepSnapshotRepository) GetTaskSteps(taskID string) []*stateful.TaskStepSnapshot {
	var dos []TaskStepSnapshotDO
	if err := r.db.Where("task_id = ?", taskID).Order("id ASC").Find(&dos).Error; err != nil {
		log.Printf("查询任务步骤快照列表失败: %v", err)
		return nil
	}

	result := make([]*stateful.TaskStepSnapshot, len(dos))
	for i := range dos {
		result[i] = taskStepSnapshotFromDO(&dos[i])
	}
	return result
}

// GetByStepTypes 根据步骤类型和状态查询步骤快照
func (r *DbBasedTaskStepSnapshotRepository) GetByStepTypes(stepType string, statuses []core.TaskStepStatus,
	contextQueryParams map[string]any) []*stateful.TaskStepSnapshot {
	query := r.db.Model(&TaskStepSnapshotDO{}).Where("type = ?", stepType)

	if len(statuses) > 0 {
		statusStrs := make([]string, len(statuses))
		for i, s := range statuses {
			statusStrs[i] = s.String()
		}
		query = query.Where("status IN ?", statusStrs)
	}

	// 应用上下文查询参数 (原始 SQL 片段)
	for _, v := range contextQueryParams {
		if s, ok := v.(string); ok {
			query = query.Where(s)
		}
	}

	var dos []TaskStepSnapshotDO
	if err := query.Find(&dos).Error; err != nil {
		log.Printf("按类型查询任务步骤快照失败: %v", err)
		return nil
	}

	result := make([]*stateful.TaskStepSnapshot, len(dos))
	for i := range dos {
		result[i] = taskStepSnapshotFromDO(&dos[i])
	}
	return result
}

// 辅助函数

func stringToTaskStepStatus(s string) core.TaskStepStatus {
	switch s {
	case "PENDING":
		return core.StepPending
	case "RUNNING":
		return core.StepRunning
	case "SUCCESS":
		return core.StepSuccess
	case "SKIPPED":
		return core.StepSkipped
	case "FAILURE":
		return core.StepFailure
	case "INTERRUPTED":
		return core.StepInterrupted
	default:
		return core.TaskStepStatus(-1)
	}
}

// 转换器: 领域对象 -> DO
func taskStepSnapshotToDO(snapshot *stateful.TaskStepSnapshot) *TaskStepSnapshotDO {
	do := &TaskStepSnapshotDO{
		ID:          snapshot.ID,
		GmtCreate:   snapshot.GmtCreate,
		GmtModified: snapshot.GmtModified,
		TaskID:      snapshot.TaskID,
		Name:        snapshot.Name,
		Type:        snapshot.Type,
		Status:      snapshot.Status.String(),
	}

	// 序列化 taskStepContext
	if snapshot.TaskStepContext != nil {
		if b, err := json.Marshal(snapshot.TaskStepContext); err == nil {
			do.TaskStepContext = string(b)
		}
	}

	// 序列化 description
	if b, err := json.Marshal(snapshot.Description); err == nil {
		do.Description = string(b)
	}

	return do
}

// 转换器: DO -> 领域对象
func taskStepSnapshotFromDO(do *TaskStepSnapshotDO) *stateful.TaskStepSnapshot {
	if do == nil {
		return nil
	}

	snapshot := &stateful.TaskStepSnapshot{
		ID:          do.ID,
		GmtCreate:   do.GmtCreate,
		GmtModified: do.GmtModified,
		TaskID:      do.TaskID,
		Name:        do.Name,
		Type:        do.Type,
		Status:      stringToTaskStepStatus(do.Status),
	}

	// 反序列化 taskStepContext
	if do.TaskStepContext != "" {
		var ctx map[string]any
		if err := json.Unmarshal([]byte(do.TaskStepContext), &ctx); err == nil {
			snapshot.TaskStepContext = ctx
		}
	}

	// 反序列化 description
	if do.Description != "" {
		var desc core.StepDescription
		if err := json.Unmarshal([]byte(do.Description), &desc); err == nil {
			snapshot.Description = desc
		}
	}

	return snapshot
}

// 确保接口实现
var _ stateful.TaskStepSnapshotRepository = (*DbBasedTaskStepSnapshotRepository)(nil)
