package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
)

func TestCleanupReverseOrderAndErrors(t *testing.T) {
	var cleanup Cleanup
	var order []int
	want := errors.New("关闭失败")
	cleanup.Add(func() error { order = append(order, 1); return nil })
	cleanup.Add(func() error { order = append(order, 2); return want })
	cleanup.Add(func() error { order = append(order, 3); return nil })
	if err := cleanup.Close(); !errors.Is(err, want) || !slices.Equal(order, []int{3, 2, 1}) {
		t.Fatal("清理顺序或错误聚合不正确")
	}
	if err := cleanup.Close(); err != nil || len(order) != 3 {
		t.Fatal("重复关闭不应再次释放资源")
	}
}

func TestShutdownWaitsForRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	entered, release, stopping := make(chan struct{}), make(chan struct{}), make(chan struct{})
	finished, clientDone := make(chan error, 1), make(chan error, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-release:
		case <-r.Context().Done():
		}
		_, _ = io.WriteString(w, "ok")
	})
	go func() {
		finished <- Run(ctx, listener, handler, config.App{ShutdownTimeout: 2 * time.Second}, func() { close(stopping) })
	}()
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			_, err = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
		}
		clientDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("请求未进入服务")
	}
	cancel()
	select {
	case <-stopping:
	case <-time.After(time.Second):
		t.Fatal("未先标记停止")
	}
	select {
	case err := <-finished:
		t.Fatalf("在途请求未完成却已退出：%v", err)
	default:
	}
	close(release)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("停机超时")
	}
	if err := <-clientDone; err != nil {
		t.Fatal(err)
	}
}

func TestShutdownDeadlineCancelsRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	entered, canceled := make(chan struct{}), make(chan struct{})
	finished := make(chan error, 1)
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); close(canceled) })
	go func() {
		finished <- Run(ctx, listener, handler, config.App{ShutdownTimeout: 20 * time.Millisecond}, func() {})
	}()
	go func() {
		resp, err := (&http.Client{Timeout: time.Second}).Get("http://" + listener.Addr().String())
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("请求未进入")
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("应返回停机超时：%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("没有有界退出")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("未取消请求上下文")
	}
}

func TestListenerFailureStillDrainsRequests(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered, release, stopping := make(chan struct{}), make(chan struct{}), make(chan struct{})
	finished, clientDone := make(chan error, 1), make(chan error, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = io.WriteString(w, "ok")
	})
	go func() {
		finished <- Run(t.Context(), listener, handler, config.App{ShutdownTimeout: time.Second}, func() { close(stopping) })
	}()
	go func() {
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Get("http://" + listener.Addr().String())
		if err == nil {
			_, err = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
		}
		clientDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("请求未进入")
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-stopping:
	case <-time.After(time.Second):
		t.Fatal("监听器失败未触发停机")
	}
	select {
	case err := <-finished:
		t.Fatalf("监听器失败跳过了在途请求排空：%v", err)
	default:
	}
	close(release)
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("未保留监听器错误")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("服务未退出")
	}
	if err := <-clientDone; err != nil {
		t.Fatal(err)
	}
}
