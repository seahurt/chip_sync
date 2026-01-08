// Package logger 提供日志管理功能
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger 日志管理器
type Logger struct {
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

	return nil
}

// logBuffer 用于捕获日志到内存
func (l *Logger) writeToBuffer(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logLines = append(l.logLines, msg)

	// 保持最大行数限制
	if len(l.logLines) > l.maxLines {
		l.logLines = l.logLines[len(l.logLines)-l.maxLines:]
	}
}

// rotateIfNeeded 检查并执行日志轮转
func (l *Logger) rotateIfNeeded() error {
	l.mu.Lock()
	defer l.mu.Unlock()

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

// formatLog 格式化日志
func (l *Logger) formatLog(level, msg string, args ...any) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	content := msg
	if len(args) > 0 {
		content = fmt.Sprintf(msg, args...)
	}
	return fmt.Sprintf("[%s] %s: %s\n", timestamp, level, content)
}

// log 统一记录日志
func (l *Logger) log(level, msg string, args ...any) {
	l.rotateIfNeeded()

	formatted := l.formatLog(level, msg, args...)

	// 写入文件
	l.mu.Lock()
	if l.file != nil {
		io.WriteString(l.file, formatted)
	}
	l.mu.Unlock()

	// 写入内存缓冲
	l.writeToBuffer(formatted)

	// 同时也输出到标准输出
	fmt.Print(formatted)
}

// Info 记录信息级别日志
func (l *Logger) Info(msg string, args ...any) {
	l.log("INFO", msg, args...)
}

// Error 记录错误级别日志
func (l *Logger) Error(msg string, args ...any) {
	l.log("ERROR", msg, args...)
}

// Warn 记录警告级别日志
func (l *Logger) Warn(msg string, args ...any) {
	l.log("WARN", msg, args...)
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
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
