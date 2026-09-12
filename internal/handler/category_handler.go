package handler

import (
	"github.com/gin-gonic/gin"

	"igoblog/internal/apperr"
	"igoblog/internal/middleware"
	"igoblog/internal/service"
)

// CategoryHandler 分类相关 HTTP 处理器。
type CategoryHandler struct{ svc service.CategoryService }

// NewCategoryHandler 构造 CategoryHandler。
func NewCategoryHandler(svc service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

type categoryRequest struct {
	Name string `json:"name" binding:"required,max=50"`
}

// Create POST /api/v1/categories
func (h *CategoryHandler) Create(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	cat, err := h.svc.Create(c.Request.Context(), req.Name)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.Created(c, cat)
}

// Get GET /api/v1/categories/:id
func (h *CategoryHandler) Get(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	cat, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, cat)
}

// List GET /api/v1/categories
func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.svc.List(c.Request.Context())
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, categories)
}

// Update PUT /api/v1/categories/:id
func (h *CategoryHandler) Update(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	cat, err := h.svc.Update(c.Request.Context(), id, req.Name)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, cat)
}

// Delete DELETE /api/v1/categories/:id
func (h *CategoryHandler) Delete(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, nil)
}
