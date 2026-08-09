// SPDX-License-Identifier: GPL-3.0-only
//go:build windows

package main

import "os/exec"

func configureProviderProcess(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return cmd.Process.Kill()
	}
}
