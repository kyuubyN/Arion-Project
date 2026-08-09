// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kyuubyN/Arion-Project/backend/security"
	"github.com/kyuubyN/Arion-Project/providerkit"
)

const (
	providerResponseLimit = 8 * 1024 * 1024
	providerStderrLimit   = 32 * 1024
)

var providerMethodCapability = map[string]string{
	"provider.describe":  "",
	"provider.health":    "",
	"catalog.search":     "catalog.search",
	"collection.resolve": "collection.resolve",
	"item.resolve":       "item.resolve",
}

type providerRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type providerRPCResponse struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      string            `json:"id"`
	Result  json.RawMessage   `json:"result,omitempty"`
	Error   *ProviderRPCError `json:"error,omitempty"`
}

type ProviderRPCError = providerkit.RPCError
type ProviderCatalogSearchParams = providerkit.CatalogSearchParams
type ProviderCatalogSearchResult = providerkit.CatalogSearchResult
type ProviderHealthResult = providerkit.HealthResult
type ProviderCatalogItem = providerkit.CatalogItem
type ProviderVariant = providerkit.Variant
type ProviderCollectionResolveParams = providerkit.CollectionResolveParams
type ProviderCollectionResult = providerkit.CollectionResult
type ProviderItem = providerkit.Item
type ProviderItemResolveParams = providerkit.ItemResolveParams
type ProviderItemResolveResult = providerkit.ItemResolveResult

func CallProvider(ctx context.Context, installation ProviderInstallation, method string, params, target any) error {
	capability, supported := providerMethodCapability[method]
	if !supported {
		return fmt.Errorf("unsupported provider method %q", method)
	}
	if !installation.Enabled {
		return errors.New("provider is disabled")
	}
	if capability != "" && !hasProviderCapability(installation.Capabilities, capability) {
		return fmt.Errorf("provider did not declare capability %q", capability)
	}

	timeout := providerMethodTimeout(method)
	callContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request := providerRPCRequest{JSONRPC: "2.0", ID: "1", Method: method, Params: params}
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	var rawResponse []byte
	kind := installation.Kind
	if kind == "" {
		kind = ProviderLocalProcess
	}
	switch kind {
	case ProviderLocalProcess:
		rawResponse, err = callLocalProcessProvider(callContext, installation, payload, timeout)
	case ProviderWebsite:
		rawResponse, err = callWebsiteProvider(callContext, installation, payload)
	default:
		return fmt.Errorf("unsupported provider kind %q", installation.Kind)
	}
	if err != nil {
		return err
	}
	return decodeProviderRPCResponse(rawResponse, request.ID, method, target)
}

func callLocalProcessProvider(callContext context.Context, installation ProviderInstallation, payload []byte, timeout time.Duration) ([]byte, error) {
	candidate, err := ReadProviderManifest(installation.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("provider manifest is no longer valid: %w", err)
	}
	if !candidate.Ready || candidate.Manifest.ID != installation.ID || candidate.Executable != installation.Executable {
		return nil, errors.New("provider installation changed after registration; register it again")
	}

	cmd := exec.CommandContext(callContext, candidate.Executable)
	cmd.Dir = candidate.RootPath
	cmd.Env = providerEnvironment()
	configureProviderProcess(cmd)
	cmd.Stdin = bytes.NewReader(payload)
	stdout := &cappedBuffer{limit: providerResponseLimit}
	stderr := &cappedBuffer{limit: providerStderrLimit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		if callContext.Err() != nil {
			return nil, fmt.Errorf("provider %s timed out after %s", installation.Name, timeout)
		}
		if stdout.exceeded {
			return nil, errors.New("provider response exceeded the 8 MiB limit")
		}
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return nil, fmt.Errorf("provider process failed: %w: %s", err, message)
		}
		return nil, fmt.Errorf("provider process failed: %w", err)
	}
	if stdout.exceeded {
		return nil, errors.New("provider response exceeded the 8 MiB limit")
	}
	return append([]byte(nil), stdout.Bytes()...), nil
}

