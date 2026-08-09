// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrIndexAlreadyRunning = errors.New("media index is already running")
	ErrUnsafeMediaRoot     = errors.New("refusing to index a broad or unsafe media root")
)

var videoExtensions = map[string]bool{
	".mp4": true, ".mkv": true, ".webm": true, ".mov": true,
	".m4v": true, ".avi": true, ".mpg": true, ".mpeg": true,
}

type IndexStatus struct {
	Running      bool      `json:"running"`
	FilesFound   int       `json:"files_found"`
	FilesIndexed int       `json:"files_indexed"`
	CurrentPath  string    `json:"current_path,omitempty"`
	LastError    string    `json:"last_error,omitempty"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	FinishedAt   time.Time `json:"finished_at,omitempty"`
	FFprobeReady bool      `json:"ffprobe_ready"`
	FFmpegReady  bool      `json:"ffmpeg_ready"`
}

type MediaIndexer struct {
	mu     sync.RWMutex
	store  *GalleryStore
	status IndexStatus
}

func NewMediaIndexer(store *GalleryStore) *MediaIndexer {
	_, probeErr := exec.LookPath("ffprobe")
	_, ffmpegErr := exec.LookPath("ffmpeg")
	return &MediaIndexer{
		store:  store,
		status: IndexStatus{FFprobeReady: probeErr == nil, FFmpegReady: ffmpegErr == nil},
	}
}

func (i *MediaIndexer) Status() IndexStatus {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.status
}

func (i *MediaIndexer) Start(roots []string) error {
	i.mu.Lock()
	if i.status.Running {
		i.mu.Unlock()
		return ErrIndexAlreadyRunning
	}
	cleanRoots := uniqueCleanPaths(roots)
	for _, root := range cleanRoots {
		if err := validateMediaRoot(root); err != nil {
			i.mu.Unlock()
			return fmt.Errorf("%s: %w", root, err)
		}
	}
	i.status.Running = true
	i.status.FilesFound = 0
	i.status.FilesIndexed = 0
	i.status.CurrentPath = ""
	i.status.LastError = ""
	i.status.StartedAt = time.Now()
	i.status.FinishedAt = time.Time{}
	i.mu.Unlock()

	go i.scan(cleanRoots)
	return nil
}

func (i *MediaIndexer) scan(roots []string) {
	collections := make(map[string]*Collection)
	var scanErr error
	for _, root := range roots {
		walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil // An unreadable item must not abort the rest of the private library.
			}
			if path != root && entry.IsDir() && shouldSkipDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() || !videoExtensions[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			i.updateStatus(func(status *IndexStatus) {
				status.FilesFound++
				status.CurrentPath = path
			})
			item, itemErr := i.indexFile(root, path)
			if itemErr != nil {
				return nil
			}
			collectionRoot, collectionTitle := collectionLocation(root, path)
			collectionID := stableID("collection", collectionRoot)
			collection := collections[collectionID]
			if collection == nil {
				now := time.Now()
				collection = &Collection{
					ID:        collectionID,
					Title:     collectionTitle,
					Kind:      CollectionLocalFolder,
					SourceID:  "local",
					RootPath:  collectionRoot,
					Items:     []MediaItem{},
					AddedAt:   now,
					UpdatedAt: now,
				}
				collections[collectionID] = collection
			}
			item.CollectionID = collectionID
			collection.Items = append(collection.Items, item)
			if collection.ArtworkURL == "" && item.ThumbnailPath != "" {
				collection.ArtworkURL = "/api/media/thumbnail?id=" + item.ID
			}
			i.updateStatus(func(status *IndexStatus) { status.FilesIndexed++ })
			return nil
		})
		if walkErr != nil && scanErr == nil {
			scanErr = walkErr
		}
	}
	if scanErr == nil {
		scanErr = i.store.ReplaceLocalCollections(collections, roots)
	}
	i.updateStatus(func(status *IndexStatus) {
		status.Running = false
		status.CurrentPath = ""
		status.FinishedAt = time.Now()
		if scanErr != nil {
			status.LastError = scanErr.Error()
		}
	})
}

func (i *MediaIndexer) indexFile(root, path string) (MediaItem, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return MediaItem{}, errors.New("not a regular media file")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return MediaItem{}, err
	}
	itemID := stableID("media", abs)
	item := MediaItem{
		ID:           itemID,
		Title:        displayTitle(filepath.Base(abs)),
		Kind:         MediaVideo,
		SourceID:     "local",
		Path:         abs,
		MIMEType:     mediaMIME(abs),
		Size:         info.Size(),
		ModifiedUnix: info.ModTime().Unix(),
		AddedAt:      time.Now(),
	}
	i.probeMedia(&item)
	i.createThumbnail(&item)
	return item, nil
}

func (i *MediaIndexer) probeMedia(item *MediaItem) {
	if !i.Status().FFprobeReady {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration:stream=codec_type,width,height", "-of", "json", "--", item.Path).Output()
	if err != nil {
		return
	}
	var payload struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}
	if json.Unmarshal(output, &payload) != nil {
		return
	}
	if duration, err := strconv.ParseFloat(payload.Format.Duration, 64); err == nil {
		item.Duration = int(duration + 0.5)
	}
	for _, stream := range payload.Streams {
		if stream.CodecType == "video" {
			item.Width = stream.Width
			item.Height = stream.Height
			break
		}
	}
}

func (i *MediaIndexer) createThumbnail(item *MediaItem) {
	if !i.Status().FFmpegReady {
		return
	}
	cache, err := arionCacheDir()
	if err != nil {
		return
	}
	dir := filepath.Join(cache, "thumbnails")
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	target := filepath.Join(dir, item.ID+".jpg")
	if info, err := os.Stat(target); err == nil && info.Size() > 0 && info.ModTime().Unix() >= item.ModifiedUnix {
		item.ThumbnailPath = target
		return
	}
	seek := "5"
	if item.Duration > 0 && item.Duration < 20 {
		seek = "1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-ss", seek, "-i", item.Path, "-frames:v", "1", "-vf", "scale=640:-2", "-q:v", "3", "-y", target).Run()
	if err == nil {
		_ = os.Chmod(target, 0o600)
		item.ThumbnailPath = target
	}
}

func (i *MediaIndexer) updateStatus(fn func(*IndexStatus)) {
	i.mu.Lock()
	defer i.mu.Unlock()
	fn(&i.status)
}

func validateMediaRoot(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("media root is not a directory")
	}
	volume := filepath.VolumeName(root) + string(filepath.Separator)
	if filepath.Clean(root) == filepath.Clean(volume) {
		return ErrUnsafeMediaRoot
	}
	home, _ := os.UserHomeDir()
	if home != "" && filepath.Clean(root) == filepath.Clean(home) {
		return ErrUnsafeMediaRoot
	}
	return nil
}

func suggestedMediaRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}
	candidates := []string{filepath.Join(home, "Videos"), filepath.Join(home, "Vídeos")}
	result := make([]string, 0, len(candidates))
	for _, candidate := range uniqueCleanPaths(candidates) {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			result = append(result, candidate)
		}
	}
	return result
}

func shouldSkipDirectory(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch strings.ToLower(name) {
	case "node_modules", "vendor", "cache", "tmp", "temp":
		return true
	default:
		return false
	}
}

func collectionLocation(root, mediaPath string) (string, string) {
	rel, err := filepath.Rel(root, mediaPath)
	if err != nil {
		return root, filepath.Base(root)
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) <= 1 {
		return root, filepath.Base(root)
	}
	collectionRoot := filepath.Join(root, parts[0])
	return collectionRoot, displayTitle(parts[0])
}

func stableID(namespace, value string) string {
	sum := sha256.Sum256([]byte(namespace + "\x00" + filepath.Clean(value)))
	return namespace + "_" + hex.EncodeToString(sum[:12])
}

func displayTitle(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.NewReplacer("_", " ", ".", " ").Replace(name)
	return strings.Join(strings.Fields(name), " ")
}

func mediaMIME(path string) string {
	if value := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); value != "" {
		return value
	}
	return "video/mp4"
}
