// SPDX-License-Identifier: GPL-3.0-only

package providerkit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerServesManifestAndRPC(t *testing.T) {
	handler, err := NewHandler(testService())
	if err != nil {
		t.Fatal(err)
	}
	manifestRequest := httptest.NewRequest(http.MethodGet, WellKnownManifestPath, nil)
	manifestResponse := httptest.NewRecorder()
	handler.ServeHTTP(manifestResponse, manifestRequest)
	if manifestResponse.Code != http.StatusOK || manifestResponse.Header().Get("Content-Type") != "application/arion-provider+json" {
		t.Fatalf("unexpected manifest response: %d %#v", manifestResponse.Code, manifestResponse.Header())
	}
	if _, err := DecodeManifest(manifestResponse.Body.Bytes()); err != nil {
		t.Fatal(err)
	}

	rpcRequest := httptest.NewRequest(http.MethodPost, "/arion/rpc", bytes.NewBufferString(`{"jsonrpc":"2.0","id":"test-1","method":"catalog.search","params":{"query":"demo","limit":5,"mode":"preview"}}`))
	rpcRequest.Header.Set("Content-Type", "application/json")
	rpcResponse := httptest.NewRecorder()
	handler.ServeHTTP(rpcResponse, rpcRequest)
	if rpcResponse.Code != http.StatusOK {
		t.Fatalf("RPC status = %d: %s", rpcResponse.Code, rpcResponse.Body.String())
	}
	var envelope struct {
		Result CatalogSearchResult `json:"result"`
		Error  *RPCError           `json:"error"`
	}
	if err := json.Unmarshal(rpcResponse.Body.Bytes(), &envelope); err != nil || envelope.Error != nil || len(envelope.Result.Items) != 1 {
		t.Fatalf("unexpected RPC response: %s, err = %v", rpcResponse.Body, err)
	}
}

func TestHandlerRejectsUnknownParamsAndUndeclaredMethods(t *testing.T) {
	service := testService()
	service.Manifest.Capabilities = []string{"catalog.search"}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		`{"jsonrpc":"2.0","id":"1","method":"catalog.search","params":{"query":"demo","unknown":true}}`,
		`{"jsonrpc":"2.0","id":"2","method":"item.resolve","params":{"reference":"demo"}}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/arion/rpc", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"error"`)) {
			t.Fatalf("expected JSON-RPC error, got %d %s", response.Code, response.Body)
		}
	}
}

func TestManifestValidationRejectsUnsafeRPCPath(t *testing.T) {
	manifest := testService().Manifest
	manifest.RPCPath = "https://other.example/rpc"
	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("expected cross-origin RPC URL to be rejected")
	}
}

func TestNormalizePublicOriginRejectsMalformedHostname(t *testing.T) {
	for _, rawURL := range []string{"https://a..example", "https://-bad.example", "https://bad-.example"} {
		if _, err := NormalizePublicOrigin(rawURL); err == nil {
			t.Fatalf("expected %q to be rejected", rawURL)
		}
	}
}

func testService() Service {
	return Service{
		Manifest: Manifest{
			SchemaVersion: SchemaVersion, Kind: WebsiteKind, ID: "example.web", Name: "Example",
			Version: "1.0.0", ProtocolVersion: ProtocolVersion, RPCPath: "/arion/rpc",
			Capabilities: []string{"catalog.search", "collection.resolve", "item.resolve"},
		},
		Health: func(context.Context) (HealthResult, error) { return HealthResult{Status: "ok"}, nil },
		Search: func(_ context.Context, params CatalogSearchParams) (CatalogSearchResult, error) {
			return CatalogSearchResult{Items: []CatalogItem{{ID: "demo", Title: "Demo", Variants: []Variant{{ID: "default", Label: "Default", Reference: "demo"}}}}}, nil
		},
		ResolveCollection: func(context.Context, CollectionResolveParams) (CollectionResult, error) {
			return CollectionResult{ID: "demo", Title: "Demo", Items: []Item{{ID: "1", Title: "Video 1", Reference: "demo:1"}}}, nil
		},
		ResolveItem: func(context.Context, ItemResolveParams) (ItemResolveResult, error) {
			return ItemResolveResult{URL: "https://media.example/video.mp4", MIMEType: "video/mp4"}, nil
		},
	}
}
