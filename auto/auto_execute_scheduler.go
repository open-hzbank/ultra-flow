package auto

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"hzbank.com.cn/ultra-flow/core"
	"hzbank.com.cn/ultra-flow/ops"
	"hzbank.com.cn/ultra-flow/stateful"
)

const (
	taskScanNumLimit = 500
	roundTimeout     = 120 * time.Second
	poolSize         = 50
	scheduleInterval = 5 * time.Second
)

// AutoExecuteScheduler 基于定时调度的流程任务自动调度器
// 对齐 Java SpringAutoExecuteScheduler: 定时扫描存量可自动执行的步骤, 并发调度对应任务执行
type AutoExecuteScheduler struct {
	taskSnapshotRepo     stateful.TaskSnapshotRepository
	taskStepSnapshotRepo stateful.TaskStepSnapshotRepository
	opsTaskRunnerFactory *ops.OpsTaskRunnerFactory
	stopCh               chan struct{}
	wg                   sync.WaitGroup
}

func NewAutoExecuteScheduler(
	taskSnapshotRepo stateful.TaskSnapshotRepository,
	taskStepSnapshotRepo stateful.TaskStepSnapshotRepository,
	opsTaskRunnerFactory *ops.OpsTaskRunnerFactory,
) *AutoExecuteScheduler {
	return &AutoExecuteScheduler{
		taskSnapshotRepo:     taskSnapshotRepo,
		taskStepSnapshotRepo: taskStepSnapshotRepo,
		opsTaskRunnerFactory: opsTaskRunnerFactory,
		stopCh:               make(chan struct{}),
	}
}

// Start 启动定时调度, 固定间隔 5 秒
func (s *AutoExecuteScheduler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(scheduleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				results := s.AutoExecute()
				errCount := 0
				for _, r := range results {
					if r.Status == core.TaskFailure {
						errCount++
					}
				}
				log.Printf("[auto-execute] scheduled total=%d, error=%d", len(results), errCount)
			case <-s.stopCh:
				return
			}
		}
	}()
}

// Stop 停止调度
func (s *AutoExecuteScheduler) Stop() {
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	s.wg.Wait()
}

// GetAutoExecutingTasks 获取需要自动执行的任务列表
// 查询类型为 autoExecute 且处于活跃状态的步骤, 过滤 allowAutoExecute=true 且 nextExecuteTime<=now,
// 再批量获取对应的任务快照
func (s *AutoExecuteScheduler) GetAutoExecutingTasks() []*stateful.TaskSnapshot {
	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")
	contextQueryParams := map[string]any{
		"allowAutoExecute": "JSON_CONTAINS(task_step_context, 'true', '$.allowAutoExecute')",
		"nextExecuteTime":  "JSON_UNQUOTE(JSON_EXTRACT(task_step_context, '$.nextExecuteTime')) <= '" + nowStr + "'",
	}

	steps := s.taskStepSnapshotRepo.GetByStepTypes(
		AutoExecuteTaskStepType,
		core.ActiveStepStatuses(),
		contextQueryParams,
	)

	taskIDSet := make(map[string]struct{})
	for _, step := range steps {
		if !isAutoExecuteReady(step, now) {
			continue
		}
		taskIDSet[step.TaskID] = struct{}{}
	}

	if len(taskIDSet) == 0 {
		return nil
	}

	taskIDs := make([]string, 0, len(taskIDSet))
	for id := range taskIDSet {
		taskIDs = append(taskIDs, id)
	}
	return s.taskSnapshotRepo.BatchGet(taskIDs, taskScanNumLimit)
}

// isAutoExecuteReady 判断步骤是否满足自动执行条件: allowAutoExecute=true 且 nextExecuteTime<=now
func isAutoExecuteReady(step *stateful.TaskStepSnapshot, now time.Time) bool {
	allowVal, ok := step.TaskStepContext[AllowAutoExecute]
	if !ok {
		return false
	}
	allow, ok := allowVal.(bool)
	if !ok || !allow {
		return false
	}

	nextTimeVal, ok := step.TaskStepContext[NextExecuteTime]
	if !ok {
		return true
	}
	nextTime, ok := nextTimeVal.(time.Time)
	if !ok {
		return true
	}
	return !nextTime.After(now)
}

// AutoExecute 扫描并并发执行所有需要自动执行的任务, 单轮超时 120 秒
func (s *AutoExecuteScheduler) AutoExecute() []core.TaskResult {
	tasks := s.GetAutoExecutingTasks()
	if len(tasks) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), roundTimeout)
	defer cancel()

	var mu sync.Mutex
	var results []core.TaskResult
	var wg sync.WaitGroup

	sem := make(chan struct{}, poolSize)

	for _, task := range tasks {
		select {
		case <-ctx.Done():
			log.Printf("[auto-execute] round timeout: %v", roundTimeout)
			break
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(snapshot *stateful.TaskSnapshot) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[auto-execute] task error, taskId=%s, panic=%v", snapshot.TaskID, r)
					mu.Lock()
					results = append(results, core.TaskResult{
						TaskID:  snapshot.TaskID,
						Status:  core.TaskFailure,
						Error:   errors.New("auto execute panic"),
						Message: "auto execute panic",
					})
					mu.Unlock()
				}
			}()

			builder := ops.GetOpsTaskBuilder(snapshot.Name, snapshot.Type)
			runner := s.opsTaskRunnerFactory.SynchronizedTaskRunner(snapshot.TaskID)
			result := runner.Execute(builder, func(b ops.OpsTaskBuilder) core.Task {
				return b.InternalResumeBuild(snapshot.TaskID)
			})

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(task)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		log.Printf("[auto-execute] round timeout: %v", roundTimeout)
	}

	return results
}