func callWebsiteProvider(callContext context.Context, installation ProviderInstallation, payload []byte) ([]byte, error) {
	if err := validateWebsiteInstallation(installation); err != nil {
		return nil, err
	}
	guard := security.NewSSRFGuard(nil)
	if _, err := guard.ValidateURL(installation.RPCURL); err != nil {
		return nil, fmt.Errorf("website provider endpoint was blocked: %w", err)
	}
	client := websiteHTTPClient(guard.NewSecureHTTPClient(40 * time.Second))
	return callWebsiteProviderWithClient(callContext, installation, payload, client)
}

func callWebsiteProviderWithClient(callContext context.Context, installation ProviderInstallation, payload []byte, client *http.Client) ([]byte, error) {
	request, err := http.NewRequestWithContext(callContext, http.MethodPost, installation.RPCURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Arion/0.4 provider-protocol/1")
	response, err := client.Do(request)
	if err != nil {
		if callContext.Err() != nil {
			return nil, fmt.Errorf("website provider %s timed out", installation.Name)
		}
		return nil, fmt.Errorf("website provider request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("website provider returned HTTP %d", response.StatusCode)
	}
	if !isProviderJSONContentType(response.Header.Get("Content-Type")) {
		return nil, errors.New("website provider response must use a JSON content type")
	}
	body, err := security.BoundedRead(response.Body, providerResponseLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid website provider response: %w", err)
	}
	return body, nil
}

func decodeProviderRPCResponse(rawResponse []byte, requestID, method string, target any) error {

	decoder := json.NewDecoder(bytes.NewReader(rawResponse))
	decoder.DisallowUnknownFields()
	var response providerRPCResponse
	if err := decoder.Decode(&response); err != nil {
		return fmt.Errorf("invalid provider response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("provider returned more than one JSON-RPC response")
	}
	if response.JSONRPC != "2.0" || response.ID != requestID {
		return errors.New("provider returned a mismatched JSON-RPC response")
	}
	if response.Error != nil {
		return response.Error
	}
	if len(response.Result) == 0 {
		return errors.New("provider response has no result")
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(response.Result, target); err != nil {
		return fmt.Errorf("invalid result for %s: %w", method, err)
	}
	if err := validateProviderResult(method, target); err != nil {
		return err
	}
	return nil
}

func validateProviderResult(method string, target any) error {
	return providerkit.ValidateResult(method, target)
}

func providerMethodTimeout(method string) time.Duration {
	switch method {
	case "provider.describe", "provider.health":
		return 8 * time.Second
	case "catalog.search":
		return 25 * time.Second
	default:
		return 35 * time.Second
	}
}

func hasProviderCapability(capabilities []string, wanted string) bool {
	for _, capability := range capabilities {
		if capability == wanted {
			return true
		}
	}
	return false
}

func providerEnvironment() []string {
	allowed := []string{
		"PATH", "HOME", "LANG", "LC_ALL", "TMPDIR",
		"SSL_CERT_FILE", "SSL_CERT_DIR",
		"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME",
		// Windows processes need these variables to locate system DLLs, temporary
		// storage and the user's standard data directories. The allowlist still
		// excludes Arion's session token and unrelated application secrets.
		"SystemRoot", "ComSpec", "PATHEXT", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "TEMP", "TMP",
	}
	environment := []string{"ARION_PROVIDER_PROTOCOL=1"}
	for _, name := range allowed {
		if value, present := os.LookupEnv(name); present {
			environment = append(environment, name+"="+value)
		}
	}
	return environment
}

type cappedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *cappedBuffer) Write(data []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.exceeded = true
		return 0, errors.New("output limit exceeded")
	}
	if len(data) > remaining {
		_, _ = b.Buffer.Write(data[:remaining])
		b.exceeded = true
		return remaining, errors.New("output limit exceeded")
	}
	return b.Buffer.Write(data)
}
