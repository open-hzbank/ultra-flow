package db

import (
	"encoding/json"
	"log"
	"time"
	"hzbank.com.cn/ultra-flow/core"
	"hzbank.com.cn/ultra-flow/stateful"
	"hzbank.com.cn/ultra-flow/support"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DbBasedTaskSnapshotRepository 基于数据库的任务快照仓库实现
type DbBasedTaskSnapshotRepository struct {
	db *gorm.DB
}

// NewDbBasedTaskSnapshotRepository 创建基于数据库的任务快照仓库
func NewDbBasedTaskSnapshotRepository(db *gorm.DB) *DbBasedTaskSnapshotRepository {
	return &DbBasedTaskSnapshotRepository{db: db}
}

// SaveSnapshot 保存任务快照
func (r *DbBasedTaskSnapshotRepository) SaveSnapshot(snapshot *stateful.TaskSnapshot) {
	do := taskSnapshotToDO(snapshot)

	var existing TaskSnapshotDO
	err := r.db.Where("task_id = ?", snapshot.TaskID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if err := r.db.Create(do).Error; err != nil {
			log.Printf("创建任务快照失败: %v", err)
		}
		return
	}
	if err != nil {
		log.Printf("查询任务快照失败: %v", err)
		return
	}

	do.ID = existing.ID
	if err := r.db.Save(do).Error; err != nil {
		log.Printf("更新任务快照失败: %v", err)
	}
}

// Get 根据任务ID获取任务快照
func (r *DbBasedTaskSnapshotRepository) Get(taskID string) *stateful.TaskSnapshot {
	var do TaskSnapshotDO
	if err := r.db.Where("task_id = ?", taskID).First(&do).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("查询任务快照失败: %v", err)
		}
		return nil
	}
	return taskSnapshotFromDO(&do)
}

// BatchGet 批量获取任务快照
func (r *DbBasedTaskSnapshotRepository) BatchGet(taskIDs []string, limit int) []*stateful.TaskSnapshot {
	if len(taskIDs) == 0 {
		return []*stateful.TaskSnapshot{}
	}

	if limit > 0 && limit < len(taskIDs) {
		taskIDs = taskIDs[:limit]
	}

	var dos []TaskSnapshotDO
	if err := r.db.Where("task_id IN ?", taskIDs).Find(&dos).Error; err != nil {
		log.Printf("批量查询任务快照失败: %v", err)
		return []*stateful.TaskSnapshot{}
	}

	result := make([]*stateful.TaskSnapshot, len(dos))
	for i := range dos {
		result[i] = taskSnapshotFromDO(&dos[i])
	}
	return result
}

// FindByIdempotentID 根据幂等ID查找任务快照
func (r *DbBasedTaskSnapshotRepository) FindByIdempotentID(idempotentID, taskName, taskType string) *stateful.TaskSnapshot {
	var do TaskSnapshotDO
	err := r.db.Where("idempotent_id = ? AND name = ? AND type = ?", idempotentID, taskName, taskType).First(&do).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("查询任务快照失败: %v", err)
		}
		return nil
	}
	return taskSnapshotFromDO(&do)
}

// FindBySubject 根据主体信息查找任务快照
func (r *DbBasedTaskSnapshotRepository) FindBySubject(publishEnv core.Env, subjectQueryParams map[string]any, taskName, taskType string, activeStatuses []core.TaskStatus) *stateful.TaskSnapshot {
	query := r.db.Model(&TaskSnapshotDO{})

	// 应用主体查询参数 (原始 SQL 片段)
	for _, v := range subjectQueryParams {
		if s, ok := v.(string); ok {
			query = query.Where(s)
		}
	}

	statusStrs := taskStatusesToStrings(activeStatuses)
	if len(statusStrs) > 0 {
		query = query.Where("status IN ?", statusStrs)
	}

	var do TaskSnapshotDO
	err := query.Where("publish_env = ? AND name = ? AND type = ?", string(publishEnv), taskName, taskType).
		Order("gmt_modified DESC").
		Limit(1).
		First(&do).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("查询任务快照失败: %v", err)
		}
		return nil
	}
	return taskSnapshotFromDO(&do)
}

