// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	providerManifestName = "arion-provider.json"
	providerProtocol     = 1
	maxManifestDepth     = 4
	maxManifestBytes     = 1024 * 1024
)

var providerIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)

var providerCapabilities = map[string]struct{}{
	"catalog.search":     {},
	"collection.resolve": {},
	"item.resolve":       {},
}

type ProviderManifest struct {
	SchemaVersion   int      `json:"schema_version"`
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	ProtocolVersion int      `json:"protocol_version"`
	Executable      string   `json:"executable"`
	Capabilities    []string `json:"capabilities"`
	Author          string   `json:"author,omitempty"`
	License         string   `json:"license,omitempty"`
	Homepage        string   `json:"homepage,omitempty"`
}

type ProviderCandidate struct {
	Manifest     ProviderManifest `json:"manifest"`
	ManifestPath string           `json:"manifest_path"`
	RootPath     string           `json:"root_path"`
	Executable   string           `json:"executable,omitempty"`
	Ready        bool             `json:"ready"`
	Status       string           `json:"status"`
}

func DiscoverProviders(root string) ([]ProviderCandidate, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := validateProviderRoot(absRoot); err != nil {
		return nil, err
	}
	candidates := []ProviderCandidate{}
	err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			return nil
		}
		depth := 0
		if rel != "." {
			depth = len(strings.Split(rel, string(filepath.Separator)))
		}
		if entry.IsDir() {
			if path != absRoot && (depth > maxManifestDepth || shouldSkipProviderDirectory(entry.Name())) {
				return filepath.SkipDir
			}
			return nil
		}
		if depth > maxManifestDepth+1 || entry.Name() != providerManifestName {
			return nil
		}
		candidate, readErr := ReadProviderManifest(path)
		if readErr == nil {
			candidates = append(candidates, candidate)
		}
		return nil
	})
	return candidates, err
}

func ReadProviderManifest(path string) (ProviderCandidate, error) {
	absManifest, err := filepath.Abs(path)
	if err != nil {
		return ProviderCandidate{}, err
	}
	file, err := os.Open(absManifest)
	if err != nil {
		return ProviderCandidate{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxManifestBytes))
	decoder.DisallowUnknownFields()
	var manifest ProviderManifest
	if err := decoder.Decode(&manifest); err != nil {
		return ProviderCandidate{}, fmt.Errorf("invalid provider manifest: %w", err)
	}
	if err := validateProviderManifest(manifest); err != nil {
		return ProviderCandidate{}, err
	}
	root := filepath.Dir(absManifest)
	executable := ""
	ready := false
	status := "Executável não informado"
	if manifest.Executable != "" {
		executable, err = resolveProviderExecutable(root, manifest.Executable)
		if err != nil {
			status = err.Error()
		} else if info, statErr := os.Stat(executable); statErr != nil {
			status = "Código-fonte detectado; o executável ainda precisa ser preparado"
		} else if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			status = "Executável sem permissão de execução"
		} else {
			ready = true
			status = "Pronto para ativar"
		}
	}
	return ProviderCandidate{
		Manifest: manifest, ManifestPath: absManifest, RootPath: root,
		Executable: executable, Ready: ready, Status: status,
	}, nil
}

func validateProviderManifest(manifest ProviderManifest) error {
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("unsupported manifest schema %d", manifest.SchemaVersion)
	}
	if manifest.ProtocolVersion != providerProtocol {
		return fmt.Errorf("unsupported provider protocol %d", manifest.ProtocolVersion)
	}
	if !providerIDPattern.MatchString(manifest.ID) {
		return errors.New("provider id must use lowercase letters, numbers, dots, dashes or underscores")
	}
	if strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Version) == "" {
		return errors.New("provider name and version are required")
	}
	seenCapabilities := make(map[string]struct{}, len(manifest.Capabilities))
	for _, capability := range manifest.Capabilities {
		if _, supported := providerCapabilities[capability]; !supported {
			return fmt.Errorf("unsupported provider capability %q", capability)
		}
		if _, duplicate := seenCapabilities[capability]; duplicate {
			return fmt.Errorf("duplicate provider capability %q", capability)
		}
		seenCapabilities[capability] = struct{}{}
	}
	return nil
}

func resolveProviderExecutable(root, declared string) (string, error) {
	if filepath.IsAbs(declared) {
		return "", errors.New("provider executable must be relative to its manifest")
	}
	resolved, err := filepath.Abs(filepath.Join(root, declared))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || startsWithParent(rel) {
		return "", errors.New("provider executable escapes its manifest directory")
	}
	if _, statErr := os.Stat(resolved); statErr == nil {
		realRoot, rootErr := filepath.EvalSymlinks(root)
		realExecutable, executableErr := filepath.EvalSymlinks(resolved)
		if rootErr != nil || executableErr != nil {
			return "", errors.New("provider executable symlink could not be validated")
		}
		realRel, relErr := filepath.Rel(realRoot, realExecutable)
		if relErr != nil || realRel == ".." || startsWithParent(realRel) {
			return "", errors.New("provider executable symlink escapes its manifest directory")
		}
	}
	return resolved, nil
}

func validateProviderRoot(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("provider search path is not a directory")
	}
	volume := filepath.VolumeName(root) + string(filepath.Separator)
	if filepath.Clean(root) == filepath.Clean(volume) {
		return errors.New("refusing to scan an entire filesystem for providers")
	}
	return nil
}

func shouldSkipProviderDirectory(name string) bool {
	if name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" {
		return true
	}
	return strings.HasPrefix(name, ".")
}

func candidateInstallation(candidate ProviderCandidate) ProviderInstallation {
	return ProviderInstallation{
		Kind: ProviderLocalProcess,
		ID:   candidate.Manifest.ID, Name: candidate.Manifest.Name,
		Version: candidate.Manifest.Version, ManifestPath: candidate.ManifestPath,
		RootPath: candidate.RootPath, Executable: candidate.Executable,
		Capabilities: append([]string(nil), candidate.Manifest.Capabilities...),
		Enabled:      candidate.Ready,
	}
}
