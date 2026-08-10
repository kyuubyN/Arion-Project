// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kyuubyN/Arion-Project/backend/player"
	"github.com/kyuubyN/Arion-Project/backend/security"
)

const maxJSONBody = 1024 * 1024

type API struct {
	gallery      *GalleryStore
	settings     *SettingsStore
	indexer      *MediaIndexer
	players      *player.PlayerService
	processes    *player.ProcessManager
	audit        *security.AuditLogger
	artwork      *ArtworkCache
	downloader   *DownloadManager
	sessionToken string
	shutdown     chan struct{}
}

func NewAPI(gallery *GalleryStore, settings *SettingsStore, indexer *MediaIndexer, players *player.PlayerService, processes *player.ProcessManager, audit *security.AuditLogger, artwork *ArtworkCache, token string, shutdown chan struct{}) *API {
	dm := NewDownloadManager(gallery)
	return &API{gallery: gallery, settings: settings, indexer: indexer, players: players, processes: processes, audit: audit, artwork: artwork, downloader: dm, sessionToken: token, shutdown: shutdown}
}

func (a *API) Routes(frontendDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/collections", a.handleCollections)
	mux.HandleFunc("/api/collections/remove", a.handleRemoveCollection)
	mux.HandleFunc("/api/index/suggestions", a.handleIndexSuggestions)
	mux.HandleFunc("/api/index/scan", a.handleIndexScan)
	mux.HandleFunc("/api/index/status", a.handleIndexStatus)
	mux.HandleFunc("/api/media/local", a.handleLocalMedia)
	mux.HandleFunc("/api/media/stream", a.handleMediaStreamProxy)
	mux.HandleFunc("/api/media/download", a.handleStartDownloadTask)
	mux.HandleFunc("/api/media/download/status", a.handleDownloadTaskStatus)
	mux.HandleFunc("/api/media/thumbnail", a.handleThumbnail)
	mux.HandleFunc("/api/media/artwork", a.handleArtwork)
	mux.HandleFunc("/api/history/update", a.handleUpdateProgress)
	mux.HandleFunc("/api/settings", a.handleSettings)
	mux.HandleFunc("/api/providers", a.handleProviders)
	mux.HandleFunc("/api/providers/discover", a.handleDiscoverProviders)
	mux.HandleFunc("/api/providers/register", a.handleRegisterProvider)
	mux.HandleFunc("/api/providers/web/probe", a.handleProbeWebsiteProvider)
	mux.HandleFunc("/api/providers/web/register", a.handleRegisterWebsiteProvider)
	mux.HandleFunc("/api/providers/health", a.handleProviderHealth)
	mux.HandleFunc("/api/providers/search", a.handleProviderSearch)
	mux.HandleFunc("/api/providers/collection", a.handleProviderCollection)
	mux.HandleFunc("/api/providers/import", a.handleProviderImport)
	mux.HandleFunc("/api/providers/item", a.handleProviderItem)
	mux.HandleFunc("/api/providers/play", a.handleProviderPlay)
	mux.HandleFunc("/api/player/list", a.handlePlayers)
	mux.HandleFunc("/api/player/play", a.handlePlay)
	mux.HandleFunc("/api/open-folder", a.handleOpenFolder)
	mux.HandleFunc("/api/security/events", a.handleSecurityEvents)
	mux.HandleFunc("/api/shutdown", a.handleShutdown)

	files := http.FileServer(http.Dir(frontendDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; media-src 'self' blob:; style-src 'self'; script-src 'self'; connect-src 'self'; frame-src 'none'; object-src 'none'; base-uri 'none'; form-action 'self'")
		files.ServeHTTP(w, r)
	})
	return a.securityMiddleware(mux)
}

