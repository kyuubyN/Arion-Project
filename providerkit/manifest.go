// SPDX-License-Identifier: GPL-3.0-only

// Package providerkit contains the public, provider-neutral Arion protocol.
// Website providers can use it without importing the Arion application backend.
package providerkit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const (
	SchemaVersion         = 1
	ProtocolVersion       = 1
	WebsiteKind           = "website"
	WellKnownManifestPath = "/.well-known/arion-provider.json"
	MaxManifestBytes      = 256 * 1024
)

var (
	providerIDPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
	publicHostnamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)
	capabilities          = map[string]struct{}{
		"catalog.search":     {},
		"collection.resolve": {},
		"item.resolve":       {},
	}
)

type Manifest struct {
	SchemaVersion   int      `json:"schema_version"`
	Kind            string   `json:"kind"`
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	ProtocolVersion int      `json:"protocol_version"`
	RPCPath         string   `json:"rpc_path"`
	Capabilities    []string `json:"capabilities"`
	Author          string   `json:"author,omitempty"`
	License         string   `json:"license,omitempty"`
	Homepage        string   `json:"homepage,omitempty"`
}

func DecodeManifest(data []byte) (Manifest, error) {
	if len(data) > MaxManifestBytes {
		return Manifest{}, fmt.Errorf("manifest exceeds %d bytes", MaxManifestBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("invalid website provider manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return Manifest{}, errors.New("website provider manifest contains more than one JSON value")
	} else if !errors.Is(err, io.EOF) {
		return Manifest{}, fmt.Errorf("invalid website provider manifest: %w", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func ValidateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported manifest schema %d", manifest.SchemaVersion)
	}
	if manifest.Kind != WebsiteKind {
		return errors.New("website provider manifest kind must be website")
	}
	if manifest.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("unsupported provider protocol %d", manifest.ProtocolVersion)
	}
	if !providerIDPattern.MatchString(manifest.ID) {
		return errors.New("provider id must use lowercase letters, numbers, dots, dashes or underscores")
	}
	if strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Version) == "" {
		return errors.New("provider name and version are required")
	}
	if len(manifest.Name) > 200 || len(manifest.Version) > 64 || len(manifest.Author) > 500 || len(manifest.License) > 200 || len(manifest.Homepage) > 2048 || len(manifest.RPCPath) > 2048 {
		return errors.New("website provider manifest contains an oversized text field")
	}
	seen := make(map[string]struct{}, len(manifest.Capabilities))
	for _, capability := range manifest.Capabilities {
		if _, supported := capabilities[capability]; !supported {
			return fmt.Errorf("unsupported provider capability %q", capability)
		}
		if _, duplicate := seen[capability]; duplicate {
			return fmt.Errorf("duplicate provider capability %q", capability)
		}
		seen[capability] = struct{}{}
	}
	if strings.TrimSpace(manifest.RPCPath) == "" {
		return errors.New("website provider manifest requires rpc_path")
	}
	if _, err := ResolveRPCURL("https://provider.invalid", manifest.RPCPath); err != nil {
		return err
	}
	if manifest.RPCPath == "/" || manifest.RPCPath == WellKnownManifestPath {
		return errors.New("rpc_path cannot overlap the well-known manifest")
	}
	return nil
}

// NormalizePublicOrigin accepts a pasted site URL but returns only its HTTPS origin.
func NormalizePublicOrigin(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("website URL is required")
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", errors.New("website providers require a valid HTTPS URL")
	}
	if parsed.User != nil {
		return "", errors.New("website provider URLs cannot contain credentials")
	}
	hostname := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if hostname == "localhost" || net.ParseIP(hostname) != nil || !validPublicHostname(hostname) {
		return "", errors.New("website providers require a public domain name, not a local name or IP address")
	}
	port := parsed.Port()
	if port != "" && port != "443" {
		return "", errors.New("website providers are restricted to the standard HTTPS port")
	}
	return "https://" + hostname, nil
}

func validPublicHostname(hostname string) bool {
	if len(hostname) > 253 || !strings.Contains(hostname, ".") || !publicHostnamePattern.MatchString(hostname) {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if len(label) == 0 || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}
	return true
}

func ResolveRPCURL(origin, declaredPath string) (string, error) {
	parsed, err := url.Parse(declaredPath)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
		return "", errors.New("website provider rpc_path must be an absolute same-origin path without query or fragment")
	}
	if cleaned := path.Clean(parsed.Path); cleaned != parsed.Path {
		return "", errors.New("website provider rpc_path must not contain traversal or redundant segments")
	}
	return origin + parsed.EscapedPath(), nil
}

func Fingerprint(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func IsJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && (mediaType == "application/json" || mediaType == "application/arion-provider+json")
}

func HasCapability(list []string, wanted string) bool {
	for _, capability := range list {
		if capability == wanted {
			return true
		}
	}
	return false
}
