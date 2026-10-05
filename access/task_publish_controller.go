package access

import (
	"encoding/json"
	"time"

	"github.com/emicklei/go-restful/v3"

	"github.com/open-hzbank/ultra-flow-scenes/unified"
	"github.com/open-hzbank/ultra-flow/arrange"
	"github.com/open-hzbank/ultra-flow/arrange/view"
	"github.com/open-hzbank/ultra-flow/auto"
	"github.com/open-hzbank/ultra-flow/control"
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/ops"
	"github.com/open-hzbank/ultra-flow/stateful"
	"github.com/open-hzbank/ultra-flow/support"
)

const (
	errTaskCannotStart    = "任务无法在当前状态下开始"
	errTaskCannotReCancel = "发布任务无法重复取消"
	errTaskCannotExecute  = "发布任务当前状态无法执行"
)

// TaskPublishController 发布任务 HTTP 控制器
type TaskPublishController struct {
	flowPublishService *unified.FlowPublishService
	taskPreviewService *view.TaskPreviewService
}

func NewTaskPublishController(
	flowPublishService *unified.FlowPublishService,
	taskPreviewService *view.TaskPreviewService,
) *TaskPublishController {
	return &TaskPublishController{
		flowPublishService: flowPublishService,
		taskPreviewService: taskPreviewService,
	}
}

// RegisterRoutes 注册 restful 路由
func (c *TaskPublishController) RegisterRoutes(container *restful.Container) {
	ws := &restful.WebService{}
	ws.Path("/flow/publish").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/submitAndStart").To(c.submitAndStart))
	ws.Route(ws.POST("/submit").To(c.submit))
	ws.Route(ws.POST("/start").To(c.start).Consumes("*/*"))
	ws.Route(ws.POST("/resume").To(c.resume).Consumes("*/*"))
	ws.Route(ws.POST("/cancel").To(c.cancel).Consumes("*/*"))
	ws.Route(ws.POST("/interrupt").To(c.interrupt).Consumes("*/*"))
	ws.Route(ws.GET("/task").To(c.task))
	ws.Route(ws.POST("/tasks").To(c.tasks))

	container.Add(ws)
}

// submitAndStart 提交并开始任务
func (c *TaskPublishController) submitAndStart(req *restful.Request, resp *restful.Response) {
	var publishReq FlowPublishRequest
	if err := req.ReadEntity(&publishReq); err != nil {
		_ = resp.WriteAsJson(FailureResponse[string]("INVALID_REQUEST", err.Error()))
		return
	}

	submitResp := c.doSubmit(&publishReq)
	if !submitResp.Success {
		_ = resp.WriteAsJson(submitResp)
		return
	}

	c.doStart(resp, submitResp.Result)
}

// submit 仅提交 API 发布任务
func (c *TaskPublishController) submit(req *restful.Request, resp *restful.Response) {
	var publishReq FlowPublishRequest
	if err := req.ReadEntity(&publishReq); err != nil {
		_ = resp.WriteAsJson(FailureResponse[string]("INVALID_REQUEST", err.Error()))
		return
	}

	result := c.doSubmit(&publishReq)
	_ = resp.WriteAsJson(result)
}

func (c *TaskPublishController) doSubmit(publishReq *FlowPublishRequest) Response[string] {
	publishTypes := make(map[string]core.PublishType, len(publishReq.FlowConfigs))
	for _, fc := range publishReq.FlowConfigs {
		publishTypes[fc.Name] = publishReq.PublishType
	}

	result := c.flowPublishService.CreateTask(
		publishReq.FlowConfigs,
		publishTypes,
		publishReq.Biz,
		publishReq.UserID,
		publishReq.PublishEnv,
		publishReq.IdempotentID,
		publishReq.PublishReason,
		publishReq.EmergencyPublish,
		core.TaskTypePublish,
	)
	return WrapResponse(result)
}

