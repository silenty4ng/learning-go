package service

import (
	"context"
	"testing"
	"time"
)

// fakeRepoForView 记录 IncrViewCounts 的调用结果。
type fakeRepoForView struct {
	mockArticleRepo
	got    map[int64]int64
	called int
}

func (f *fakeRepoForView) IncrViewCounts(_ context.Context, counts map[int64]int64) error {
	f.called++
	for id, n := range counts {
		f.got[id] += n
	}
	return nil
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

	if repo.called == 0 {
		t.Fatal("flush was never called on Close")
	}
	if repo.got[1] != 5 || repo.got[2] != 1 {
		t.Fatalf("flushed counts = %v, want {1:5, 2:1}", repo.got)
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
	for time.Now().Before(deadline) && repo.got[1] != 1 {
		time.Sleep(5 * time.Millisecond)
	}
	for id := int64(1); id <= 10; id++ {
		if repo.got[id] != 1 {
			t.Fatalf("article %d count = %d, want 1 (flushed=%v)", id, repo.got[id], repo.got)
		}
	}
	// 缓冲已排空，此时投递的第 11 篇会被 Close 冲刷
	vc.Add(11)
	vc.Close() // 关停时冲刷剩余
	if repo.got[11] != 1 {
		t.Fatalf("article 11 should be flushed on Close, got %d", repo.got[11])
	}
}

// TestViewCounterCloseIdempotent 验证 Close 可重复调用不 panic。
func TestViewCounterCloseIdempotent(t *testing.T) {
	repo := &fakeRepoForView{mockArticleRepo: *newMockArticleRepo(), got: map[int64]int64{}}
	vc := NewViewCounter(repo, time.Hour, 10)
	vc.Close()
	vc.Close() // 第二次调用应为空操作
}
