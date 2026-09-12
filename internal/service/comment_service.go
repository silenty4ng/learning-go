package service

import (
	"context"
	"strings"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
	"igoblog/internal/repository"
)

// commentService 是 CommentService 的实现。
type commentService struct {
	comments repository.CommentRepository
	articles repository.ArticleRepository
}

// NewCommentService 构造评论服务。
func NewCommentService(c repository.CommentRepository, a repository.ArticleRepository) CommentService {
	return &commentService{comments: c, articles: a}
}

func (s *commentService) Create(ctx context.Context, userID, articleID int64, content string) (*model.Comment, error) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > 1000 {
		return nil, apperr.BadRequest("评论内容不能为空且不超过 1000 字符")
	}
	// 确认文章存在（不存在时透传 404）
	if _, err := s.articles.GetByID(ctx, articleID); err != nil {
		return nil, err
	}
	c := &model.Comment{ArticleID: articleID, UserID: userID, Content: content}
	if err := s.comments.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *commentService) List(ctx context.Context, articleID int64, page, pageSize int) ([]model.Comment, int64, error) {
	// 确认文章存在
	if _, err := s.articles.GetByID(ctx, articleID); err != nil {
		return nil, 0, err
	}
	return s.comments.ListByArticle(ctx, articleID, page, pageSize)
}

// Delete 删除评论：评论者本人、文章作者或管理员均有权限。
func (s *commentService) Delete(ctx context.Context, op model.Operator, commentID int64) error {
	c, err := s.comments.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c.UserID == op.ID || op.IsAdmin() {
		return s.comments.Delete(ctx, commentID)
	}
	// 非评论者：仅当其为文章作者时可删
	a, err := s.articles.GetByID(ctx, c.ArticleID)
	if err != nil {
		return err
	}
	if a.AuthorID == op.ID {
		return s.comments.Delete(ctx, commentID)
	}
	return apperr.Forbidden("只有评论者本人、文章作者或管理员可以删除评论")
}
