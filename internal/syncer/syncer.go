// Package syncer 提供 rsync 同步功能
package syncer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chip_sync/internal/config"
	"chip_sync/internal/logger"
)

// Status 同步状态
type Status int32

const (
	StatusIdle    Status = iota // 空闲
	StatusRunning               // 运行中
	StatusError                 // 错误
)

func (s Status) String() string {
	switch s {
	case StatusIdle:
		return "空闲"
	case StatusRunning:
		return "同步中"
	case StatusError:
		return "错误"
	default:
		return "未知"
	}
}

// SyncResult 同步结果
type SyncResult struct {
	Success   bool      `json:"success"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Error     string    `json:"error,omitempty"`
	Output    string    `json:"output,omitempty"`
}

// Syncer rsync 同步器
type Syncer struct {
	cfg        *config.Config
	logger     *logger.Logger
	status     atomic.Int32
	dirty      atomic.Bool
	mu         sync.Mutex
	cancel     context.CancelFunc
	lastResult *SyncResult
	resultMu   sync.RWMutex
}

// NewSyncer 创建同步器
func NewSyncer(cfg *config.Config, log *logger.Logger) *Syncer {
	s := &Syncer{
		cfg:    cfg,
		logger: log,
	}
	s.status.Store(int32(StatusIdle))
	return s
}

// UpdateConfig 更新配置
func (s *Syncer) UpdateConfig(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

// GetStatus 获取当前状态
func (s *Syncer) GetStatus() Status {
	return Status(s.status.Load())
}

// GetStatusString 获取状态字符串
func (s *Syncer) GetStatusString() string {
	return s.GetStatus().String()
}

// IsDirty 检查是否有待处理的同步请求
func (s *Syncer) IsDirty() bool {
	return s.dirty.Load()
}

// MarkDirty 标记为脏状态（有待处理同步）
func (s *Syncer) MarkDirty() {
	s.dirty.Store(true)
	if s.logger != nil {
		s.logger.Info("标记 Dirty 状态，等待当前同步完成后再次同步")
	}
}

// GetLastResult 获取最近一次同步结果
func (s *Syncer) GetLastResult() *SyncResult {
	s.resultMu.RLock()
	defer s.resultMu.RUnlock()
	if s.lastResult == nil {
		return nil
	}
	result := *s.lastResult
	return &result
}

// buildCommand 构建 rsync 命令
func (s *Syncer) buildCommand(ctx context.Context, localDir string) (*exec.Cmd, func(), error) {
	// 构建远程目标路径
	// 格式: rsync://user@host:port/module/path
	remotePath := fmt.Sprintf("rsync://%s@%s:%d/%s/",
		s.cfg.Username,
		s.cfg.RemoteHost,
		s.cfg.RemotePort,
		s.cfg.RemoteModule,
	)

	args := []string{
		"-avz",
		"--partial",
		"--inplace",
		"--progress",
	}

	var cleanup func() = func() {}

	// 添加密码文件参数
	if s.cfg.Password != "" {
		// 创建临时密码文件
		tmpFile, err := os.CreateTemp("", "rsync_pass_*.txt")
		if err != nil {
			return nil, nil, fmt.Errorf("创建临时密码文件失败: %w", err)
		}

		// 写入密码
		if _, err := tmpFile.WriteString(s.cfg.Password); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return nil, nil, fmt.Errorf("写入临时密码文件失败: %w", err)
		}

		// rsync 要求密码文件权限为 600
		if err := tmpFile.Chmod(0600); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return nil, nil, fmt.Errorf("设置临时密码文件权限失败: %w", err)
		}

		tmpFile.Close()
		args = append(args, "--password-file="+tmpFile.Name())
		cleanup = func() {
			os.Remove(tmpFile.Name())
		}
	}

	// 添加源路径和目标路径
	args = append(args, localDir+"/", remotePath)

	cmd := exec.CommandContext(ctx, s.cfg.RsyncPath, args...)
	return cmd, cleanup, nil
}

// Sync 执行同步
func (s *Syncer) Sync(ctx context.Context) error {
	// 尝试获取锁
	if !s.mu.TryLock() {
		// 已有同步在运行，标记 dirty
		s.MarkDirty()
		return fmt.Errorf("同步任务正在运行中")
	}
	defer s.mu.Unlock()

	// 清除 dirty 标记
	s.dirty.Store(false)

	// 设置状态为运行中
	s.status.Store(int32(StatusRunning))
	defer func() {
		if s.dirty.Load() {
			// 如果有新的同步请求，保持运行状态
			return
		}
		// 根据结果设置最终状态
		if s.lastResult != nil && !s.lastResult.Success {
			s.status.Store(int32(StatusError))
		} else {
			s.status.Store(int32(StatusIdle))
		}
	}()

	result := &SyncResult{
		StartTime: time.Now(),
	}

	// 验证配置
	if err := s.cfg.Validate(); err != nil {
		result.EndTime = time.Now()
		result.Success = false
		result.Error = fmt.Sprintf("配置验证失败: %v", err)
		s.setResult(result)
		s.logger.Error("同步失败", "error", result.Error)
		return fmt.Errorf(result.Error)
	}

	s.logger.Info("开始同步", "local_path", s.cfg.LocalPath)

	// 获取芯片子目录列表
	dirs, err := s.getChipDirs()
	if err != nil {
		result.EndTime = time.Now()
		result.Success = false
		result.Error = fmt.Sprintf("获取芯片目录失败: %v", err)
		s.setResult(result)
		s.logger.Error("同步失败", "error", result.Error)
		return fmt.Errorf(result.Error)
	}

	if len(dirs) == 0 {
		s.logger.Info("没有找到芯片目录")
		result.EndTime = time.Now()
		result.Success = true
		result.Output = "没有找到芯片目录"
		s.setResult(result)
		return nil
	}

	// 创建可取消的上下文
	syncCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	var outputs []string
	var hasError bool

	// 同步每个芯片目录
	for _, dir := range dirs {
		select {
		case <-syncCtx.Done():
			result.EndTime = time.Now()
			result.Success = false
			result.Error = "同步被取消"
			s.setResult(result)
			return syncCtx.Err()
		default:
		}

		output, err := s.syncDir(syncCtx, dir)
		if err != nil {
			hasError = true
			s.logger.Error("同步目录失败", "dir", dir, "error", err)
			outputs = append(outputs, fmt.Sprintf("[%s] 失败: %v", filepath.Base(dir), err))
		} else {
			s.logger.Info("同步目录成功", "dir", dir)
			outputs = append(outputs, fmt.Sprintf("[%s] 成功", filepath.Base(dir)))
		}

		if output != "" {
			outputs = append(outputs, output)
		}
	}

	result.EndTime = time.Now()
	result.Success = !hasError
	result.Output = strings.Join(outputs, "\n")
	if hasError {
		result.Error = "部分目录同步失败"
	}
	s.setResult(result)

	s.logger.Info("同步完成", "success", result.Success, "duration", result.EndTime.Sub(result.StartTime))

	return nil
}

// syncDir 同步单个目录
func (s *Syncer) syncDir(ctx context.Context, dir string) (string, error) {
	cmd, cleanup, err := s.buildCommand(ctx, dir)
	if err != nil {
		return "", err
	}
	defer cleanup()

	s.logger.Info("执行 rsync", "cmd", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("rsync 执行失败: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

// getChipDirs 获取芯片子目录列表
func (s *Syncer) getChipDirs() ([]string, error) {
	entries, err := os.ReadDir(s.cfg.LocalPath)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, filepath.Join(s.cfg.LocalPath, entry.Name()))
		}
	}

	return dirs, nil
}

// setResult 设置同步结果
func (s *Syncer) setResult(result *SyncResult) {
	s.resultMu.Lock()
	defer s.resultMu.Unlock()
	s.lastResult = result
}

// Cancel 取消当前同步
func (s *Syncer) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}