func (a *API) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			http.Error(w, `{"error":"invalid host"}`, http.StatusForbidden)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/health" {
			provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if provided == "" && (r.URL.Path == "/api/media/local" || r.URL.Path == "/api/media/stream" || r.URL.Path == "/api/media/thumbnail" || r.URL.Path == "/api/media/artwork") {
				provided = r.URL.Query().Get("session")
			}
			if len(provided) != len(a.sessionToken) || subtle.ConstantTimeCompare([]byte(provided), []byte(a.sessionToken)) != 1 {
				a.audit.LogEvent(security.EventAuthFailure, security.SeverityWarning, "LocalAPI", "invalid_or_missing_session", r.URL.Path)
				http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" && !sameLocalOrigin(origin, r.Host) {
				http.Error(w, `{"error":"cross-origin request denied"}`, http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) handleHealth(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok", "product": "Arion Media Gallery", "schema_version": gallerySchemaVersion})
}

func (a *API) handleCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	collections := a.gallery.Collections()
	jsonResponse(w, http.StatusOK, map[string]any{"collections": collections})

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, col := range collections {
			if (col.ArtworkURL == "" || col.Description == "") && col.Title != "" {
				if aniMeta, err := FetchAniListMetadata(ctx, col.Title); err == nil && aniMeta != nil {
					if col.ArtworkURL == "" && aniMeta.ArtworkURL != "" {
						col.ArtworkURL = aniMeta.ArtworkURL
					}
					if col.Description == "" && aniMeta.Description != "" {
						col.Description = aniMeta.Description
					}
					_ = a.gallery.UpsertProviderCollection(col)
				}
			}
		}
	}()
}

func (a *API) handleRemoveCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}
	if err := a.gallery.RemoveCollection(r.URL.Query().Get("id")); err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]bool{"removed": true})
}

func (a *API) handleIndexSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"suggestions": suggestedMediaRoots()})
}

func (a *API) handleIndexScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Roots []string `json:"roots"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	if len(request.Roots) == 0 {
		jsonError(w, http.StatusBadRequest, errors.New("select at least one media folder"))
		return
	}
	settings := a.settings.Get()
	settings.MediaRoots = uniqueCleanPaths(append(settings.MediaRoots, request.Roots...))
	if err := a.settings.Save(settings); err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}
	if err := a.indexer.Start(settings.MediaRoots); err != nil {
		jsonError(w, http.StatusConflict, err)
		return
	}
	jsonResponse(w, http.StatusAccepted, a.indexer.Status())
}

func (a *API) handleIndexStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	jsonResponse(w, http.StatusOK, a.indexer.Status())
}

func (a *API) handleLocalMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	item, err := a.gallery.MediaItem(r.URL.Query().Get("id"))
	if err != nil || item.Path == "" {
		jsonError(w, http.StatusNotFound, ErrMediaNotFound)
		return
	}
	file, err := os.Open(item.Path)
	if err != nil {
		jsonError(w, http.StatusNotFound, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		jsonError(w, http.StatusNotFound, ErrMediaNotFound)
		return
	}
	w.Header().Set("Content-Type", item.MIMEType)
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, filepath.Base(item.Path), info.ModTime(), file)
}

func (a *API) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	item, err := a.gallery.MediaItem(r.URL.Query().Get("id"))
	if err != nil || item.ThumbnailPath == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(item.ThumbnailPath)))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, item.ThumbnailPath)
}

func (a *API) handleArtwork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	path, contentType, err := a.artwork.Get(r.Context(), r.URL.Query().Get("url"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "artwork", info.ModTime(), file)
}

func (a *API) handleUpdateProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ItemID  string `json:"item_id"`
		Seconds int    `json:"seconds"`
		Watched bool   `json:"watched"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	if err := a.gallery.UpdateProgress(request.ItemID, request.Seconds, request.Watched); err != nil {
		jsonError(w, http.StatusNotFound, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]bool{"updated": true})
}

