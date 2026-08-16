// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type SettingsStore struct {
	mu       sync.RWMutex
	filePath string
	value    AppSettings
}

func NewSettingsStore() (*SettingsStore, error) {
	dir, err := arionConfigDir()
	if err != nil {
		return nil, err
	}
	store := &SettingsStore{
		filePath: filepath.Join(dir, "settings-v2.json"),
		value: AppSettings{
			Language:      "en",
			DefaultPlayer: "integrated",
			MediaRoots:    []string{},
			Providers:     []ProviderInstallation{},
			Privacy: PrivacySettings{
				TelemetryEnabled: false,
				WebSessionMode:   "private",
				KeepWebHistory:   false,
			},
		},
	}
	if data, readErr := os.ReadFile(store.filePath); readErr == nil {
		_ = json.Unmarshal(data, &store.value)
	} else if !os.IsNotExist(readErr) {
		return nil, readErr
	} else if err := writeJSONAtomic(store.filePath, store.value); err != nil {
		return nil, err
	}
	store.normalizeLocked()
	return store, nil
}

func (s *SettingsStore) Get() AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSettings(s.value)
}

func (s *SettingsStore) Save(value AppSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = cloneSettings(value)
	s.normalizeLocked()
	return writeJSONAtomic(s.filePath, s.value)
}

func (s *SettingsStore) AddProvider(provider ProviderInstallation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for i := range s.value.Providers {
		if s.value.Providers[i].ID == provider.ID {
			if providerIdentity(s.value.Providers[i]) != providerIdentity(provider) {
				return errors.New("provider id is already registered by another installation; remove it before replacing it")
			}
			s.value.Providers[i] = provider
			found = true
			break
		}
	}
	if !found {
		s.value.Providers = append(s.value.Providers, provider)
	}
	s.normalizeLocked()
	return writeJSONAtomic(s.filePath, s.value)
}

func (s *SettingsStore) RemoveProvider(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	providers := s.value.Providers[:0]
	for _, provider := range s.value.Providers {
		if provider.ID != id {
			providers = append(providers, provider)
		}
	}
	if len(providers) == len(s.value.Providers) {
		return errors.New("provider not found")
	}
	s.value.Providers = providers
	return writeJSONAtomic(s.filePath, s.value)
}

func (s *SettingsStore) normalizeLocked() {
	if s.value.Language != "en" && s.value.Language != "pt-BR" && s.value.Language != "pt" {
		s.value.Language = "en"
	}
	if s.value.Language == "pt" {
		s.value.Language = "pt-BR"
	}
	if s.value.DefaultPlayer == "" {
		s.value.DefaultPlayer = "integrated"
	}
	if s.value.Privacy.WebSessionMode != "persistent" {
		s.value.Privacy.WebSessionMode = "private"
	}
	// Arion has no telemetry transport. Keep this field explicit and fail closed.
	s.value.Privacy.TelemetryEnabled = false
	s.value.MediaRoots = uniqueCleanPaths(s.value.MediaRoots)
	for i := range s.value.Providers {
		if s.value.Providers[i].Kind == "" {
			s.value.Providers[i].Kind = ProviderLocalProcess
		}
	}
	sort.SliceStable(s.value.Providers, func(i, j int) bool {
		return s.value.Providers[i].Name < s.value.Providers[j].Name
	})
}

func providerIdentity(provider ProviderInstallation) string {
	if provider.Kind == ProviderWebsite {
		return string(ProviderWebsite) + ":" + provider.Origin
	}
	return string(ProviderLocalProcess) + ":" + provider.ManifestPath
}

func cloneSettings(value AppSettings) AppSettings {
	value.MediaRoots = append([]string(nil), value.MediaRoots...)
	value.Providers = append([]ProviderInstallation(nil), value.Providers...)
	for i := range value.Providers {
		value.Providers[i].Capabilities = append([]string(nil), value.Providers[i].Capabilities...)
	}
	return value
}

func uniqueCleanPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		result = append(result, abs)
	}
	return result
}
