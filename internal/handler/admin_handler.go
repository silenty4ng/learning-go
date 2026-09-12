package handler

import (
	"github.com/gin-gonic/gin"

	"igoblog/internal/apperr"
	"igoblog/internal/middleware"
	"igoblog/internal/model"
	"igoblog/internal/service"
)

// AdminHandler 管理员功能 HTTP 处理器：用户管理与站点设置。
type AdminHandler struct{ svc service.AdminService }

// NewAdminHandler 构造 AdminHandler。
func NewAdminHandler(svc service.AdminService) *AdminHandler { return &AdminHandler{svc: svc} }

// ListUsers GET /api/v1/admin/users?page=&page_size=
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, pageSize := normalizePage(
		queryInt(c, "page", 1),
		queryInt(c, "page_size", 20),
	)
	users, total, err := h.svc.ListUsers(c.Request.Context(), page, pageSize)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, gin.H{
		"items":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

type roleRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateUserRole PUT /api/v1/admin/users/:id/role
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	u, err := h.svc.UpdateUserRole(c.Request.Context(), middleware.UserID(c), id, req.Role)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, u)
}

// DeleteUser DELETE /api/v1/admin/users/:id
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	if err := h.svc.DeleteUser(c.Request.Context(), middleware.UserID(c), id); err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, nil)
}

// GetSiteSettings GET /api/v1/admin/settings（另有公开版 /api/v1/site/settings）
func (h *AdminHandler) GetSiteSettings(c *gin.Context) {
	s, err := h.svc.GetSiteSettings(c.Request.Context())
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, s)
}

type settingsRequest struct {
	SiteTitle           string `json:"site_title" binding:"required,max=50"`
	SiteDescription     string `json:"site_description" binding:"max=200"`
	RegistrationEnabled bool   `json:"registration_enabled"`
}

// UpdateSiteSettings PUT /api/v1/admin/settings
func (h *AdminHandler) UpdateSiteSettings(c *gin.Context) {
	var req settingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	s, err := h.svc.UpdateSiteSettings(c.Request.Context(), model.SiteSettings{
		SiteTitle:           req.SiteTitle,
		SiteDescription:     req.SiteDescription,
		RegistrationEnabled: req.RegistrationEnabled,
	})
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, s)
}
