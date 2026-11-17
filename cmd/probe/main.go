// Command probe 为不含 Shell 的生产镜像提供 readiness 探针。
package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	client := &http.Client{Timeout: 2 * time.Second}
	reply, err := client.Get("http://127.0.0.1:8080/api/health/ready")
	if err != nil {
		os.Exit(1)
	}
	_ = reply.Body.Close()
	if reply.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
