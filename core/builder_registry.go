package core

import "sync"

// TaskStepBuilder 注册表: 全限定名 → 构建器实例
// 解决 Go 无法像 Java 那样通过反射 (Class.forName) 根据字符串创建实例的问题
var (
	taskStepBuilderRegistry   = make(map[string]TaskStepBuilder)
	taskStepBuilderRegistryMu sync.RWMutex
)

// RegisterTaskStepBuilder 注册一个 TaskStepBuilder 实例
// fqn 为构建器的全限定名, 如 "hzbank.com.cn/ultra-flow/padding.NoneTaskStepBuilder"
func RegisterTaskStepBuilder(fqn string, builder TaskStepBuilder) {
	taskStepBuilderRegistryMu.Lock()
	defer taskStepBuilderRegistryMu.Unlock()
	taskStepBuilderRegistry[fqn] = builder
}

// GetTaskStepBuilder 根据全限定名获取已注册的 TaskStepBuilder 实例
func GetTaskStepBuilder(fqn string) (TaskStepBuilder, bool) {
	taskStepBuilderRegistryMu.RLock()
	defer taskStepBuilderRegistryMu.RUnlock()
	b, ok := taskStepBuilderRegistry[fqn]
	return b, ok
}
