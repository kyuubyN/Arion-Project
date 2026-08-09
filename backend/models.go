// SPDX-License-Identifier: GPL-3.0-only

package main

import "time"

const gallerySchemaVersion = 1

type CollectionKind string

const (
	CollectionLocalFolder CollectionKind = "local_folder"
	CollectionWeb         CollectionKind = "web"
	CollectionProvider    CollectionKind = "provider"
)

type MediaKind string

const (
	MediaVideo      MediaKind = "video"
	MediaShortVideo MediaKind = "short_video"
	MediaEpisode    MediaKind = "episode"
	MediaWebVideo   MediaKind = "web_video"
)

// Collection is the provider-neutral unit displayed in the Arion gallery.
// Local folders, web channels and provider collections all use the same shape
// and differ only by Kind and SourceID.
type Collection struct {
	ID                string         `json:"id"`
	Title             string         `json:"title"`
	Kind              CollectionKind `json:"kind"`
	SourceID          string         `json:"source_id"`
	RootPath          string         `json:"root_path,omitempty"`
	Description       string         `json:"description,omitempty"`
	ArtworkURL        string         `json:"artwork_url,omitempty"`
	ProviderReference string         `json:"provider_reference,omitempty"`
	Items             []MediaItem    `json:"items"`
	AddedAt           time.Time      `json:"added_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type MediaItem struct {
	ID                string    `json:"id"`
	CollectionID      string    `json:"collection_id"`
	Title             string    `json:"title"`
	Kind              MediaKind `json:"kind"`
	SourceID          string    `json:"source_id"`
	Path              string    `json:"path,omitempty"`
	URL               string    `json:"url,omitempty"`
	ProviderReference string    `json:"provider_reference,omitempty"`
	MIMEType          string    `json:"mime_type,omitempty"`
	ThumbnailPath     string    `json:"-"`
	Duration          int       `json:"duration_seconds,omitempty"`
	Width             int       `json:"width,omitempty"`
	Height            int       `json:"height,omitempty"`
	Size              int64     `json:"size_bytes,omitempty"`
	ModifiedUnix      int64     `json:"modified_unix,omitempty"`
	PlaybackTime      int       `json:"playback_time,omitempty"`
	Watched           bool      `json:"watched"`
	AddedAt           time.Time `json:"added_at"`
}

type GalleryData struct {
	SchemaVersion int                    `json:"schema_version"`
	Collections   map[string]*Collection `json:"collections"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type PrivacySettings struct {
	TelemetryEnabled bool   `json:"telemetry_enabled"`
	WebSessionMode   string `json:"web_session_mode"`
	KeepWebHistory   bool   `json:"keep_web_history"`
}

type ProviderKind string

const (
	ProviderLocalProcess ProviderKind = "local_process"
	ProviderWebsite      ProviderKind = "website"
)

type ProviderInstallation struct {
	Kind         ProviderKind `json:"kind,omitempty"`
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	ManifestPath string       `json:"manifest_path,omitempty"`
	RootPath     string       `json:"root_path,omitempty"`
	Executable   string       `json:"executable,omitempty"`
	Origin       string       `json:"origin,omitempty"`
	ManifestURL  string       `json:"manifest_url,omitempty"`
	RPCURL       string       `json:"rpc_url,omitempty"`
	Capabilities []string     `json:"capabilities,omitempty"`
	Enabled      bool         `json:"enabled"`
}

type AppSettings struct {
	DefaultPlayer string                 `json:"default_player"`
	MediaRoots    []string               `json:"media_roots"`
	Providers     []ProviderInstallation `json:"providers"`
	Privacy       PrivacySettings        `json:"privacy"`
}
