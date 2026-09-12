// Package service 实现业务逻辑层。
//
// 设计要点：
//   - 每个 Service 定义为接口，handler 只依赖接口，便于替换与 mock；
//   - Service 依赖 repository 接口而非具体实现（依赖倒置）；
//   - 业务校验（权限、存在性、参数语义）集中在本层完成。
package service

import (
	"context"

	"igoblog/internal/model"
)

// ArticleInput 创建/更新文章的输入参数。
type ArticleInput struct {
	Title      string
	Content    string
	CategoryID int64
	TagIDs     []int64
}

// UserService 用户注册/登录业务接口。
type UserService interface {
	Register(ctx context.Context, username, password string) (*model.User, error)
	// Login 校验凭据并签发 JWT，成功返回 token。
	Login(ctx context.Context, username, password string) (string, error)
	// UpdateProfile 修改当前用户的个人资料（昵称；空字符串表示清除）。
	UpdateProfile(ctx context.Context, userID int64, nickname string) (*model.User, error)
	// Profile 查询用户完整资料（/users/me）。
	Profile(ctx context.Context, userID int64) (*model.User, error)
}

// ArticleService 文章业务接口。
// 需要权限判断的方法接收 model.Operator（操作者身份）：
// 普通用户只能操作自己的文章，管理员（超管）可操作任何文章。
type ArticleService interface {
	Create(ctx context.Context, userID int64, in ArticleInput) (*model.Article, error)
	Get(ctx context.Context, id int64) (*model.Article, error)
	Update(ctx context.Context, op model.Operator, id int64, in ArticleInput) (*model.Article, error)
	Delete(ctx context.Context, op model.Operator, id int64) error
	List(ctx context.Context, f model.ArticleFilter) ([]model.Article, int64, error)
}

// CategoryService 分类业务接口。
type CategoryService interface {
	Create(ctx context.Context, name string) (*model.Category, error)
	Get(ctx context.Context, id int64) (*model.Category, error)
	List(ctx context.Context) ([]model.Category, error)
	Update(ctx context.Context, id int64, name string) (*model.Category, error)
	Delete(ctx context.Context, id int64) error
}

// TagService 标签业务接口。
type TagService interface {
	Create(ctx context.Context, name string) (*model.Tag, error)
	Get(ctx context.Context, id int64) (*model.Tag, error)
	List(ctx context.Context) ([]model.Tag, error)
	Delete(ctx context.Context, id int64) error
}

// CommentService 评论业务接口。
type CommentService interface {
	Create(ctx context.Context, userID, articleID int64, content string) (*model.Comment, error)
	List(ctx context.Context, articleID int64, page, pageSize int) ([]model.Comment, int64, error)
	// Delete 删除评论：评论者本人、文章作者或管理员可操作。
	Delete(ctx context.Context, op model.Operator, commentID int64) error
}

// AdminService 管理员业务接口：用户管理与站点设置。
type AdminService interface {
	// ListUsers 分页列出全部用户。
	ListUsers(ctx context.Context, page, pageSize int) ([]model.User, int64, error)
	// UpdateUserRole 修改用户角色；不允许自我降级，且站点至少保留一名管理员。
	UpdateUserRole(ctx context.Context, operatorID, targetID int64, role string) (*model.User, error)
	// DeleteUser 删除用户及其全部内容；不允许删除自己，且至少保留一名管理员。
	DeleteUser(ctx context.Context, operatorID, targetID int64) error
	// GetSiteSettings 读取站点设置。
	GetSiteSettings(ctx context.Context) (model.SiteSettings, error)
	// UpdateSiteSettings 更新站点设置，返回更新后的值。
	UpdateSiteSettings(ctx context.Context, s model.SiteSettings) (model.SiteSettings, error)
}
