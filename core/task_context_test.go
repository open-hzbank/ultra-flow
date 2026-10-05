package core

import (
	"encoding/json"
	"testing"
)

func TestTaskContext_NonStringContextContent(t *testing.T) {
	taskContext := NewTaskContext(nil, make(map[string]any))
	taskContext.Put("taskCancelMark", true)

	contextMap := taskContext.GetContext()
	jsonData, err := json.Marshal(contextMap)
	if err != nil {
		t.Fatalf("Failed to marshal context: %v", err)
	}

	var parsedMap map[string]any
	if err := json.Unmarshal(jsonData, &parsedMap); err != nil {
		t.Fatalf("Failed to unmarshal context: %v", err)
	}

	taskCancelling, ok := parsedMap["taskCancelMark"].(bool)
	if !ok {
		t.Fatal("Expected taskCancelMark to be bool")
	}
	if !taskCancelling {
		t.Error("Expected taskCancelling to be true")
	}
}
