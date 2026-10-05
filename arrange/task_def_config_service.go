package arrange

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/open-hzbank/ultra-flow/arrange/view"
	"github.com/open-hzbank/ultra-flow/core"
	"github.com/open-hzbank/ultra-flow/padding"
)

// TaskDefConfigService 任务定义的配置服务: 规则订阅及获取
type TaskDefConfigService struct {
	mu              sync.RWMutex
	flowDefinitions []MatchableDefinition
}

// MatchableDefinition 可匹配的流程定义
type MatchableDefinition struct {
	MatchingMetas       map[string]string
	TaskStageDefinition *TaskStageDefinition
}

func (m *MatchableDefinition) IsMatch(condition map[string]string) bool {
	if len(condition) != len(m.MatchingMetas) {
		return false
	}
	for k, v := range m.MatchingMetas {
		if condition[k] != v {
			return false
		}
	}
	return true
}

// FlowConfig 流程配置
type FlowConfig struct {
	// 流程定义的描述 (目前仅用于配置平台展示方便管理)
	Desc string `json:"desc"`
	// 匹配目标 stageConfigs 的元信息标签
	MatchingMetas map[string]string `json:"matchingMetas"`
	Stages        []StageConfig     `json:"stages"`
}

// StageConfig 阶段配置
type StageConfig struct {
	Stage string            `json:"stage"`
	Steps []StepArrangement `json:"steps"`
	// 是否需要在上一个阶段完成时, 自动执行当前阶段内的步骤
	// 因为很多场景下无法预知上一个阶段完成的时间, 阶段的自动执行可能带来不可控的风险, 故本开关默认关闭
	// 如有特殊需求 (如任务后置写基线的逻辑) 可自行配置开启
	AutoExecute bool `json:"autoExecute"`
}

// StepArrangement 步骤编排配置
// StepConfigs 和 Stages 必须有一个是 nil, 另一个不为 nil
type StepArrangement struct {
	// 原子步骤 (正向 & 补偿)
	StepConfigs *AtomicStepConfigs `json:"stepConfigs"`
	// 具有子编排逻辑的次级阶段
	Stages []StageConfig   `json:"stages"`
	Type   StepArrangeType `json:"type"`
}

// AtomicStepConfigs 原子步骤配置
type AtomicStepConfigs struct {
	Normal     *TaskStepConfig `json:"normal"`
	Compensate *TaskStepConfig `json:"compensate"`
}

// TaskStepConfig 步骤配置
type TaskStepConfig struct {
	Name            string `json:"name"`
	TaskStepBuilder string `json:"taskStepBuilder"`
	// 步骤是否需要持有状态
	// 考虑到大部分步骤皆需要持有状态, 本开关默认开启
	// 如有特殊需求 (如极简单的轻量化场景, 不想额外引入持久化的相关依赖) 可自行配置关闭
	Stateful bool `json:"stateful"`
	// 步骤是否是整个流程的卡点 (即步骤不会自动执行完成, 必须由用户交互处理)
	// NOTICE: 这只是一个申明型的配置, 不会向流程引擎本身引入任何新特性,
	// 它向展示层申明这个步骤会引起流程阻断, 需要用户关注并及时处理
	Blockable bool `json:"blockable"`
	// 步骤是否需要流程引擎持续的状态追踪并自动更新
	// 考虑到很多场景下步骤执行皆是异步的, 需要及时同步其执行状态, 故本开关默认开启
	// 如有特殊需求 (如非异步的纯本地逻辑) 可自行配置关闭
	AutoUpdate bool `json:"autoUpdate"`
	// 步骤状态自动追踪的间隔 (单位: 秒, 默认 5s)
	// 当 AutoUpdate = true 时有效
	AutoUpdateIntervalSecs int `json:"autoUpdateIntervalSeconds"`
	// 步骤状态自动追踪执行失败重试次数阈值 (默认 120, 基于 1-5-10 故障快恢标准)
	// 当 AutoUpdate = true 时有效
	AutoUpdateErrorRetry int `json:"autoUpdateErrorRetryTimes"`
	// 步骤状态自动追踪执行的最大次数 (默认 17280 次, 相当于一天)
	AutoUpdateMaxTimes int `json:"autoUpdateMaxTimes"`
}

