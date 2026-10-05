package unified

import (
	"encoding/json"

	"github.com/open-hzbank/ultra-flow/core"
)

// FlowPublishContext 统一流量编排发布上下文
type FlowPublishContext struct {
	// Biz 业务场景 (业务域)
	Biz string `json:"biz"`
	// IdempotentID 幂等 id
	IdempotentID  string   `json:"idempotentId"`
	PublishReason string   `json:"publishReason"`
	PublishEnv    core.Env `json:"publishEnv"`
	// TaskType 发布/回滚
	TaskType core.TaskType `json:"taskType"`
	// PublishTypes 每个流控策略对应的发布类型
	//
	// 回滚场景无法对所有流控策略统一使用相同的 publishType, 必须分策略分别指定, 例如:
	//   - 发布: stg1: 首次发布 -> ONLINE, stg2: 非首次发布 -> ONLINE
	//   - 回滚: stg1: OFFLINE, stg2: ONLINE
	PublishTypes map[string]core.PublishType `json:"publishTypes"`
	// EmergencyPublish 是否紧急发布
	EmergencyPublish bool         `json:"emergencyPublish"`
	FlowConfigs      []FlowConfig `json:"flowConfigs"`
	Creator          string       `json:"creator"`
}

// FlowConfig 流控策略配置
type FlowConfig struct {
	Name           string         `json:"name"`
	Type           FlowConfigType `json:"type"`
	TargetConfig   string         `json:"targetConfig,omitempty"`
	RollbackConfig string         `json:"rollbackConfig,omitempty"`
}

// FlowConfigType 流控策略类型
type FlowConfigType string

const (
	FlowConfigRateLimit      FlowConfigType = "RATE_LIMIT"
	FlowConfigCircuitBreaker FlowConfigType = "CIRCUIT_BREAKER"
	FlowConfigTagRoute       FlowConfigType = "TAG_ROUTE"
)

// ParseConfig 解析策略配置
func (t FlowConfigType) ParseConfig(config string) FlowStrategy {
	switch t {
	case FlowConfigRateLimit:
		var s RateLimitStrategy
		if err := json.Unmarshal([]byte(config), &s); err != nil {
			return nil
		}
		return &s
	case FlowConfigCircuitBreaker:
		return nil
	case FlowConfigTagRoute:
		return nil
	default:
		return nil
	}
}

// FlowStrategy 流控策略接口
type FlowStrategy interface {
	GetName() string
	// GetApi 适用接口
	GetApi() string
	GetStatus() StrategyStatus
}

// RateLimitStrategy 限流策略
type RateLimitStrategy struct {
	Name       string         `json:"name"`
	Api        string         `json:"api"`
	Status     StrategyStatus `json:"status"`
	Count      int            `json:"count"`
	TimeWindow string         `json:"timeWindow"`
}

func (s *RateLimitStrategy) GetName() string           { return s.Name }
func (s *RateLimitStrategy) GetApi() string            { return s.Api }
func (s *RateLimitStrategy) GetStatus() StrategyStatus { return s.Status }

// StrategyStatus 策略启停状态
type StrategyStatus string

const (
	StrategyOpen  StrategyStatus = "OPEN"
	StrategyClose StrategyStatus = "CLOSE"
)
