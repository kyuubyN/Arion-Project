// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIndexerGroupsTopLevelFoldersIntoCollections(t *testing.T) {
	root := t.TempDir()
	family := filepath.Join(root, "Família")
	if err := os.MkdirAll(family, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(family, "festa.mp4"), []byte("test video placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(family, "not-media.txt"), []byte("ignored"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := newGalleryStoreAt(filepath.Join(t.TempDir(), "gallery.json"))
	if err != nil {
		t.Fatal(err)
	}
	indexer := NewMediaIndexer(store)
	// Avoid invoking external codecs for the synthetic fixture.
	indexer.status.FFprobeReady = false
	indexer.status.FFmpegReady = false
	if err := indexer.Start([]string{root}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for indexer.Status().Running && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	collections := store.Collections()
	if len(collections) != 1 {
		t.Fatalf("collections = %d, want 1", len(collections))
	}
	if collections[0].Title != "Família" || len(collections[0].Items) != 1 {
		t.Fatalf("unexpected collection: %+v", collections[0])
	}
}

func TestIndexerRefusesFilesystemRoot(t *testing.T) {
	if err := validateMediaRoot(string(filepath.Separator)); err != ErrUnsafeMediaRoot {
		t.Fatalf("validateMediaRoot error = %v, want ErrUnsafeMediaRoot", err)
	}
}
