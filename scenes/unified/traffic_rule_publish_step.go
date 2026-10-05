package unified

import (
	"strings"

	"github.com/open-hzbank/ultra-flow/async"
	"github.com/open-hzbank/ultra-flow/core"
)

const (
	stepTitle    = "流控策略发布"
	publishIDKey = "publishId"
)

// TrafficRulePublishStep 流控策略发布通用控制步骤
type TrafficRulePublishStep struct {
	*async.ExtAsyncTaskStep

	// targetName 目标调度体的唯一名称
	targetName     string
	ruleType       RuleType
	targetStrategy FlowStrategy
}

// NewTrafficRulePublishStep 创建流控策略发布步骤
func NewTrafficRulePublishStep(stepConfigName string, taskCtx *core.TaskContext, publishData *FlowPublishContext) *TrafficRulePublishStep {
	splits := parseStepConfigNamed(stepConfigName)
	targetName := splits[1]
	ruleType := RuleTypeFrom(splits[0])
	targetStrategy := parseTargetStrategy(publishData, targetName, ruleType)

	s := &TrafficRulePublishStep{
		targetName:     targetName,
		ruleType:       ruleType,
		targetStrategy: targetStrategy,
	}

	s.ExtAsyncTaskStep = async.NewExtAsyncTaskStep(
		stepConfigName,
		taskCtx,
		s.isExtTaskSubmitted,
		s.getExtTaskResult,
		s.submitExtTask,
	)
	return s
}

func parseStepConfigNamed(stepConfigName string) []string {
	splits := strings.Split(stepConfigName, "^")
	parts := make([]string, 0, len(splits))
	for _, s := range splits {
		if s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) != 2 {
		panic("stepConfigName 格式不符合要求: " + stepConfigName)
	}
	return parts
}

func parseTargetStrategy(publishData *FlowPublishContext, targetName string, ruleType RuleType) FlowStrategy {
	for _, flowConfig := range publishData.FlowConfigs {
		if flowConfig.Name == targetName {
			var config string
			if ruleType == RuleTypeNormal {
				config = flowConfig.TargetConfig
			} else {
				config = flowConfig.RollbackConfig
			}
			strategy := flowConfig.Type.ParseConfig(config)
			if strategy != nil {
				return strategy
			}
		}
	}
	panic("找不到对应的流控策略: " + targetName)
}

func (s *TrafficRulePublishStep) isExtTaskSubmitted() bool {
	return s.getPublishID() != ""
}

func (s *TrafficRulePublishStep) getExtTaskResult() core.TaskStepResult {
	_ = s.getPublishID()
	queryTimesVal := s.GetTaskStepContext().Get("queryTimes")
	queryTimes := 0
	if queryTimesVal != nil {
		if v, ok := queryTimesVal.(int); ok {
			queryTimes = v
		}
	}
	queryTimes++
	s.GetTaskStepContext().Put("queryTimes", queryTimes)

	if queryTimes == 5 {
		return core.NewStepResult(core.StepSuccess)
	}
	return core.NewStepResult(core.StepRunning)
}

func (s *TrafficRulePublishStep) submitExtTask() core.TaskStepResult {
	// todo 调用流控中心接口发布
	s.GetTaskStepContext().Put(publishIDKey, "mock")
	return core.NewStepResult(core.StepRunning)
}

func (s *TrafficRulePublishStep) getPublishID() string {
	val := s.GetTaskStepContext().Get(publishIDKey)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// Describe 步骤描述
func (s *TrafficRulePublishStep) Describe() core.StepDescription {
	return core.NewStepDescription(stepTitle, map[string]any{
		"targetName":     s.targetName,
		"ruleType":       s.ruleType.String(),
		"publishId":      s.getPublishID(),
		"targetStrategy": s.targetStrategy,
	})
}

// GetType 步骤类型
func (s *TrafficRulePublishStep) GetType() string {
	return "TRAFFIC_RULE_PUBLISH"
}

// RuleType 规则类型
type RuleType int

const (
	// RuleTypeNormal 正常规则
	RuleTypeNormal RuleType = iota
	// RuleTypeCompensate 补偿规则
	RuleTypeCompensate
)

// String 规则类型字符串
func (r RuleType) String() string {
	switch r {
	case RuleTypeNormal:
		return "normal"
	case RuleTypeCompensate:
		return "compensate"
	default:
		return "unknown"
	}
}

// RuleTypeFrom 从字符串解析规则类型
func RuleTypeFrom(name string) RuleType {
	switch name {
	case "normal":
		return RuleTypeNormal
	case "compensate":
		return RuleTypeCompensate
	default:
		panic("找不到对应的枚举值: " + name)
	}
}

// TrafficRulePublishStepBuilder 流控策略发布步骤构建器
type TrafficRulePublishStepBuilder struct{}

// NewTrafficRulePublishStepBuilder 创建步骤构建器
func NewTrafficRulePublishStepBuilder() *TrafficRulePublishStepBuilder {
	return &TrafficRulePublishStepBuilder{}
}

// Build 构建步骤
func (b *TrafficRulePublishStepBuilder) Build(name string, taskCtx *core.TaskContext, deps any) core.TaskStep {
	taskDeps := deps.(*FlowTaskDependencies)
	return NewTrafficRulePublishStep(name, taskCtx, taskDeps.PublishContext)
}

// TaskStepTitle 步骤标题
func (b *TrafficRulePublishStepBuilder) TaskStepTitle() string {
	return stepTitle
}
