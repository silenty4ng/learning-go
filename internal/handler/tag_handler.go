package handler

import (
	"github.com/gin-gonic/gin"

	"igoblog/internal/apperr"
	"igoblog/internal/middleware"
	"igoblog/internal/service"
)

// TagHandler 标签相关 HTTP 处理器。
type TagHandler struct{ svc service.TagService }

// NewTagHandler 构造 TagHandler。
func NewTagHandler(svc service.TagService) *TagHandler { return &TagHandler{svc: svc} }

type tagRequest struct {
	Name string `json:"name" binding:"required,max=30"`
}

// Create POST /api/v1/tags
func (h *TagHandler) Create(c *gin.Context) {
	var req tagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	t, err := h.svc.Create(c.Request.Context(), req.Name)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.Created(c, t)
}

// Get GET /api/v1/tags/:id
func (h *TagHandler) Get(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	t, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, t)
}

// List GET /api/v1/tags
func (h *TagHandler) List(c *gin.Context) {
	tags, err := h.svc.List(c.Request.Context())
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, tags)
}

// Delete DELETE /api/v1/tags/:id
func (h *TagHandler) Delete(c *gin.Context) {
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