// start 仅开始任务
func (c *TaskPublishController) start(req *restful.Request, resp *restful.Response) {
	taskID := req.QueryParameter("taskId")
	if taskID == "" {
		_ = resp.WriteAsJson(FailureResponse[core.TaskResult]("INVALID_REQUEST", "taskId 不能为空"))
		return
	}
	c.doStart(resp, taskID)
}

func (c *TaskPublishController) doStart(resp *restful.Response, taskID string) {
	taskResult := c.flowPublishService.GetTask(taskID)
	if !taskResult.IsSuccess() {
		_ = resp.WriteAsJson(FailureResponse[core.TaskResult](taskResult.GetErrorCode(), taskResult.GetErrorMessage()))
		return
	}

	task := taskResult.GetData()
	if !task.Status.UnStarted() {
		_ = resp.WriteAsJson(FailureResponse[core.TaskResult](errTaskCannotStart, task.Status.String()))
		return
	}

	exeResult := c.flowPublishService.ExecuteTaskWithTaskContext(taskID, map[string]any{
		control.ManualFireNextStepSignal: true,
	})
	_ = resp.WriteAsJson(SuccessResponseWithData(exeResult))
}

// resume 恢复执行任务
func (c *TaskPublishController) resume(req *restful.Request, resp *restful.Response) {
	taskID := req.QueryParameter("taskId")
	if taskID == "" {
		_ = resp.WriteAsJson(FailureResponse[core.TaskResult]("INVALID_REQUEST", "taskId 不能为空"))
		return
	}

	taskResult := c.flowPublishService.GetTask(taskID)
	if !taskResult.IsSuccess() {
		_ = resp.WriteAsJson(FailureResponse[core.TaskResult](taskResult.GetErrorCode(), taskResult.GetErrorMessage()))
		return
	}

	task := taskResult.GetData()
	if !task.Status.IsActive() {
		_ = resp.WriteAsJson(FailureResponse[core.TaskResult](errTaskCannotExecute, task.Status.String()))
		return
	}

	stepConfigs := c.buildResumeStepContext(taskID)
	var exeResult core.TaskResult
	if len(stepConfigs) > 0 {
		exeResult = c.flowPublishService.ExecuteTaskWithStepConfigs(taskID, stepConfigs)
	} else if c.noTaskStepsRunning(taskID) {
		exeResult = c.flowPublishService.ExecuteTaskWithTaskContext(taskID, map[string]any{
			control.ManualFireNextStepSignal: true,
		})
	} else {
		exeResult = c.flowPublishService.ExecuteTask(taskID)
	}

	if exeResult.Status.IsFail() {
		_ = resp.WriteAsJson(FailureResponse[core.TaskResult]("", exeResult.Message))
		return
	}
	_ = resp.WriteAsJson(SuccessResponseWithData(exeResult))
}

// buildResumeStepContext 构建执行任务的 stepContext
// 目前只针对可分批步骤, 在其可执行下一批次时自动添加 FIRE_NEXT_BATCH_SIGNAL 信号
func (c *TaskPublishController) buildResumeStepContext(taskID string) []core.StepExecuteConfig {
	batchableStepNames := c.getBatchableStepTypes(taskID)

	taskResult := c.flowPublishService.GetTask(taskID)
	stepResult := c.flowPublishService.GetTaskSteps(taskID)
	if !taskResult.IsSuccess() || !stepResult.IsSuccess() {
		return nil
	}

	task := taskResult.GetData()
	taskSteps := stepResult.GetData()

	runningBatchableSteps := getRunningBatchableSteps(taskSteps, batchableStepNames)
	var configs []core.StepExecuteConfig
	for _, batchableStep := range runningBatchableSteps {
		if !ops.HasRunningBatchInStep(task, batchableStep) {
			configs = append(configs, core.StepExecuteConfig{
				StepName: batchableStep.Name,
				Key:      control.FireNextBatchSignal,
				Value:    true,
			})
		}
	}
	return configs
}

func (c *TaskPublishController) noTaskStepsRunning(taskID string) bool {
	stepResult := c.flowPublishService.GetTaskSteps(taskID)
	if !stepResult.IsSuccess() {
		return false
	}
	steps := stepResult.GetData()
	for _, step := range steps {
		if !step.Status.NormalEnded() {
			return false
		}
	}
	return true
}

