// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/kyuubyN/Arion-Project/backend/security"
	"github.com/kyuubyN/Arion-Project/providerkit"
)

const (
	websiteProviderManifestPath = providerkit.WellKnownManifestPath
	maxWebsiteManifestBytes     = providerkit.MaxManifestBytes
)

type WebsiteProviderManifest = providerkit.Manifest

type WebsiteProviderCandidate struct {
	Manifest    WebsiteProviderManifest `json:"manifest"`
	Origin      string                  `json:"origin"`
	ManifestURL string                  `json:"manifest_url"`
	RPCURL      string                  `json:"rpc_url"`
	Fingerprint string                  `json:"fingerprint"`
	Ready       bool                    `json:"ready"`
	Status      string                  `json:"status"`
}

func ProbeWebsiteProvider(ctx context.Context, rawURL string, audit *security.AuditLogger) (WebsiteProviderCandidate, error) {
	guard := security.NewSSRFGuard(audit)
	client := websiteHTTPClient(guard.NewSecureHTTPClient(12 * time.Second))
	return probeWebsiteProviderWithClient(ctx, rawURL, client, func(target string) error {
		_, err := guard.ValidateURL(target)
		return err
	})
}

func probeWebsiteProviderWithClient(ctx context.Context, rawURL string, client *http.Client, validateURL func(string) error) (WebsiteProviderCandidate, error) {
	origin, err := normalizeWebsiteOrigin(rawURL)
	if err != nil {
		return WebsiteProviderCandidate{}, err
	}
	manifestURL := origin + websiteProviderManifestPath
	if validateURL != nil {
		if err := validateURL(manifestURL); err != nil {
			return WebsiteProviderCandidate{}, fmt.Errorf("website provider destination was blocked: %w", err)
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return WebsiteProviderCandidate{}, err
	}
	request.Header.Set("Accept", "application/arion-provider+json, application/json")
	request.Header.Set("User-Agent", "Arion/0.4.1 provider-protocol/1")
	response, err := client.Do(request)
	if err != nil {
		return WebsiteProviderCandidate{}, fmt.Errorf("could not read the website provider manifest: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return WebsiteProviderCandidate{}, errors.New("this site does not expose a compatible Arion provider manifest")
	}
	if response.StatusCode != http.StatusOK {
		return WebsiteProviderCandidate{}, fmt.Errorf("website provider manifest returned HTTP %d", response.StatusCode)
	}
	if !isProviderJSONContentType(response.Header.Get("Content-Type")) {
		return WebsiteProviderCandidate{}, errors.New("website provider manifest must use a JSON content type")
	}
	body, err := security.BoundedRead(response.Body, maxWebsiteManifestBytes)
	if err != nil {
		return WebsiteProviderCandidate{}, fmt.Errorf("invalid website provider manifest: %w", err)
	}
	manifest, err := decodeWebsiteProviderManifest(body)
	if err != nil {
		return WebsiteProviderCandidate{}, err
	}
	rpcURL, err := websiteRPCURL(origin, manifest.RPCPath)
	if err != nil {
		return WebsiteProviderCandidate{}, err
	}
	if validateURL != nil {
		if err := validateURL(rpcURL); err != nil {
			return WebsiteProviderCandidate{}, fmt.Errorf("website provider endpoint was blocked: %w", err)
		}
	}
	return WebsiteProviderCandidate{
		Manifest: manifest, Origin: origin, ManifestURL: manifestURL, RPCURL: rpcURL,
		Fingerprint: manifestFingerprint(body), Ready: true, Status: "Compatível e pronto para ativar",
	}, nil
}

func manifestFingerprint(data []byte) string {
	return providerkit.Fingerprint(data)
}

func decodeWebsiteProviderManifest(data []byte) (WebsiteProviderManifest, error) {
	return providerkit.DecodeManifest(data)
}

func validateWebsiteProviderManifest(manifest WebsiteProviderManifest) error {
	return providerkit.ValidateManifest(manifest)
}

func normalizeWebsiteOrigin(rawURL string) (string, error) {
	return providerkit.NormalizePublicOrigin(rawURL)
}

func websiteRPCURL(origin, declaredPath string) (string, error) {
	return providerkit.ResolveRPCURL(origin, declaredPath)
}

func websiteHTTPClient(base *http.Client) *http.Client {
	client := *base
	client.Jar = nil
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errors.New("website provider redirects are not allowed")
	}
	return &client
}

func isProviderJSONContentType(value string) bool {
	return providerkit.IsJSONContentType(value)
}

func websiteCandidateInstallation(candidate WebsiteProviderCandidate) ProviderInstallation {
	return ProviderInstallation{
		Kind: ProviderWebsite, ID: candidate.Manifest.ID, Name: candidate.Manifest.Name,
		Version: candidate.Manifest.Version, Origin: candidate.Origin,
		ManifestURL: candidate.ManifestURL, RPCURL: candidate.RPCURL,
		Capabilities: append([]string(nil), candidate.Manifest.Capabilities...), Enabled: true,
	}
}

func validateWebsiteInstallation(installation ProviderInstallation) error {
	origin, err := normalizeWebsiteOrigin(installation.Origin)
	if err != nil || origin != installation.Origin {
		return errors.New("website provider origin is no longer valid; register it again")
	}
	if installation.ManifestURL != origin+websiteProviderManifestPath {
		return errors.New("website provider manifest location changed after registration; register it again")
	}
	rpc, err := url.Parse(installation.RPCURL)
	if err != nil || rpc.Scheme != "https" || rpc.User != nil || rpc.RawQuery != "" || rpc.Fragment != "" || rpc.Path == "" {
		return errors.New("website provider endpoint is no longer valid; register it again")
	}
	rpcOrigin, err := normalizeWebsiteOrigin(installation.RPCURL)
	if err != nil || rpcOrigin != origin || path.Clean(rpc.Path) != rpc.Path || strings.HasPrefix(rpc.Path, "//") {
		return errors.New("website provider endpoint must remain on its registered HTTPS origin")
	}
	return nil
}
