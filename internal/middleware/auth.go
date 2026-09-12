package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
	"igoblog/internal/pkg/jwtutil"
	"igoblog/internal/repository"
)

// Gin context 中存放认证信息的键。
const (
	ctxUserID   = "auth.user_id"
	ctxUsername = "auth.username"
	ctxRole     = "auth.role"
)

// Auth 封装 JWT 认证。
// 与无状态纯 Token 校验不同，这里在解析 Token 后回查用户表：
// 既把最新角色注入 context，也让被删除用户的 Token 立即失效。
type Auth struct {
	secret string
	users  repository.UserRepository
}

// NewAuth 构造认证器（依赖用户仓储以回查最新角色与存在性）。
func NewAuth(secret string, users repository.UserRepository) *Auth {
	return &Auth{secret: secret, users: users}
}

// Require 返回校验 Bearer Token 的 Gin 中间件，
// 校验通过后将用户信息（含角色）注入 context，供后续 handler 取用。
func (a *Auth) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(raw, "Bearer ")
		if !ok || token == "" {
			Fail(c, apperr.Unauthorized("缺少 Authorization: Bearer <token>"))
			c.Abort()
			return
		}
		claims, err := jwtutil.Parse(a.secret, token)
		if err != nil {
			Fail(c, apperr.Unauthorized("登录状态无效或已过期"))
			c.Abort()
			return
		}
		// 回查数据库：角色以库内最新值为准，账号被删则 Token 失效
		u, err := a.users.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			Fail(c, apperr.Unauthorized("账号不存在或已失效"))
			c.Abort()
			return
		}
		c.Set(ctxUserID, u.ID)
		c.Set(ctxUsername, u.Username)
		c.Set(ctxRole, u.Role)
		c.Next()
	}
}

// RequireAdmin 管理员守卫，必须挂在 Require 之后使用。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if Role(c) != model.RoleAdmin {
			Fail(c, apperr.Forbidden("需要管理员权限"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// UserID 取出当前登录用户 ID（须在 Require 中间件之后调用）。
func UserID(c *gin.Context) int64 {
	v, _ := c.Get(ctxUserID)
	id, _ := v.(int64)
	return id
}

// Username 取出当前登录用户名。
func Username(c *gin.Context) string {
	v, _ := c.Get(ctxUsername)
	s, _ := v.(string)
	return s
}

// Role 取出当前登录用户角色。
func Role(c *gin.Context) string {
	v, _ := c.Get(ctxRole)
	s, _ := v.(string)
	return s
}

// Operator 从 context 构造操作者身份，传给 service 层做权限判断。
func Operator(c *gin.Context) model.Operator {
	return model.Operator{ID: UserID(c), Role: Role(c)}
}
