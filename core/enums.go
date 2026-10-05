package core

import (
	"encoding/json"
	"fmt"
)

// ExecuteMode 执行模式
type ExecuteMode int

const (
	ModePublish ExecuteMode = iota // 正向发布
	ModeCancel                     // 逆向取消
)

// PublishType 发布类型
type PublishType int

const (
	PublishOnline  PublishType = iota // 上线
	PublishOffline                     // 下线
)

var publishTypeNames = map[PublishType]string{
	PublishOnline:  "ONLINE",
	PublishOffline: "OFFLINE",
}

var publishTypeValues = map[string]PublishType{
	"ONLINE":  PublishOnline,
	"OFFLINE": PublishOffline,
}

func (p PublishType) MarshalJSON() ([]byte, error) {
	if name, ok := publishTypeNames[p]; ok {
		return json.Marshal(name)
	}
	return json.Marshal("UNKNOWN")
}

func (p *PublishType) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	if v, ok := publishTypeValues[name]; ok {
		*p = v
		return nil
	}
	return fmt.Errorf("unknown PublishType: %s", name)
}

// TaskType 任务类型
type TaskType int

const (
	// 发布
	TaskTypePublish TaskType = iota
	// 回滚
	TaskTypeRollback
)

var taskTypeNames = map[TaskType]string{
	TaskTypePublish:  "PUBLISH",
	TaskTypeRollback: "ROLLBACK",
}

var taskTypeValues = map[string]TaskType{
	"PUBLISH":  TaskTypePublish,
	"ROLLBACK": TaskTypeRollback,
}

func (t TaskType) MarshalJSON() ([]byte, error) {
	if name, ok := taskTypeNames[t]; ok {
		return json.Marshal(name)
	}
	return json.Marshal("UNKNOWN")
}

func (t *TaskType) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	if v, ok := taskTypeValues[name]; ok {
		*t = v
		return nil
	}
	return fmt.Errorf("unknown TaskType: %s", name)
}

// StringToTaskType 将字符串转换为 TaskType
func StringToTaskType(name string) (TaskType, bool) {
	v, ok := taskTypeValues[name]
	return v, ok
}

// TargetPublishScene 单个发布对象的发布行为场景
type TargetPublishScene int

const (
	TargetFirstPublish  TargetPublishScene = iota
	TargetUpdatePublish
	TargetOffline
)

// TaskPublishScene 发布场景
type TaskPublishScene int

const (
	// 待发布对象全部都是首次发布
	AllFirstPublish TaskPublishScene = iota
	// 待发布对象存在非首次发布
	ExistsUpdatePublish
	// 发布对象全部为下线操作
	AllOffline
)
