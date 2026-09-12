package service

import (
	"context"
	"strings"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
	"igoblog/internal/repository"
)

// ViewIncrer 浏览量统计的投递抽象：业务侧只依赖该接口，
// 具体实现（异步批量写入器）见 view_counter.go。
type ViewIncrer interface {
	// Add 非阻塞地投递一次浏览行为。
	Add(articleID int64)
}

// articleService 是 ArticleService 的实现。
type articleService struct {
	articles   repository.ArticleRepository
	categories repository.CategoryRepository
	tags       repository.TagRepository
	view       ViewIncrer
}

// NewArticleService 构造文章服务。
func NewArticleService(
	a repository.ArticleRepository,
	c repository.CategoryRepository,
	t repository.TagRepository,
	v ViewIncrer,
) ArticleService {
	return &articleService{articles: a, categories: c, tags: t, view: v}
}

func (s *articleService) Create(ctx context.Context, userID int64, in ArticleInput) (*model.Article, error) {
	if err := s.validateInput(ctx, in); err != nil {
		return nil, err
	}
	a := &model.Article{
		Title:      in.Title,
		Content:    in.Content,
		AuthorID:   userID,
		CategoryID: in.CategoryID,
	}
	if err := s.articles.Create(ctx, a); err != nil {
		return nil, err
	}
	if err := s.tags.SetArticleTags(ctx, a.ID, in.TagIDs); err != nil {
		return nil, err
	}
	// 回读以填充联表展示字段（作者名、分类名、标签）
	return s.articles.GetByID(ctx, a.ID)
}

// Get 查看文章详情，同时向 ViewCounter 非阻塞投递一次浏览量（异步落库）。
func (s *articleService) Get(ctx context.Context, id int64) (*model.Article, error) {
	a, err := s.articles.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.view.Add(id)
	return a, nil
}

func (s *articleService) Update(ctx context.Context, op model.Operator, id int64, in ArticleInput) (*model.Article, error) {
	a, err := s.articles.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// 作者本人可改；管理员（超管）可管理任何文章
	if a.AuthorID != op.ID && !op.IsAdmin() {
		return nil, apperr.Forbidden("只有作者本人或管理员可以修改文章")
	}
	if err := s.validateInput(ctx, in); err != nil {
		return nil, err
	}
	a.Title, a.Content, a.CategoryID = in.Title, in.Content, in.CategoryID
	if err := s.articles.Update(ctx, a); err != nil {
		return nil, err
	}
	if err := s.tags.SetArticleTags(ctx, a.ID, in.TagIDs); err != nil {
		return nil, err
	}
	return s.articles.GetByID(ctx, a.ID)
}

func (s *articleService) Delete(ctx context.Context, op model.Operator, id int64) error {
	a, err := s.articles.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a.AuthorID != op.ID && !op.IsAdmin() {
		return apperr.Forbidden("只有作者本人或管理员可以删除文章")
	}
	return s.articles.Delete(ctx, id)
}

func (s *articleService) List(ctx context.Context, f model.ArticleFilter) ([]model.Article, int64, error) {
	return s.articles.List(ctx, f)
}

// validateInput 校验输入语义并确认分类与标签存在。
func (s *articleService) validateInput(ctx context.Context, in ArticleInput) error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || len(in.Title) > 200 {
		return apperr.BadRequest("标题不能为空且不超过 200 字符")
	}
	if strings.TrimSpace(in.Content) == "" {
		return apperr.BadRequest("正文不能为空")
	}
	if _, err := s.categories.GetByID(ctx, in.CategoryID); err != nil {
		return apperr.BadRequest("指定的分类不存在")
	}
	if len(in.TagIDs) > 20 {
		return apperr.BadRequest("标签数量不能超过 20 个")
	}
	for _, id := range in.TagIDs {
		if _, err := s.tags.GetByID(ctx, id); err != nil {
			return apperr.BadRequest("标签 %d 不存在", id)
		}
	}
	return nil
}
