// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var ErrMediaNotFound = errors.New("media item not found")

type GalleryStore struct {
	mu       sync.RWMutex
	filePath string
	data     GalleryData
}

func NewGalleryStore() (*GalleryStore, error) {
	dir, err := arionConfigDir()
	if err != nil {
		return nil, err
	}
	return newGalleryStoreAt(filepath.Join(dir, "gallery.json"))
}

func newGalleryStoreAt(filePath string) (*GalleryStore, error) {
	store := &GalleryStore{
		filePath: filePath,
		data: GalleryData{
			SchemaVersion: gallerySchemaVersion,
			Collections:   make(map[string]*Collection),
			UpdatedAt:     time.Now(),
		},
	}
	if raw, readErr := os.ReadFile(store.filePath); readErr == nil {
		if err := json.Unmarshal(raw, &store.data); err != nil {
			return nil, err
		}
		if store.data.Collections == nil {
			store.data.Collections = make(map[string]*Collection)
		}
	} else if !os.IsNotExist(readErr) {
		return nil, readErr
	} else if err := store.saveLocked(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *GalleryStore) Collections() []*Collection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Collection, 0, len(s.data.Collections))
	for _, collection := range s.data.Collections {
		result = append(result, cloneCollection(collection))
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

func (s *GalleryStore) Collection(id string) *Collection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneCollection(s.data.Collections[id])
}

func (s *GalleryStore) ReplaceLocalCollections(collections map[string]*Collection, scannedRoots []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	roots := uniqueCleanPaths(scannedRoots)
	for id, existing := range s.data.Collections {
		if existing.Kind != CollectionLocalFolder || !pathUnderAnyRoot(existing.RootPath, roots) {
			continue
		}
		if _, present := collections[id]; !present {
			delete(s.data.Collections, id)
		}
	}
	for id, incoming := range collections {
		if previous := s.data.Collections[id]; previous != nil {
			progress := make(map[string]MediaItem, len(previous.Items))
			for _, item := range previous.Items {
				progress[item.ID] = item
			}
			for i := range incoming.Items {
				if old, ok := progress[incoming.Items[i].ID]; ok {
					incoming.Items[i].PlaybackTime = old.PlaybackTime
					incoming.Items[i].Watched = old.Watched
					incoming.Items[i].AddedAt = old.AddedAt
				}
			}
			incoming.AddedAt = previous.AddedAt
		}
		s.data.Collections[id] = cloneCollection(incoming)
	}
	s.data.UpdatedAt = time.Now()
	return s.saveLocked()
}

func (s *GalleryStore) RemoveCollection(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Collections, id)
	s.data.UpdatedAt = time.Now()
	return s.saveLocked()
}

func (s *GalleryStore) UpsertProviderCollection(collection *Collection) error {
	if collection == nil || collection.ID == "" || collection.Kind != CollectionProvider || collection.SourceID == "" {
		return errors.New("invalid provider collection")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if previous := s.data.Collections[collection.ID]; previous != nil {
		progress := make(map[string]MediaItem, len(previous.Items))
		for _, item := range previous.Items {
			progress[item.ID] = item
		}
		for index := range collection.Items {
			if old, present := progress[collection.Items[index].ID]; present {
				collection.Items[index].PlaybackTime = old.PlaybackTime
				collection.Items[index].Watched = old.Watched
				collection.Items[index].AddedAt = old.AddedAt
			}
		}
		collection.AddedAt = previous.AddedAt
	}
	s.data.Collections[collection.ID] = cloneCollection(collection)
	s.data.UpdatedAt = time.Now()
	return s.saveLocked()
}

func (s *GalleryStore) UpdateProgress(itemID string, seconds int, watched bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, collection := range s.data.Collections {
		for i := range collection.Items {
			if collection.Items[i].ID != itemID {
				continue
			}
			if seconds < 0 {
				seconds = 0
			}
			collection.Items[i].PlaybackTime = seconds
			collection.Items[i].Watched = watched
			collection.UpdatedAt = time.Now()
			s.data.UpdatedAt = time.Now()
			return s.saveLocked()
		}
	}
	return ErrMediaNotFound
}

func (s *GalleryStore) MediaItem(id string) (MediaItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, collection := range s.data.Collections {
		for _, item := range collection.Items {
			if item.ID == id {
				return item, nil
			}
		}
	}
	return MediaItem{}, ErrMediaNotFound
}

func (s *GalleryStore) saveLocked() error {
	s.data.SchemaVersion = gallerySchemaVersion
	return writeJSONAtomic(s.filePath, s.data)
}

func cloneCollection(collection *Collection) *Collection {
	if collection == nil {
		return nil
	}
	clone := *collection
	clone.Items = append([]MediaItem(nil), collection.Items...)
	return &clone
}

func pathUnderAnyRoot(path string, roots []string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, path)
		if err == nil && rel != ".." && rel != "" && !startsWithParent(rel) {
			return true
		}
		if err == nil && rel == "." {
			return true
		}
	}
	return false
}

func startsWithParent(rel string) bool {
	return len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator)
}
