package arrange

import (
	"encoding/json"
	"testing"
)

func TestLoadCompensateAwareConfig(t *testing.T) {
	flowConfig := `[
		{
			"stage": "前置准备",
			"steps": [
				{
					"normal": {
						"name": "zzz",
						"taskStepBuilder": "com.mtop"
					},
					"compensate": {
						"name": "zzz",
						"taskStepBuilder": "com.mtop"
					}
				},
				{
					"stages": [{
						"stage": "前置校验",
						"steps": [{
								"normal": {
									"name": "securityCheck",
									"taskStepBuilder": "com.mtop"
								},
								"compensate": {
									"name": "zzz",
									"taskStepBuilder": "com.mtop"
								}
							},
							{
								"normal": {
									"name": "complianceCheck",
									"taskStepBuilder": "com.mtop"
								},
								"compensate": {
									"name": "zzz",
									"taskStepBuilder": "com.mtop"
								}
							}
						]
					}]
				}
			]
		},
		{
			"stage": "API 发布",
			"steps": [
				{
					"normal": {
						"name": "routePublish",
						"taskStepBuilder": "com.mtop"
					},
					"compensate": {
						"name": "zzz",
						"taskStepBuilder": "com.mtop"
					}
				},
				{
					"stages": [{
							"stage": "bpms 审批",
							"steps": [{
								"normal": {
									"name": "zzz",
									"taskStepBuilder": "com.mtop"
								},
								"compensate": {
									"name": "zzz",
									"taskStepBuilder": "com.mtop"
								}
							}]
						},
						{
							"stage": "元信息发布",
							"steps": [{
								"normal": {
									"name": "zzz",
									"taskStepBuilder": "com.mtop"
								},
								"compensate": {
									"name": "zzz",
									"taskStepBuilder": "com.mtop"
								}
							}]
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

	if len(stageConfigs) != 2 {
		t.Fatalf("expected 2 stage configs, got %d", len(stageConfigs))
	}
}
