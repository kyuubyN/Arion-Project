// SPDX-License-Identifier: GPL-3.0-only

package main

import "testing"

func TestArtworkTypeRejectsActiveOrNonImageContent(t *testing.T) {
	if got := normalizedArtworkType("image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script/></svg>`)); got != "" {
		t.Fatalf("SVG must not be accepted, got %q", got)
	}
	if got := normalizedArtworkType("text/html", []byte(`<html></html>`)); got != "" {
		t.Fatalf("HTML must not be accepted, got %q", got)
	}
}

func TestArtworkTypeAcceptsPNGSignature(t *testing.T) {
	raw := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 32)...)
	if got := normalizedArtworkType("application/octet-stream", raw); got != "image/png" {
		t.Fatalf("got %q", got)
	}
}
