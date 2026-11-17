package log

import (
	"sync"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/scheduler"
	"go.uber.org/zap"
)

func TestJobLogFullQueueIsObservableAndNonBlocking(t *testing.T) {
	s := &Service{queue: make(chan scheduler.Event, 1), log: zap.NewNop()}
	s.Submit(scheduler.Event{})
	var wg sync.WaitGroup
	start := time.Now()
	for range 10 {
		wg.Go(func() {
			for range 1000 {
				s.Submit(scheduler.Event{})
			}
		})
	}
	wg.Wait()
	if time.Since(start) > time.Second || s.Dropped() != 10000 {
		t.Fatal("满队列阻塞或丢弃计数错误")
	}
}
