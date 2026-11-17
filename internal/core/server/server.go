// Package server 管理 HTTP 服务的有界优雅停机。
package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
)

// Run 接收已绑定的监听器，取消时先标记停止，再等待在途请求。
func Run(ctx context.Context, listener net.Listener, handler http.Handler, cfg config.App, stopping func()) error {
	requestCtx, cancelRequests := context.WithCancel(context.Background())
	defer cancelRequests()
	srv := &http.Server{
		Handler: handler, ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout, IdleTimeout: cfg.IdleTimeout,
		BaseContext: func(net.Listener) context.Context { return requestCtx },
	}
	finished := make(chan error, 1)
	var workers sync.WaitGroup
	workers.Go(func() { finished <- srv.Serve(listener) })
	defer workers.Wait()
	var serveErr error
	select {
	case serveErr = <-finished:
	case <-ctx.Done():
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	// 即便监听器异常退出，也要等待已有请求，不能提前关闭数据库。
	stopping()
	shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		cancelRequests()
		return errors.Join(serveErr, err, srv.Close())
	}
	return serveErr
}

// Cleanup 按逆序释放已获得的资源，单个失败不跳过后续释放。
type Cleanup struct {
	functions []func() error
}

// Add 登记资源释放函数。
func (c *Cleanup) Add(fn func() error) { c.functions = append(c.functions, fn) }

// Close 释放资源并聚合错误；重复关闭为空操作。
func (c *Cleanup) Close() error {
	var err error
	for i := len(c.functions) - 1; i >= 0; i-- {
		err = errors.Join(err, c.functions[i]())
	}
	c.functions = nil
	return err
}
