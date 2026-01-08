// Package scheduler 提供定时调度功能
package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"chip_sync/internal/logger"
	"chip_sync/internal/syncer"
)

// Scheduler 定时调度器
type Scheduler struct {
	syncer   *syncer.Syncer
	logger   *logger.Logger
	interval time.Duration
	running  atomic.Bool
	mu       sync.Mutex
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewScheduler 创建调度器
func NewScheduler(s *syncer.Syncer, log *logger.Logger, intervalSeconds int) *Scheduler {
	return &Scheduler{
		syncer:   s,
		logger:   log,
		interval: time.Duration(intervalSeconds) * time.Second,
	}
}

// UpdateInterval 更新同步间隔
func (s *Scheduler) UpdateInterval(seconds int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = time.Duration(seconds) * time.Second
	s.logger.Info("更新同步间隔", "seconds", seconds)
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running.Store(true)

	s.wg.Add(1)
	go s.run(ctx)

	s.logger.Info("调度器已启动", "interval", s.interval)
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running.Load() {
		return
	}

	s.running.Store(false)
	if s.cancel != nil {
		s.cancel()
	}

	// 等待调度器退出
	s.wg.Wait()
	s.logger.Info("调度器已停止")
}

// IsRunning 检查调度器是否运行中
func (s *Scheduler) IsRunning() bool {
	return s.running.Load()
}

// run 调度器主循环
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	// 启动时立即执行一次同步
	s.triggerSync(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.triggerSync(ctx)

			// 检查是否需要更新间隔
			s.mu.Lock()
			newInterval := s.interval
			s.mu.Unlock()
			ticker.Reset(newInterval)
		}
	}
}

// triggerSync 触发同步任务
func (s *Scheduler) triggerSync(ctx context.Context) {
	// 检查当前是否有同步在运行
	if s.syncer.GetStatus() == syncer.StatusRunning {
		s.syncer.MarkDirty()
		s.logger.Info("同步任务正在运行，标记 Dirty 状态")
		return
	}

	s.logger.Info("调度器触发同步")

	// 异步执行同步
	go func() {
		for {
			if err := s.syncer.Sync(ctx); err != nil {
				s.logger.Error("同步执行失败", "error", err)
			}

			// 检查是否有 dirty 标记，如果有则再次同步
			if s.syncer.IsDirty() {
				s.logger.Info("检测到 Dirty 标记，立即再次同步")
				continue
			}
			break
		}
	}()
}

// TriggerNow 立即触发一次同步
func (s *Scheduler) TriggerNow() {
	ctx := context.Background()
	go s.triggerSync(ctx)
}
