package service

import (
	"context"
	"strings"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
	"igoblog/internal/repository"
)

// categoryService 是 CategoryService 的实现。
type categoryService struct {
	repo repository.CategoryRepository
}

// NewCategoryService 构造分类服务。
func NewCategoryService(repo repository.CategoryRepository) CategoryService {
	return &categoryService{repo: repo}
}

func (s *categoryService) Create(ctx context.Context, name string) (*model.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 50 {
		return nil, apperr.BadRequest("分类名不能为空且不超过 50 字符")
	}
	c := &model.Category{Name: name}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *categoryService) Get(ctx context.Context, id int64) (*model.Category, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *categoryService) List(ctx context.Context) ([]model.Category, error) {
	return s.repo.List(ctx)
}

func (s *categoryService) Update(ctx context.Context, id int64, name string) (*model.Category, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 50 {
		return nil, apperr.BadRequest("分类名不能为空且不超过 50 字符")
	}
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	c.Name = name
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// Delete 删除分类；分类下仍有文章时拒绝删除。
func (s *categoryService) Delete(ctx context.Context, id int64) error {
	has, err := s.repo.HasArticles(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return apperr.Conflict("该分类下仍有文章，无法删除")
	}
	return s.repo.Delete(ctx, id)
}
