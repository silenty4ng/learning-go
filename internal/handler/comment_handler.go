package handler

import (
	"github.com/gin-gonic/gin"

	"igoblog/internal/apperr"
	"igoblog/internal/middleware"
	"igoblog/internal/service"
)

// CommentHandler 评论相关 HTTP 处理器。
type CommentHandler struct{ svc service.CommentService }

// NewCommentHandler 构造 CommentHandler。
func NewCommentHandler(svc service.CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

type commentRequest struct {
	Content string `json:"content" binding:"required,max=1000"`
}

// Create POST /api/v1/articles/:id/comments
func (h *CommentHandler) Create(c *gin.Context) {
	articleID, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	var req commentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	cm, err := h.svc.Create(c.Request.Context(), middleware.UserID(c), articleID, req.Content)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.Created(c, cm)
}

// List GET /api/v1/articles/:id/comments?page=1&page_size=10
func (h *CommentHandler) List(c *gin.Context) {
	articleID, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	page, pageSize := normalizePage(
		queryInt(c, "page", 1),
		queryInt(c, "page_size", 10),
	)
	comments, total, err := h.svc.List(c.Request.Context(), articleID, page, pageSize)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, gin.H{
		"items":     comments,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Delete DELETE /api/v1/comments/:id
func (h *CommentHandler) Delete(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	if err := h.svc.Delete(c.Request.Context(), middleware.Operator(c), id); err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, nil)
}
