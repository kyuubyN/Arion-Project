// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kyuubyN/Arion-Project/providerkit"
)

func main() {
	listenAddress := os.Getenv("ARION_EXAMPLE_LISTEN")
	if listenAddress == "" {
		listenAddress = "127.0.0.1:9080"
	}
	mediaURL := os.Getenv("ARION_EXAMPLE_MEDIA_URL")
	if mediaURL == "" {
		mediaURL = "https://media.example.invalid/video.mp4"
	}
	service := providerkit.Service{
		Manifest: providerkit.Manifest{
			SchemaVersion:   providerkit.SchemaVersion,
			Kind:            providerkit.WebsiteKind,
			ID:              "example.web",
			Name:            "Arion Web Provider Example",
			Version:         "1.0.0",
			ProtocolVersion: providerkit.ProtocolVersion,
			RPCPath:         "/arion/rpc",
			Capabilities:    []string{"catalog.search", "collection.resolve", "item.resolve"},
			Author:          "Arion contributors",
			License:         "GPL-3.0-only",
		},
		Health: func(context.Context) (providerkit.HealthResult, error) {
			return providerkit.HealthResult{Status: "ok", Message: "Example provider is ready"}, nil
		},
		Search: func(_ context.Context, params providerkit.CatalogSearchParams) (providerkit.CatalogSearchResult, error) {
			return providerkit.CatalogSearchResult{Items: []providerkit.CatalogItem{{
				ID: "example-collection", Title: "Example Collection",
				Description: "Neutral sample data returned by the public provider kit.", Kind: "video_collection",
				Variants: []providerkit.Variant{{ID: "default", Label: "Default", Languages: []string{"und"}, Reference: "collection:example"}},
			}}}, nil
		},
		ResolveCollection: func(context.Context, providerkit.CollectionResolveParams) (providerkit.CollectionResult, error) {
			return providerkit.CollectionResult{
				ID: "example-collection", Title: "Example Collection",
				Description: "A provider-neutral collection used only for protocol development.",
				Items:       []providerkit.Item{{ID: "example-video", Title: "Example Video", Reference: "item:example"}},
			}, nil
		},
		ResolveItem: func(context.Context, providerkit.ItemResolveParams) (providerkit.ItemResolveResult, error) {
			return providerkit.ItemResolveResult{URL: mediaURL, MIMEType: "video/mp4"}, nil
		},
	}
	handler, err := providerkit.NewHandler(service)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr: listenAddress, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 35 * time.Second, IdleTimeout: 60 * time.Second,
	}
	log.Printf("development provider listening on http://%s", listenAddress)
	log.Print("Arion production discovery requires a public HTTPS reverse proxy")
	log.Fatal(server.ListenAndServe())
}
