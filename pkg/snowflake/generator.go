// Package snowflake 封装雪花 ID，节点编号由部署配置显式分配。
package snowflake

import (
	"errors"
	"sync"
	"time"

	base "github.com/bwmarrin/snowflake"
)

// Generator 在单个进程内串行生成 ID，并拒绝观察到的墙上时钟回拨。
type Generator struct {
	mu       sync.Mutex
	node     *base.Node
	clock    func() time.Time
	lastWall int64
}

// New 校验节点范围；不同应用实例必须配置不同节点，不能自动协调分配。
func New(nodeID int64) (*Generator, error) {
	if nodeID < 0 || nodeID > 1023 {
		return nil, errors.New("雪花节点编号必须在 0 到 1023 之间")
	}
	node, err := base.NewNode(nodeID)
	if err != nil {
		return nil, err
	}
	return &Generator{node: node, clock: time.Now}, nil
}

// Next 发生时钟回拨时返回错误，调用方不能用零值继续持久化。
func (g *Generator) Next() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.clock().UnixMilli()
	if now < g.lastWall {
		return 0, errors.New("检测到时钟回拨，拒绝生成 ID")
	}
	id := g.node.Generate().Int64()
	g.lastWall = now
	return id, nil
}
