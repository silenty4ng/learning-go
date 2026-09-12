package handler

import (
	"github.com/gin-gonic/gin"

	"igoblog/internal/apperr"
	"igoblog/internal/middleware"
	"igoblog/internal/model"
	"igoblog/internal/service"
)

// ArticleHandler 文章相关 HTTP 处理器。
type ArticleHandler struct{ svc service.ArticleService }

// NewArticleHandler 构造 ArticleHandler。
func NewArticleHandler(svc service.ArticleService) *ArticleHandler { return &ArticleHandler{svc: svc} }

type articleRequest struct {
	Title      string  `json:"title" binding:"required,max=200"`
	Content    string  `json:"content" binding:"required"`
	CategoryID int64   `json:"category_id" binding:"required,gt=0"`
	TagIDs     []int64 `json:"tag_ids"`
}

// Create POST /api/v1/articles
func (h *ArticleHandler) Create(c *gin.Context) {
	var req articleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	a, err := h.svc.Create(c.Request.Context(), middleware.UserID(c), service.ArticleInput{
		Title:      req.Title,
		Content:    req.Content,
		CategoryID: req.CategoryID,
		TagIDs:     req.TagIDs,
	})
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.Created(c, a)
}

// Get GET /api/v1/articles/:id
func (h *ArticleHandler) Get(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	a, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, a)
}

// Update PUT /api/v1/articles/:id
func (h *ArticleHandler) Update(c *gin.Context) {
	id, ok := paramInt(c, "id")
	if !ok {
		middleware.Fail(c, idError())
		return
	}
	var req articleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, apperr.BadRequest("参数错误：%v", err))
		return
	}
	a, err := h.svc.Update(c.Request.Context(), middleware.Operator(c), id, service.ArticleInput{
		Title:      req.Title,
		Content:    req.Content,
		CategoryID: req.CategoryID,
		TagIDs:     req.TagIDs,
	})
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, a)
}

// Delete DELETE /api/v1/articles/:id
func (h *ArticleHandler) Delete(c *gin.Context) {
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

// List GET /api/v1/articles?page=1&page_size=10&category_id=1&tag_id=2&keyword=go
func (h *ArticleHandler) List(c *gin.Context) {
	page, pageSize := normalizePage(
		queryInt(c, "page", 1),
		queryInt(c, "page_size", 10),
	)
	f := model.ArticleFilter{
		CategoryID: queryInt(c, "category_id", 0),
		TagID:      queryInt(c, "tag_id", 0),
		Keyword:    c.Query("keyword"),
		Page:       page,
		PageSize:   pageSize,
	}
	articles, total, err := h.svc.List(c.Request.Context(), f)
	if err != nil {
		middleware.Fail(c, err)
		return
	}
	middleware.OK(c, gin.H{
		"items":     articles,
		"total":     total,
		"page":      f.Page,
		"page_size": f.PageSize,
	})
}
