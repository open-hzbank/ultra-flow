package support

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/open-hzbank/ultra-flow/core"
)

// Publishes 发布相关工具方法
type Publishes struct{}

// TaskOrderUrlKey 任务工单 URL 的 key (对应 Java: TASK_ORDER_URL_KEY)
const TaskOrderUrlKey = "taskOrderUrl"

const flowIDLength = 16

// CalcTaskType 根据发布类型计算任务类型
func CalcTaskType(publishType core.PublishType) string {
	switch publishType {
	case core.PublishOnline:
		return "publish"
	case core.PublishOffline:
		return "rollback"
	default:
		return "unknown"
	}
}

// CalcExecuteMode 根据任务类型计算执行模式
func CalcExecuteMode(taskType string) core.ExecuteMode {
	switch taskType {
	case "publish":
		return core.ModePublish
	case "rollback":
		return core.ModeCancel
	default:
		return core.ModePublish
	}
}

// BuildFlowID 构造随机任务 ID
func BuildFlowID() string {
	b := make([]byte, flowIDLength)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ToKeyMap 将字符串切片转换为 map[string]int (值统一为 1)
func ToKeyMap(items []string) map[string]int {
	m := make(map[string]int, len(items))
	for _, item := range items {
		m[item] = 1
	}
	return m
}

// ClearErrorTips 清除步骤的错误提示信息
func ClearErrorTips(stepCtx *core.TaskStepContext) {
	stepCtx.Put("taskErrorTips", "")
}

// TaskTypeToString 将 TaskType 转换为字符串
func TaskTypeToString(t core.TaskType) string {
	switch t {
	case core.TaskTypePublish:
		return "PUBLISH"
	case core.TaskTypeRollback:
		return "ROLLBACK"
	default:
		return "UNKNOWN"
	}
}
