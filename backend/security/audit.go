// SPDX-License-Identifier: GPL-3.0-only

package security

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/url"
	"regexp"
	"sync"
	"time"
)

type EventType string
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

const (
	EventAppStarted           EventType = "APP_STARTED"
	EventAppStopped           EventType = "APP_STOPPED"
	EventAPIStarted           EventType = "API_STARTED"
	EventAuthSuccess          EventType = "AUTH_SUCCESS"
	EventAuthFailure          EventType = "AUTH_FAILURE"
	EventCapabilityDenied     EventType = "CAPABILITY_DENIED"
	EventSSRFBlocked          EventType = "SSRF_BLOCKED"
	EventPathTraversalBlocked EventType = "PATH_TRAVERSAL_BLOCKED"
	EventPlayerStarted        EventType = "PLAYER_STARTED"
	EventPlayerStopped        EventType = "PLAYER_STOPPED"
	EventPlayerFailed         EventType = "PLAYER_FAILED"
	EventDownloadStarted      EventType = "DOWNLOAD_STARTED"
	EventDownloadCompleted    EventType = "DOWNLOAD_COMPLETED"
	EventDownloadFailed       EventType = "DOWNLOAD_FAILED"
)

type SecurityEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	EventType EventType `json:"event_type"`
	Severity  Severity  `json:"severity"`
	Component string    `json:"component"`
	Reason    string    `json:"reason"`
	Details   string    `json:"details"`
	RequestID string    `json:"request_id,omitempty"`
}

type AuditLogger struct {
	mu     sync.RWMutex
	events []SecurityEvent
	maxLen int
}

func NewAuditLogger(maxLen int) *AuditLogger {
	if maxLen <= 0 {
		maxLen = 100
	}
	return &AuditLogger{
		events: make([]SecurityEvent, 0, maxLen),
		maxLen: maxLen,
	}
}

func (a *AuditLogger) LogEvent(eventType EventType, severity Severity, component, reason, details string) SecurityEvent {
	event := SecurityEvent{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		EventType: eventType,
		Severity:  severity,
		Component: component,
		Reason:    reason,
		Details:   RedactURL(details),
	}

	a.mu.Lock()
	if len(a.events) >= a.maxLen {
		a.events = a.events[1:] // drop oldest
	}
	a.events = append(a.events, event)
	a.mu.Unlock()

	log.Printf("[AUDIT LOG] [%s] [%s] %s: %s (%s)", severity, eventType, component, reason, event.Details)
	return event
}

func (a *AuditLogger) GetRecentEvents() []SecurityEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	copied := make([]SecurityEvent, len(a.events))
	copy(copied, a.events)
	return copied
}

func generateEventID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RedactURL redacts sensitive tokens, passwords, and keys from URL parameters in logs.
func RedactURL(raw string) string {
	if raw == "" {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		// Fallback regex redaction
		re := regexp.MustCompile(`(?i)(token|auth|key|secret|password|bearer)=([^&]+)`)
		return re.ReplaceAllString(raw, "${1}=[REDACTED]")
	}

	query := parsed.Query()
	modified := false
	sensitiveKeys := []string{"token", "auth", "key", "secret", "password", "bearer", "signature", "sig"}

	for _, k := range sensitiveKeys {
		if query.Has(k) {
			query.Set(k, "[REDACTED]")
			modified = true
		}
	}

	if modified {
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}

	return raw
}
