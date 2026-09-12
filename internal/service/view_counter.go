package service

import (
	"context"
	"log"
	"sync"
	"time"

	"igoblog/internal/repository"
)

// ViewCounter 是 ViewIncrer 的并发实现：异步批量浏览量写入器。
//
// 并发设计说明（Go 并发原语的标准用法）：
//   - 带缓冲 channel 作为生产者/消费者队列，Add 非阻塞投递，
//     缓冲满时直接丢弃（浏览统计允许有损，绝不拖慢请求路径）；
//   - 后台 goroutine 以「满批 or 定时」双触发条件批量落库，
//     将高频单条 UPDATE 合并为低频批量事务；
//   - Close 幂等：关闭 channel 触发消费者排空缓冲后退出，
//     WaitGroup 保证关停时数据不丢。
type ViewCounter struct {
	ch         chan int64
	repo       repository.ArticleRepository
	flushEvery time.Duration
	batch      int
	wg         sync.WaitGroup
	closeOnce  sync.Once
}

// NewViewCounter 创建并启动后台写入 goroutine。
// batch 同时决定 channel 缓冲容量与批量落库阈值。
func NewViewCounter(repo repository.ArticleRepository, flushEvery time.Duration, batch int) *ViewCounter {
	if batch <= 0 {
		batch = 100
	}
	if flushEvery <= 0 {
		flushEvery = 5 * time.Second
	}
	vc := &ViewCounter{
		ch:         make(chan int64, batch),
		repo:       repo,
		flushEvery: flushEvery,
		batch:      batch,
	}
	vc.wg.Add(1)
	go vc.run()
	return vc
}

// Add 非阻塞投递一次浏览量；缓冲满时丢弃（select + default 的典型用法）。
func (vc *ViewCounter) Add(articleID int64) {
	select {
	case vc.ch <- articleID:
	default:
		// 缓冲已满：丢弃本次统计，保证读接口不被写压力阻塞
	}
}

// run 是后台消费者：汇总 channel 中的浏览量并周期性落库。
func (vc *ViewCounter) run() {
	defer vc.wg.Done()
	ticker := time.NewTicker(vc.flushEvery)
	defer ticker.Stop()

	pending := make(map[int64]int64, vc.batch)
	for {
		select {
		case id, ok := <-vc.ch:
			if !ok {
				// channel 已关闭：冲刷剩余数据后退出
				vc.flush(pending)
				return
			}
			pending[id]++
			if len(pending) >= vc.batch {
				vc.flush(pending)
			}
		case <-ticker.C:
			vc.flush(pending)
		}
	}
}

// flush 将汇总结果批量写入数据库并清空缓冲。
func (vc *ViewCounter) flush(pending map[int64]int64) {
	if len(pending) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := vc.repo.IncrViewCounts(ctx, pending); err != nil {
		// 统计数据允许有损：记录日志，不中断服务
		log.Printf("[viewcounter] flush %d items failed: %v", len(pending), err)
	}
	clear(pending)
}

// Close 关停写入器：关闭 channel，等待后台 goroutine 排空退出。幂等。
func (vc *ViewCounter) Close() {
	vc.closeOnce.Do(func() {
		close(vc.ch)
		vc.wg.Wait()
	})
}
