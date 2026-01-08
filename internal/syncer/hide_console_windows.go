//go:build windows
// +build windows

package syncer

import (
	"os/exec"
	"syscall"
)

// hideConsoleForCmd 设置命令行在 Windows 下无窗口运行
func hideConsoleForCmd(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	// CREATE_NO_WINDOW = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
}
