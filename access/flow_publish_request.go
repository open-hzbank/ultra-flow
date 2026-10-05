package access

import (
	"github.com/open-hzbank/ultra-flow-scenes/unified"
	"github.com/open-hzbank/ultra-flow/core"
)

// FlowPublishRequest 流量发布请求
type FlowPublishRequest struct {
	// Biz 业务域
	Biz string `json:"biz"`
	// FlowConfigs 待发布的流控策略列表
	FlowConfigs []unified.FlowConfig `json:"flowConfigs"`
	// PublishEnv 目标发布环境
	PublishEnv core.Env `json:"publishEnv"`
	// PublishReason 发布原因
	PublishReason string `json:"publishReason"`
	// PublishType 发布类型
	PublishType core.PublishType `json:"publishType"`
	// UserID 发布用户
	UserID string `json:"userId"`
	// IdempotentID 幂等 id (选填)
	// 如传则依据此判断是否已提交, 否则认为是一次新的发布请求
	IdempotentID string `json:"idempotentId"`
	// EmergencyPublish 是否需要紧急发布
	EmergencyPublish bool `json:"emergencyPublish"`
}
