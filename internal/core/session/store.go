// Package session 管理单 Redis 上的原子会话状态，不保存原始令牌。
package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	token "github.com/StarsSkyCat/Stars-Admin/pkg/jwt"
	"github.com/redis/go-redis/v9"
)

// Store 所有会话变更共享同一脚本，跨键操作仅支持单 Redis。
type Store struct {
	client  *redis.Client
	prefix  string
	timeout time.Duration
}

// New 接收由应用统一管理的 Redis 连接。
func New(client *redis.Client, prefix string, timeout time.Duration) *Store {
	return &Store{client, prefix, timeout}
}

// Digest 使用摘要比对 Refresh，避免 Redis 保存可直接使用的令牌。
func Digest(raw string) string { sum := sha256.Sum256([]byte(raw)); return hex.EncodeToString(sum[:]) }

const script = `
local mode,prefix,user,sid,version,access,refresh,digest = unpack(ARGV)
local base=prefix..'auth:'
local index=base..'sessions:'..user
local now=tonumber(ARGV[11])
local function key(kind,s) return base..kind..':'..user..':'..s end
local function drop(s)
 local sk=key('session',s)
 local aj=redis.call('HGET',sk,'access')
 if aj then redis.call('DEL',key('token',aj)) end
 redis.call('DEL',sk,key('refresh',s),prefix..'online:'..user..':'..s)
 redis.call('ZREM',index,s)
end
local function trim()
 for _,s in ipairs(redis.call('ZRANGEBYSCORE',index,'-inf',now)) do drop(s) end
end
local function write()
 local aexp,sexp=tonumber(ARGV[9]),tonumber(ARGV[10])
 local sk=key('session',sid)
 local created=redis.call('HGET',sk,'created') or ARGV[11]
 redis.call('HSET',sk,'version',version,'access',access,'refresh',refresh,'expires',ARGV[10],'created',created)
 redis.call('PEXPIREAT',sk,sexp)
 redis.call('SET',key('token',access),sid,'PXAT',aexp)
 redis.call('SET',key('refresh',sid),digest,'PXAT',sexp)
 redis.call('SET',prefix..'online:'..user..':'..sid,sid,'PXAT',aexp)
 redis.call('ZADD',index,sexp,sid)
 local last=redis.call('ZRANGE',index,-1,-1,'WITHSCORES')
 if #last>0 then redis.call('PEXPIREAT',index,tonumber(last[2])) end
end
local sk=key('session',sid)
if mode=='login' then
 trim()
 local members=redis.call('ZRANGE',index,0,-1)
 if ARGV[13]=='0' then for _,s in ipairs(members) do drop(s) end
 else
  while redis.call('ZCARD',index)>=tonumber(ARGV[12]) do
   local oldest=nil
   local oldestTime=nil
   for _,s in ipairs(redis.call('ZRANGE',index,0,-1)) do
    local created=tonumber(redis.call('HGET',key('session',s),'created') or '0')
    if not oldestTime or created<oldestTime then oldest=s;oldestTime=created end
   end
   if oldest then drop(oldest) else break end
  end
 end
 write()
 return 1
elseif mode=='rotate' then
 if redis.call('HGET',sk,'version')~=version or redis.call('HGET',sk,'refresh')~=ARGV[14] or redis.call('GET',key('refresh',sid))~=ARGV[15] then return 0 end
 if redis.call('HGET',sk,'expires')~=ARGV[10] then return 0 end
 local old=redis.call('HGET',sk,'access')
 if old then redis.call('DEL',key('token',old)) end
 write()
 return 1
elseif mode=='check' then
 if redis.call('HGET',sk,'version')==version and redis.call('HGET',sk,'access')==access and redis.call('GET',key('token',access))==sid then return 1 end
 return 0
elseif mode=='logout' then
 if redis.call('HGET',sk,'version')~=version or redis.call('HGET',sk,'access')~=access then return 0 end
 drop(sid)
 return 1
elseif mode=='revoke' then
 trim()
 for _,s in ipairs(redis.call('ZRANGE',index,0,-1)) do
  local v=redis.call('HGET',key('session',s),'version')
  if not v or tonumber(v)<tonumber(version) then drop(s) end
 end
 return 1
end
return 0
`

func (s *Store) eval(ctx context.Context, mode, user, sid string, version int64, tokens token.Tokens, maxSessions int, multi bool, oldJTI, oldRaw string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var accessEnd, sessionEnd int64
	if tokens.Access.ExpiresAt != nil {
		accessEnd = tokens.Access.ExpiresAt.UnixMilli()
	}
	if tokens.Refresh.ExpiresAt != nil {
		sessionEnd = tokens.Refresh.ExpiresAt.UnixMilli()
	}
	multiValue := "0"
	if multi {
		multiValue = "1"
	}
	value, err := s.client.Eval(ctx, script, []string{s.prefix + "auth:sessions:" + user}, mode, s.prefix, user, sid, strconv.FormatInt(version, 10), tokens.Access.ID, tokens.Refresh.ID, Digest(tokens.RefreshToken), accessEnd, sessionEnd, time.Now().UnixMilli(), maxSessions, multiValue, oldJTI, Digest(oldRaw)).Int()
	return value == 1, err
}

// Login 原子清理旧会话、限制数量并建立候选令牌状态。
func (s *Store) Login(ctx context.Context, id int64, version int64, tokens token.Tokens, maxSessions int, multi bool) error {
	_, err := s.eval(ctx, "login", strconv.FormatInt(id, 10), tokens.Access.SID, version, tokens, maxSessions, multi, "", "")
	return err
}

// Rotate 核对原 Refresh 并原子换入候选令牌；失败不能返回候选值。
func (s *Store) Rotate(ctx context.Context, id, version int64, old *token.Claims, raw string, tokens token.Tokens) (bool, error) {
	return s.eval(ctx, "rotate", strconv.FormatInt(id, 10), old.SID, version, tokens, 0, false, old.ID, raw)
}

// Validate 同时核对会话版本、当前 jti 和 Access 登录态。
func (s *Store) Validate(ctx context.Context, id, version int64, claims *token.Claims) (bool, error) {
	return s.eval(ctx, "check", strconv.FormatInt(id, 10), claims.SID, version, token.Tokens{Access: *claims}, 0, false, "", "")
}

// Logout 只删除当前有效 Access 对应的会话。
func (s *Store) Logout(ctx context.Context, id, version int64, claims *token.Claims) (bool, error) {
	return s.eval(ctx, "logout", strconv.FormatInt(id, 10), claims.SID, version, token.Tokens{Access: *claims}, 0, false, "", "")
}

// RevokeBefore 只清理旧版本会话，重试不得删除撤销后建立的新会话。
func (s *Store) RevokeBefore(ctx context.Context, id, version int64) error {
	_, err := s.eval(ctx, "revoke", strconv.FormatInt(id, 10), "", version, token.Tokens{}, 0, false, "", "")
	return err
}