// FindRunningTasks 查找运行中的任务
func (r *DbBasedTaskSnapshotRepository) FindRunningTasks(taskName, taskType string) []*stateful.TaskSnapshot {
	activeStatuses := core.GetActiveStatuses()
	statusStrs := taskStatusesToStrings(activeStatuses)

	var dos []TaskSnapshotDO
	if err := r.db.Where("name = ? AND type = ? AND status IN ?", taskName, taskType, statusStrs).Find(&dos).Error; err != nil {
		log.Printf("查询运行中任务失败: %v", err)
		return []*stateful.TaskSnapshot{}
	}

	result := make([]*stateful.TaskSnapshot, len(dos))
	for i := range dos {
		result[i] = taskSnapshotFromDO(&dos[i])
	}
	return result
}

// SearchLatestOne 搜索最新的一条任务快照
func (r *DbBasedTaskSnapshotRepository) SearchLatestOne(taskName, taskType string, publishEnv core.Env, tenant string, statuses []core.TaskStatus, subjectQueryParams map[string]any) *stateful.TaskSnapshot {
	query := r.db.Model(&TaskSnapshotDO{})

	// 应用主体查询参数 (原始 SQL 片段)
	for _, v := range subjectQueryParams {
		if s, ok := v.(string); ok {
			query = query.Where(s)
		}
	}

	query = query.Where("name = ? AND type = ?", taskName, taskType)
	if publishEnv != "" {
		query = query.Where("publish_env = ?", string(publishEnv))
	}

	statusStrs := taskStatusesToStrings(statuses)
	if len(statusStrs) > 0 {
		query = query.Where("status IN ?", statusStrs)
	}

	var do TaskSnapshotDO
	err := query.Order("gmt_modified DESC").Limit(1).First(&do).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("搜索最新任务快照失败: %v", err)
		}
		return nil
	}
	return taskSnapshotFromDO(&do)
}

// Search 分页搜索任务快照
func (r *DbBasedTaskSnapshotRepository) Search(creator string, publishEnvs []core.Env, order support.Order, subjectQueryParams [][2]string, targetStatuses []core.TaskStatus, pageNum, pageSize int, fromTime, toTime time.Time, taskName, taskType string) support.PageResult[stateful.TaskSnapshot] {
	query := r.db.Model(&TaskSnapshotDO{})

	// 应用主体查询参数
	for _, pair := range subjectQueryParams {
		query = query.Where(pair[0], pair[1])
	}

	if creator != "" {
		query = query.Where("creator = ?", creator)
	}

	if len(publishEnvs) > 0 {
		envStrs := make([]string, len(publishEnvs))
		for i, e := range publishEnvs {
			envStrs[i] = string(e)
		}
		query = query.Where("publish_env IN ?", envStrs)
	}

	if !fromTime.IsZero() {
		query = query.Where("gmt_create > ?", fromTime)
	}
	if !toTime.IsZero() {
		query = query.Where("gmt_create < ?", toTime)
	}

	if taskName != "" {
		query = query.Where("name = ?", taskName)
	}
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}

	statusStrs := taskStatusesToStrings(targetStatuses)
	if len(statusStrs) > 0 {
		query = query.Where("status IN ?", statusStrs)
	}

	// 排序
	if order == support.OrderAsc {
		query = query.Order("gmt_modified ASC")
	} else {
		query = query.Order("gmt_modified DESC")
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("计算任务快照总数失败: %v", err)
		return support.EmptyPageResult[stateful.TaskSnapshot]()
	}

	if total == 0 {
		return support.EmptyPageResult[stateful.TaskSnapshot]()
	}

	// 分页
	offset := (pageNum - 1) * pageSize
	var dos []TaskSnapshotDO
	if err := query.Offset(offset).Limit(pageSize).Find(&dos).Error; err != nil {
		log.Printf("分页查询任务快照失败: %v", err)
		return support.EmptyPageResult[stateful.TaskSnapshot]()
	}

	data := make([]stateful.TaskSnapshot, len(dos))
	for i := range dos {
		snapshot := taskSnapshotFromDO(&dos[i])
		if snapshot != nil {
			data[i] = *snapshot
		}
	}

	return support.NewPageResult(data, total, pageNum, pageSize)
}

