// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGalleryStorePersistsNeutralCollectionAndProgress(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "gallery.json")
	store, err := newGalleryStoreAt(storePath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	collection := &Collection{
		ID: "collection_one", Title: "Viagem", Kind: CollectionLocalFolder,
		SourceID: "local", RootPath: "/media/viagem", AddedAt: now, UpdatedAt: now,
		Items: []MediaItem{{ID: "media_one", CollectionID: "collection_one", Title: "Praia", Kind: MediaVideo, SourceID: "local", Path: "/media/viagem/praia.mp4", AddedAt: now}},
	}
	if err := store.ReplaceLocalCollections(map[string]*Collection{collection.ID: collection}, []string{"/media"}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateProgress("media_one", 42, true); err != nil {
		t.Fatal(err)
	}
	reloaded, err := newGalleryStoreAt(storePath)
	if err != nil {
		t.Fatal(err)
	}
	item, err := reloaded.MediaItem("media_one")
	if err != nil {
		t.Fatal(err)
	}
	if item.PlaybackTime != 42 || !item.Watched {
		t.Fatalf("progress not persisted: %+v", item)
	}
	info, err := os.Stat(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("gallery permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestGalleryRejectsUnknownMedia(t *testing.T) {
	store, err := newGalleryStoreAt(filepath.Join(t.TempDir(), "gallery.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MediaItem("missing"); err != ErrMediaNotFound {
		t.Fatalf("MediaItem error = %v, want ErrMediaNotFound", err)
	}
}
