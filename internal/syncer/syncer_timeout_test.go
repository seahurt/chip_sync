//go:build !windows

package syncer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chip_sync/internal/config"
	"chip_sync/internal/logger"
)

func TestSyncDirTimeout(t *testing.T) {
	// 使用一个不会自行结束的 rsync 替身，验证进程级超时能释放等待。
	rsync := filepath.Join(t.TempDir(), "rsync-stall.sh")
	if err := os.WriteFile(rsync, []byte("#!/bin/sh\nexec sleep 3\n"), 0755); err != nil {
		t.Fatalf("创建 rsync 替身失败: %v", err)
	}

	log, err := logger.NewLogger(filepath.Join(t.TempDir(), "syncer.log"))
	if err != nil {
		t.Fatalf("创建测试日志失败: %v", err)
	}
	defer log.Close()

	s := NewSyncer(&config.Config{
		RsyncPath:           rsync,
		RemoteHost:          "127.0.0.1",
		RemotePort:          873,
		RemoteModule:        "test",
		RsyncTimeoutSeconds: 1,
	}, log)

	start := time.Now()
	_, err = s.syncDir(context.Background(), t.TempDir())
	elapsed := time.Since(start)

	if err == nil || !strings.Contains(err.Error(), "rsync 执行超时") {
		t.Fatalf("期望 rsync 超时错误，实际: %v", err)
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("rsync 超时未及时返回，耗时: %v", elapsed)
	}
}
