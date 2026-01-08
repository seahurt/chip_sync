package main

import (
	"context"
	"os"
	"path/filepath"

	"chip_sync/internal/config"
	"chip_sync/internal/logger"
	"chip_sync/internal/scheduler"
	"chip_sync/internal/syncer"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 应用程序主结构
type App struct {
	ctx       context.Context
	cfgMgr    *config.Manager
	logger    *logger.Logger
	syncer    *syncer.Syncer
	scheduler *scheduler.Scheduler
}

// NewApp 创建应用程序实例
func NewApp() *App {
	return &App{}
}

// startup 应用启动时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 获取应用数据目录
	dataDir := a.getAppDataDir()

	// 初始化配置管理器
	cfgMgr, err := config.NewManager(dataDir)
	if err != nil {
		runtime.LogError(ctx, "初始化配置管理器失败: "+err.Error())
		return
	}
	a.cfgMgr = cfgMgr

	// 加载配置
	if err := a.cfgMgr.Load(); err != nil {
		runtime.LogWarning(ctx, "加载配置失败: "+err.Error())
	}

	cfg := a.cfgMgr.Get()

	// 设置默认日志路径
	if cfg.LogPath == "" {
		cfg.LogPath = filepath.Join(dataDir, "seqsync.log")
		a.cfgMgr.Update(cfg)
	}

	// 初始化日志系统
	log, err := logger.NewLogger(cfg.LogPath)
	if err != nil {
		runtime.LogError(ctx, "初始化日志系统失败: "+err.Error())
		return
	}
	a.logger = log
	a.logger.Info("SeqSync Windows 启动")

	// 初始化同步器
	a.syncer = syncer.NewSyncer(&cfg, a.logger)

	// 初始化调度器
	a.scheduler = scheduler.NewScheduler(a.syncer, a.logger, cfg.SyncIntervalSeconds)
}

// shutdown 应用关闭时调用
func (a *App) shutdown(ctx context.Context) {
	if a.scheduler != nil {
		a.scheduler.Stop()
	}
	if a.syncer != nil {
		a.syncer.Cancel()
	}
	if a.logger != nil {
		a.logger.Info("SeqSync Windows 关闭")
		a.logger.Close()
	}
}

// getAppDataDir 获取应用数据目录
func (a *App) getAppDataDir() string {
	// 使用用户配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		// 回退到可执行文件所在目录
		exe, _ := os.Executable()
		return filepath.Dir(exe)
	}
	return filepath.Join(configDir, "SeqSync")
}

// GetConfig 获取当前配置
func (a *App) GetConfig() config.Config {
	if a.cfgMgr == nil {
		return config.Config{}
	}
	return a.cfgMgr.Get()
}

// SaveConfig 保存配置
func (a *App) SaveConfig(cfg config.Config) error {
	if a.cfgMgr == nil {
		return nil
	}

	// 更新配置
	if err := a.cfgMgr.Update(cfg); err != nil {
		a.logger.Error("保存配置失败", "error", err)
		return err
	}

	// 更新同步器配置
	if a.syncer != nil {
		a.syncer.UpdateConfig(&cfg)
	}

	// 更新调度器间隔
	if a.scheduler != nil {
		a.scheduler.UpdateInterval(cfg.SyncIntervalSeconds)
	}

	a.logger.Info("配置已保存")
	return nil
}

// SelectDirectory 选择目录
func (a *App) SelectDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择芯片父目录",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// StartSync 手动触发同步
func (a *App) StartSync() error {
	if a.syncer == nil {
		return nil
	}

	a.logger.Info("手动触发同步")
	go func() {
		if err := a.syncer.Sync(context.Background()); err != nil {
			a.logger.Error("同步失败", "error", err)
		}
	}()
	return nil
}

// GetSyncStatus 获取同步状态
func (a *App) GetSyncStatus() map[string]interface{} {
	result := map[string]interface{}{
		"status":    "空闲",
		"isDirty":   false,
		"isRunning": false,
	}

	if a.syncer != nil {
		result["status"] = a.syncer.GetStatusString()
		result["isDirty"] = a.syncer.IsDirty()
		result["isRunning"] = a.syncer.GetStatus() == syncer.StatusRunning

		if lastResult := a.syncer.GetLastResult(); lastResult != nil {
			result["lastResult"] = lastResult
		}
	}

	if a.scheduler != nil {
		result["schedulerRunning"] = a.scheduler.IsRunning()
	}

	return result
}

// GetLogs 获取最近日志
func (a *App) GetLogs() []string {
	if a.logger == nil {
		return []string{}
	}
	return a.logger.GetRecentLogs()
}

// StartScheduler 启动调度器
func (a *App) StartScheduler() {
	if a.scheduler != nil {
		a.scheduler.Start()
	}
}

// StopScheduler 停止调度器
func (a *App) StopScheduler() {
	if a.scheduler != nil {
		a.scheduler.Stop()
	}
}

// GetChipDirs 获取芯片目录列表
func (a *App) GetChipDirs() ([]string, error) {
	cfg := a.cfgMgr.Get()
	if cfg.LocalPath == "" {
		return []string{}, nil
	}

	entries, err := os.ReadDir(cfg.LocalPath)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name()[0] != '.' {
			dirs = append(dirs, entry.Name())
		}
	}
	return dirs, nil
}
