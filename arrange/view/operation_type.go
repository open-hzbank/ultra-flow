package view

// OperationType 任务可以执行的操作状态
type OperationType string

const (
	// OpAllowFireNextBatch 可以发布步骤中的下一批次
	OpAllowFireNextBatch OperationType = "ALLOW_FIRE_NEXT_BATCH"
	// OpForbidFireNextBatch 不能发布步骤中的下一批次
	OpForbidFireNextBatch OperationType = "FORBID_FIRE_NEXT_BATCH"
	// OpAllowFireFirstStep 可以开始发布首个步骤
	OpAllowFireFirstStep OperationType = "ALLOW_FIRE_FIRST_STEP"
	// OpAllowFireNextStep 可以发布下一步骤 (非首个)
	OpAllowFireNextStep OperationType = "ALLOW_FIRE_NEXT_STEP"
	// OpForbidFireNextStep 不能发布下一步骤
	OpForbidFireNextStep OperationType = "FORBID_FIRE_NEXT_STEP"
	// OpForbidByBlockedSteps 流程被卡点步骤阻断
	OpForbidByBlockedSteps OperationType = "FORBID_BY_BLOCKED_STEPS"
)
