// SPDX-License-Identifier: GPL-3.0-only
//go:build windows

package player

import (
	"os/exec"
	"syscall"
)

func configureChildProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func stopChildProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	// Go cannot deliver POSIX termination signals on Windows. TerminateProcess
	// is deterministic and prevents an external player from surviving Arion.
	return cmd.Process.Kill()
}
