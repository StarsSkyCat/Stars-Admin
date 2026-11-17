package session

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/model"
	token "github.com/StarsSkyCat/Stars-Admin/pkg/jwt"
)

// Online 是会话摘要，不包含 Access、Refresh、jti 或哈希。
type Online struct {
	UserID     model.ID  `json:"userId"`
	SID        string    `json:"sid"`
	Username   string    `json:"username"`
	IP         string    `json:"ip"`
	Browser    string    `json:"browser"`
	LoginAt    time.Time `json:"loginAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	Version    int64     `json:"-"`
}

// Touch 仅更新当前有效会话摘要，不创建已撤销键、不延长任何 TTL。
func (s *Store) Touch(ctx context.Context, id int64, claims *token.Claims, username, ip, browser string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	base := s.prefix + "auth:"
	user := strconv.FormatInt(id, 10)
	sessionKey := base + "session:" + user + ":" + claims.SID
	_, err := s.client.Eval(ctx, `if redis.call('HGET',KEYS[1],'access')~=ARGV[1] or redis.call('EXISTS',KEYS[2])==0 then return 0 end
redis.call('HSETNX',KEYS[1],'username',ARGV[2]);redis.call('HSETNX',KEYS[1],'ip',ARGV[3]);redis.call('HSETNX',KEYS[1],'browser',ARGV[4]);redis.call('HSET',KEYS[1],'lastSeen',ARGV[5]);return 1`, []string{sessionKey, s.prefix + "online:" + user + ":" + claims.SID}, claims.ID, username, ip, browser, time.Now().UnixMilli()).Result()
	return err
}

// OnlineList 通过 SCAN 读取并用 Lua 核对当前 Access、在线摘要及会话。
func (s *Store) OnlineList(ctx context.Context) ([]Online, error) {
	out := []Online{}
	var cursor uint64
	for {
		keys, next, err := s.client.Scan(ctx, cursor, s.prefix+"online:*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			parts := strings.Split(strings.TrimPrefix(key, s.prefix+"online:"), ":")
			if len(parts) != 2 {
				continue
			}
			id, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil || id <= 0 {
				continue
			}
			sk := s.prefix + "auth:session:" + parts[0] + ":" + parts[1]
			values, err := s.client.Eval(ctx, `local a=redis.call('HGET',KEYS[1],'access');if not a or redis.call('GET',ARGV[1]..a)~=ARGV[2] or redis.call('EXISTS',KEYS[2])==0 then return {} end;return redis.call('HMGET',KEYS[1],'version','username','ip','browser','created','lastSeen')`, []string{sk, key}, s.prefix+"auth:token:"+parts[0]+":", parts[1]).Slice()
			if err != nil {
				return nil, err
			}
			if len(values) != 6 {
				continue
			}
			get := func(i int) string { v, _ := values[i].(string); return v }
			version, _ := strconv.ParseInt(get(0), 10, 64)
			created, _ := strconv.ParseInt(get(4), 10, 64)
			last, _ := strconv.ParseInt(get(5), 10, 64)
			out = append(out, Online{UserID: model.ID(id), SID: parts[1], Username: get(1), IP: get(2), Browser: get(3), LoginAt: time.UnixMilli(created).UTC(), LastSeenAt: time.UnixMilli(last).UTC(), Version: version})
		}
		cursor = next
		if cursor == 0 {
			return out, nil
		}
	}
}
