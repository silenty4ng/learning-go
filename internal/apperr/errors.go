// Package apperr 定义统一的业务错误类型。
//
// 设计要点（体现 Go 错误处理思想）：
//  1. AppError 实现 error 接口，可像普通错误一样在调用链中传递；
//  2. 通过 Unwrap 支持标准库 errors.Is / errors.As 进行错误判断；
//  3. 携带 HTTP 状态码与业务错误码，供 handler 层统一转换为 JSON 响应。
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// 业务错误码常量。
const (
	CodeInternal     = "INTERNAL_ERROR"
	CodeValidation   = "VALIDATION_ERROR"
	CodeNotFound     = "NOT_FOUND"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeConflict     = "CONFLICT"
	CodeRateLimited  = "RATE_LIMITED"
)

// AppError 业务错误：携带 HTTP 状态码、业务错误码与用户可读消息。
type AppError struct {
	Status int    // HTTP 状态码
	Code   string // 业务错误码
	Msg    string // 面向用户的可读消息
	err    error  // 底层错误（可选），用于保留错误链
}

// Error 实现 error 接口。
func (e *AppError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.err)
	}
	return e.Msg
}

// Unwrap 支持标准库 errors.As / errors.Is 解包。
func (e *AppError) Unwrap() error { return e.err }

// New 构造一个业务错误。
func New(status int, code, msg string) *AppError {
	return &AppError{Status: status, Code: code, Msg: msg}
}

// Wrap 在业务错误上包装底层错误，保留原始错误链便于排查。
func Wrap(status int, code, msg string, err error) *AppError {
	return &AppError{Status: status, Code: code, Msg: msg, err: err}
}

// From 将任意 error 规范化为 *AppError：
// 已是 *AppError 则原样返回；否则包装为 500 内部错误。
func From(err error) *AppError {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	return Wrap(http.StatusInternalServerError, CodeInternal, "服务器内部错误", err)
}

// ---------- 常用业务错误构造器 ----------

// BadRequest 参数校验失败（400）。
func BadRequest(format string, args ...any) *AppError {
	return New(http.StatusBadRequest, CodeValidation, fmt.Sprintf(format, args...))
}

// NotFound 资源不存在（404）。
func NotFound(entity string) *AppError {
	return New(http.StatusNotFound, CodeNotFound, entity+"不存在")
}

// Unauthorized 未认证或认证失效（401）。
func Unauthorized(msg string) *AppError {
	return New(http.StatusUnauthorized, CodeUnauthorized, msg)
}

// Forbidden 已认证但无权限（403）。
func Forbidden(msg string) *AppError {
	return New(http.StatusForbidden, CodeForbidden, msg)
}

// Conflict 资源冲突，如用户名重复（409）。
func Conflict(format string, args ...any) *AppError {
	return New(http.StatusConflict, CodeConflict, fmt.Sprintf(format, args...))
}

// Internal 内部错误，包装底层 err 以便日志排查（500）。
func Internal(err error) *AppError {
	return Wrap(http.StatusInternalServerError, CodeInternal, "服务器内部错误", err)
}
