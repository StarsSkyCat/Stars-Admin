//go:build integration

package session

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"github.com/StarsSkyCat/Stars-Admin/internal/testutil"
	token "github.com/StarsSkyCat/Stars-Admin/pkg/jwt"
)

func TestAtomicRotationRevocationAndSessionLimits(t *testing.T) {
	cfg, err := config.Load("../../../configs")
	if err != nil {
		t.Fatal(err)
	}
	deps := testutil.Open(t, cfg)
	store := New(deps.Redis, cfg.Redis.KeyPrefix, cfg.Redis.Timeout)
	tokens := token.New("session-test-secret", "session-test", time.Minute)
	issue := func() token.Tokens {
		t.Helper()
		pair, err := tokens.Issue(42, "alice", "", time.Now().Add(time.Hour), false)
		if err != nil {
			t.Fatal(err)
		}
		return pair
	}
	pair := issue()
	if err := store.Login(t.Context(), 42, 1, pair, 5, true); err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int64
	var wg sync.WaitGroup
	var winner token.Tokens
	var mu sync.Mutex
	for range 40 {
		wg.Go(func() {
			candidate, e := tokens.Issue(42, "alice", pair.Refresh.SID, pair.Refresh.ExpiresAt.Time, false)
			if e != nil {
				t.Error(e)
				return
			}
			ok, e := store.Rotate(t.Context(), 42, 1, &pair.Refresh, pair.RefreshToken, candidate)
			if e != nil {
				t.Error(e)
			}
			if ok {
				wins.Add(1)
				mu.Lock()
				winner = candidate
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("并发刷新成功数量 %d", wins.Load())
	}
	if ok, err := store.Validate(t.Context(), 42, 1, &pair.Access); err != nil || ok {
		t.Fatal("旧 Access 未撤销")
	}
	if ok, err := store.Validate(t.Context(), 42, 1, &winner.Access); err != nil || !ok {
		t.Fatal("新 Access 不可用")
	}
	// Refresh 与版本撤销交错时，所有旧版本会话最终必须消失。
	for range 20 {
		wg.Go(func() {
			candidate, e := tokens.Issue(42, "alice", winner.Refresh.SID, winner.Refresh.ExpiresAt.Time, false)
			if e != nil {
				t.Error(e)
				return
			}
			if _, err := store.Rotate(t.Context(), 42, 1, &winner.Refresh, winner.RefreshToken, candidate); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Go(func() {
		if err := store.RevokeBefore(t.Context(), 42, 2); err != nil {
			t.Error(err)
		}
	})
	wg.Wait()
	if ok, err := store.Validate(t.Context(), 42, 1, &winner.Access); err != nil || ok {
		t.Fatal("刷新不能复活已撤销会话")
	}
	newPair := issue()
	if err := store.Login(t.Context(), 42, 2, newPair, 5, true); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeBefore(t.Context(), 42, 2); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.Validate(t.Context(), 42, 2, &newPair.Access); err != nil || !ok {
		t.Fatal("撤销重试误删新版本会话")
	}
	all := []token.Tokens{}
	for range 6 {
		p := issue()
		if err := store.Login(t.Context(), 42, 2, p, 5, true); err != nil {
			t.Fatal(err)
		}
		all = append(all, p)
		time.Sleep(time.Millisecond)
	}
	if ok, _ := store.Validate(t.Context(), 42, 2, &all[0].Access); ok {
		t.Fatal("超额会话没有撤销最早登录")
	}
	single := issue()
	if err := store.Login(t.Context(), 42, 2, single, 5, false); err != nil {
		t.Fatal(err)
	}
	if ok, _ := store.Validate(t.Context(), 42, 2, &all[5].Access); ok {
		t.Fatal("单点登录未清旧会话")
	}
	if ok, err := store.Logout(t.Context(), 42, 2, &single.Access); err != nil || !ok {
		t.Fatal("登出失败")
	}
	if ok, _ := store.Validate(t.Context(), 42, 2, &single.Access); ok {
		t.Fatal("登出后 Access 仍有效")
	}
	// 存储内不应出现原始令牌，键必须有有限 TTL。
	cursor := uint64(0)
	for {
		keys, next, err := deps.Redis.Scan(t.Context(), cursor, cfg.Redis.KeyPrefix+"*", 100).Result()
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			ttl, err := deps.Redis.PTTL(t.Context(), key).Result()
			if err != nil || ttl < 0 {
				t.Fatal("认证键缺少 TTL")
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
}
