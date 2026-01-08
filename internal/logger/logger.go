// Package logger 提供日志管理功能
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger 日志管理器
type Logger struct {
	logger   *slog.Logger
	file     *os.File
	logPath  string
	maxSize  int64 // 最大文件大小（字节）
	mu       sync.Mutex
	logLines []string // 保存最近的日志用于 UI 显示
	maxLines int
}

// NewLogger 创建日志管理器
func NewLogger(logPath string) (*Logger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	l := &Logger{
		logPath:  logPath,
		maxSize:  10 * 1024 * 1024, // 默认 10MB
		logLines: make([]string, 0, 100),
		maxLines: 100,
	}

	if err := l.openLogFile(); err != nil {
		return nil, err
	}

	return l, nil
}

// openLogFile 打开日志文件
func (l *Logger) openLogFile() error {
	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	l.file = file

	// 创建双输出：文件 + 内存缓冲
	handler := slog.NewTextHandler(io.MultiWriter(file, &logBuffer{l: l}), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	l.logger = slog.New(handler)

	return nil
}

// logBuffer 实现 io.Writer，用于捕获日志到内存
type logBuffer struct {
	l *Logger
}

func (b *logBuffer) Write(p []byte) (n int, err error) {
	b.l.mu.Lock()
	defer b.l.mu.Unlock()

	line := string(p)
	b.l.logLines = append(b.l.logLines, line)

	// 保持最大行数限制
	if len(b.l.logLines) > b.l.maxLines {
		b.l.logLines = b.l.logLines[len(b.l.logLines)-b.l.maxLines:]
	}

	return len(p), nil
}

// rotateIfNeeded 检查并执行日志轮转
func (l *Logger) rotateIfNeeded() error {
	info, err := l.file.Stat()
	if err != nil {
		return err
	}

	if info.Size() < l.maxSize {
		return nil
	}

	// 关闭当前文件
	l.file.Close()

	// 重命名为带时间戳的备份
	backupPath := fmt.Sprintf("%s.%s", l.logPath, time.Now().Format("20060102150405"))
	if err := os.Rename(l.logPath, backupPath); err != nil {
		return fmt.Errorf("日志轮转失败: %w", err)
	}

	// 重新打开日志文件
	return l.openLogFile()
}

// Info 记录信息级别日志
func (l *Logger) Info(msg string, args ...any) {
	l.rotateIfNeeded()
	l.logger.Info(msg, args...)
}

// Error 记录错误级别日志
func (l *Logger) Error(msg string, args ...any) {
	l.rotateIfNeeded()
	l.logger.Error(msg, args...)
}

// Warn 记录警告级别日志
func (l *Logger) Warn(msg string, args ...any) {
	l.rotateIfNeeded()
	l.logger.Warn(msg, args...)
}

// GetRecentLogs 获取最近的日志行
func (l *Logger) GetRecentLogs() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	result := make([]string, len(l.logLines))
	copy(result, l.logLines)
	return result
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
