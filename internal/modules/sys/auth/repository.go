package auth

import (
	"context"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/session"
	"github.com/redis/go-redis/v9"
)

type repository struct {
	cache   *redis.Client
	prefix  string
	timeout time.Duration
}

func (r repository) putCaptcha(ctx context.Context, id, answer string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.cache.Set(ctx, r.prefix+"captcha:"+id, session.Digest(answer), 2*time.Minute).Err()
}
func (r repository) captcha(ctx context.Context, id, answer string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	digest, err := r.cache.GetDel(ctx, r.prefix+"captcha:"+id).Result()
	if err == redis.Nil {
		return false, nil
	}
	return err == nil && digest == session.Digest(answer), err
}
func (r repository) locked(ctx context.Context, name string, threshold int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	n, err := r.cache.Get(ctx, r.prefix+"auth:fail:"+name).Int()
	if err == redis.Nil {
		return false, nil
	}
	return n >= threshold, err
}
func (r repository) fail(ctx context.Context, name string, threshold, minutes int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	n, err := r.cache.Eval(ctx, `local n=redis.call('INCR',KEYS[1]);if n==1 or n==tonumber(ARGV[1]) then redis.call('PEXPIRE',KEYS[1],ARGV[2]) end;return n`, []string{r.prefix + "auth:fail:" + name}, threshold, time.Duration(minutes)*time.Minute/time.Millisecond).Int()
	return n >= threshold, err
}
func (r repository) clearFailures(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.cache.Del(ctx, r.prefix+"auth:fail:"+name).Err()
}
func (r repository) rate(ctx context.Context, subject string, limit int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	n, err := r.cache.Eval(ctx, `local n=redis.call('INCR',KEYS[1]);if n==1 then redis.call('PEXPIRE',KEYS[1],60000) end;return n`, []string{r.prefix + "limit:auth:" + subject}, limit).Int()
	return n <= limit, err
}
