package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"igoblog/internal/apperr"
)

// RateLimiter 基于令牌桶、按客户端 IP 维度的限流器。
//
// 并发设计：map 读写由 sync.Mutex 保护；
// 每个客户端独立一个 *rate.Limiter，互不影响。
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	limit    rate.Limit
	burst    int
}

// NewRateLimiter 构造限流器：limit 为每秒平均请求数，burst 为突发容量。
func NewRateLimiter(limit float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		limit:    rate.Limit(limit),
		burst:    burst,
	}
}

// get 取出（或惰性创建）指定 IP 的限流器。
func (rl *RateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	l, ok := rl.limiters[ip]
	if !ok {
		l = rate.NewLimiter(rl.limit, rl.burst)
		rl.limiters[ip] = l
	}
	return l
}

// Middleware 返回 Gin 限流中间件。
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.get(c.ClientIP()).Allow() {
			c.Header("Retry-After", "1")
			Fail(c, apperr.New(http.StatusTooManyRequests, apperr.CodeRateLimited, "请求过于频繁，请稍后再试"))
			c.Abort()
			return
		}
		c.Next()
	}
}
