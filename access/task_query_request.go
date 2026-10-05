package access

import (
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/support"
)

// TaskQueryRequest 任务查询请求
type TaskQueryRequest struct {
	support.PageRequest
	// Biz 业务域
	Biz string `json:"biz"`
	// Creator 创建人
	Creator string `json:"creator"`
	// PublishEnvs 发布环境列表
	PublishEnvs []core.Env `json:"publishEnvs"`
	// TargetStatuses 目标状态列表
	TargetStatuses []core.TaskStatus `json:"targetStatuses"`
	// Order 排序方向
	Order support.Order `json:"order"`
	// FromTime 起始时间
	FromTime string `json:"fromTime"`
	// ToTime 结束时间
	ToTime string `json:"toTime"`
	// ConfigNames 配置名称列表
	ConfigNames []string `json:"configNames"`
}
