// SPDX-License-Identifier: GPL-3.0-only

package player

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
)

var (
	ErrPlayerNotInstalled = errors.New("player: requested video player binary is not installed on this system")
	ErrPlayerStartFailed  = errors.New("player: failed to launch external player process")
)

type PlayerCapabilities struct {
	Video                bool `json:"video"`
	Audio                bool `json:"audio"`
	Subtitles            bool `json:"subtitles"`
	Fullscreen           bool `json:"fullscreen"`
	ExternalProcess      bool `json:"external_process"`
	HardwareAcceleration bool `json:"hardware_acceleration"`
	Seek                 bool `json:"seek"`
	Volume               bool `json:"volume"`
	Pause                bool `json:"pause"`
}

type PlayerProvider interface {
	Name() string
	IsInstalled() bool
	Version() string
	ExecutablePath() string
	Capabilities() PlayerCapabilities
	Launch(ctx context.Context, streamURL string, headers map[string]string, title string) error
	Stop() error
	Status() string
}

type PlayerService struct {
	mu             sync.RWMutex
	providers      map[string]PlayerProvider
	processManager *ProcessManager
	activeProvider PlayerProvider
}

func NewPlayerService(pm *ProcessManager) *PlayerService {
	ps := &PlayerService{
		providers:      make(map[string]PlayerProvider),
		processManager: pm,
	}

	// Register built-in providers
	ps.RegisterProvider(NewIntegratedPlayer())
	ps.RegisterProvider(NewMPVProvider(pm))
	if runtime.GOOS == "linux" {
		ps.RegisterProvider(NewCelluloidProvider(pm))
	}

	return ps
}

func (s *PlayerService) RegisterProvider(p PlayerProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[p.Name()] = p
}

func (s *PlayerService) GetProvider(name string) (PlayerProvider, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.providers[name]
	return p, ok
}

func (s *PlayerService) ListProviders() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{})
	for name, p := range s.providers {
		result[name] = map[string]interface{}{
			"name":            p.Name(),
			"installed":       p.IsInstalled(),
			"version":         p.Version(),
			"executable_path": p.ExecutablePath(),
			"capabilities":    p.Capabilities(),
			"status":          p.Status(),
		}
	}
	return result
}

func (s *PlayerService) Play(ctx context.Context, providerName, streamURL string, headers map[string]string, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.providers[providerName]
	if !ok {
		// Fallback to MPV or Integrated
		if mpv, exists := s.providers["mpv"]; exists && mpv.IsInstalled() {
			p = mpv
		} else {
			p = s.providers["integrated"]
		}
	}

	if !p.IsInstalled() && p.Name() != "integrated" {
		return fmt.Errorf("%w: %s", ErrPlayerNotInstalled, p.Name())
	}

	// Stop current active provider if running
	if s.activeProvider != nil && s.activeProvider.Status() == "playing" {
		_ = s.activeProvider.Stop()
	}

	s.activeProvider = p
	return p.Launch(ctx, streamURL, headers, title)
}

func (s *PlayerService) TestPlayer(ctx context.Context, providerName string) error {
	p, ok := s.GetProvider(providerName)
	if !ok {
		return fmt.Errorf("player: provider %s not found", providerName)
	}

	if !p.IsInstalled() {
		return fmt.Errorf("player: provider %s is not installed", providerName)
	}

	// Controlled test stream URL
	testURL := "https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8"
	return p.Launch(ctx, testURL, nil, "ARION Player Test - "+p.Name())
}
