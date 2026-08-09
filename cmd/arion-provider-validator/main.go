// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/kyuubyN/Arion-Project/backend/security"
	"github.com/kyuubyN/Arion-Project/providerkit"
)

const maxRPCResponseBytes = 8 * 1024 * 1024

type validationReport struct {
	Manifest         providerkit.Manifest
	Origin           string
	RPCURL           string
	Health           providerkit.HealthResult
	TransportChecked bool
}

func main() {
	filePath := flag.String("file", "", "validate a local website provider manifest")
	rawURL := flag.String("url", "", "validate a public HTTPS provider and its RPC transport")
	timeout := flag.Duration("timeout", 15*time.Second, "network timeout")
	flag.Parse()
	if (*filePath == "") == (*rawURL == "") {
		fmt.Fprintln(os.Stderr, "use exactly one of -file or -url")
		flag.Usage()
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	var report validationReport
	var err error
	if *filePath != "" {
		report, err = validateManifestFile(*filePath)
	} else {
		guard := security.NewSSRFGuard(nil)
		client := guard.NewSecureHTTPClient(*timeout)
		client.Jar = nil
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return errors.New("provider redirects are not allowed")
		}
		report, err = validateRemoteProvider(ctx, *rawURL, client, func(target string) error {
			_, err := guard.ValidateURL(target)
			return err
		})
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "INVALID: %v\n", err)
		os.Exit(1)
	}
	printReport(report)
}

func validateManifestFile(path string) (validationReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return validationReport{}, err
	}
	defer file.Close()
	data, err := security.BoundedRead(file, providerkit.MaxManifestBytes)
	if err != nil {
		return validationReport{}, err
	}
	manifest, err := providerkit.DecodeManifest(data)
	if err != nil {
		return validationReport{}, err
	}
	return validationReport{Manifest: manifest}, nil
}

func validateRemoteProvider(ctx context.Context, rawURL string, client *http.Client, validateURL func(string) error) (validationReport, error) {
	origin, err := providerkit.NormalizePublicOrigin(rawURL)
	if err != nil {
		return validationReport{}, err
	}
	manifestURL := origin + providerkit.WellKnownManifestPath
	if validateURL != nil {
		if err := validateURL(manifestURL); err != nil {
			return validationReport{}, fmt.Errorf("manifest destination blocked: %w", err)
		}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return validationReport{}, err
	}
	request.Header.Set("Accept", "application/arion-provider+json, application/json")
	request.Header.Set("User-Agent", "Arion-Provider-Validator/0.4.1")
	response, err := client.Do(request)
	if err != nil {
		return validationReport{}, fmt.Errorf("manifest request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return validationReport{}, fmt.Errorf("manifest returned HTTP %d", response.StatusCode)
	}
	if !providerkit.IsJSONContentType(response.Header.Get("Content-Type")) {
		return validationReport{}, errors.New("manifest response does not use a supported JSON content type")
	}
	data, err := security.BoundedRead(response.Body, providerkit.MaxManifestBytes)
	if err != nil {
		return validationReport{}, err
	}
	manifest, err := providerkit.DecodeManifest(data)
	if err != nil {
		return validationReport{}, err
	}
	rpcURL, err := providerkit.ResolveRPCURL(origin, manifest.RPCPath)
	if err != nil {
		return validationReport{}, err
	}
	if validateURL != nil {
		if err := validateURL(rpcURL); err != nil {
			return validationReport{}, fmt.Errorf("RPC destination blocked: %w", err)
		}
	}
	var described providerkit.Manifest
	if err := callRPC(ctx, client, rpcURL, "validator-describe", "provider.describe", nil, &described); err != nil {
		return validationReport{}, err
	}
	if err := providerkit.ValidateManifest(described); err != nil || !reflect.DeepEqual(described, manifest) {
		return validationReport{}, errors.New("provider.describe does not match the discovered manifest")
	}
	var health providerkit.HealthResult
	if err := callRPC(ctx, client, rpcURL, "validator-health", "provider.health", nil, &health); err != nil {
		return validationReport{}, err
	}
	if err := providerkit.ValidateResult("provider.health", &health); err != nil {
		return validationReport{}, err
	}
	return validationReport{Manifest: manifest, Origin: origin, RPCURL: rpcURL, Health: health, TransportChecked: true}, nil
}

func callRPC(ctx context.Context, client *http.Client, rpcURL, id, method string, params any, target any) error {
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Arion-Provider-Validator/0.4.1")
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", method, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP %d", method, response.StatusCode)
	}
	if !providerkit.IsJSONContentType(response.Header.Get("Content-Type")) {
		return fmt.Errorf("%s response does not use application/json", method)
	}
	data, err := security.BoundedRead(response.Body, maxRPCResponseBytes)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var envelope struct {
		JSONRPC string                `json:"jsonrpc"`
		ID      string                `json:"id"`
		Result  json.RawMessage       `json:"result,omitempty"`
		Error   *providerkit.RPCError `json:"error,omitempty"`
	}
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("invalid %s response: %w", method, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s returned more than one JSON value", method)
	}
	if envelope.JSONRPC != "2.0" || envelope.ID != id {
		return fmt.Errorf("%s returned a mismatched JSON-RPC envelope", method)
	}
	if envelope.Error != nil {
		return envelope.Error
	}
	if len(envelope.Result) == 0 {
		return fmt.Errorf("%s returned no result", method)
	}
	resultDecoder := json.NewDecoder(bytes.NewReader(envelope.Result))
	resultDecoder.DisallowUnknownFields()
	if err := resultDecoder.Decode(target); err != nil {
		return fmt.Errorf("invalid %s result: %w", method, err)
	}
	return nil
}

func printReport(report validationReport) {
	fmt.Printf("OK manifesto: %s (%s)\n", report.Manifest.Name, report.Manifest.ID)
	fmt.Printf("OK protocolo: %d | capacidades: %s\n", report.Manifest.ProtocolVersion, strings.Join(report.Manifest.Capabilities, ", "))
	if report.TransportChecked {
		fmt.Printf("OK origem HTTPS: %s\n", report.Origin)
		fmt.Printf("OK RPC mesma origem: %s\n", report.RPCURL)
		fmt.Printf("OK provider.describe\n")
		fmt.Printf("OK provider.health: %s", report.Health.Status)
		if report.Health.Message != "" {
			fmt.Printf(" — %s", report.Health.Message)
		}
		fmt.Println()
	} else {
		fmt.Println("AVISO transporte não testado; use -url após publicar em HTTPS")
	}
}
