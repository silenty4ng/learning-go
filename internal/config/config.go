// Package config 负责加载服务配置：环境变量优先，未设置时使用默认值。
package config

import (
	"os"
	"strconv"
	"time"
)

// Config 服务运行配置。
type Config struct {
	Port      string        // HTTP 监听端口
	DBPath    string        // SQLite 数据库文件路径
	JWTSecret string        // JWT 签名密钥
	JWTExpire time.Duration // Token 有效期

	RateLimit float64 // 令牌桶速率（每秒请求数）
	RateBurst int     // 令牌桶容量（突发上限）

	ViewFlushInterval time.Duration // 浏览量批量落库间隔
	ViewBatchSize     int           // 浏览量缓冲批大小（也是 channel 容量）
}

// Load 从环境变量加载配置，缺省值适合本地开发。
func Load() *Config {
	return &Config{
		Port:              getEnv("BLOG_PORT", "8080"),
		DBPath:            getEnv("BLOG_DB_PATH", "blog.db"),
		JWTSecret:         getEnv("BLOG_JWT_SECRET", "dev-secret-change-me"),
		JWTExpire:         getEnvDuration("BLOG_JWT_EXPIRE", 24*time.Hour),
		RateLimit:         getEnvFloat("BLOG_RATE_LIMIT", 50),
		RateBurst:         getEnvInt("BLOG_RATE_BURST", 100),
		ViewFlushInterval: getEnvDuration("BLOG_VIEW_FLUSH_INTERVAL", 5*time.Second),
		ViewBatchSize:     getEnvInt("BLOG_VIEW_BATCH_SIZE", 100),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
