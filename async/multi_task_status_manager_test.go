package async

import (
	"encoding/json"
	"testing"

	"github.com/open-hzbank/ultra-flow/core"
)

func TestMultiTaskStatusManager_UnSubmittedTaskSnapshotKeys(t *testing.T) {
	// 测试场景 1: 部分任务未提交
	jsonData1 := `{
		"ONLINE_tao": {"status": "SUCCESS", "taskId": "311239"},
		"OFFLINE_tao": {"status": "PENDING"},
		"ONLINE_baichuan": {"status": "PENDING"},
		"OFFLINE_baichuan": {"status": "RUNNING", "taskId": "311238"}
	}`
	manager1 := initTestManager(jsonData1)
	unsubmitted1 := manager1.UnSubmittedTaskSnapshotKeys()
	if len(unsubmitted1) != 2 {
		t.Errorf("Expected 2 unsubmitted tasks, got %d", len(unsubmitted1))
	}
	if manager1.AllTasksSubmitted() {
		t.Error("Expected AllTasksSubmitted to be false")
	}

	// 测试场景 2: 所有任务已提交
	jsonData2 := `{
		"ONLINE_tao": {"status": "SUCCESS", "taskId": "311239"},
		"OFFLINE_tao": {"status": "RUNNING", "taskId": "311238"},
		"ONLINE_baichuan": {"status": "FAILURE", "taskId": "311229"},
		"OFFLINE_baichuan": {"status": "RUNNING", "taskId": "311239"}
	}`
	manager2 := initTestManager(jsonData2)
	unsubmitted2 := manager2.UnSubmittedTaskSnapshotKeys()
	if len(unsubmitted2) != 0 {
		t.Errorf("Expected 0 unsubmitted tasks, got %d", len(unsubmitted2))
	}
	if !manager2.AllTasksSubmitted() {
		t.Error("Expected AllTasksSubmitted to be true")
	}
}

func TestMultiTaskStatusManager_CalcMultiTaskStatus(t *testing.T) {
	// 测试场景 1: RUNNING (混合 PENDING 和 RUNNING)
	jsonData1 := `{
		"ONLINE_tao": {"status": "SUCCESS", "taskId": "311239"},
		"OFFLINE_tao": {"status": "PENDING"},
		"ONLINE_baichuan": {"status": "PENDING"},
		"OFFLINE_baichuan": {"status": "RUNNING", "taskId": "311238"}
	}`
	manager1 := initTestManager(jsonData1)
	if status := manager1.CalcMultiTaskStatus(); status != TaskStatusRunning {
		t.Errorf("Expected RUNNING, got %v", status)
	}

	// 测试场景 2: SKIPPED (空)
	manager2 := initTestManager(`{}`)
	if status := manager2.CalcMultiTaskStatus(); status != TaskStatusSkipped {
		t.Errorf("Expected SKIPPED, got %v", status)
	}

	// 测试场景 3: FAILURE (有 FAILURE 状态)
	jsonData3 := `{
		"ONLINE_tao": {"status": "SUCCESS", "taskId": "311239"},
		"OFFLINE_tao": {"status": "PENDING"},
		"ONLINE_baichuan": {"status": "FAILURE"},
		"OFFLINE_baichuan": {"status": "RUNNING", "taskId": "311238"}
	}`
	manager3 := initTestManager(jsonData3)
	if status := manager3.CalcMultiTaskStatus(); status != TaskStatusFailure {
		t.Errorf("Expected FAILURE, got %v", status)
	}

	// 测试场景 4: SUCCESS (全部 SUCCESS)
	jsonData4 := `{
		"ONLINE_tao": {"status": "SUCCESS", "taskId": "311239"},
		"OFFLINE_baichuan": {"status": "SUCCESS", "taskId": "311238"}
	}`
	manager4 := initTestManager(jsonData4)
	if status := manager4.CalcMultiTaskStatus(); status != TaskStatusSuccess {
		t.Errorf("Expected SUCCESS, got %v", status)
	}

	// 测试场景 5: CANCELED (混合 SUCCESS 和 CANCELED)
	jsonData5 := `{
		"ONLINE_tao": {"status": "SUCCESS", "taskId": "311239"},
		"OFFLINE_tao": {"status": "CANCELED", "taskId": "311230"},
		"ONLINE_baichuan": {"status": "CANCELED", "taskId": "312239"},
		"OFFLINE_baichuan": {"status": "SUCCESS", "taskId": "311238"}
	}`
	manager5 := initTestManager(jsonData5)
	if status := manager5.CalcMultiTaskStatus(); status != TaskStatusCanceled {
		t.Errorf("Expected CANCELED, got %v", status)
	}
}

func initTestManager(serializedTaskSnapshots string) *MultiTaskStatusManager {
	context := map[string]any{"wmccTaskSnapshots": serializedTaskSnapshots}
	taskContext := core.NewTaskContext(nil, context)
	return NewTaskBasedManager(taskContext, "wmccTaskSnapshots", []string{})
}

// 辅助函数: 解析 JSON 到 map
func parseJSON(t *testing.T, data string) map[string]*MultiTaskSnapshot {
	var result map[string]*MultiTaskSnapshot
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	return result
}
