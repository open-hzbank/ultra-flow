package access

import "hzbank.com.cn/ultra-flow/support"

// Response 通用 HTTP 响应包装
type Response[T any] struct {
	Success bool   `json:"success"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Result  T      `json:"result,omitempty"`
}

// WrapResponse 将业务 Result 包装为 HTTP Response
func WrapResponse[T any](result support.Result[T]) Response[T] {
	return Response[T]{
		Success: result.IsSuccess(),
		Code:    result.GetErrorCode(),
		Message: result.GetErrorMessage(),
		Result:  result.GetData(),
	}
}

// SuccessResponse 成功响应 (无数据)
func SuccessResponse[T any]() Response[T] {
	return Response[T]{Success: true}
}

// SuccessResponseWithData 成功响应 (携带数据)
func SuccessResponseWithData[T any](data T) Response[T] {
	return Response[T]{Success: true, Result: data}
}

// FailureResponse 失败响应
func FailureResponse[T any](code, message string) Response[T] {
	return Response[T]{Success: false, Code: code, Message: message}
}
