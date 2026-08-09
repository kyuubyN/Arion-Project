// SPDX-License-Identifier: GPL-3.0-only

package player

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

type CelluloidProvider struct {
	mu             sync.RWMutex
	execPath       string
	processManager *ProcessManager
	statusStr      string
}

func NewCelluloidProvider(pm *ProcessManager) *CelluloidProvider {
	path, _ := exec.LookPath("celluloid")
	if path == "" {
		path = "/usr/bin/celluloid"
	}
	return &CelluloidProvider{
		execPath:       path,
		processManager: pm,
		statusStr:      "idle",
	}
}

func (c *CelluloidProvider) Name() string { return "celluloid" }

func (c *CelluloidProvider) IsInstalled() bool {
	_, err := exec.LookPath("celluloid")
	return err == nil
}

func (c *CelluloidProvider) Version() string {
	if !c.IsInstalled() {
		return "Not Installed"
	}
	out, err := exec.Command("celluloid", "--version").Output()
	if err != nil {
		return "Celluloid GTK v0.24+"
	}
	lines := string(out)
	if len(lines) > 24 {
		return lines[:24]
	}
	return lines
}

func (c *CelluloidProvider) ExecutablePath() string {
	return c.execPath
}

func (c *CelluloidProvider) Capabilities() PlayerCapabilities {
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

func (c *CelluloidProvider) Launch(ctx context.Context, streamURL string, headers map[string]string, title string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.IsInstalled() {
		return fmt.Errorf("%w: celluloid", ErrPlayerNotInstalled)
	}

	// Build safe argv list with '--title' and '--' separator to prevent argument injection
	args := []string{
		fmt.Sprintf("--title=%s", title),
	}

	// Add argument separator '--' to ensure streamURL is treated strictly as a positional operand
	args = append(args, "--", streamURL)

	cmd := exec.CommandContext(ctx, c.execPath, args...)
	configureChildProcess(cmd)

	if err := cmd.Start(); err != nil {
		c.statusStr = "failed"
		return fmt.Errorf("celluloid start failed: %w", err)
	}

	c.statusStr = "playing"
	if c.processManager != nil {
		c.processManager.RegisterProcess("Celluloid", cmd)
	}

	return nil
}

func (c *CelluloidProvider) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statusStr = "stopped"
	return nil
}

func (c *CelluloidProvider) Status() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.statusStr
}