func (a *API) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, http.StatusOK, a.settings.Get())
	case http.MethodPost:
		var value AppSettings
		if err := decodeJSON(r, &value); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		if err := a.settings.Save(value); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		jsonResponse(w, http.StatusOK, a.settings.Get())
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *API) handleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, http.StatusOK, map[string]any{"providers": a.settings.Get().Providers, "protocol_version": providerProtocol})
	case http.MethodDelete:
		if err := a.settings.RemoveProvider(r.URL.Query().Get("id")); err != nil {
			jsonError(w, http.StatusNotFound, err)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]bool{"removed": true})
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodDelete)
	}
}

func (a *API) handleDiscoverProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	candidates, err := DiscoverProviders(request.Path)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"candidates": candidates})
}

func (a *API) handleRegisterProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ManifestPath string `json:"manifest_path"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	candidate, err := ReadProviderManifest(request.ManifestPath)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	installation := candidateInstallation(candidate)
	if err := a.settings.AddProvider(installation); err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusOK, installation)
}

func (a *API) handleProbeWebsiteProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		URL string `json:"url"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	candidate, err := ProbeWebsiteProvider(r.Context(), request.URL, a.audit)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	jsonResponse(w, http.StatusOK, candidate)
}

func (a *API) handleRegisterWebsiteProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		URL         string `json:"url"`
		Fingerprint string `json:"fingerprint"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	// Probe again server-side so activation never trusts candidate data from the UI.
	candidate, err := ProbeWebsiteProvider(r.Context(), request.URL, a.audit)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	if len(request.Fingerprint) != len(candidate.Fingerprint) || subtle.ConstantTimeCompare([]byte(request.Fingerprint), []byte(candidate.Fingerprint)) != 1 {
		jsonError(w, http.StatusConflict, errors.New("website provider manifest changed after verification; review it again before activating"))
		return
	}
	installation := websiteCandidateInstallation(candidate)
	if err := a.settings.AddProvider(installation); err != nil {
		jsonError(w, http.StatusConflict, err)
		return
	}
	jsonResponse(w, http.StatusOK, installation)
}

func (a *API) handleProviderHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ProviderID string `json:"provider_id"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	provider, err := configuredProvider(a.settings.Get(), request.ProviderID)
	if err != nil {
		jsonError(w, http.StatusNotFound, err)
		return
	}
	var result ProviderHealthResult
	if err := CallProvider(r.Context(), provider, "provider.health", nil, &result); err != nil {
		jsonError(w, http.StatusBadGateway, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

type providerSearchSource struct {
	ProviderID   string                `json:"provider_id"`
	ProviderName string                `json:"provider_name"`
	Items        []ProviderCatalogItem `json:"items"`
	Error        string                `json:"error,omitempty"`
}

func (a *API) handleProviderSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Query       string   `json:"query"`
		Limit       int      `json:"limit"`
		Preview     bool     `json:"preview,omitempty"`
		ProviderIDs []string `json:"provider_ids,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	request.Query = strings.TrimSpace(request.Query)
	if len(request.Query) < 2 || len(request.Query) > 200 {
		jsonError(w, http.StatusBadRequest, errors.New("query must contain between 2 and 200 characters"))
		return
	}
	if request.Limit <= 0 || request.Limit > 50 {
		request.Limit = 20
	}
	wanted := make(map[string]bool, len(request.ProviderIDs))
	for _, id := range request.ProviderIDs {
		wanted[id] = true
	}
	providers := a.settings.Get().Providers
	sources := make([]providerSearchSource, 0, len(providers))
	var mu sync.Mutex
	var wait sync.WaitGroup
	for _, provider := range providers {
		if !provider.Enabled || !hasProviderCapability(provider.Capabilities, "catalog.search") || (len(wanted) > 0 && !wanted[provider.ID]) {
			continue
		}
		provider := provider
		wait.Add(1)
		go func() {
			defer wait.Done()
			var result ProviderCatalogSearchResult
			mode := "complete"
			if request.Preview {
				mode = "preview"
			}
			err := CallProvider(r.Context(), provider, "catalog.search", ProviderCatalogSearchParams{Query: request.Query, Limit: request.Limit, Mode: mode}, &result)
			source := providerSearchSource{ProviderID: provider.ID, ProviderName: provider.Name, Items: result.Items}
			if err != nil {
				source.Error = err.Error()
				source.Items = []ProviderCatalogItem{}
			}
			mu.Lock()
			sources = append(sources, source)
			mu.Unlock()
		}()
	}
	wait.Wait()
	sort.SliceStable(sources, func(i, j int) bool { return sources[i].ProviderName < sources[j].ProviderName })
	jsonResponse(w, http.StatusOK, map[string]any{"query": request.Query, "sources": sources})
}

func (a *API) handleProviderCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ProviderID string `json:"provider_id"`
		Reference  string `json:"reference"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	if request.Reference == "" || len(request.Reference) > 256*1024 {
		jsonError(w, http.StatusBadRequest, errors.New("invalid provider reference"))
		return
	}
	provider, err := configuredProvider(a.settings.Get(), request.ProviderID)
	if err != nil {
		jsonError(w, http.StatusNotFound, err)
		return
	}
	var result ProviderCollectionResult
	if err := CallProvider(r.Context(), provider, "collection.resolve", ProviderCollectionResolveParams{Reference: request.Reference}, &result); err != nil {
		jsonError(w, http.StatusBadGateway, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (a *API) handleProviderImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ProviderID string `json:"provider_id"`
		Reference  string `json:"reference"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	if request.Reference == "" || len(request.Reference) > 256*1024 {
		jsonError(w, http.StatusBadRequest, errors.New("invalid provider reference"))
		return
	}
	provider, err := configuredProvider(a.settings.Get(), request.ProviderID)
	if err != nil {
		jsonError(w, http.StatusNotFound, err)
		return
	}
	var resolved ProviderCollectionResult
	if err := CallProvider(r.Context(), provider, "collection.resolve", ProviderCollectionResolveParams{Reference: request.Reference}, &resolved); err != nil {
		jsonError(w, http.StatusBadGateway, err)
		return
	}
	now := time.Now()
	collectionID := stableID("provider-collection:"+provider.ID, resolved.ID)
	collection := &Collection{
		ID: collectionID, Title: resolved.Title, Kind: CollectionProvider,
		SourceID: provider.ID, Description: resolved.Description, ArtworkURL: resolved.ArtworkURL,
		ProviderReference: request.Reference, Items: make([]MediaItem, 0, len(resolved.Items)),
		AddedAt: now, UpdatedAt: now,
	}

	if collection.ArtworkURL == "" || collection.Description == "" {
		if aniMeta, err := FetchAniListMetadata(r.Context(), collection.Title); err == nil && aniMeta != nil {
			if collection.ArtworkURL == "" {
				collection.ArtworkURL = aniMeta.ArtworkURL
			}
			if collection.Description == "" {
				collection.Description = aniMeta.Description
			}
		}
	}

	for _, sourceItem := range resolved.Items {
		collection.Items = append(collection.Items, MediaItem{
			ID:           stableID("provider-item:"+provider.ID, resolved.ID+"\x00"+sourceItem.ID),
			CollectionID: collectionID, Title: sourceItem.Title, Kind: MediaEpisode,
			SourceID: provider.ID, ProviderReference: sourceItem.Reference,
			Duration: sourceItem.Duration, AddedAt: now,
		})
	}
	if err := a.gallery.UpsertProviderCollection(collection); err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusOK, collection)
}

func (a *API) handleProviderItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ProviderID string `json:"provider_id"`
		Reference  string `json:"reference"`
		Quality    string `json:"quality,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	if request.Reference == "" || len(request.Reference) > 256*1024 {
		jsonError(w, http.StatusBadRequest, errors.New("invalid provider reference"))
		return
	}
	provider, err := configuredProvider(a.settings.Get(), request.ProviderID)
	if err != nil {
		jsonError(w, http.StatusNotFound, err)
		return
	}
	var result ProviderItemResolveResult
	if err := CallProvider(r.Context(), provider, "item.resolve", ProviderItemResolveParams{Reference: request.Reference, Quality: request.Quality}, &result); err != nil {
		jsonError(w, http.StatusBadGateway, err)
		return
	}
	guard := security.NewSSRFGuard(a.audit)
	if _, err := guard.ValidateURL(result.URL); err != nil {
		jsonError(w, http.StatusBadGateway, errors.New("provider returned an unsafe media URL"))
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (a *API) handleProviderPlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ItemID  string `json:"item_id"`
		Player  string `json:"player,omitempty"`
		Quality string `json:"quality,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	item, err := a.gallery.MediaItem(request.ItemID)
	if err != nil || item.SourceID == "local" || item.ProviderReference == "" {
		jsonError(w, http.StatusNotFound, ErrMediaNotFound)
		return
	}
	provider, err := configuredProvider(a.settings.Get(), item.SourceID)
	if err != nil {
		jsonError(w, http.StatusNotFound, err)
		return
	}
	var resolved ProviderItemResolveResult
	if err := CallProvider(r.Context(), provider, "item.resolve", ProviderItemResolveParams{Reference: item.ProviderReference, Quality: request.Quality}, &resolved); err != nil {
		jsonError(w, http.StatusBadGateway, err)
		return
	}
	guard := security.NewSSRFGuard(a.audit)
	if _, err := guard.ValidateURL(resolved.URL); err != nil {
		jsonError(w, http.StatusBadGateway, errors.New("provider returned an unsafe media URL"))
		return
	}
	for name, value := range resolved.Headers {
		if !validMediaHeader(name, value) {
			jsonError(w, http.StatusBadGateway, errors.New("provider returned an unsafe media header"))
			return
		}
	}
	playerName := request.Player
	// The media URL stays server-side.  Passing an arbitrary URL back into a
	// local proxy would turn this endpoint into an SSRF primitive and expose
	// short-lived provider URLs in the renderer.
	streamProxyURL := "/api/media/stream?id=" + url.QueryEscape(item.ID) + "&session=" + url.QueryEscape(a.sessionToken)
	if playerName == "integrated" {
		jsonResponse(w, http.StatusOK, map[string]any{
			"status":  "resolved",
			"player":  "integrated",
			"url":     streamProxyURL,
			"title":   item.Title,
			"item_id": item.ID,
		})
		return
	}
	if playerName == "" {
		playerName = firstInstalledExternalPlayer(a.players)
		if playerName == "" {
			jsonResponse(w, http.StatusOK, map[string]any{
				"status":  "resolved",
				"player":  "integrated",
				"url":     streamProxyURL,
				"title":   item.Title,
				"item_id": item.ID,
			})
			return
		}
	}
	if err := a.players.Play(context.Background(), playerName, resolved.URL, resolved.Headers, item.Title); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"status": "playing", "player": playerName, "url": resolved.URL})
}

func firstInstalledExternalPlayer(players *player.PlayerService) string {
	for _, name := range []string{"mpv", "celluloid"} {
		if candidate, present := players.GetProvider(name); present && candidate.IsInstalled() {
			return name
		}
	}
	return ""
}

func validMediaHeader(name, value string) bool {
	if name == "" || len(name) > 128 || len(value) > 8192 || strings.ContainsAny(value, "\r\n") {
		return false
	}
	for _, character := range name {
		if !(character >= 'a' && character <= 'z') && !(character >= 'A' && character <= 'Z') && !(character >= '0' && character <= '9') && !strings.ContainsRune("!#$%&'*+-.^_`|~", character) {
			return false
		}
	}
	return true
}

func configuredProvider(settings AppSettings, id string) (ProviderInstallation, error) {
	for _, provider := range settings.Providers {
		if provider.ID == id && provider.Enabled {
			return provider, nil
		}
	}
	return ProviderInstallation{}, errors.New("enabled provider not found")
}

func (a *API) handlePlayers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	jsonResponse(w, http.StatusOK, a.players.ListProviders())
}