// cancel 任务取消
func (c *TaskPublishController) cancel(req *restful.Request, resp *restful.Response) {
	taskID := req.QueryParameter("taskId")
	if taskID == "" {
		_ = resp.WriteAsJson(FailureResponse[any]("INVALID_REQUEST", "taskId 不能为空"))
		return
	}

	taskResult := c.flowPublishService.GetTask(taskID)
	if !taskResult.IsSuccess() {
		_ = resp.WriteAsJson(FailureResponse[any](taskResult.GetErrorCode(), taskResult.GetErrorMessage()))
		return
	}

	task := taskResult.GetData()
	if !task.Status.CanCancel() {
		_ = resp.WriteAsJson(FailureResponse[any](errTaskCannotReCancel, task.Status.String()))
		return
	}

	c.flowPublishService.CancelTask(taskID)
	_ = resp.WriteAsJson(SuccessResponse[any]())
}

// interrupt 任务中断 (前端不要对用户透出, 管理员操作即可)
func (c *TaskPublishController) interrupt(req *restful.Request, resp *restful.Response) {
	taskID := req.QueryParameter("taskId")
	if taskID == "" {
		_ = resp.WriteAsJson(FailureResponse[any]("INVALID_REQUEST", "taskId 不能为空"))
		return
	}

	c.flowPublishService.InterruptTask(taskID)
	_ = resp.WriteAsJson(SuccessResponse[any]())
}

// task 任务详情
func (c *TaskPublishController) task(req *restful.Request, resp *restful.Response) {
	taskID := req.QueryParameter("taskId")
	if taskID == "" {
		_ = resp.WriteAsJson(FailureResponse[*view.TaskOverView]("INVALID_REQUEST", "taskId 不能为空"))
		return
	}

	taskResult := c.flowPublishService.GetTask(taskID)
	if !taskResult.IsSuccess() {
		_ = resp.WriteAsJson(FailureResponse[*view.TaskOverView](taskResult.GetErrorCode(), taskResult.GetErrorMessage()))
		return
	}

	task := taskResult.GetData()
	biz := c.calcTaskBiz(task)
	firstStepDef := c.flowPublishService.GetTaskStageDefinition(biz, task.PublishEnv)
	stageNav := &arrange.StageNavigatorAdapter{Def: firstStepDef}
	taskPreview := c.taskPreviewService.TaskPreview(stageNav, core.NewJSONTaskDataSerDeser(&unified.FlowPublishContext{}), taskID)

	operationType := c.taskOperation(taskID, taskPreview)
	taskType := c.calcTaskType(task)
	canRollback := c.canTaskRollback(taskID)

	overView := view.NewTaskOverView(
		operationType,
		taskType,
		taskPreview.Task,
		taskPreview.TaskSteps,
		canRollback,
	)
	_ = resp.WriteAsJson(SuccessResponseWithData(overView))
}

// taskOperation 任务当前的可执行状态
func (c *TaskPublishController) taskOperation(taskID string, taskPreview *view.TaskPreview) view.OperationType {
	batchableStepTypes := c.getBatchableStepTypes(taskID)

	taskResult := c.flowPublishService.GetTask(taskID)
	stepResult := c.flowPublishService.GetTaskSteps(taskID)
	if !taskResult.IsSuccess() || !stepResult.IsSuccess() {
		return view.OpForbidFireNextStep
	}

	task := taskResult.GetData()
	taskSteps := stepResult.GetData()

	runningBatchableSteps := getRunningBatchableSteps(taskSteps, batchableStepTypes)
	if containsBlockingStep(taskPreview.TaskSteps, taskSteps) {
		return view.OpForbidByBlockedSteps
	}
	if len(runningBatchableSteps) > 0 {
		if !hasRunningBatchesInStep(runningBatchableSteps, task) && !c.isLastBatchFinished(task) {
			return view.OpAllowFireNextBatch
		}
		return view.OpForbidFireNextBatch
	}

	runningBatchlessSteps := getRunningBatchlessSteps(taskSteps, batchableStepTypes)
	if containsBlockingStep(taskPreview.TaskSteps, taskSteps) {
		return view.OpForbidByBlockedSteps
	}
	if allNotAllowedAutoScheduled(taskSteps) {
		return c.calcAllowFireOperationType(taskPreview, taskSteps)
	}
	if len(runningBatchlessSteps) > 0 {
		return view.OpForbidFireNextStep
	}
	if task.Status.IsActive() {
		return c.calcAllowFireOperationType(taskPreview, taskSteps)
	}
	return view.OpForbidFireNextStep
}

