package support

import "fmt"

// Result 通用结果
type Result[T any] struct {
	// 是否成功
	success      bool
	// 错误代码
	errorCode    string
	// 错误响应消息
	errorMessage string
	// 响应结果
	data         T
}

func SuccessResult[T any]() Result[T] {
	return Result[T]{success: true}
}

func SuccessResultWithData[T any](data T) Result[T] {
	return Result[T]{success: true, data: data}
}

func FailResult[T any](errorMessage string) Result[T] {
	return Result[T]{success: false, errorMessage: errorMessage}
}

func FailResultWithError[T any](err error) Result[T] {
	return Result[T]{success: false, errorMessage: err.Error()}
}

func (r Result[T]) IsSuccess() bool  { return r.success }
func (r Result[T]) GetErrorCode() string { return r.errorCode }
func (r Result[T]) GetData() T       { return r.data }
func (r Result[T]) GetErrorMessage() string { return r.errorMessage }

func (r Result[T]) String() string {
	if r.success {
		return fmt.Sprintf("Result{success=true, data=%v}", r.data)
	}
	return fmt.Sprintf("Result{success=false, error=%s}", r.errorMessage)
}
