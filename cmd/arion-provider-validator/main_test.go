// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyuubyN/Arion-Project/providerkit"
)

type handlerTransport struct {
	handler http.Handler
}

func (transport handlerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	transport.handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	response.Request = request
	return response, nil
}

func TestValidateRemoteProviderChecksManifestDescribeAndHealth(t *testing.T) {
	service := providerkit.Service{
		Manifest: providerkit.Manifest{
			SchemaVersion: 1, Kind: "website", ID: "validator.example", Name: "Validator Example",
			Version: "1.0.0", ProtocolVersion: 1, RPCPath: "/arion/rpc", Capabilities: []string{"catalog.search"},
		},
	}
	handler, err := providerkit.NewHandler(service)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: handlerTransport{handler: handler}}
	report, err := validateRemoteProvider(context.Background(), "https://provider.example/ignored", client, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.TransportChecked || report.Manifest.ID != "validator.example" || report.Health.Status != "ok" {
		t.Fatalf("unexpected report: %+v", report)
	}
}