// SelectForUpdate 悲观锁查询
func (r *DbBasedTaskSnapshotRepository) SelectForUpdate(taskID string) *stateful.TaskSnapshot {
	var do TaskSnapshotDO
	err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("task_id = ?", taskID).
		First(&do).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("悲观锁查询任务快照失败: %v", err)
		}
		return nil
	}
	return taskSnapshotFromDO(&do)
}

// 辅助函数

func taskStatusesToStrings(statuses []core.TaskStatus) []string {
	if len(statuses) == 0 {
		return nil
	}
	result := make([]string, len(statuses))
	for i, s := range statuses {
		result[i] = s.String()
	}
	return result
}

func stringToTaskStatus(s string) core.TaskStatus {
	switch s {
	case "CREATED":
		return core.TaskCreated
	case "RUNNING":
		return core.TaskRunning
	case "SUCCESS":
		return core.TaskSuccess
	case "FAILURE":
		return core.TaskFailure
	case "CANCELLING":
		return core.TaskCancelling
	case "CANCELED":
		return core.TaskCanceled
	case "INTERRUPTED":
		return core.TaskInterrupted
	default:
		return core.TaskStatus(-1)
	}
}

// 转换器: 领域对象 -> DO
func taskSnapshotToDO(snapshot *stateful.TaskSnapshot) *TaskSnapshotDO {
	do := &TaskSnapshotDO{
		ID:           snapshot.ID,
		GmtCreate:    snapshot.GmtCreate,
		GmtModified:  snapshot.GmtModified,
		TaskID:       snapshot.TaskID,
		IdempotentID: snapshot.IdempotentID,
		Name:         snapshot.Name,
		Type:         snapshot.Type,
		PublishEnv:   string(snapshot.PublishEnv),
		Status:       snapshot.Status.String(),
		Creator:      snapshot.Creator,
	}

	// 序列化 context
	if snapshot.Context != nil {
		if b, err := json.Marshal(snapshot.Context); err == nil {
			do.Context = string(b)
		}
	}

	// 序列化 subject
	if snapshot.Subject != nil {
		if b, err := json.Marshal(snapshot.Subject); err == nil {
			do.Subject = string(b)
		}
	}

	// 序列化 description
	if b, err := json.Marshal(snapshot.Description); err == nil {
		do.Description = string(b)
	}

	return do
}

// 转换器: DO -> 领域对象
func taskSnapshotFromDO(do *TaskSnapshotDO) *stateful.TaskSnapshot {
	if do == nil {
		return nil
	}

	snapshot := &stateful.TaskSnapshot{
		ID:           do.ID,
		GmtCreate:    do.GmtCreate,
		GmtModified:  do.GmtModified,
		TaskID:       do.TaskID,
		IdempotentID: do.IdempotentID,
		Name:         do.Name,
		Type:         do.Type,
		PublishEnv:   core.Env(do.PublishEnv),
		Status:       stringToTaskStatus(do.Status),
		Creator:      do.Creator,
	}

	// 反序列化 context
	if do.Context != "" {
		var ctx map[string]any
		if err := json.Unmarshal([]byte(do.Context), &ctx); err == nil {
			snapshot.Context = ctx
		}
	}

	// 反序列化 subject (保持为 any 类型)
	if do.Subject != "" {
		var subject any
		if err := json.Unmarshal([]byte(do.Subject), &subject); err == nil {
			snapshot.Subject = subject
		}
	}

	// 反序列化 description
	if do.Description != "" {
		var desc core.Description
		if err := json.Unmarshal([]byte(do.Description), &desc); err == nil {
			snapshot.Description = desc
		}
	}

	return snapshot
}

// 确保接口实现
var _ stateful.TaskSnapshotRepository = (*DbBasedTaskSnapshotRepository)(nil)
