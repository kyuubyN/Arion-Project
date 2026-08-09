// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/kyuubyN/Arion-Project/backend/player"
	"github.com/kyuubyN/Arion-Project/backend/security"
)

func main() {
	log.Println("[Arion] Starting neutral media gallery")
	gallery, err := NewGalleryStore()
	if err != nil {
		log.Fatalf("initialize gallery: %v", err)
	}
	settings, err := NewSettingsStore()
	if err != nil {
		log.Fatalf("initialize settings: %v", err)
	}
	audit := security.NewAuditLogger(250)
	artwork, err := NewArtworkCache(audit)
	if err != nil {
		log.Fatalf("initialize artwork cache: %v", err)
	}
	processes := player.NewProcessManager()
	players := player.NewPlayerService(processes)
	indexer := NewMediaIndexer(gallery)
	shutdown := make(chan struct{}, 1)
	watchContext, stopWatching := context.WithCancel(context.Background())
	defer stopWatching()
	token := os.Getenv("ARION_SESSION_TOKEN")
	if token == "" {
		token = randomToken()
	}

	frontendDir := resolveFrontendDir()
	api := NewAPI(gallery, settings, indexer, players, processes, audit, artwork, token, shutdown)
	port := os.Getenv("ARION_PORT")
	if port == "" {
		port = "0"
	}
	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port
	server := &http.Server{
		Handler:           api.Routes(frontendDir),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, shutdownSignals()...)
	go func() {
		select {
		case <-stop:
		case <-shutdown:
		}
		processes.StopAll()
		stopWatching()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	if configured := settings.Get().MediaRoots; len(configured) > 0 {
		_ = indexer.Start(configured)
	}
	go keepLocalLibraryFresh(watchContext, indexer, settings, 2*time.Minute)
	fmt.Printf("ARION_SERVER_READY_URL=http://127.0.0.1:%d/#session=%s\n", actualPort, token)
	log.Printf("[Arion] Serving local gallery on 127.0.0.1:%d", actualPort)
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func keepLocalLibraryFresh(ctx context.Context, indexer *MediaIndexer, settings *SettingsStore, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			roots := settings.Get().MediaRoots
			if len(roots) > 0 && !indexer.Status().Running {
				_ = indexer.Start(roots)
			}
		}
	}
}

func resolveFrontendDir() string {
	executable, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(executable), "..", "frontend")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate
		}
	}
	for _, candidate := range []string{"./frontend", "../frontend"} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "./frontend"
}

func randomToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("secure random source unavailable")
	}
	return hex.EncodeToString(bytes)
}
