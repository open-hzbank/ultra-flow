package main

import (
	"fmt"
	"hzbank.com.cn/ultra-flow/auto"
	"log"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"hzbank.com.cn/ultra-flow-access"
	"hzbank.com.cn/ultra-flow-scenes/unified"
	"hzbank.com.cn/ultra-flow/arrange"
	"hzbank.com.cn/ultra-flow/arrange/view"
	"hzbank.com.cn/ultra-flow/ops"
	"hzbank.com.cn/ultra-flow/stateful"
	"hzbank.com.cn/ultra-flow/stateful/mem"
	flowsync "hzbank.com.cn/ultra-flow/sync"
)

const serverPort = ":7001"

func main() {
	// 任务定义组件
	taskDefConfigService := arrange.NewTaskDefConfigService()
	mockedDefMgr := NewMockedFlowDefinitionManager(taskDefConfigService)
	if err := mockedDefMgr.InitFlowDefinitions(); err != nil {
		log.Fatalf("加载流程定义失败: %v", err)
	}

	// 任务保存组件
	taskSnapshotRepo := mem.NewMemoryTaskSnapshotRepository()
	taskStepSnapshotRepo := mem.NewMemoryTaskStepSnapshotRepository()
	transactionable := mem.NewMemoryMockedTransaction()
	taskPersistence := stateful.NewDefaultTaskPersistence(taskSnapshotRepo, taskStepSnapshotRepo)

	// 任务同步组件
	lockService := flowsync.NewMemoryLockService()
	opsTaskRunnerFactory := ops.NewOpsTaskRunnerFactory(lockService)

	// 任务调度组件
	autoScheduler := auto.NewAutoExecuteScheduler(taskSnapshotRepo, taskStepSnapshotRepo, opsTaskRunnerFactory)
	autoScheduler.Start()

	// 任务预览组件
	taskPreviewService := view.NewTaskPreviewService(taskSnapshotRepo, taskStepSnapshotRepo)

	// 具体任务类型加载
	publishTaskBuilder := unified.NewFlowPublishTaskBuilder(
		transactionable, taskPersistence, taskDefConfigService,
	)
	publishTaskBuilder.Register()
	flowPublishService := unified.NewFlowPublishService(
		taskSnapshotRepo, taskStepSnapshotRepo,
		opsTaskRunnerFactory, lockService,
		taskDefConfigService, publishTaskBuilder,
	)

	// http 接入服务组件
	container := restful.NewContainer()
	controller := access.NewTaskPublishController(flowPublishService, taskPreviewService)
	controller.RegisterRoutes(container)
	fmt.Printf("ultra-flow-go server starting on %s\n", serverPort)
	if err := http.ListenAndServe(serverPort, container); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
