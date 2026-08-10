// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kyuubyN/Arion-Project/backend/security"
)

type DownloadTaskStatus string

const (
	DownloadStatusPending     DownloadTaskStatus = "pending"
	DownloadStatusDownloading DownloadTaskStatus = "downloading"
	DownloadStatusCompleted   DownloadTaskStatus = "completed"
	DownloadStatusFailed      DownloadTaskStatus = "failed"
)

type DownloadTask struct {
	ID              string             `json:"id"`
	ItemID          string             `json:"item_id"`
	Title           string             `json:"title"`
	Status          DownloadTaskStatus `json:"status"`
	ProgressPercent float64            `json:"progress_percent"`
	DownloadedBytes int64              `json:"downloaded_bytes"`
	TotalBytes      int64              `json:"total_bytes"`
	SpeedMbps       float64            `json:"speed_mbps"`
	FilePath        string             `json:"file_path,omitempty"`
	Error           string             `json:"error,omitempty"`
	StartedAt       time.Time          `json:"started_at"`
	FinishedAt      time.Time          `json:"finished_at,omitempty"`
}

type DownloadManager struct {
	mu          sync.RWMutex
	tasks       map[string]*DownloadTask
	downloadDir string
	gallery     *GalleryStore
}

func NewDownloadManager(gallery *GalleryStore) *DownloadManager {
	userHome, err := os.UserHomeDir()
	if err != nil {
		userHome = "."
	}
	dir := filepath.Join(userHome, "Videos", "Arion")
	_ = os.MkdirAll(dir, 0755)

	return &DownloadManager{
		tasks:       make(map[string]*DownloadTask),
		downloadDir: dir,
		gallery:     gallery,
	}
}

type DownloadOptions struct {
	CollectionTitle string `json:"collection_title,omitempty"`
	ArtworkURL      string `json:"artwork_url,omitempty"`
}

func sanitizeFilename(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "")
	return strings.TrimSpace(r.Replace(name))
}

func (dm *DownloadManager) StartDownload(ctx context.Context, item MediaItem, options DownloadOptions, streamURL string, headers map[string]string) (*DownloadTask, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	taskID := fmt.Sprintf("dl-%d", time.Now().UnixNano())
	colTitle := options.CollectionTitle
	if colTitle == "" {
		colTitle = "Downloads do Arion"
	}
	cleanColTitle := sanitizeFilename(colTitle)
	cleanItemTitle := sanitizeFilename(item.Title)
	if cleanColTitle == "" || cleanColTitle == "." || cleanColTitle == ".." {
		cleanColTitle = "Downloads do Arion"
	}
	if cleanItemTitle == "" || cleanItemTitle == "." || cleanItemTitle == ".." {
		cleanItemTitle = "video"
	}

	animeDir := filepath.Join(dm.downloadDir, cleanColTitle)
	_ = os.MkdirAll(animeDir, 0755)

	filename := fmt.Sprintf("%s.mp4", cleanItemTitle)
	destPath := filepath.Join(animeDir, filename)

	task := &DownloadTask{
		ID:              taskID,
		ItemID:          item.ID,
		Title:           item.Title,
		Status:          DownloadStatusPending,
		ProgressPercent: 0,
		FilePath:        destPath,
		StartedAt:       time.Now(),
	}
	dm.tasks[taskID] = task

	go dm.runDownload(taskID, streamURL, headers, destPath, item, colTitle, animeDir, options.ArtworkURL)

	return task, nil
}

func (dm *DownloadManager) runDownload(taskID, streamURL string, headers map[string]string, destPath string, item MediaItem, collectionTitle, animeDir, artworkURL string) {
	dm.mu.Lock()
	task, exists := dm.tasks[taskID]
	if !exists {
		dm.mu.Unlock()
		return
	}
	task.Status = DownloadStatusDownloading
	dm.mu.Unlock()

	reqCtx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, streamURL, nil)
	if err != nil {
		dm.failTask(taskID, err)
		return
	}

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}
	if httpReq.Header.Get("Referer") == "" {
		if parsed, parseErr := url.Parse(streamURL); parseErr == nil {
			httpReq.Header.Set("Referer", fmt.Sprintf("%s://%s/", parsed.Scheme, parsed.Host))
		}
	}

	guard := security.NewSSRFGuard(nil)
	if _, err := guard.ValidateURL(streamURL); err != nil {
		dm.failTask(taskID, fmt.Errorf("unsafe media URL: %w", err))
		return
	}
	client := guard.NewSecureHTTPClient(2 * time.Hour)
	resp, err := client.Do(httpReq)
	if err != nil {
		dm.failTask(taskID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		dm.failTask(taskID, fmt.Errorf("servidor retornou HTTP %d", resp.StatusCode))
		return
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		dm.failTask(taskID, err)
		return
	}
	defer outFile.Close()

	totalBytes := resp.ContentLength
	dm.mu.Lock()
	task.TotalBytes = totalBytes
	dm.mu.Unlock()

	buf := make([]byte, 64*1024)
	var downloaded int64
	startTime := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			written, writeErr := outFile.Write(buf[:n])
			if writeErr != nil {
				dm.failTask(taskID, writeErr)
				return
			}
			downloaded += int64(written)

			elapsed := time.Since(startTime).Seconds()
			speedMbps := 0.0
			if elapsed > 0 {
				speedMbps = (float64(downloaded) * 8.0) / (elapsed * 1000000.0)
			}

			percent := 0.0
			if totalBytes > 0 {
				percent = (float64(downloaded) / float64(totalBytes)) * 100.0
			} else {
				percent = 50.0
			}

			dm.mu.Lock()
			task.DownloadedBytes = downloaded
			task.ProgressPercent = percent
			task.SpeedMbps = speedMbps
			dm.mu.Unlock()
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			dm.failTask(taskID, readErr)
			return
		}
	}

	dm.mu.Lock()
	task.Status = DownloadStatusCompleted
	task.ProgressPercent = 100.0
	task.FinishedAt = time.Now()
	dm.mu.Unlock()

	// Register downloaded video into local collection for Anime title
	if dm.gallery != nil {
		now := time.Now()
		colID := stableID("collection", animeDir)
		localItem := MediaItem{
			ID:           stableID("media", destPath),
			CollectionID: colID,
			Title:        item.Title,
			Kind:         MediaVideo,
			SourceID:     "local",
			Path:         destPath,
			MIMEType:     "video/mp4",
			Size:         downloaded,
			AddedAt:      now,
		}
		_ = dm.gallery.UpsertDownloadedItem(localItem, collectionTitle, animeDir, artworkURL)
	}
}

func (dm *DownloadManager) failTask(taskID string, err error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if task, exists := dm.tasks[taskID]; exists {
		task.Status = DownloadStatusFailed
		task.Error = err.Error()
		task.FinishedAt = time.Now()
	}
}

func (dm *DownloadManager) GetTask(taskID string) (*DownloadTask, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	task, exists := dm.tasks[taskID]
	if !exists {
		return nil, false
	}
	copyTask := *task
	return &copyTask, true
}

func (dm *DownloadManager) ActiveTasks() []DownloadTask {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	list := make([]DownloadTask, 0, len(dm.tasks))
	for _, task := range dm.tasks {
		list = append(list, *task)
	}
	return list
}
