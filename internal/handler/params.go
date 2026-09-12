// Package handler 实现 HTTP 层：参数绑定校验、调用 service、统一响应。
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"igoblog/internal/apperr"
)

// queryInt 读取整型查询参数，缺省或非法时返回默认值。
func queryInt(c *gin.Context, key string, def int64) int64 {
	s := c.Query(key)
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}

// paramInt 解析路径参数 :id，非法时返回 false。
func paramInt(c *gin.Context, name string) (int64, bool) {
	v, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

// normalizePage 将分页参数收敛到安全范围。
func normalizePage(page, pageSize int64) (int, int) {
	if page < 1 {
		page = 1
	}
	const maxPageSize = 50
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return int(page), int(pageSize)
}

// idError 路径参数非法的统一错误。
func idError() error {
	return apperr.BadRequest("路径参数 id 必须为正整数")
}
