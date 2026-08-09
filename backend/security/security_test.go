// SPDX-License-Identifier: GPL-3.0-only

package security

import (
	"net"
	"strings"
	"testing"
)

func TestSSRFGuard_IsRestrictedIP(t *testing.T) {
	logger := NewAuditLogger(10)
	guard := NewSSRFGuard(logger)

	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.100", true},
		{"169.254.169.254", true},
		{"::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tt := range tests {
		parsed := net.ParseIP(tt.ip)
		res := guard.IsRestrictedIP(parsed)
		if res != tt.expected {
			t.Errorf("IsRestrictedIP(%s) = %v; want %v", tt.ip, res, tt.expected)
		}
	}
}

func TestSSRFGuard_ValidateURL(t *testing.T) {
	logger := NewAuditLogger(10)
	guard := NewSSRFGuard(logger)

	badURLs := []string{
		"file:///etc/passwd",
		"gopher://127.0.0.1:70",
		"http://127.0.0.1:8080",
		"http://169.254.169.254/latest/meta-data/",
	}

	for _, u := range badURLs {
		_, err := guard.ValidateURL(u)
		if err == nil {
			t.Errorf("ValidateURL(%s) should have failed but passed", u)
		}
	}
}

func TestSafePathResolver_PathTraversal(t *testing.T) {
	logger := NewAuditLogger(10)
	resolver := NewSafePathResolver(logger)
	baseDir := "/tmp/arion_test_downloads"

	badPaths := []string{
		"../../etc/passwd",
		"../secret.txt",
		"/etc/shadow",
	}

	for _, p := range badPaths {
		_, err := resolver.ResolveSafePath(baseDir, p)
		if err == nil {
			t.Errorf("ResolveSafePath(%s, %s) should have blocked path traversal", baseDir, p)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	logger := NewAuditLogger(10)
	resolver := NewSafePathResolver(logger)

	tests := []struct {
		input    string
		expected string
	}{
		{"normal_title.mp4", "normal_title.mp4"},
		{"title/with/slashes", "title_with_slashes"},
		{"title:with*forbidden?chars", "title_with_forbidden_chars"},
		{"..", "downloaded_file"},
	}

	for _, tt := range tests {
		res := resolver.SanitizeFilename(tt.input)
		if res != tt.expected {
			t.Errorf("SanitizeFilename(%s) = %s; want %s", tt.input, res, tt.expected)
		}
	}
}

func TestRedactURL(t *testing.T) {
	raw := "https://example.com/video.m3u8?token=SECRET123&user=john"
	redacted := RedactURL(raw)
	if strings.Contains(redacted, "SECRET123") {
		t.Errorf("RedactURL failed to redact token from URL: %s", redacted)
	}
}
