// Package repository 定义数据访问层接口与 SQLite 实现。
//
// 核心设计：service 层只依赖这里的接口而非具体实现，
// 单元测试时可用内存 mock 替换，从而实现业务逻辑与存储的彻底解耦。
package repository

import (
	"context"

	"igoblog/internal/model"
)

// UserRepository 用户数据访问接口。
type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	// List 分页列出用户，返回当前页数据与总数。
	List(ctx context.Context, page, pageSize int) ([]model.User, int64, error)
	// UpdateRole 修改用户角色。
	UpdateRole(ctx context.Context, id int64, role string) error
	// UpdateNickname 修改用户昵称（空字符串表示清除昵称）。
	UpdateNickname(ctx context.Context, id int64, nickname string) error
	// Delete 删除用户及其全部文章与评论（事务内级联）。
	Delete(ctx context.Context, id int64) error
	// CountAdmins 统计管理员数量（保证站点至少保留一名管理员）。
	CountAdmins(ctx context.Context) (int, error)
}

// SettingsRepository 站点设置（键值对）数据访问接口。
type SettingsRepository interface {
	GetAll(ctx context.Context) (map[string]string, error)
	Set(ctx context.Context, key, value string) error
}

// ArticleRepository 文章数据访问接口。
type ArticleRepository interface {
	Create(ctx context.Context, a *model.Article) error
	GetByID(ctx context.Context, id int64) (*model.Article, error)
	Update(ctx context.Context, a *model.Article) error
	Delete(ctx context.Context, id int64) error
	// List 按条件分页查询，返回当前页数据与总数。
	List(ctx context.Context, f model.ArticleFilter) ([]model.Article, int64, error)
	// IncrViewCounts 批量累加浏览量，键为文章 ID，值为增量。
	IncrViewCounts(ctx context.Context, counts map[int64]int64) error
}

// CategoryRepository 分类数据访问接口。
type CategoryRepository interface {
	Create(ctx context.Context, c *model.Category) error
	GetByID(ctx context.Context, id int64) (*model.Category, error)
	List(ctx context.Context) ([]model.Category, error)
	Update(ctx context.Context, c *model.Category) error
	Delete(ctx context.Context, id int64) error
	// HasArticles 判断分类下是否还有文章，删除前校验。
	HasArticles(ctx context.Context, id int64) (bool, error)
}

// TagRepository 标签数据访问接口（含文章-标签多对多绑定）。
type TagRepository interface {
	Create(ctx context.Context, t *model.Tag) error
	GetByID(ctx context.Context, id int64) (*model.Tag, error)
	List(ctx context.Context) ([]model.Tag, error)
	Delete(ctx context.Context, id int64) error
	// SetArticleTags 全量重设文章的标签集合（先删后插，事务内完成）。
	SetArticleTags(ctx context.Context, articleID int64, tagIDs []int64) error
	// ListByArticle 查询文章的全部标签。
	ListByArticle(ctx context.Context, articleID int64) ([]model.Tag, error)
}

// CommentRepository 评论数据访问接口。
type CommentRepository interface {
	Create(ctx context.Context, c *model.Comment) error
	GetByID(ctx context.Context, id int64) (*model.Comment, error)
	// ListByArticle 按文章分页查询评论，返回当前页数据与总数。
	ListByArticle(ctx context.Context, articleID int64, page, pageSize int) ([]model.Comment, int64, error)
	Delete(ctx context.Context, id int64) error
}