// containsBlockingStep 检测在给定的 taskSteps 中是否存在卡点步骤
func containsBlockingStep(stages []*view.StageView, taskSteps []*stateful.TaskStepSnapshot) bool {
	stepNames := make(map[string]bool, len(taskSteps))
	for _, step := range taskSteps {
		stepNames[step.Name] = true
	}
	return stagesContainBlockingStep(stages, stepNames)
}

func stagesContainBlockingStep(stages []*view.StageView, stepNames map[string]bool) bool {
	for _, stage := range stages {
		for _, step := range stage.Steps {
			switch s := step.(type) {
			case *view.AtomicStepView:
				if stepNames[s.Name] && s.Blockable {
					return true
				}
			case *view.NestedStepView:
				if stagesContainBlockingStep(s.Stages, stepNames) {
					return true
				}
			}
		}
	}
	return false
}

func hasRunningBatchesInStep(runningBatchableSteps []*stateful.TaskStepSnapshot, task *stateful.TaskSnapshot) bool {
	for _, step := range runningBatchableSteps {
		if ops.HasRunningBatchInStep(task, step) {
			return true
		}
	}
	return false
}

func (c *TaskPublishController) isLastBatchFinished(_ *stateful.TaskSnapshot) bool {
	// todo 暂时没有分批的场景
	return false
}

func (c *TaskPublishController) getBatchableStepTypes(taskID string) []string {
	taskResult := c.flowPublishService.GetTask(taskID)
	if !taskResult.IsSuccess() {
		return nil
	}
	task := taskResult.GetData()
	return ops.GetBatchableStepsOfTask(task.Name, task.Type)
}

// getRunningBatchableSteps 正在执行中的可分批的步骤集合
func getRunningBatchableSteps(taskSteps []*stateful.TaskStepSnapshot, batchableStepTypes []string) []*stateful.TaskStepSnapshot {
	batchableSet := make(map[string]bool, len(batchableStepTypes))
	for _, t := range batchableStepTypes {
		batchableSet[t] = true
	}
	var result []*stateful.TaskStepSnapshot
	for _, step := range taskSteps {
		if batchableSet[step.Type] && step.Status.IsExecuting() {
			result = append(result, step)
		}
	}
	return result
}

// getRunningBatchlessSteps 正在执行中的无需分批的步骤集合
func getRunningBatchlessSteps(taskSteps []*stateful.TaskStepSnapshot, batchableStepTypes []string) []*stateful.TaskStepSnapshot {
	batchableSet := make(map[string]bool, len(batchableStepTypes))
	for _, t := range batchableStepTypes {
		batchableSet[t] = true
	}
	var result []*stateful.TaskStepSnapshot
	for _, step := range taskSteps {
		if !batchableSet[step.Type] && step.Status.IsExecuting() {
			result = append(result, step)
		}
	}
	return result
}

// allNotAllowedAutoScheduled 给定的步骤都不允许被自动执行
func allNotAllowedAutoScheduled(taskSteps []*stateful.TaskStepSnapshot) bool {
	for _, step := range taskSteps {
		if step.Type == auto.AutoExecuteTaskStepType && step.Status.IsExecuting() {
			allowAuto, ok := step.TaskStepContext[auto.AllowAutoExecute]
			if !ok {
				allowAuto = true
			}
			if allowBool, ok := allowAuto.(bool); ok && allowBool {
				return false
			}
		}
	}
	return true
}

