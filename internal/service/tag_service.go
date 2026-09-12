package service

import (
	"context"
	"strings"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
	"igoblog/internal/repository"
)

// tagService 是 TagService 的实现。
type tagService struct {
	repo repository.TagRepository
}

// NewTagService 构造标签服务。
func NewTagService(repo repository.TagRepository) TagService {
	return &tagService{repo: repo}
}

func (s *tagService) Create(ctx context.Context, name string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 30 {
		return nil, apperr.BadRequest("标签名不能为空且不超过 30 字符")
	}
	t := &model.Tag{Name: name}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *tagService) Get(ctx context.Context, id int64) (*model.Tag, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *tagService) List(ctx context.Context) ([]model.Tag, error) {
	return s.repo.List(ctx)
}

func (s *tagService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
