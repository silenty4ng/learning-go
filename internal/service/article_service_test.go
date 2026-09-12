package service

import (
	"context"
	"errors"
	"testing"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
)

// ---------- mock repository：隔离数据库，专注业务逻辑 ----------

type mockArticleRepo struct {
	articles map[int64]*model.Article
	nextID   int64
}

func newMockArticleRepo() *mockArticleRepo {
	return &mockArticleRepo{articles: map[int64]*model.Article{}, nextID: 1}
}

func (m *mockArticleRepo) Create(_ context.Context, a *model.Article) error {
	a.ID = m.nextID
	m.nextID++
	cp := *a
	m.articles[a.ID] = &cp
	return nil
}

func (m *mockArticleRepo) GetByID(_ context.Context, id int64) (*model.Article, error) {
	if a, ok := m.articles[id]; ok {
		cp := *a
		return &cp, nil
	}
	return nil, apperr.NotFound("文章")
}

func (m *mockArticleRepo) Update(_ context.Context, a *model.Article) error {
	if _, ok := m.articles[a.ID]; !ok {
		return apperr.NotFound("文章")
	}
	cp := *a
	m.articles[a.ID] = &cp
	return nil
}

func (m *mockArticleRepo) Delete(_ context.Context, id int64) error {
	if _, ok := m.articles[id]; !ok {
		return apperr.NotFound("文章")
	}
	delete(m.articles, id)
	return nil
}

func (m *mockArticleRepo) List(_ context.Context, f model.ArticleFilter) ([]model.Article, int64, error) {
	return nil, 0, errors.New("not implemented in mock")
}

func (m *mockArticleRepo) IncrViewCounts(_ context.Context, counts map[int64]int64) error {
	return nil
}

type mockCategoryRepo struct{ exists map[int64]string }

func (m *mockCategoryRepo) Create(_ context.Context, c *model.Category) error { return nil }
func (m *mockCategoryRepo) GetByID(_ context.Context, id int64) (*model.Category, error) {
	name, ok := m.exists[id]
	if !ok {
		return nil, apperr.NotFound("分类")
	}
	return &model.Category{ID: id, Name: name}, nil
}
func (m *mockCategoryRepo) List(_ context.Context) ([]model.Category, error) { return nil, nil }
func (m *mockCategoryRepo) Update(_ context.Context, c *model.Category) error { return nil }
func (m *mockCategoryRepo) Delete(_ context.Context, id int64) error          { return nil }
func (m *mockCategoryRepo) HasArticles(_ context.Context, id int64) (bool, error) {
	return false, nil
}

type mockTagRepo struct{ exists map[int64]string }

func (m *mockTagRepo) Create(_ context.Context, t *model.Tag) error { return nil }
func (m *mockTagRepo) GetByID(_ context.Context, id int64) (*model.Tag, error) {
	name, ok := m.exists[id]
	if !ok {
		return nil, apperr.NotFound("标签")
	}
	return &model.Tag{ID: id, Name: name}, nil
}
func (m *mockTagRepo) List(_ context.Context) ([]model.Tag, error)          { return nil, nil }
func (m *mockTagRepo) Delete(_ context.Context, id int64) error             { return nil }
func (m *mockTagRepo) SetArticleTags(_ context.Context, _ int64, _ []int64) error { return nil }
func (m *mockTagRepo) ListByArticle(_ context.Context, _ int64) ([]model.Tag, error) {
	return nil, nil
}

// mockViewIncrer 记录浏览量投递，验证 ArticleService.Get 的并发投递行为。
type mockViewIncrer struct{ hits map[int64]int }

func (m *mockViewIncrer) Add(id int64) { m.hits[id]++ }

// ---------- 被测辅助 ----------

func newTestArticleService() (ArticleService, *mockViewIncrer) {
	arts := newMockArticleRepo()
	cats := &mockCategoryRepo{exists: map[int64]string{1: "Tech"}}
	tags := &mockTagRepo{exists: map[int64]string{1: "go"}}
	view := &mockViewIncrer{hits: map[int64]int{}}
	svc := NewArticleService(arts, cats, tags, view)
	return svc, view
}

