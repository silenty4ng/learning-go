// Package middleware 提供 Gin 中间件与统一响应辅助。
package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"igoblog/internal/apperr"
)

// Response 统一 JSON 响应结构。
// 成功：{"message":"ok","data":...}；失败：{"code":"NOT_FOUND","message":"文章不存在"}。
type Response struct {
	Code    string `json:"code,omitempty"` // 业务错误码，成功时为空
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OK 输出 200 成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Message: "ok", Data: data})
}

// Created 输出 201 创建成功响应。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{Message: "created", Data: data})
}

// Fail 将任意错误统一转换为 JSON 响应，并记录内部错误日志。
func Fail(c *gin.Context, err error) {
	ae := apperr.From(err)
	if ae.Status == http.StatusInternalServerError {
		// 内部错误记录完整错误链便于排查；对外只暴露通用消息
		log.Printf("[error] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	}
	c.JSON(ae.Status, Response{Code: ae.Code, Message: ae.Msg})
}