func (c *TaskPublishController) calcAllowFireOperationType(taskPreview *view.TaskPreview, taskSteps []*stateful.TaskStepSnapshot) view.OperationType {
	if isFirstUnBlockableStage(taskPreview.TaskSteps, taskSteps) {
		return view.OpAllowFireFirstStep
	}
	return view.OpAllowFireNextStep
}

func isFirstUnBlockableStage(stages []*view.StageView, taskSteps []*stateful.TaskStepSnapshot) bool {
	for _, stage := range stages {
		if !containsBlockingStep([]*view.StageView{stage}, taskSteps) && stage.StageStatus().NotStarted() {
			return stage.StageOrder == 1
		}
	}
	return false
}

// tasks 任务列表
func (c *TaskPublishController) tasks(req *restful.Request, resp *restful.Response) {
	var queryReq TaskQueryRequest
	if err := req.ReadEntity(&queryReq); err != nil {
		_ = resp.WriteAsJson(FailureResponse[any]("INVALID_REQUEST", err.Error()))
		return
	}

	var fromTime, toTime time.Time
	if queryReq.FromTime != "" {
		fromTime, _ = time.Parse(time.RFC3339, queryReq.FromTime)
	}
	if queryReq.ToTime != "" {
		toTime, _ = time.Parse(time.RFC3339, queryReq.ToTime)
	}

	taskSnapshots := c.flowPublishService.SearchTasks(
		queryReq.Biz,
		queryReq.ConfigNames,
		queryReq.Creator,
		queryReq.PublishEnvs,
		queryReq.TargetStatuses,
		queryReq.Order,
		fromTime, toTime,
		queryReq.PageNum,
		queryReq.PageSize,
	)

	briefs := make([]*TaskBrief, 0, len(taskSnapshots.Data))
	for _, snapshot := range taskSnapshots.Data {
		brief := NewTaskBrief(
			snapshot.GmtCreate.UnixMilli(),
			snapshot.GmtModified.UnixMilli(),
			snapshot.TaskID,
			snapshot.PublishEnv,
			view.NewUserViewFromEmpID(snapshot.Creator),
			snapshot.Description.Reason,
			c.calcTaskBiz(&snapshot),
			c.calcTaskType(&snapshot),
			snapshot.Status,
			c.canTaskRollback(snapshot.TaskID),
		)
		briefs = append(briefs, brief)
	}

	pageResult := support.PageResult[*TaskBrief]{
		Total:    taskSnapshots.Total,
		Pages:    taskSnapshots.Pages,
		PageNum:  taskSnapshots.PageNum,
		PageSize: taskSnapshots.PageSize,
		Data:     briefs,
	}
	_ = resp.WriteAsJson(SuccessResponseWithData(pageResult))
}

func (c *TaskPublishController) calcTaskBiz(task *stateful.TaskSnapshot) string {
	subjectJSON := task.GetSubjectJSON()
	if subjectJSON == "" {
		return ""
	}
	var subject map[string]string
	if err := json.Unmarshal([]byte(subjectJSON), &subject); err != nil {
		return ""
	}
	return subject[unified.SubjectPublishBiz]
}

func (c *TaskPublishController) calcTaskType(task *stateful.TaskSnapshot) core.TaskType {
	subjectJSON := task.GetSubjectJSON()
	if subjectJSON == "" {
		return core.TaskTypePublish
	}
	var subject map[string]any
	if err := json.Unmarshal([]byte(subjectJSON), &subject); err != nil {
		return core.TaskTypePublish
	}
	taskTypeStr, _ := subject[unified.SubjectTaskType].(string)
	if taskTypeStr == "" {
		return core.TaskTypePublish
	}
	taskType, ok := core.StringToTaskType(taskTypeStr)
	if !ok {
		return core.TaskTypePublish
	}
	return taskType
}

func (c *TaskPublishController) canTaskRollback(_ string) bool {
	// todo 暂时不支持回滚的能力
	return false
}
