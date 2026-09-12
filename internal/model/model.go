// Package model 定义领域模型与查询条件。
// 模型不依赖任何框架，是各层之间共享的数据结构（贫血模型 + 各层行为分离）。
package model

import "time"

// 用户角色。
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// settings 表的键名。
const (
	SettingSiteTitle           = "site_title"
	SettingSiteDescription     = "site_description"
	SettingRegistrationEnabled = "registration_enabled"
)

// User 用户。密码仅存 bcrypt 哈希，序列化时隐藏。
// Nickname 为可选昵称：为空时各展示处回退显示用户名。
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Nickname     string    `json:"nickname"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsAdmin 判断用户是否管理员。
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// Operator 操作者身份：service 层权限判断的统一入参，
// 由 handler 从认证中间件注入的用户信息构造。
type Operator struct {
	ID   int64
	Role string
}

// IsAdmin 判断操作者是否管理员（超管可管理所有文章与评论）。
func (o Operator) IsAdmin() bool { return o.Role == RoleAdmin }

// SiteSettings 站点设置（settings 表键值对的强类型视图）。
type SiteSettings struct {
	SiteTitle           string `json:"site_title"`
	SiteDescription     string `json:"site_description"`
	RegistrationEnabled bool   `json:"registration_enabled"`
}

// Category 文章分类（一对多）。
type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Tag 文章标签（与文章多对多）。
type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Article 文章。
// Author/Category/Tags 为查询时联表填充的展示字段，写入时不依赖它们。
type Article struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	AuthorID   int64     `json:"author_id"`
	Author     string    `json:"author,omitempty"`
	CategoryID int64     `json:"category_id"`
	Category   string    `json:"category,omitempty"`
	Tags       []Tag     `json:"tags,omitempty"`
	ViewCount  int64     `json:"view_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ArticleFilter 文章列表查询条件（分页 + 筛选 + 搜索）。
type ArticleFilter struct {
	CategoryID int64  // 按分类筛选，0 表示不过滤
	TagID      int64  // 按标签筛选，0 表示不过滤
	Keyword    string // 标题/内容模糊搜索
	Page       int    // 页码，从 1 开始
	PageSize   int    // 每页条数
}

// Offset 计算 SQL LIMIT/OFFSET 的偏移量。
func (f ArticleFilter) Offset() int { return (f.Page - 1) * f.PageSize }

// Comment 评论。
type Comment struct {
	ID        int64     `json:"id"`
	ArticleID int64     `json:"article_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
