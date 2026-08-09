// SPDX-License-Identifier: GPL-3.0-only

package providerkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      string    `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("provider error %d: %s", e.Code, e.Message)
}

type CatalogSearchParams struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
	Mode  string `json:"mode,omitempty"`
}

type CatalogSearchResult struct {
	Items []CatalogItem `json:"items"`
}

type HealthResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type CatalogItem struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	AlternativeTitles []string  `json:"alternative_titles,omitempty"`
	Description       string    `json:"description,omitempty"`
	ArtworkURL        string    `json:"artwork_url,omitempty"`
	Kind              string    `json:"kind,omitempty"`
	Year              int       `json:"year,omitempty"`
	Rating            float64   `json:"rating,omitempty"`
	Genres            []string  `json:"genres,omitempty"`
	Variants          []Variant `json:"variants"`
}

type Variant struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Languages []string `json:"languages,omitempty"`
	Audio     []string `json:"audio,omitempty"`
	Quality   string   `json:"quality,omitempty"`
	Reference string   `json:"reference"`
}

type CollectionResolveParams struct {
	Reference string `json:"reference"`
}

type CollectionResult struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ArtworkURL  string `json:"artwork_url,omitempty"`
	Items       []Item `json:"items"`
}

type Item struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Number     string `json:"number,omitempty"`
	Duration   int    `json:"duration_seconds,omitempty"`
	ArtworkURL string `json:"artwork_url,omitempty"`
	Reference  string `json:"reference"`
}

type ItemResolveParams struct {
	Reference string `json:"reference"`
	Quality   string `json:"quality,omitempty"`
}

type ItemResolveResult struct {
	URL       string            `json:"url"`
	MIMEType  string            `json:"mime_type,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt string            `json:"expires_at,omitempty"`
}

func ValidateResult(method string, target any) error {
	switch result := target.(type) {
	case CatalogSearchResult:
		return validateCatalogSearchResult(result)
	case *CatalogSearchResult:
		return validateCatalogSearchResult(*result)
	case HealthResult:
		if (result.Status != "ok" && result.Status != "degraded" && result.Status != "unavailable") || len(result.Message) > 4096 {
			return errors.New("provider returned an invalid health status")
		}
	case *HealthResult:
		return ValidateResult(method, *result)
	case CollectionResult:
		if strings.TrimSpace(result.ID) == "" || strings.TrimSpace(result.Title) == "" || len(result.ID) > 512 || len(result.Title) > 1000 || len(result.Description) > 256*1024 || len(result.ArtworkURL) > 8192 || len(result.Items) > 5000 {
			return errors.New("provider returned an invalid collection")
		}
		for _, item := range result.Items {
			if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Reference) == "" || len(item.ID) > 512 || len(item.Title) > 1000 || len(item.Reference) > 256*1024 || len(item.ArtworkURL) > 8192 {
				return errors.New("provider collection item is missing id, title or reference")
			}
		}
	case *CollectionResult:
		return ValidateResult(method, *result)
	case ItemResolveResult:
		if strings.TrimSpace(result.URL) == "" || len(result.URL) > 8192 || len(result.MIMEType) > 256 || len(result.Headers) > 64 {
			return errors.New("provider returned an invalid media result")
		}
		for name, value := range result.Headers {
			if len(name) == 0 || len(name) > 128 || len(value) > 8192 || strings.ContainsAny(name+value, "\r\n") {
				return errors.New("provider returned an invalid media header")
			}
		}
	case *ItemResolveResult:
		return ValidateResult(method, *result)
	default:
		return fmt.Errorf("cannot validate result for %s", method)
	}
	return nil
}

func validateCatalogSearchResult(result CatalogSearchResult) error {
	if len(result.Items) > 100 {
		return errors.New("provider returned more than 100 catalog items")
	}
	for _, item := range result.Items {
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Title) == "" || len(item.ID) > 512 || len(item.Title) > 1000 || len(item.Description) > 256*1024 || len(item.ArtworkURL) > 8192 || len(item.AlternativeTitles) > 64 || len(item.Genres) > 64 {
			return errors.New("provider catalog item is invalid or oversized")
		}
		if len(item.Variants) > 32 {
			return errors.New("provider catalog item has too many variants")
		}
		for _, variant := range item.Variants {
			if strings.TrimSpace(variant.ID) == "" || strings.TrimSpace(variant.Label) == "" || strings.TrimSpace(variant.Reference) == "" || len(variant.ID) > 512 || len(variant.Label) > 1000 || len(variant.Reference) > 256*1024 {
				return errors.New("provider variant is invalid or oversized")
			}
		}
	}
	return nil
}
