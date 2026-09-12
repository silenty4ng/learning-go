package handler

import (
	"github.com/gin-gonic/gin"

	"igoblog/internal/apperr"
	"igoblog/internal/middleware"
	"igoblog/internal/service"
)

// UserHandler 用户注册/登录相关的 HTTP 处理器。
type UserHandler struct{ svc service.UserService }

// NewUserHandler 构造 UserHandler。
func NewUserHandler(svc service.UserService) *UserHandler { return &UserHandler{svc: svc} }

type authRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register POST /api/v1/auth/register
func (h *UserHandler) Register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	u, err := h.svc.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.Created(c, u)
}

type nicknameRequest struct {
	Nickname string `json:"nickname" binding:"max=30"`
}

// UpdateNickname PUT /api/v1/users/me：修改当前用户昵称。
func (h *UserHandler) UpdateNickname(c *gin.Context) {
	var req nicknameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	u, err := h.svc.UpdateProfile(c.Request.Context(), middleware.UserID(c), req.Nickname)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, u)
}

// Login POST /api/v1/auth/login
func (h *UserHandler) Login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	token, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, gin.H{"token": token})
}

// Me GET /api/v1/users/me：返回当前登录用户完整资料。
func (h *UserHandler) Me(c *gin.Context) {
	u, err := h.svc.Profile(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, u)
}
