package core

import (
	"encoding/json"
	"log"
	"reflect"
)

// TaskDataSerDeser 任务数据 序列化/反序列化
type TaskDataSerDeser interface {
	Serialize(data any) string
	Deserialize(raw string) any
}

// JSONTaskDataSerDeser json 序列化/反序列化
type JSONTaskDataSerDeser struct {
	classType reflect.Type
}

// NewJSONTaskDataSerDeser 创建指定类型的序列化器
// prototype 用于确定反序列化的目标类型, 传入值或指针均可 (如 FlowPublishContext{} 或 &FlowPublishContext{})
func NewJSONTaskDataSerDeser(prototype any) *JSONTaskDataSerDeser {
	t := reflect.TypeOf(prototype)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return &JSONTaskDataSerDeser{classType: t}
}

func (s *JSONTaskDataSerDeser) Serialize(data any) string {
	if data == nil {
		return ""
	}
	b, err := json.Marshal(data)
	if err != nil {
		log.Printf("json serialize error, data = %v, error = %v", data, err)
		return ""
	}
	return string(b)
}

func (s *JSONTaskDataSerDeser) Deserialize(raw string) any {
	if raw == "" {
		return nil
	}
	if s.classType != nil {
		ptr := reflect.New(s.classType)
		if err := json.Unmarshal([]byte(raw), ptr.Interface()); err != nil {
			log.Printf("json deserialize error, json = %s, class = %s, error = %v", raw, s.classType.Name(), err)
			return nil
		}
		return ptr.Interface()
	}
	var result any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		log.Printf("json deserialize error, json = %s, error = %v", raw, err)
		return nil
	}
	return result
}

// DefaultTaskDataSerDeser 返回默认的无类型序列化器 (反序列化为 map[string]any)
func DefaultTaskDataSerDeser() TaskDataSerDeser {
	return &JSONTaskDataSerDeser{}
}
