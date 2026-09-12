package repository

import (
	"context"
	"path/filepath"
	"testing"

	"igoblog/internal/database"
	"igoblog/internal/model"
)

// newTestDB 每个测试独享一个临时 SQLite 文件，互不干扰。
func newTestDB(t *testing.T) *testEnv {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &testEnv{
		db:         db,
		users:      NewUserRepository(db),
		articles:   NewArticleRepository(db),
		categories: NewCategoryRepository(db),
		tags:       NewTagRepository(db),
		comments:   NewCommentRepository(db),
	}
}

type testEnv struct {
	db         interface{ Close() error }
	users      UserRepository
	articles   ArticleRepository
	categories CategoryRepository
	tags       TagRepository
	comments   CommentRepository
}

// seed 创建基础用户/分类/标签，返回各资源 ID。
func (e *testEnv) seed(t *testing.T) (userID, categoryID, tagID int64) {
	t.Helper()
	ctx := context.Background()

	u := &model.User{Username: "alice", PasswordHash: "hash"}
	if err := e.users.Create(ctx, u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	c := &model.Category{Name: "Tech"}
	if err := e.categories.Create(ctx, c); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	tg := &model.Tag{Name: "go"}
	if err := e.tags.Create(ctx, tg); err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	return u.ID, c.ID, tg.ID
}

func TestArticleLifecycle(t *testing.T) {
	env := newTestDB(t)
	userID, catID, tagID := env.seed(t)
	ctx := context.Background()

	// 创建
	a := &model.Article{Title: "Hello Go", Content: "concurrency & interfaces", AuthorID: userID, CategoryID: catID}
	if err := env.articles.Create(ctx, a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := env.tags.SetArticleTags(ctx, a.ID, []int64{tagID}); err != nil {
		t.Fatalf("SetArticleTags: %v", err)
	}

	// 查询：联表字段与标签均正确
	got, err := env.articles.GetByID(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Author != "alice" || got.Category != "Tech" {
		t.Fatalf("joined fields wrong: %+v", got)
	}
	if len(got.Tags) != 1 || got.Tags[0].Name != "go" {
		t.Fatalf("tags wrong: %+v", got.Tags)
	}

	// 列表 + 关键词搜索
	list, total, err := env.articles.List(ctx, model.ArticleFilter{Keyword: "concurrency", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("keyword search wrong: total=%d list=%+v", total, list)
	}

	// 按标签筛选
	list, total, err = env.articles.List(ctx, model.ArticleFilter{TagID: tagID, Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Fatalf("tag filter wrong: err=%v total=%d", err, total)
	}

	// 按分类筛选
	list, total, err = env.articles.List(ctx, model.ArticleFilter{CategoryID: catID, Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Fatalf("category filter wrong: err=%v total=%d", err, total)
	}

	// 更新
	got.Title = "Hello Go v2"
	if err := env.articles.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	again, _ := env.articles.GetByID(ctx, a.ID)
	if again.Title != "Hello Go v2" {
		t.Fatalf("update not persisted: %+v", again)
	}

	// 删除
	if err := env.articles.Delete(ctx, a.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := env.articles.GetByID(ctx, a.ID); err == nil {
		t.Fatal("article should be gone")
	}
}

func TestIncrViewCounts(t *testing.T) {
	env := newTestDB(t)
	userID, catID, _ := env.seed(t)
	ctx := context.Background()

	a := &model.Article{Title: "t1", Content: "c1", AuthorID: userID, CategoryID: catID}
	if err := env.articles.Create(ctx, a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	b := &model.Article{Title: "t2", Content: "c2", AuthorID: userID, CategoryID: catID}
	if err := env.articles.Create(ctx, b); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := env.articles.IncrViewCounts(ctx, map[int64]int64{a.ID: 3, b.ID: 2}); err != nil {
		t.Fatalf("IncrViewCounts: %v", err)
	}
	// 多次累加
	if err := env.articles.IncrViewCounts(ctx, map[int64]int64{a.ID: 1}); err != nil {
		t.Fatalf("IncrViewCounts: %v", err)
	}

	got, _ := env.articles.GetByID(ctx, a.ID)
	if got.ViewCount != 4 {
		t.Fatalf("view count = %d, want 4", got.ViewCount)
	}
	got, _ = env.articles.GetByID(ctx, b.ID)
	if got.ViewCount != 2 {
		t.Fatalf("view count = %d, want 2", got.ViewCount)
	}
}

func TestUserUniqueConstraint(t *testing.T) {
	env := newTestDB(t)
	ctx := context.Background()

	u1 := &model.User{Username: "bob", PasswordHash: "h1"}
	if err := env.users.Create(ctx, u1); err != nil {
		t.Fatalf("Create: %v", err)
	}
	u2 := &model.User{Username: "bob", PasswordHash: "h2"}
	if err := env.users.Create(ctx, u2); err == nil {
		t.Fatal("duplicate username should fail")
	}
}

func TestCommentListPagination(t *testing.T) {
	env := newTestDB(t)
	userID, catID, _ := env.seed(t)
	ctx := context.Background()

	a := &model.Article{Title: "t", Content: "c", AuthorID: userID, CategoryID: catID}
	if err := env.articles.Create(ctx, a); err != nil {
		t.Fatalf("Create article: %v", err)
	}
	// 造 3 条评论
	for i := 0; i < 3; i++ {
		cm := &model.Comment{ArticleID: a.ID, UserID: userID, Content: "comment"}
		if err := env.comments.Create(ctx, cm); err != nil {
			t.Fatalf("Create comment: %v", err)
		}
	}

	// 第一页 2 条
	items, total, err := env.comments.ListByArticle(ctx, a.ID, 1, 2)
	if err != nil || total != 3 || len(items) != 2 {
		t.Fatalf("page1: err=%v total=%d len=%d", err, total, len(items))
	}
	if items[0].Username != "alice" {
		t.Fatalf("joined username = %q, want alice", items[0].Username)
	}
	// 第二页 1 条
	items, total, err = env.comments.ListByArticle(ctx, a.ID, 2, 2)
	if err != nil || total != 3 || len(items) != 1 {
		t.Fatalf("page2: err=%v total=%d len=%d", err, total, len(items))
	}
}
