// Package syncer 提供 rsync 同步功能
package syncer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
		s.logger.Infof("标记 Dirty 状态，等待当前同步完成后再次同步")
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

// convertToMSYS2Path 将Windows路径转换为MSYS2格式
// 例如: C:\Users\test 转换为 /c/Users/test
func convertToMSYS2Path(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}

	// 将反斜杠转换为正斜杠
	path = strings.ReplaceAll(path, "\\", "/")

	// 匹配驱动器号（如 C: 变为 /c）
	re := regexp.MustCompile(`^([A-Za-z]):`)
	path = re.ReplaceAllStringFunc(path, func(match string) string {
		return "/" + strings.ToLower(string(match[0]))
	})

	return path
}

// buildCommand 构建 rsync 命令
func (s *Syncer) buildCommand(ctx context.Context, localDir string) (*exec.Cmd, func(), error) {
	// 提取芯片目录名称（在路径转换前提取，确保正确获取目录名）
	chipDirName := filepath.Base(localDir)

	// 在Windows上将路径转换为MSYS2格式
	localDir = convertToMSYS2Path(localDir)
	// 构建远程目标路径
	// 格式: rsync://user@host:port/module/chipDirName/
	// 确保每个芯片目录同步到远程对应的子目录中
	remotePath := fmt.Sprintf("rsync://%s@%s:%d/%s/%s/",
		s.cfg.Username,
		s.cfg.RemoteHost,
		s.cfg.RemotePort,
		s.cfg.RemoteModule,
		chipDirName,
	)

	args := []string{
		"-avz",
		"--partial",
		"--inplace",
		"--progress",
	}

	// 添加源路径和目标路径
	args = append(args, localDir+"/", remotePath)

	cmd := exec.CommandContext(ctx, s.cfg.RsyncPath, args...)

	// 隐藏命令行窗口（由OS-specific helper处理）
	hideConsoleForCmd(cmd)

	// 通过环境变量设置密码
	if s.cfg.Password != "" {
		cmd.Env = append(os.Environ(), "RSYNC_PASSWORD="+s.cfg.Password)
	}

	return cmd, func() {}, nil
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
		s.logger.Errorf("同步失败: %s", result.Error)
		return fmt.Errorf(result.Error)
	}

	s.logger.Infof("开始同步, local_path: %s", s.cfg.LocalPath)

	// 获取芯片子目录列表
	dirs, err := s.getChipDirs()
	if err != nil {
		result.EndTime = time.Now()
		result.Success = false
		result.Error = fmt.Sprintf("获取芯片目录失败: %v", err)
		s.setResult(result)
		s.logger.Errorf("同步失败: %s", result.Error)
		return fmt.Errorf(result.Error)
	}

	if len(dirs) == 0 {
		s.logger.Infof("没有找到芯片目录")
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

		// 检查目录是否已稳定，如果稳定则跳过
		if s.isChipDirStable(dir) {
			outputs = append(outputs, fmt.Sprintf("[%s] 已稳定，跳过", filepath.Base(dir)))
			continue
		}

		output, err := s.syncDir(syncCtx, dir)
		if err != nil {
			hasError = true
			s.logger.Errorf("同步目录失败, dir: %s, error: %v", dir, err)
			outputs = append(outputs, fmt.Sprintf("[%s] 失败: %v", filepath.Base(dir), err))
		} else {
			s.logger.Infof("同步目录成功, dir: %s", dir)
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

	s.logger.Infof("同步完成, success: %v, duration: %v", result.Success, result.EndTime.Sub(result.StartTime))

	return nil
}

// syncDir 同步单个目录
func (s *Syncer) syncDir(ctx context.Context, dir string) (string, error) {
	cmd, cleanup, err := s.buildCommand(ctx, dir)
	if err != nil {
		return "", err
	}
	defer cleanup()

	s.logger.Infof("执行 rsync, cmd: %s", cmd.String())

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

// getLatestModTime 递归获取目录下所有文件的最新修改时间
func getLatestModTime(dir string) (time.Time, error) {
	var latest time.Time

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过隐藏文件和目录
		if strings.HasPrefix(info.Name(), ".") && path != dir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// 只检查文件的修改时间
		if !info.IsDir() {
			if info.ModTime().After(latest) {
				latest = info.ModTime()
			}
		}
		return nil
	})

	return latest, err
}

// isChipDirStable 判断芯片目录是否已稳定（超过阈值时间无修改）
func (s *Syncer) isChipDirStable(dir string) bool {
	latest, err := getLatestModTime(dir)
	if err != nil {
		s.logger.Errorf("获取目录修改时间失败, dir: %s, error: %v", dir, err)
		return false // 有错误时不跳过，继续同步
	}

	// 如果目录为空或没有文件，不视为稳定
	if latest.IsZero() {
		return false
	}

	stableThreshold := time.Duration(s.cfg.StableHours) * time.Hour
	isStable := time.Since(latest) > stableThreshold

	if isStable {
		s.logger.Infof("芯片目录已稳定，跳过同步, dir: %s, last_modified: %s, stable_hours: %d",
			filepath.Base(dir),
			latest.Format("2006-01-02 15:04:05"),
			s.cfg.StableHours)
	}

	return isStable
}
