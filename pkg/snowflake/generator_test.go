package snowflake

import (
	"sync"
	"testing"
	"time"
)

func TestNodeAndConcurrentUniqueness(t *testing.T) {
	for _, node := range []int64{-1, 1024} {
		if _, err := New(node); err == nil {
			t.Fatal("非法节点未拒绝")
		}
	}
	g, err := New(0)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, 1000)
	var wg sync.WaitGroup
	for i := range len(ids) {
		wg.Go(func() {
			id, err := g.Next()
			if err != nil {
				t.Error(err)
				return
			}
			ids[i] = id
		})
	}
	wg.Wait()
	seen := make(map[int64]bool)
	for _, id := range ids {
		if id <= 1 || seen[id] {
			t.Fatal("ID 无效或重复")
		}
		seen[id] = true
	}
}

func TestRollbackRefusesAndRecovers(t *testing.T) {
	g, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	first, err := g.Next()
	if err != nil {
		t.Fatal(err)
	}
	old := g.lastWall
	g.clock = func() time.Time { return time.UnixMilli(old - 1) }
	if _, err := g.Next(); err == nil {
		t.Fatal("时钟回拨应拒绝生成")
	}
	g.clock = time.Now
	next, err := g.Next()
	if err != nil || next <= first {
		t.Fatal("时钟恢复后应生成更大的 ID")
	}
}
