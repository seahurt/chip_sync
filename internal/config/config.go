// Package config 提供配置管理功能
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config 定义应用程序配置结构
type Config struct {
	// RsyncPath rsync 可执行文件路径
	RsyncPath string `json:"rsync_path"`
	// RemoteHost 远程服务器地址 (hostname or IP)
	RemoteHost string `json:"remote_host"`
	// RemotePort rsync daemon 端口，默认 873
	RemotePort int `json:"remote_port"`
	// RemoteModule rsync 模块名
	RemoteModule string `json:"remote_module"`
	// Username rsync 用户名
	Username string `json:"username"`
	// Password 直接输入的密码
	Password string `json:"password"`
	// LocalPath 本地芯片父目录路径
	LocalPath string `json:"local_path"`
	// SyncIntervalSeconds 同步间隔（秒），默认 300（5分钟）
	SyncIntervalSeconds int `json:"sync_interval_seconds"`
	// LogPath 日志文件路径
	LogPath string `json:"log_path"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		RsyncPath:           "rsync",
		RemotePort:          873,
		SyncIntervalSeconds: 300,
	}
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.RsyncPath == "" {
		return errors.New("rsync_path 不能为空")
	}
	if c.RemoteHost == "" {
		return errors.New("remote_host 不能为空")
	}
	if c.RemoteModule == "" {
		return errors.New("remote_module 不能为空")
	}
	if c.LocalPath == "" {
		return errors.New("local_path 不能为空")
	}
	if c.SyncIntervalSeconds < 10 {
		return errors.New("sync_interval_seconds 不能小于 10 秒")
	}

	// 检查本地目录是否存在
	if info, err := os.Stat(c.LocalPath); err != nil {
		return fmt.Errorf("local_path 无效: %w", err)
	} else if !info.IsDir() {
		return errors.New("local_path 必须是目录")
	}

	// 密码已直接在配置中提供，无需检查文件

	return nil
}

// Manager 配置管理器
type Manager struct {
	configPath string
	config     *Config
}

// NewManager 创建配置管理器
func NewManager(configDir string) (*Manager, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}

	return &Manager{
		configPath: filepath.Join(configDir, "config.json"),
		config:     DefaultConfig(),
	}, nil
}

// Load 从文件加载配置
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，使用默认配置
			return nil
		}
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := json.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}

// Save 保存配置到文件
func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// Get 获取当前配置（返回副本）
func (m *Manager) Get() Config {
	return *m.config
}

// Update 更新配置
func (m *Manager) Update(cfg Config) error {
	m.config = &cfg
	return m.Save()
}

// GetConfigPath 返回配置文件路径
func (m *Manager) GetConfigPath() string {
	return m.configPath
}
