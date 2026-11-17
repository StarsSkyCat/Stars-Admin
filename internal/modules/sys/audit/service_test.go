package audit

import (
	"sync"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/middleware"
	"go.uber.org/zap"
)

func TestFullAuditQueueDoesNotBlockAndCountsDrops(t *testing.T) {
	s := &Service{worker: &worker{queue: make(chan middleware.AuditEvent, 1), log: zap.NewNop()}}
	s.Submit(middleware.AuditEvent{})
	start := time.Now()
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 1000 {
				s.Submit(middleware.AuditEvent{})
			}
		})
	}
	wg.Wait()
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("满队列阻塞请求 %s", elapsed)
	}
	if s.worker.dropped.Load() != 10000 {
		t.Fatalf("丢弃计数不正确 %d", s.worker.dropped.Load())
	}
	if len(s.worker.queue) != 1 {
		t.Fatal("队列容量没有保持有界")
	}
}
