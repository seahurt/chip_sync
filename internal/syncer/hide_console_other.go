//go:build !windows
// +build !windows

package syncer

import "os/exec"

// hideConsoleForCmd 在非 Windows 平台为 no-op
func hideConsoleForCmd(cmd *exec.Cmd) {}
