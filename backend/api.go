// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
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
	sessionToken string
	shutdown     chan struct{}
}

func NewAPI(gallery *GalleryStore, settings *SettingsStore, indexer *MediaIndexer, players *player.PlayerService, processes *player.ProcessManager, audit *security.AuditLogger, artwork *ArtworkCache, token string, shutdown chan struct{}) *API {
	return &API{gallery: gallery, settings: settings, indexer: indexer, players: players, processes: processes, audit: audit, artwork: artwork, sessionToken: token, shutdown: shutdown}
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
			if provided == "" && (r.URL.Path == "/api/media/local" || r.URL.Path == "/api/media/thumbnail" || r.URL.Path == "/api/media/artwork") {
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
	jsonResponse(w, http.StatusOK, map[string]any{"collections": a.gallery.Collections()})
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
	if playerName == "" || playerName == "integrated" {
		playerName = firstInstalledExternalPlayer(a.players)
	}
	if playerName == "" {
		jsonError(w, http.StatusConflict, errors.New("install MPV or Celluloid to play provider streams securely"))
		return
	}
	if err := a.players.Play(context.Background(), playerName, resolved.URL, resolved.Headers, item.Title); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "playing", "player": playerName})
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