func (a *API) handlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ItemID string `json:"item_id"`
		Player string `json:"player"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	item, err := a.gallery.MediaItem(request.ItemID)
	if err != nil || item.Path == "" {
		jsonError(w, http.StatusNotFound, ErrMediaNotFound)
		return
	}
	if request.Player == "" {
		request.Player = a.settings.Get().DefaultPlayer
	}
	if err := a.players.Play(context.Background(), request.Player, item.Path, nil, item.Title); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]bool{"started": true})
}

func (a *API) handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	abs, err := filepath.Abs(request.Path)
	if err != nil || !allowedConfiguredPath(abs, a.settings.Get()) {
		jsonError(w, http.StatusForbidden, errors.New("folder is not an authorized media or provider root"))
		return
	}
	command := "xdg-open"
	args := []string{abs}
	if runtime.GOOS == "darwin" {
		command = "open"
	} else if runtime.GOOS == "windows" {
		command, args = "explorer", []string{abs}
	}
	if err := exec.Command(command, args...).Start(); err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]bool{"opened": true})
}

func (a *API) handleSecurityEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"events": a.audit.GetRecentEvents()})
}

func (a *API) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "shutting_down"})
	select {
	case a.shutdown <- struct{}{}:
	default:
	}
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxJSONBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func jsonError(w http.ResponseWriter, status int, err error) {
	jsonResponse(w, status, map[string]string{"error": err.Error()})
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	jsonError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

func isLoopbackHost(hostPort string) bool {
	host := hostPort
	if parsed, err := url.Parse("http://" + hostPort); err == nil {
		host = parsed.Hostname()
	}
	return host == "127.0.0.1" || host == "localhost" || host == "[::1]" || host == "::1"
}

func sameLocalOrigin(origin, requestHost string) bool {
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Scheme == "http" && parsed.Host == requestHost && isLoopbackHost(parsed.Host)
}

func allowedConfiguredPath(path string, settings AppSettings) bool {
	for _, root := range settings.MediaRoots {
		if filepath.Clean(path) == filepath.Clean(root) {
			return true
		}
	}
	for _, provider := range settings.Providers {
		if filepath.Clean(path) == filepath.Clean(provider.RootPath) {
			return true
		}
	}
	return false
}

func isHopByHopHeader(header string) bool {
	h := http.CanonicalHeaderKey(header)
	return h == "Connection" ||
		h == "Keep-Alive" ||
		h == "Proxy-Authenticate" ||
		h == "Proxy-Authorization" ||
		h == "Te" ||
		h == "Trailers" ||
		h == "Transfer-Encoding" ||
		h == "Upgrade"
}

func (a *API) handleMediaStreamProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
		return
	}
	item, err := a.gallery.MediaItem(itemID)
	if err != nil || item.SourceID == "local" || item.ProviderReference == "" {
		http.NotFound(w, r)
		return
	}
	provider, err := configuredProvider(a.settings.Get(), item.SourceID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var resolved ProviderItemResolveResult
	if err := CallProvider(r.Context(), provider, "item.resolve", ProviderItemResolveParams{Reference: item.ProviderReference}, &resolved); err != nil {
		http.Error(w, `{"error":"provider stream resolution failed"}`, http.StatusBadGateway)
		return
	}
	if resolved.URL == "" {
		http.Error(w, `{"error":"empty stream url"}`, http.StatusBadGateway)
		return
	}
	guard := security.NewSSRFGuard(a.audit)
	if _, err := guard.ValidateURL(resolved.URL); err != nil {
		http.Error(w, `{"error":"provider returned an unsafe media URL"}`, http.StatusBadGateway)
		return
	}
	for name, value := range resolved.Headers {
		if !validMediaHeader(name, value) {
			http.Error(w, `{"error":"provider returned an unsafe media header"}`, http.StatusBadGateway)
			return
		}
	}

	reqCtx, cancel := context.WithTimeout(r.Context(), 4*time.Hour)
	defer cancel()

	proxyReq, err := http.NewRequestWithContext(reqCtx, r.Method, resolved.URL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range resolved.Headers {
		proxyReq.Header.Set(k, v)
	}
	if proxyReq.Header.Get("User-Agent") == "" {
		proxyReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}
	if proxyReq.Header.Get("Referer") == "" {
		if parsed, err := url.Parse(resolved.URL); err == nil {
			proxyReq.Header.Set("Referer", fmt.Sprintf("%s://%s/", parsed.Scheme, parsed.Host))
		}
	}
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		proxyReq.Header.Set("Range", rangeHeader)
	}

	client := guard.NewSecureHTTPClient(4 * time.Hour)
	if transport, ok := client.Transport.(*http.Transport); ok {
		transport.DisableCompression = true
	}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		if isHopByHopHeader(k) {
			continue
		}
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	if w.Header().Get("Content-Type") == "" || w.Header().Get("Content-Type") == "application/octet-stream" {
		w.Header().Set("Content-Type", "video/mp4")
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (a *API) handleStartDownloadTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		ItemID          string `json:"item_id"`
		ProviderID      string `json:"provider_id,omitempty"`
		Reference       string `json:"reference,omitempty"`
		Title           string `json:"title,omitempty"`
		CollectionID    string `json:"collection_id,omitempty"`
		CollectionTitle string `json:"collection_title,omitempty"`
		ArtworkURL      string `json:"artwork_url,omitempty"`
	}
	if err := decodeJSON(r, &request); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var item MediaItem
	var err error
	options := DownloadOptions{
		CollectionTitle: request.CollectionTitle,
		ArtworkURL:      request.ArtworkURL,
	}

	if request.ItemID != "" {
		item, err = a.gallery.MediaItem(request.ItemID)
		if err == nil && item.CollectionID != "" {
			if parentCol := a.gallery.Collection(item.CollectionID); parentCol != nil {
				if options.CollectionTitle == "" {
					options.CollectionTitle = parentCol.Title
				}
				if options.ArtworkURL == "" {
					options.ArtworkURL = parentCol.ArtworkURL
				}
			}
		}
	}

	if (err != nil || item.ID == "") && request.ProviderID != "" && request.Reference != "" {
		item = MediaItem{
			ID:                stableID("provider-item:"+request.ProviderID, request.Reference),
			Title:             request.Title,
			Kind:              MediaEpisode,
			SourceID:          request.ProviderID,
			ProviderReference: request.Reference,
		}
		err = nil
	}

	if err != nil || item.SourceID == "" || item.SourceID == "local" || item.ProviderReference == "" {
		jsonError(w, http.StatusNotFound, ErrMediaNotFound)
		return
	}
	provider, err := configuredProvider(a.settings.Get(), item.SourceID)
	if err != nil {
		jsonError(w, http.StatusNotFound, err)
		return
	}
	var resolved ProviderItemResolveResult
	if err := CallProvider(r.Context(), provider, "item.resolve", ProviderItemResolveParams{Reference: item.ProviderReference}, &resolved); err != nil {
		jsonError(w, http.StatusBadGateway, err)
		return
	}
	guard := security.NewSSRFGuard(a.audit)
	if _, err := guard.ValidateURL(resolved.URL); err != nil {
		jsonError(w, http.StatusBadGateway, errors.New("provider returned an unsafe media URL"))
		return
	}
	for name, value := range resolved.Headers {
		if !validMediaHeader(name, value) {
			jsonError(w, http.StatusBadGateway, errors.New("provider returned an unsafe media header"))
			return
		}
	}

	task, err := a.downloader.StartDownload(r.Context(), item, options, resolved.URL, resolved.Headers)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusAccepted, task)
}

func (a *API) handleDownloadTaskStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		jsonResponse(w, http.StatusOK, map[string]any{"tasks": a.downloader.ActiveTasks()})
		return
	}
	task, found := a.downloader.GetTask(taskID)
	if !found {
		jsonError(w, http.StatusNotFound, errors.New("download task not found"))
		return
	}
	jsonResponse(w, http.StatusOK, task)
}