// UnmarshalJSON 自定义反序列化, 先设置与 Java 版本一致的默认值再反序列化
func (c *TaskStepConfig) UnmarshalJSON(data []byte) error {
	c.Stateful = true
	c.AutoUpdate = true
	c.AutoUpdateIntervalSecs = 5
	c.AutoUpdateErrorRetry = 120
	c.AutoUpdateMaxTimes = 24 * 3600 / 5
	type Alias TaskStepConfig
	return json.Unmarshal(data, (*Alias)(c))
}

// DefaultTaskStepConfig 创建带默认值的步骤配置 (对齐 Java 版本字段默认值)
func DefaultTaskStepConfig() TaskStepConfig {
	return TaskStepConfig{
		Stateful:               true,
		AutoUpdate:             true,
		AutoUpdateIntervalSecs: 5,
		AutoUpdateErrorRetry:   120,
		AutoUpdateMaxTimes:     24 * 3600 / 5,
	}
}

// StepArrangeType 步骤编排类型
type StepArrangeType string

const (
	StepArrangeAtomic StepArrangeType = "ATOMIC"
	StepArrangeNested StepArrangeType = "NESTED"
)

// UnmarshalJSON 自定义反序列化, 支持两种 JSON 格式:
//  1. 嵌套格式: {"stepConfigs": {"normal": {...}, "compensate": {...}}}
//  2. 扁平格式: {"normal": {...}, "compensate": {...}} (对齐 Java fastjson2 构造器反序列化)
//
// 同时支持显式指定 "stages" 的嵌套编排格式
// 当 Type 未显式指定时, 根据填充的字段自动推断类型
func (sa *StepArrangement) UnmarshalJSON(data []byte) error {
	type Alias StepArrangement
	aux := &struct {
		*Alias
		Normal     *TaskStepConfig `json:"normal"`
		Compensate *TaskStepConfig `json:"compensate"`
	}{
		Alias: (*Alias)(sa),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	// 扁平格式: normal/compensate 直接出现在顶层, 自动组装为 StepConfigs
	if sa.StepConfigs == nil && aux.Normal != nil {
		sa.StepConfigs = &AtomicStepConfigs{
			Normal:     aux.Normal,
			Compensate: aux.Compensate,
		}
	}
	if sa.Type == "" {
		if sa.StepConfigs != nil {
			sa.Type = StepArrangeAtomic
		} else if len(sa.Stages) > 0 {
			sa.Type = StepArrangeNested
		}
	}
	return nil
}

// StepBuilderRegistry 步骤构建器注册表
var (
	stepBuilderRegistry   = make(map[string]func() StepDefinition)
	stepBuilderRegistryMu sync.RWMutex
)

// RegisterStepBuilder 注册步骤构建器工厂
func RegisterStepBuilder(builderClass string, factory func() StepDefinition) {
	stepBuilderRegistryMu.Lock()
	defer stepBuilderRegistryMu.Unlock()
	stepBuilderRegistry[builderClass] = factory
}

func NewTaskDefConfigService() *TaskDefConfigService {
	return &TaskDefConfigService{}
}

// GetFlowDefinition 获取匹配的流程定义
func (s *TaskDefConfigService) GetFlowDefinition(matchingCondition map[string]string) (*TaskStageDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, def := range s.flowDefinitions {
		if def.IsMatch(matchingCondition) {
			return def.TaskStageDefinition, true
		}
	}
	return nil, false
}

// LoadAll 加载所有流程定义
func (s *TaskDefConfigService) LoadAll(rawFlowConfigs string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var flowConfigs []FlowConfig
	if err := json.Unmarshal([]byte(rawFlowConfigs), &flowConfigs); err != nil {
		log.Printf("[TaskDefConfigService] process configs exception, config = %s, error = %v", rawFlowConfigs, err)
		panic(fmt.Sprintf("process flow definitions exception: %v", err))
	}

	var tmpDefs []MatchableDefinition
	for _, fc := range flowConfigs {
		stageDef := buildTaskStageDefinition(fc.Stages)
		tmpDefs = append(tmpDefs, MatchableDefinition{
			MatchingMetas:       fc.MatchingMetas,
			TaskStageDefinition: stageDef,
		})
	}
	s.flowDefinitions = tmpDefs
	log.Printf("[TaskDefConfigService] received new configs, count = %d", len(tmpDefs))
}

// Load 加载单个流程定义
func (s *TaskDefConfigService) Load(rawFlowConfig string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var fc FlowConfig
	if err := json.Unmarshal([]byte(rawFlowConfig), &fc); err != nil {
		log.Printf("[TaskDefConfigService] process config exception, config = %s, error = %v", rawFlowConfig, err)
		panic(fmt.Sprintf("process flow definition exception: %v", err))
	}

	stageDef := buildTaskStageDefinition(fc.Stages)
	newDef := MatchableDefinition{
		MatchingMetas:       fc.MatchingMetas,
		TaskStageDefinition: stageDef,
	}

	for i, existing := range s.flowDefinitions {
		if existing.IsMatch(fc.MatchingMetas) {
			s.flowDefinitions[i] = newDef
			return
		}
	}
	s.flowDefinitions = append(s.flowDefinitions, newDef)
}

func buildTaskStageDefinition(stageConfigs []StageConfig) *TaskStageDefinition {
	var currentDef *TaskStageDefinition
	// 最终要返回的目标: 步骤定义配置链的头指针
	var firstDef *TaskStageDefinition

	for order, sc := range stageConfigs {
		prev := currentDef
		currentDef = doBuildTaskStageDefinition(order, len(stageConfigs), sc)
		if prev != nil {
			prev.SetNext(currentDef)
		} else {
			firstDef = currentDef
		}
	}
	return firstDef
}

func doBuildTaskStageDefinition(order, totalStageCount int, stageConfig StageConfig) *TaskStageDefinition {
	var definitions []StepDefinition
	for _, step := range stageConfig.Steps {
		normalDef, compensateDef := step.buildDefinition(order, totalStageCount, stageConfig)
		definitions = append(definitions, normalDef, compensateDef)
	}
	return BuildCompensableDefinition(stageConfig.AutoExecute, definitions)
}

// buildDefinition 构建步骤定义, 返回 (normalDefinition, compensateDefinition)
func (sa *StepArrangement) buildDefinition(order, totalStageCount int, stageConfig StageConfig) (StepDefinition, StepDefinition) {
	switch sa.Type {
	case StepArrangeAtomic:
		return sa.buildAtomicDefinition(order, totalStageCount, stageConfig)
	case StepArrangeNested:
		return sa.buildNestedDefinition(order, totalStageCount, stageConfig)
	default:
		panic(fmt.Sprintf("unknown step arrange type: %s", sa.Type))
	}
}

func (sa *StepArrangement) buildAtomicDefinition(order, totalStageCount int, stageConfig StageConfig) (StepDefinition, StepDefinition) {
	normalConfig := sa.StepConfigs.Normal
	compensateConfig := sa.StepConfigs.Compensate
	if compensateConfig == nil {
		defaultCfg := DefaultTaskStepConfig()
		defaultCfg.Name = padding.NoneStepName
		defaultCfg.TaskStepBuilder = "padding.NoneTaskStepBuilder"
		compensateConfig = &defaultCfg
	}

	normalStage := view.Stage{StageOrder: order, StageName: stageConfig.Stage}
	compensateStage := view.Stage{
		StageOrder: calcCompensateStageOrder(totalStageCount, order),
		StageName:  "补偿: " + stageConfig.Stage,
	}

	normalBuilder, ok := core.GetTaskStepBuilder(normalConfig.TaskStepBuilder)
	if !ok {
		panic(fmt.Sprintf("TaskStepBuilder not registered: %s", normalConfig.TaskStepBuilder))
	}
	normalDef := &AtomicStepDefinition{
		StageInfo:              normalStage,
		Name:                   normalConfig.Name,
		TaskStepBuilder:        normalBuilder,
		Stateful:               normalConfig.Stateful,
		AutoUpdate:             normalConfig.AutoUpdate,
		AutoUpdateIntervalSecs: normalConfig.AutoUpdateIntervalSecs,
		AutoUpdateErrorRetry:   normalConfig.AutoUpdateErrorRetry,
		AutoUpdateMaxTimes:     normalConfig.AutoUpdateMaxTimes,
		Blockable:              normalConfig.Blockable,
	}

	compensateBuilder, ok := core.GetTaskStepBuilder(compensateConfig.TaskStepBuilder)
	if !ok {
		panic(fmt.Sprintf("TaskStepBuilder not registered: %s", compensateConfig.TaskStepBuilder))
	}
	compensateDef := &CompensateAtomicStepDefinition{
		AtomicStepDefinition: AtomicStepDefinition{
			StageInfo:              compensateStage,
			Name:                   compensateConfig.Name,
			TaskStepBuilder:        compensateBuilder,
			Stateful:               compensateConfig.Stateful,
			AutoUpdate:             compensateConfig.AutoUpdate,
			AutoUpdateIntervalSecs: compensateConfig.AutoUpdateIntervalSecs,
			AutoUpdateErrorRetry:   compensateConfig.AutoUpdateErrorRetry,
			AutoUpdateMaxTimes:     compensateConfig.AutoUpdateMaxTimes,
			Blockable:              compensateConfig.Blockable,
		},
		OriginDefinition: normalDef,
	}

	return normalDef, compensateDef
}

func (sa *StepArrangement) buildNestedDefinition(order, totalStageCount int, stageConfig StageConfig) (StepDefinition, StepDefinition) {
	nestedDef := buildTaskStageDefinition(sa.Stages)

	normalStage := view.Stage{StageOrder: order, StageName: stageConfig.Stage}
	compensateStage := view.Stage{
		StageOrder: calcCompensateStageOrder(totalStageCount, order),
		StageName:  "补偿: " + stageConfig.Stage,
	}

	normalDef := &NestedStepDefinition{
		StageInfo:        normalStage,
		NestedDefinition: nestedDef,
	}
	compensateDef := &NestedStepDefinition{
		StageInfo:        compensateStage,
		NestedDefinition: BuildCompensableDefinition(false, nil),
	}

	return normalDef, compensateDef
}

// calcCompensateStageOrder 给定原始阶段的 stageOrder, 计算对应的补偿阶段的 compensateStageOrder
//
// 原始阶段及对应补偿阶段呈轴对称分布, 例如 totalStageCount = 3 的场景:
//   - 0 (原始阶段 stageOrder = 0)
//   - 1 (原始阶段 stageOrder = 1)
//   - 2 (原始阶段 stageOrder = 2)
//   - 3 (原始阶段 2 的补偿阶段: 3 + (2 - 2))
//   - 4 (原始阶段 1 的补偿阶段: 3 + (2 - 1))
//   - 5 (原始阶段 0 的补偿阶段: 3 + (2 - 0))
//
// 计算公式: compensateStageOrder = totalStageCount + [(totalStageCount -1) - normalStageOrder]
func calcCompensateStageOrder(totalStageCount, normalStageOrder int) int {
	return 2*totalStageCount - 1 - normalStageOrder
}
