package arrange

import (
	"encoding/json"
	"testing"
)

func TestBuildTaskStageDefinitions(t *testing.T) {
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
										"taskStepBuilder": "hzbank.com.cn/ultra-flow/padding.FixedStatusTaskBuilder"
									},
									"compensate": {
										"name": "a_Compensate",
										"taskStepBuilder": "hzbank.com.cn/ultra-flow/padding.FixedStatusTaskBuilder"
									}
								}
							]
						},
						{
							"stage": "0_1",
							"steps": [
								{
									"normal": {
										"name": "b",
										"taskStepBuilder": "hzbank.com.cn/ultra-flow/padding.FixedStatusTaskBuilder"
									},
									"compensate": {
										"name": "b_Compensate",
										"taskStepBuilder": "hzbank.com.cn/ultra-flow/padding.FixedStatusTaskBuilder"
									}
								}
							]
						}
					]
				},
				{
					"stages": [
						{
							"stage": "1_0",
							"steps": [
								{
									"normal": {
										"name": "c",
										"taskStepBuilder": "hzbank.com.cn/ultra-flow/padding.FixedStatusTaskBuilder"
									},
									"compensate": {
										"name": "c_Compensate",
										"taskStepBuilder": "hzbank.com.cn/ultra-flow/padding.FixedStatusTaskBuilder"
									}
								}
							]
						},
						{
							"stage": "1_1",
							"steps": [
								{
									"normal": {
										"name": "d",
										"taskStepBuilder": "hzbank.com.cn/ultra-flow/padding.FixedStatusTaskBuilder"
									},
									"compensate": {
										"name": "d_Compensate",
										"taskStepBuilder": "hzbank.com.cn/ultra-flow/padding.FixedStatusTaskBuilder"
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

	if len(stageDefinition.Definitions) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(stageDefinition.Definitions))
	}

	// definition[0]
	def0 := stageDefinition.Definitions[0]
	if def0.GetStage().StageName != "test" {
		t.Errorf("[0] expected stageName=test, got %s", def0.GetStage().StageName)
	}
	if def0.GetStage().StageOrder != 0 {
		t.Errorf("[0] expected stageOrder=0, got %d", def0.GetStage().StageOrder)
	}
	if def0.GetType() != StepDefNested {
		t.Errorf("[0] expected type=NESTED, got %v", def0.GetType())
	}

	// definition[1]
	def1 := stageDefinition.Definitions[1]
	if def1.GetStage().StageName != "test" {
		t.Errorf("[1] expected stageName=test, got %s", def1.GetStage().StageName)
	}
	if def1.GetStage().StageOrder != 0 {
		t.Errorf("[1] expected stageOrder=0, got %d", def1.GetStage().StageOrder)
	}
	if def1.GetType() != StepDefNested {
		t.Errorf("[1] expected type=NESTED, got %v", def1.GetType())
	}
}
