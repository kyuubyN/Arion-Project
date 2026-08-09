// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestDiscoverProvidersUsesNeutralManifest(t *testing.T) {
	root := t.TempDir()
	providerDir := filepath.Join(root, "example")
	if err := os.MkdirAll(filepath.Join(providerDir, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(providerDir, "bin", "provider")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schema_version":1,"id":"example.local","name":"Example","version":"1.0.0","protocol_version":1,"executable":"bin/provider","capabilities":["catalog.search"]}`
	if err := os.WriteFile(filepath.Join(providerDir, providerManifestName), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	candidates, err := DiscoverProviders(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || !candidates[0].Ready || candidates[0].Manifest.ID != "example.local" {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
}

func TestProviderExecutableCannotEscapeManifestDirectory(t *testing.T) {
	if _, err := resolveProviderExecutable(t.TempDir(), "../outside"); err == nil {
		t.Fatal("expected escaping executable to be rejected")
	}
}

func TestProviderExecutableSymlinkCannotEscapeManifestDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(outside, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "provider")); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveProviderExecutable(root, "provider"); err == nil {
		t.Fatal("expected escaping executable symlink to be rejected")
	}
}

func TestCallProviderJSONRPC(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test provider uses a POSIX shell")
	}
	installation := testProviderInstallation(t, `#!/bin/sh
read request
printf '%s\n' '{"jsonrpc":"2.0","id":"1","result":{"items":[{"id":"demo","title":"Demo","variants":[]}]}}'
`)
	var result ProviderCatalogSearchResult
	if err := CallProvider(context.Background(), installation, "catalog.search", ProviderCatalogSearchParams{Query: "demo"}, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Title != "Demo" {
		t.Fatalf("unexpected provider result: %#v", result)
	}
}

func TestCallProviderEnforcesCapabilities(t *testing.T) {
	installation := ProviderInstallation{Enabled: true}
	if err := CallProvider(context.Background(), installation, "catalog.search", nil, nil); err == nil {
		t.Fatal("expected undeclared capability to be denied")
	}
}

func TestCallProviderHonorsParentDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test provider uses a POSIX shell")
	}
	installation := testProviderInstallation(t, "#!/bin/sh\nread request\nsleep 2\n")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := CallProvider(ctx, installation, "catalog.search", ProviderCatalogSearchParams{Query: "demo"}, nil); err == nil {
		t.Fatal("expected provider timeout")
	}
	if time.Since(started) > time.Second {
		t.Fatal("provider process was not terminated with its context")
	}
}

func TestProviderIdentityCannotBeSilentlyReplaced(t *testing.T) {
	store := &SettingsStore{
		filePath: filepath.Join(t.TempDir(), "settings.json"),
		value: AppSettings{Providers: []ProviderInstallation{{
			Kind: ProviderLocalProcess, ID: "example.shared", Name: "Local", ManifestPath: "/provider/arion-provider.json",
		}}},
	}
	website := ProviderInstallation{Kind: ProviderWebsite, ID: "example.shared", Name: "Remote", Origin: "https://media.example"}
	if err := store.AddProvider(website); err == nil {
		t.Fatal("expected cross-kind provider replacement to be rejected")
	}
	if err := store.RemoveProvider("example.shared"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddProvider(website); err != nil {
		t.Fatal(err)
	}
}

func testProviderInstallation(t *testing.T, script string) ProviderInstallation {
	t.Helper()
	root := t.TempDir()
	executable := filepath.Join(root, "provider")
	if err := os.WriteFile(executable, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, providerManifestName)
	manifest := `{"schema_version":1,"id":"example.local","name":"Example","version":"1.0.0","protocol_version":1,"executable":"provider","capabilities":["catalog.search"]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	candidate, err := ReadProviderManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	return candidateInstallation(candidate)
}
