package ops

import (
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk"
	"github.com/zshell-zhang/ultra-flow-go/flow-sdk/stateful"
)

// OpsTask 运维任务
type OpsTask struct {
	*stateful.StatefulTask
	globalBusData     interface{}
	taskDataSerDeser TaskDataSerDeser
}

// NewOpsTask 创建运维任务
func NewOpsTask(taskId, name, type_, creator string, subject interface{}, publishEnv flowsdk.Env, idempotentId string, description flowsdk.Description, taskPersistence stateful.TaskPersistence) *OpsTask {
	task := &OpsTask{
		StatefulTask: stateful.NewStatefulTask(taskId, name, type_, creator, subject, publishEnv, idempotentId, description, taskPersistence),
	}

	// 恢复数据
	task.recoverData()

	return task
}

// SetGlobalBusData 设置全局业务数据
func (t *OpsTask) SetGlobalBusData(data interface{}) {
	t.globalBusData = data
}

// GetGlobalBusData 获取全局业务数据
func (t *OpsTask) GetGlobalBusData() interface{} {
	return t.globalBusData
}

// SetTaskDataSerDeser 设置任务数据序列化/反序列化器
func (t *OpsTask) SetTaskDataSerDeser(serDeser TaskDataSerDeser) {
	t.taskDataSerDeser = serDeser
}

// Execute 执行任务
func (t *OpsTask) Execute() flowsdk.TaskResult {
	result := t.StatefulTask.Execute()
	// 保存数据
	t.saveData()
	return result
}

// recoverData 恢复数据
func (t *OpsTask) recoverData() {
	if t.taskDataSerDeser != nil {
		data := t.taskDataSerDeser.Deserialize(t.GetTaskId())
		if data != nil {
			t.globalBusData = data
		}
	}
}

// saveData 保存数据
func (t *OpsTask) saveData() {
	if t.taskDataSerDeser != nil && t.globalBusData != nil {
		t.taskDataSerDeser.Serialize(t.GetTaskId(), t.globalBusData)
	}
}

// CompensableOpsTask 可补偿的运维任务
type CompensableOpsTask struct {
	*OpsTask
}

// NewCompensableOpsTask 创建可补偿的运维任务
func NewCompensableOpsTask(taskId, name, type_, creator string, subject interface{}, publishEnv flowsdk.Env, idempotentId string, description flowsdk.Description, taskPersistence stateful.TaskPersistence) *CompensableOpsTask {
	task := &CompensableOpsTask{
		OpsTask: NewOpsTask(taskId, name, type_, creator, subject, publishEnv, idempotentId, description, taskPersistence),
	}

	return task
}

// TaskDataSerDeser 任务数据序列化/反序列化接口
type TaskDataSerDeser interface {
	// Serialize 序列化任务数据
	Serialize(taskId string, data interface{})
	// Deserialize 反序列化任务数据
	Deserialize(taskId string) interface{}
}
