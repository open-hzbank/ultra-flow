package arrange

import (
	"encoding/json"
	"testing"
)

func TestBuildTaskStageDefinitions2(t *testing.T) {
	flowConfig := `[
		{
			"stage": "test",
			"steps": [
				{
					"stages": [
						{
							"stage": "0_0",
							"steps": [
								{
									"normal": {
										"name": "a",
										"taskStepBuilder": "github.com/open-hzbank/ultra-flow/padding.FixedStatusTaskBuilder"
									},
									"compensate": {
										"name": "a_Compensate",
										"taskStepBuilder": "github.com/open-hzbank/ultra-flow/padding.FixedStatusTaskBuilder"
									}
								}
							]
						}
					]
				}
			]
		},
		{
			"stage": "test2",
			"steps": [
				{
					"stages": [
						{
							"stage": "0_1",
							"steps": [
								{
									"normal": {
										"name": "b",
										"taskStepBuilder": "github.com/open-hzbank/ultra-flow/padding.FixedStatusTaskBuilder"
									},
									"compensate": {
										"name": "b_Compensate",
										"taskStepBuilder": "github.com/open-hzbank/ultra-flow/padding.FixedStatusTaskBuilder"
									}
								}
							]
						}
					]
				}
			]
		}
	]`

	var stageConfigs []StageConfig
	if err := json.Unmarshal([]byte(flowConfig), &stageConfigs); err != nil {
		t.Fatalf("failed to unmarshal stage configs: %v", err)
	}

	stageDefinition := buildTaskStageDefinition(stageConfigs)

	if len(stageDefinition.Definitions) != 1 {
		t.Fatalf("expected 2 definitions, got %d", len(stageDefinition.Definitions))
	}

	compensateDef0 := stageDefinition.compensate.TaskStageDefinition.Definitions[0]
	nestedCompensateDef, ok := compensateDef0.(*NestedStepDefinition)
	if ok {
		if len(nestedCompensateDef.NestedDefinition.Definitions) == 0 {
			t.Errorf("[1] size expected > 0, got %v", nestedCompensateDef.NestedDefinition.Definitions)
		}
	}
	subCompensateDef0 := nestedCompensateDef.NestedDefinition.Definitions[0]
	subAtomicCompensateDef, ok := subCompensateDef0.(*CompensateAtomicStepDefinition)
	if ok {
		if subAtomicCompensateDef.Name != "b_Compensate" {
			t.Errorf("[1] size expected > 0, got %v", nestedCompensateDef.NestedDefinition.Definitions)
		}
	}
	deff := stageDefinition.compensate.TaskStageDefinition.NextDef.Definitions[0]
	subAtomicDeff, ok := deff.(*CompensateAtomicStepDefinition)
	if ok {
		if subAtomicDeff.Name != "a_Compensate" {
			t.Errorf("[1] size expected > 0, got %v", nestedCompensateDef.NestedDefinition.Definitions)
		}
	}
}
