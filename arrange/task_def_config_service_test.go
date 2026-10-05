package arrange

import (
	"testing"
)

func TestGetFlowDefinition(t *testing.T) {
	flowConfig := `[
		{
			"desc": "test",
			"matchingMetas": {
				"a": "1",
				"b": "2",
				"c": "3"
			},
			"stages": [
				{
					"stage": "0",
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
	]`

	configService := NewTaskDefConfigService()
	configService.LoadAll(flowConfig)

	// match: all 3 keys present
	_, found := configService.GetFlowDefinition(map[string]string{"a": "1", "b": "2", "c": "3"})
	if !found {
		t.Error("expected to find flow definition with matching metas {a:1, b:2, c:3}")
	}

	// no match: missing key "c"
	_, found = configService.GetFlowDefinition(map[string]string{"a": "1", "b": "2"})
	if found {
		t.Error("expected not to find flow definition with metas {a:1, b:2} (missing c)")
	}

	// no match: different value for "c"
	_, found = configService.GetFlowDefinition(map[string]string{"a": "1", "b": "2", "c": "4"})
	if found {
		t.Error("expected not to find flow definition with metas {a:1, b:2, c:4}")
	}
}
