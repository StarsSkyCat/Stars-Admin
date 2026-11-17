// Package logger 提供结构化日志与可关闭的轮转文件。
package logger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// New 创建日志及关闭函数，不输出配置值或敏感连接信息。
func New(c config.Log) (*zap.Logger, func() error, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(c.Level)); err != nil {
		return nil, nil, fmt.Errorf("日志级别无效: %w", err)
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	if c.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
	stdout := zapcore.Lock(zapcore.AddSync(os.Stdout))
	sinks := []zapcore.WriteSyncer{stdout}
	var file *lumberjack.Logger
	if c.File != "" {
		if err := os.MkdirAll(filepath.Dir(c.File), 0o750); err != nil {
			return nil, nil, fmt.Errorf("创建日志目录失败: %w", err)
		}
		file = &lumberjack.Logger{Filename: c.File, MaxSize: c.MaxSize, MaxBackups: c.MaxBackups, MaxAge: c.MaxAge, Compress: true}
		sinks = append(sinks, zapcore.AddSync(file))
	}
	log := zap.New(zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(sinks...), level), zap.AddCaller())
	closeLog := func() error {
		err := log.Sync()
		// 容器标准输出通常是管道，不支持 fsync。
		if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTTY) {
			err = nil
		}
		if file != nil {
			err = errors.Join(err, file.Close())
		}
		return err
	}
	return log, closeLog, nil
}
