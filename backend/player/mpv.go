// SPDX-License-Identifier: GPL-3.0-only

package player

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

type MPVProvider struct {
	mu             sync.RWMutex
	execPath       string
	processManager *ProcessManager
	statusStr      string
}

func NewMPVProvider(pm *ProcessManager) *MPVProvider {
	path, _ := exec.LookPath("mpv")
	if path == "" {
		path = "mpv"
	}
	return &MPVProvider{
		execPath:       path,
		processManager: pm,
		statusStr:      "idle",
	}
}

func (m *MPVProvider) Name() string { return "mpv" }

func (m *MPVProvider) IsInstalled() bool {
	_, err := exec.LookPath("mpv")
	return err == nil
}

func (m *MPVProvider) Version() string {
	if !m.IsInstalled() {
		return "Not Installed"
	}
	out, err := exec.Command("mpv", "--version").Output()
	if err != nil {
		return "MPV v0.35+"
	}
	lines := string(out)
	if len(lines) > 20 {
		return lines[:20]
	}
	return lines
}

func (m *MPVProvider) ExecutablePath() string {
	return m.execPath
}

func (m *MPVProvider) Capabilities() PlayerCapabilities {
	return PlayerCapabilities{
		Video:                true,
		Audio:                true,
		Subtitles:            true,
		Fullscreen:           true,
		ExternalProcess:      true,
		HardwareAcceleration: true,
		Seek:                 true,
		Volume:               true,
		Pause:                true,
	}
}

func (m *MPVProvider) Launch(ctx context.Context, streamURL string, headers map[string]string, title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.IsInstalled() {
		return fmt.Errorf("%w: mpv", ErrPlayerNotInstalled)
	}

	// Build safe argv list using strict flag allowlist and '--' separator to prevent argument injection
	args := []string{
		"--no-terminal",
		"--really-quiet",
		fmt.Sprintf("--force-media-title=%s", title),
	}

	for k, v := range headers {
		args = append(args, fmt.Sprintf("--http-header-fields=%s: %s", k, v))
	}

	// Add argument separator '--' to ensure streamURL is treated strictly as a positional operand
	args = append(args, "--", streamURL)

	cmd := exec.CommandContext(ctx, m.execPath, args...)
	configureChildProcess(cmd)

	if err := cmd.Start(); err != nil {
		m.statusStr = "failed"
		return fmt.Errorf("mpv start failed: %w", err)
	}

	m.statusStr = "playing"
	if m.processManager != nil {
		m.processManager.RegisterProcess("MPV", cmd)
	}

	return nil
}

func (m *MPVProvider) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusStr = "stopped"
	return nil
}

func (m *MPVProvider) Status() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statusStr
}