func mustCreate(t *testing.T, svc ArticleService, userID int64, in ArticleInput) *model.Article {
	t.Helper()
	a, err := svc.Create(context.Background(), userID, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return a
}

func validInput() ArticleInput {
	return ArticleInput{Title: "hello", Content: "world", CategoryID: 1, TagIDs: []int64{1}}
}

// opUser / opAdmin 构造操作者身份。
func opUser(id int64) model.Operator   { return model.Operator{ID: id, Role: model.RoleUser} }
func opAdmin(id int64) model.Operator  { return model.Operator{ID: id, Role: model.RoleAdmin} }

// ---------- 测试用例 ----------

func TestArticleCreateValidation(t *testing.T) {
	svc, _ := newTestArticleService()
	ctx := context.Background()

	if _, err := svc.Create(ctx, 1, ArticleInput{Title: "", Content: "c", CategoryID: 1}); err == nil {
		t.Fatal("empty title should fail")
	}
	if _, err := svc.Create(ctx, 1, ArticleInput{Title: "t", Content: "c", CategoryID: 999}); err == nil {
		t.Fatal("missing category should fail")
	}
	if _, err := svc.Create(ctx, 1, ArticleInput{Title: "t", Content: "c", CategoryID: 1, TagIDs: []int64{999}}); err == nil {
		t.Fatal("missing tag should fail")
	}
}

func TestArticleAuthorPermission(t *testing.T) {
	svc, _ := newTestArticleService()
	ctx := context.Background()
	a := mustCreate(t, svc, 100, validInput())

	// 非作者更新 -> 403
	if _, err := svc.Update(ctx, opUser(200), a.ID, validInput()); err == nil {
		t.Fatal("update by non-author should fail")
	}
	// 作者更新 -> 成功
	updated, err := svc.Update(ctx, opUser(100), a.ID, ArticleInput{Title: "new", Content: "new content", CategoryID: 1})
	if err != nil {
		t.Fatalf("update by author error = %v", err)
	}
	if updated.Title != "new" {
		t.Fatalf("Title = %q, want new", updated.Title)
	}
	// 非作者删除 -> 403
	var ae *apperr.AppError
	err = svc.Delete(ctx, opUser(200), a.ID)
	if !errors.As(err, &ae) || ae.Status != 403 {
		t.Fatalf("want 403 Forbidden, got %v", err)
	}
	// 管理员可删除任何人的文章 -> 成功
	if err := svc.Delete(ctx, opAdmin(999), a.ID); err != nil {
		t.Fatalf("delete by admin error = %v", err)
	}
	if _, err := svc.Get(ctx, a.ID); err == nil {
		t.Fatal("article should be gone after delete")
	}
}

func TestArticleAdminCanUpdateAny(t *testing.T) {
	svc, _ := newTestArticleService()
	ctx := context.Background()
	a := mustCreate(t, svc, 100, validInput())

	// 管理员可修改他人文章
	updated, err := svc.Update(ctx, opAdmin(999), a.ID, ArticleInput{Title: "admin edit", Content: "x", CategoryID: 1})
	if err != nil {
		t.Fatalf("update by admin error = %v", err)
	}
	if updated.Title != "admin edit" {
		t.Fatalf("Title = %q", updated.Title)
	}
}

func TestArticleGetDeliversViewCount(t *testing.T) {
	svc, view := newTestArticleService()
	ctx := context.Background()
	a := mustCreate(t, svc, 1, validInput())

	for i := 0; i < 3; i++ {
		if _, err := svc.Get(ctx, a.ID); err != nil {
			t.Fatalf("Get() error = %v", err)
		}
	}
	if view.hits[a.ID] != 3 {
		t.Fatalf("view hits = %d, want 3", view.hits[a.ID])
	}
}
