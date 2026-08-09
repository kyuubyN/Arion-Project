// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestNormalizeWebsiteOriginAcceptsOnlyPublicHTTPSDomains(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "media.example/path?q=ignored", want: "https://media.example", ok: true},
		{input: "https://MEDIA.EXAMPLE/collection", want: "https://media.example", ok: true},
		{input: "http://media.example", ok: false},
		{input: "https://localhost", ok: false},
		{input: "https://127.0.0.1", ok: false},
		{input: "https://user:pass@media.example", ok: false},
		{input: "https://media.example:8443", ok: false},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := normalizeWebsiteOrigin(test.input)
			if (err == nil) != test.ok || got != test.want {
				t.Fatalf("origin = %q, err = %v", got, err)
			}
		})
	}
}

func TestProbeWebsiteProviderReadsOnlyWellKnownManifest(t *testing.T) {
	manifest := `{"schema_version":1,"kind":"website","id":"example.web","name":"Example","version":"1.0.0","protocol_version":1,"rpc_path":"/arion/rpc","capabilities":["catalog.search"]}`
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.String() != "https://media.example/.well-known/arion-provider.json" {
			t.Fatalf("unexpected manifest request: %s %s", request.Method, request.URL)
		}
		return jsonHTTPResponse(http.StatusOK, manifest), nil
	})}
	candidate, err := probeWebsiteProviderWithClient(context.Background(), "https://media.example/arbitrary/page", client, func(target string) error {
		if !strings.HasPrefix(target, "https://media.example/") {
			t.Fatalf("validated a different origin: %s", target)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.RPCURL != "https://media.example/arion/rpc" || !candidate.Ready {
		t.Fatalf("unexpected candidate: %+v", candidate)
	}
}

func TestWebsiteManifestRejectsCrossOriginOrAmbiguousRPCPath(t *testing.T) {
	for _, rpcPath := range []string{"https://other.example/rpc", "//other.example/rpc", "/a/../rpc", "/rpc?token=value"} {
		if _, err := websiteRPCURL("https://media.example", rpcPath); err == nil {
			t.Fatalf("expected rpc path %q to be rejected", rpcPath)
		}
	}
}

func TestWebsiteProviderRPCUsesJSONWithoutCookies(t *testing.T) {
	installation := ProviderInstallation{Kind: ProviderWebsite, ID: "example.web", Name: "Example", RPCURL: "https://media.example/arion/rpc"}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/json" || request.Header.Get("Cookie") != "" || request.Header.Get("Authorization") != "" {
			t.Fatalf("unsafe or invalid RPC request: %#v", request.Header)
		}
		return jsonHTTPResponse(http.StatusOK, `{"jsonrpc":"2.0","id":"1","result":{"status":"ok"}}`), nil
	})}
	payload := []byte(`{"jsonrpc":"2.0","id":"1","method":"provider.health"}`)
	raw, err := callWebsiteProviderWithClient(context.Background(), installation, payload, client)
	if err != nil {
		t.Fatal(err)
	}
	var health ProviderHealthResult
	if err := decodeProviderRPCResponse(raw, "1", "provider.health", &health); err != nil || health.Status != "ok" {
		t.Fatalf("health = %+v, err = %v", health, err)
	}
}

func TestWebsiteProviderClientRejectsEveryRedirect(t *testing.T) {
	client := websiteHTTPClient(&http.Client{})
	request, _ := http.NewRequest(http.MethodGet, "https://media.example/elsewhere", nil)
	if err := client.CheckRedirect(request, nil); err == nil {
		t.Fatal("expected redirect to be rejected")
	}
}

func jsonHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
