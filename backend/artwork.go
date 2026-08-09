// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kyuubyN/Arion-Project/backend/security"
)

const maxArtworkBytes = 6 * 1024 * 1024

type ArtworkCache struct {
	mu     sync.Mutex
	root   string
	guard  *security.SSRFGuard
	client *http.Client
}

func NewArtworkCache(audit *security.AuditLogger) (*ArtworkCache, error) {
	cacheRoot, err := arionCacheDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(cacheRoot, "artwork")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	guard := security.NewSSRFGuard(audit)
	return &ArtworkCache{root: root, guard: guard, client: guard.NewSecureHTTPClient(15 * time.Second)}, nil
}

func (c *ArtworkCache) Get(ctx context.Context, rawURL string) (string, string, error) {
	if c == nil || c.guard == nil || c.client == nil {
		return "", "", errors.New("artwork cache is unavailable")
	}
	if len(rawURL) == 0 || len(rawURL) > 8192 {
		return "", "", errors.New("invalid artwork URL")
	}
	if _, err := c.guard.ValidateURL(rawURL); err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(rawURL))
	target := filepath.Join(c.root, hex.EncodeToString(sum[:])+".img")

	c.mu.Lock()
	defer c.mu.Unlock()
	if contentType, ok := cachedArtworkType(target); ok {
		return target, contentType, nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return "", "", err
	}
	request.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/gif;q=0.8")
	response, err := c.client.Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("artwork server returned %s", response.Status)
	}
	if response.ContentLength > maxArtworkBytes {
		return "", "", errors.New("artwork exceeds the 6 MiB limit")
	}
	raw, err := security.BoundedRead(response.Body, maxArtworkBytes)
	if err != nil {
		return "", "", err
	}
	contentType := normalizedArtworkType(response.Header.Get("Content-Type"), raw)
	if contentType == "" {
		return "", "", errors.New("provider artwork is not a supported raster image")
	}
	temporary, err := os.CreateTemp(c.root, ".artwork-*.tmp")
	if err != nil {
		return "", "", err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return "", "", err
	}
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return "", "", err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return "", "", err
	}
	if err := temporary.Close(); err != nil {
		return "", "", err
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return "", "", err
	}
	return target, contentType, nil
}

func cachedArtworkType(path string) (string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxArtworkBytes {
		return "", false
	}
	prefix := make([]byte, 512)
	count, _ := file.Read(prefix)
	contentType := normalizedArtworkType("", prefix[:count])
	return contentType, contentType != ""
}

func normalizedArtworkType(header string, raw []byte) string {
	header = strings.ToLower(strings.TrimSpace(strings.Split(header, ";")[0]))
	detected := strings.ToLower(http.DetectContentType(raw))
	allowed := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/gif": true,
		"image/webp": true, "image/avif": true,
	}
	if allowed[detected] {
		return detected
	}
	if (header == "image/webp" || header == "image/avif") && len(raw) >= 12 {
		return header
	}
	return ""
}
