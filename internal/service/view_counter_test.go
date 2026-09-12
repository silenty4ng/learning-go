package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeRepoForView 记录 IncrViewCounts 的调用结果。
// IncrViewCounts 由 ViewCounter 的后台 goroutine 调用，
// 测试主 goroutine 会并发读取结果，因此必须加锁保护。
type fakeRepoForView struct {
	mockArticleRepo
	mu     sync.Mutex
	got    map[int64]int64
	called int
}

func (f *fakeRepoForView) IncrViewCounts(_ context.Context, counts map[int64]int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called++
	for id, n := range counts {
		f.got[id] += n
	}
	return nil
}

// snapshot 返回当前累计值的副本，供测试安全读取。
func (f *fakeRepoForView) snapshot() map[int64]int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[int64]int64, len(f.got))
	for id, n := range f.got {
		out[id] = n
	}
	return out
}

// TestViewCounterFlushOnClose 验证：Add 后未满批，Close 时冲刷缓冲不丢数据。
func TestViewCounterFlushOnClose(t *testing.T) {
	repo := &fakeRepoForView{mockArticleRepo: *newMockArticleRepo(), got: map[int64]int64{}}
	vc := NewViewCounter(repo, time.Hour, 1000) // 满批阈值/间隔拉大，仅靠 Close 触发冲刷

	for i := 0; i < 5; i++ {
		vc.Add(1)
	}
	vc.Add(2)
	vc.Close() // 必须冲刷全部缓冲

	got := repo.snapshot()
	if len(got) == 0 {
		t.Fatal("flush was never called on Close")
	}
	if got[1] != 5 || got[2] != 1 {
		t.Fatalf("flushed counts = %v, want {1:5, 2:1}", got)
	}
}

// TestViewCounterFlushOnBatchFull 验证：去重后的文章数达到批大小阈值时自动落库。
func TestViewCounterFlushOnBatchFull(t *testing.T) {
	repo := &fakeRepoForView{mockArticleRepo: *newMockArticleRepo(), got: map[int64]int64{}}
	vc := NewViewCounter(repo, time.Hour, 10)

	// 投递 10 篇不同文章（去重后 len(pending) 达到阈值）
	for id := int64(1); id <= 10; id++ {
		vc.Add(id)
	}
	// 等待后台 goroutine 消费（带超时的轮询，避免测试抖动）
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && repo.snapshot()[1] != 1 {
		time.Sleep(5 * time.Millisecond)
	}
	got := repo.snapshot()
	for id := int64(1); id <= 10; id++ {
		if got[id] != 1 {
			t.Fatalf("article %d count = %d, want 1 (flushed=%v)", id, got[id], got)
		}
	}
	// 缓冲已排空，此时投递的第 11 篇会被 Close 冲刷
	vc.Add(11)
	vc.Close() // 关停时冲刷剩余
	if got := repo.snapshot(); got[11] != 1 {
		t.Fatalf("article 11 should be flushed on Close, got %d", got[11])
	}
}

// TestViewCounterCloseIdempotent 验证 Close 可重复调用不 panic。
func TestViewCounterCloseIdempotent(t *testing.T) {
	repo := &fakeRepoForView{mockArticleRepo: *newMockArticleRepo(), got: map[int64]int64{}}
	vc := NewViewCounter(repo, time.Hour, 10)
	vc.Close()
	vc.Close() // 第二次调用应为空操作
}
