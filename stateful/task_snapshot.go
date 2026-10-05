package stateful

import (
	"github.com/open-hzbank/ultra-flow/core"
	"time"
)

// TaskSnapshot 任务快照
type TaskSnapshot struct {
	ID          int64
	GmtCreate   time.Time
	GmtModified time.Time

	TaskID string
	// IdempotentID 请求源幂等 id
	IdempotentID string
	// Name 任务名称
	Name string
	// Type 任务类型
	Type string
	// Subject 任务目标主体
	// 不对用户直接透出, 消费该字段必须通过 GetSubjectJSON 方法
	Subject any
	// PublishEnv 发布目标环境
	PublishEnv core.Env
	// Context 任务上下文
	Context map[string]any
	// Creator 创建人
	Creator string
	// Status CREATED、RUNNING、SUCCESS、FAILURE、CANCEL、CANCELLING
	Status core.TaskStatus
	// Description 任务步骤当前的描述
	Description core.Description
}

// GetSubjectJSON 返回 subject 的 JSON 字符串形式
//   - 如果 subject 是普通对象, 则返回其标准 json 串
//   - 如果 subject 原本就是字符串, 则返回的 json 串是: "subject", 需解掉双引号
//
// 流程开发者不能直接使用 subject, 而是必须通过此方法获取 subject 的 json 串形式,
// 因为 subject 经 serialize 再 deserialize 拿到的对象已经不是用户原始放入的类型,
// 为避免直接返回 subject 导致返回类型不明确, 此处通过统一 json 序列化返回明确类型
func (s *TaskSnapshot) GetSubjectJSON() string {
	if s.Subject == nil {
		return ""
	}
	ser := core.DefaultTaskDataSerDeser()
	return ser.Serialize(s.Subject)
}
