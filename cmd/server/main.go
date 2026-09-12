// cmd/server/main.go 是服务入口：
// 完成手工依赖注入（无 DI 框架，贴合 Go 惯例），并实现优雅关停。
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"igoblog/internal/config"
	"igoblog/internal/database"
	"igoblog/internal/handler"
	"igoblog/internal/middleware"
	"igoblog/internal/repository"
	"igoblog/internal/service"
)

func main() {
	cfg := config.Load()

	// 1. 数据库
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// 1.5 首次建库时创建默认管理员，账号密码打印到控制台
	adminUser, adminPass, created, err := database.EnsureAdmin(db)
	if err != nil {
		log.Fatalf("ensure admin: %v", err)
	}
	if created {
		log.Printf("========================================================")
		log.Printf("  已创建默认管理员账号（首次启动）")
		log.Printf("  用户名: %s", adminUser)
		log.Printf("  密  码: %s", adminPass)
		log.Printf("  请登录后妥善保管，密码仅本次显示")
		log.Printf("========================================================")
	}

	// 2. repository 层（数据访问）
	userRepo := repository.NewUserRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	tagRepo := repository.NewTagRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	// 3. 并发组件：浏览量异步批量写入器（关停时须冲刷）
	viewCounter := service.NewViewCounter(articleRepo, cfg.ViewFlushInterval, cfg.ViewBatchSize)

	// 4. service 层（业务逻辑，依赖 repository 接口）
	deps := handler.Deps{
		User:     service.NewUserService(userRepo, settingsRepo, cfg.JWTSecret, cfg.JWTExpire),
		Article:  service.NewArticleService(articleRepo, categoryRepo, tagRepo, viewCounter),
		Category: service.NewCategoryService(categoryRepo),
		Tag:      service.NewTagService(tagRepo),
		Comment:  service.NewCommentService(commentRepo, articleRepo),
		Admin:    service.NewAdminService(userRepo, settingsRepo),
		UserRepo: userRepo,
	}

	// 5. HTTP 层
	gin.SetMode(gin.ReleaseMode)
	router := handler.NewRouter(deps, cfg.JWTSecret, middleware.NewRateLimiter(cfg.RateLimit, cfg.RateBurst))
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// 6. 启动并监听退出信号
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("iGoBlog listening on :%s (db=%s)", cfg.Port, cfg.DBPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down ...")

	// 优雅关停：先停止接收新请求，再冲刷浏览量缓冲，最后释放资源
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	viewCounter.Close()
	log.Println("bye")
}
