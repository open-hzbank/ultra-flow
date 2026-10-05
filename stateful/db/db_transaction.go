package db

import (
	"errors"
	"github.com/open-hzbank/ultra-flow/stateful"
	"log"

	"gorm.io/gorm"
)

// DbTransaction 基于数据库的事务实现
type DbTransaction struct {
	db *gorm.DB
}

// NewDbTransaction 创建基于数据库的事务管理器
func NewDbTransaction(db *gorm.DB) *DbTransaction {
	return &DbTransaction{db: db}
}

// Execute 在事务中执行动作
func (t *DbTransaction) Execute(action stateful.TransactionAction) any {
	var result any
	err := t.db.Transaction(func(tx *gorm.DB) error {
		txStatus := &dbTxStatus{tx: tx}
		result = action(txStatus)
		if txStatus.rollbackOnly {
			return errors.New("transaction rolled back")
		}
		return nil
	})
	if err != nil {
		log.Printf("事务执行失败: %v", err)
	}
	return result
}

// dbTxStatus 数据库事务状态
type dbTxStatus struct {
	tx           *gorm.DB
	rollbackOnly bool
}

// Rollback 标记事务需要回滚
func (s *dbTxStatus) Rollback() {
	s.rollbackOnly = true
}

// 确保接口实现
var _ stateful.Transactionable = (*DbTransaction)(nil)
