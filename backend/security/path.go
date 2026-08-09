// SPDX-License-Identifier: GPL-3.0-only

package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrPathTraversal = errors.New("security: path traversal attempt detected (attempt to escape allowed directory)")
	ErrSymlinkEscape = errors.New("security: symlink target escapes allowed base directory")
	ErrEmptyFilename = errors.New("security: filename is empty after sanitization")
)

type SafePathResolver struct {
	auditLogger *AuditLogger
}

func NewSafePathResolver(logger *AuditLogger) *SafePathResolver {
	return &SafePathResolver{
		auditLogger: logger,
	}
}

// ExpandHome expands the leading tilde ~ to user home directory.
func ExpandHome(pathStr string) string {
	if pathStr == "" {
		return pathStr
	}
	if strings.HasPrefix(pathStr, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, pathStr[1:])
		}
	}
	return pathStr
}

// ResolveSafePath validates that targetRelativePath stays strictly inside allowedBaseDir.
func (s *SafePathResolver) ResolveSafePath(allowedBaseDir, targetRelativePath string) (string, error) {
	baseExpanded := ExpandHome(allowedBaseDir)
	absBase, err := filepath.Abs(filepath.Clean(baseExpanded))
	if err != nil {
		return "", fmt.Errorf("invalid base directory: %w", err)
	}

	// Reject explicit traversal patterns before processing
	if strings.Contains(targetRelativePath, "..") || strings.HasPrefix(targetRelativePath, "/") || strings.Contains(targetRelativePath, "\x00") {
		s.recordBlocked("path_traversal_attempt", targetRelativePath)
		return "", ErrPathTraversal
	}

	sanitizedRelative := s.SanitizePathComponents(targetRelativePath)
	fullPath := filepath.Join(absBase, sanitizedRelative)
	absTarget, err := filepath.Abs(filepath.Clean(fullPath))
	if err != nil {
		return "", fmt.Errorf("invalid target path: %w", err)
	}

	// Verify that absTarget starts with absBase
	if !strings.HasPrefix(absTarget, absBase) {
		s.recordBlocked("path_traversal_escape", targetRelativePath)
		return "", ErrPathTraversal
	}

	// Symlink evaluation: verify real path target does not escape absBase
	evalBase, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		evalBase = absBase
	}

	if _, statErr := os.Lstat(absTarget); statErr == nil {
		evalTarget, evalErr := filepath.EvalSymlinks(absTarget)
		if evalErr == nil {
			if !strings.HasPrefix(evalTarget, evalBase) {
				s.recordBlocked("symlink_escape", targetRelativePath)
				return "", ErrSymlinkEscape
			}
		}
	}

	return absTarget, nil
}

// SanitizeFilename strips forbidden filesystem characters, null bytes, and path separators.
func (s *SafePathResolver) SanitizeFilename(filename string) string {
	if filename == "" {
		return "downloaded_file"
	}

	// Remove null bytes and control characters
	reControl := regexp.MustCompile(`[\x00-\x1F\x7F]`)
	cleaned := reControl.ReplaceAllString(filename, "")

	// Replace forbidden path characters with underscores
	reForbidden := regexp.MustCompile(`[\\/:*?"<>|]`)
	cleaned = reForbidden.ReplaceAllString(cleaned, "_")

	// Strip leading/trailing dots and spaces
	cleaned = strings.Trim(cleaned, ". ")

	// Prevent reserved filenames (CON, PRN, AUX, NUL, COM1-9, LPT1-9)
	upper := strings.ToUpper(cleaned)
	reserved := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	}

	if reserved[upper] || cleaned == ".." || cleaned == "." || cleaned == "" {
		cleaned = "downloaded_file"
	}

	return cleaned
}

func (s *SafePathResolver) SanitizePathComponents(pathStr string) string {
	parts := strings.Split(pathStr, "/")
	var cleanParts []string
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			continue
		}
		cleanParts = append(cleanParts, s.SanitizeFilename(p))
	}
	return strings.Join(cleanParts, "/")
}

func (s *SafePathResolver) recordBlocked(reason, path string) {
	if s.auditLogger != nil {
		s.auditLogger.LogEvent(EventPathTraversalBlocked, SeverityHigh, "SafePathResolver", reason, path)
	}
}
