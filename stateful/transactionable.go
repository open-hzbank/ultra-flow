package stateful

// Transactionable 针对数据 & 状态的事务规范定义
// 不涉及事务的具体实现方式, 由子类自行扩展
type Transactionable interface {
	Execute(action TransactionAction) any
}

// TransactionAction 事务内执行的动作
type TransactionAction func(status TxStatus) any

// TxStatus 事务状态
type TxStatus interface {
	Rollback()
}

// NoopTransactionable 无事务实现 (直接执行)
type NoopTransactionable struct{}

func (t *NoopTransactionable) Execute(action TransactionAction) any {
	return action(nil)
}
