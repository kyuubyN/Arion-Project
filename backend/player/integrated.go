// SPDX-License-Identifier: GPL-3.0-only

package player

import (
	"context"
	"sync"
)

type IntegratedPlayer struct {
	mu        sync.RWMutex
	statusStr string
}

func NewIntegratedPlayer() *IntegratedPlayer {
	return &IntegratedPlayer{
		statusStr: "idle",
	}
}

func (i *IntegratedPlayer) Name() string { return "integrated" }

func (i *IntegratedPlayer) IsInstalled() bool { return true }

func (i *IntegratedPlayer) Version() string { return "Web HLS v1.5" }

func (i *IntegratedPlayer) ExecutablePath() string { return "Built-in HTML5 Web Player" }

func (i *IntegratedPlayer) Capabilities() PlayerCapabilities {
	return PlayerCapabilities{
		Video:                true,
		Audio:                true,
		Subtitles:            true,
		Fullscreen:           true,
		ExternalProcess:      false,
		HardwareAcceleration: true,
		Seek:                 true,
		Volume:               true,
		Pause:                true,
	}
}

func (i *IntegratedPlayer) Launch(ctx context.Context, streamURL string, headers map[string]string, title string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.statusStr = "playing"
	return nil
}

func (i *IntegratedPlayer) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.statusStr = "stopped"
	return nil
}

func (i *IntegratedPlayer) Status() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.statusStr
}
