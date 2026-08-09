// SPDX-License-Identifier: GPL-3.0-only

package security

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
)

type Capability string

const (
	CapLibraryRead    Capability = "library:read"
	CapLibraryWrite   Capability = "library:write"
	CapSearchExecute  Capability = "search:execute"
	CapStreamResolve  Capability = "stream:resolve"
	CapPlayerLaunch   Capability = "player:launch"
	CapPlayerStop     Capability = "player:stop"
	CapDownloadStart  Capability = "download:start"
	CapDownloadCancel Capability = "download:cancel"
	CapHistoryRead    Capability = "history:read"
	CapHistoryWrite   Capability = "history:write"
	CapSettingsRead   Capability = "settings:read"
	CapSettingsWrite  Capability = "settings:write"
	CapSecurityRead   Capability = "security:read"
)

type Session struct {
	Token        string
	Capabilities map[Capability]bool
}

type SessionManager struct {
	mu           sync.RWMutex
	primaryToken string
	capabilities map[Capability]bool
	auditLogger  *AuditLogger
}

func NewSessionManager(logger *AuditLogger) *SessionManager {
	token := generateHighEntropyToken()
	allCaps := map[Capability]bool{
		CapLibraryRead:    true,
		CapLibraryWrite:   true,
		CapSearchExecute:  true,
		CapStreamResolve:  true,
		CapPlayerLaunch:   true,
		CapPlayerStop:     true,
		CapDownloadStart:  true,
		CapDownloadCancel: true,
		CapHistoryRead:    true,
		CapHistoryWrite:   true,
		CapSettingsRead:   true,
		CapSettingsWrite:  true,
		CapSecurityRead:   true,
	}

	return &SessionManager{
		primaryToken: token,
		capabilities: allCaps,
		auditLogger:  logger,
	}
}

func (s *SessionManager) PrimaryToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.primaryToken
}

func (s *SessionManager) ValidateRequest(r *http.Request, requiredCap Capability) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Tokens are accepted only through the Authorization header. Query-string
	// credentials leak into history, diagnostics and referrers.
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if !strings.HasPrefix(authHeader, "Bearer ") || token == "" || token != s.primaryToken {
		if s.auditLogger != nil {
			s.auditLogger.LogEvent(EventAuthFailure, SeverityWarning, "SessionManager", "invalid_token", r.URL.Path)
		}
		return false
	}

	if !s.capabilities[requiredCap] {
		if s.auditLogger != nil {
			s.auditLogger.LogEvent(EventCapabilityDenied, SeverityHigh, "SessionManager", string(requiredCap), r.URL.Path)
		}
		return false
	}

	return true
}

func generateHighEntropyToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic("security: cryptographic random source unavailable")
	}
	return hex.EncodeToString(b)
}
