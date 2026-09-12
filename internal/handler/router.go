package handler

import (
	"path/filepath"

	"github.com/gin-gonic/gin"

	"igoblog/internal/middleware"
	"igoblog/internal/repository"
	"igoblog/internal/service"
)

// Deps 聚合 handler 层依赖的业务服务接口（依赖注入的载体）。
type Deps struct {
	User     service.UserService
	Article  service.ArticleService
	Category service.CategoryService
	Tag      service.TagService
	Comment  service.CommentService
	Admin    service.AdminService
	// UserRepo 供认证中间件回查用户最新角色与存在性。
	UserRepo repository.UserRepository
}

// handlers 聚合各资源处理器。
type handlers struct {
	user     *UserHandler
	article  *ArticleHandler
	category *CategoryHandler
	tag      *TagHandler
	comment  *CommentHandler
	admin    *AdminHandler
}

// NewRouter 装配路由、中间件与依赖注入，返回可运行的 Gin Engine。
func NewRouter(d Deps, jwtSecret string, rl *middleware.RateLimiter) *gin.Engine {
	h := &handlers{
		user:     NewUserHandler(d.User),
		article:  NewArticleHandler(d.Article),
		category: NewCategoryHandler(d.Category),
		tag:      NewTagHandler(d.Tag),
		comment:  NewCommentHandler(d.Comment),
		admin:    NewAdminHandler(d.Admin),
	}

	r := gin.New()
	r.Use(gin.Recovery(), rl.Middleware())
	auth := middleware.NewAuth(jwtSecret, d.UserRepo).Require()

	api := r.Group("/api/v1")
	{
		// 认证
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", h.user.Register)
			authGroup.POST("/login", h.user.Login)
		}
		api.GET("/users/me", auth, h.user.Me)
		api.PUT("/users/me", auth, h.user.UpdateNickname)

		// 文章
		arts := api.Group("/articles")
		{
			arts.GET("", h.article.List)
			arts.GET("/:id", h.article.Get)
			arts.POST("", auth, h.article.Create)
			arts.PUT("/:id", auth, h.article.Update)
			arts.DELETE("/:id", auth, h.article.Delete)

			// 评论（挂在文章资源之下）
			arts.GET("/:id/comments", h.comment.List)
			arts.POST("/:id/comments", auth, h.comment.Create)
		}
		api.DELETE("/comments/:id", auth, h.comment.Delete)

		// 站点设置（公开只读：前台需要感知注册开关与站点信息）
		api.GET("/site/settings", h.admin.GetSiteSettings)

		// 管理员端点：认证 + 管理员守卫双重保护
		adminGroup := api.Group("/admin", auth, middleware.RequireAdmin())
		{
			adminGroup.GET("/users", h.admin.ListUsers)
			adminGroup.PUT("/users/:id/role", h.admin.UpdateUserRole)
			adminGroup.DELETE("/users/:id", h.admin.DeleteUser)
			adminGroup.GET("/settings", h.admin.GetSiteSettings)
			adminGroup.PUT("/settings", h.admin.UpdateSiteSettings)
		}

		// 分类
		cats := api.Group("/categories")
		{
			cats.GET("", h.category.List)
			cats.GET("/:id", h.category.Get)
			cats.POST("", auth, h.category.Create)
			cats.PUT("/:id", auth, h.category.Update)
			cats.DELETE("/:id", auth, h.category.Delete)
		}

		// 标签
		tags := api.Group("/tags")
		{
			tags.GET("", h.tag.List)
			tags.GET("/:id", h.tag.Get)
			tags.POST("", auth, h.tag.Create)
			tags.DELETE("/:id", auth, h.tag.Delete)
		}
	}

	// 静态站点托管：web/ 目录下的原生前端（与 API 同端口同源）
	// 运行目录须为项目根目录（web/ 相对路径）。
	const webDir = "web"
	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(webDir, "index.html"))
	})
	r.GET("/admin", func(c *gin.Context) {
		c.File(filepath.Join(webDir, "admin.html"))
	})
	r.Static("/assets", filepath.Join(webDir, "assets"))

	// 健康检查
	r.GET("/healthz", func(c *gin.Context) { middleware.OK(c, gin.H{"status": "up"}) })

	return r
}
