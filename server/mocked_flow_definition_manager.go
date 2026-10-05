package main

import (
	"os"
	"path/filepath"
	"runtime"

	"hzbank.com.cn/ultra-flow/arrange"
)

// MockedFlowDefinitionManager 从 JSON 文件加载流程定义
type MockedFlowDefinitionManager struct {
	taskDefConfigService *arrange.TaskDefConfigService
}

func NewMockedFlowDefinitionManager(taskDefConfigService *arrange.TaskDefConfigService) *MockedFlowDefinitionManager {
	return &MockedFlowDefinitionManager{taskDefConfigService: taskDefConfigService}
}

func (m *MockedFlowDefinitionManager) InitFlowDefinitions() error {
	_, filename, _, _ := runtime.Caller(0)
	resourcePath := filepath.Join(filepath.Dir(filename), "resources", "flow_definitions.json")

	content, err := os.ReadFile(resourcePath)
	if err != nil {
		return err
	}

	m.taskDefConfigService.LoadAll(string(content))
	return nil
}
