// SPDX-License-Identifier: GPL-3.0-only

package providerkit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const maxRPCRequestBytes = 1024 * 1024

type Service struct {
	Manifest          Manifest
	Health            func(context.Context) (HealthResult, error)
	Search            func(context.Context, CatalogSearchParams) (CatalogSearchResult, error)
	ResolveCollection func(context.Context, CollectionResolveParams) (CollectionResult, error)
	ResolveItem       func(context.Context, ItemResolveParams) (ItemResolveResult, error)
}

// NewHandler creates an HTTP handler for the well-known manifest and its RPC endpoint.
// Authentication, rate limits and public TLS termination remain the provider operator's responsibility.
func NewHandler(service Service) (http.Handler, error) {
	if err := ValidateManifest(service.Manifest); err != nil {
		return nil, err
	}
	manifestJSON, err := json.Marshal(service.Manifest)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc(WellKnownManifestPath, func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			response.Header().Set("Allow", "GET, HEAD")
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "application/arion-provider+json")
		response.Header().Set("Cache-Control", "public, max-age=300")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.WriteHeader(http.StatusOK)
		if request.Method == http.MethodGet {
			_, _ = response.Write(manifestJSON)
		}
	})
	mux.HandleFunc(service.Manifest.RPCPath, func(response http.ResponseWriter, request *http.Request) {
		serveRPC(service, response, request)
	})
	return mux, nil
}

func serveRPC(service Service, response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", "POST")
		writeRPCError(response, "", http.StatusMethodNotAllowed, -32600, "JSON-RPC requires POST")
		return
	}
	if !IsJSONContentType(request.Header.Get("Content-Type")) {
		writeRPCError(response, "", http.StatusUnsupportedMediaType, -32600, "Content-Type must be application/json")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(response, request.Body, maxRPCRequestBytes))
	if err != nil {
		writeRPCError(response, "", http.StatusRequestEntityTooLarge, -32600, "request body is invalid or too large")
		return
	}
	var rpcRequest RPCRequest
	if err := decodeStrictJSON(body, &rpcRequest); err != nil {
		writeRPCError(response, "", http.StatusBadRequest, -32700, "invalid JSON-RPC request")
		return
	}
	if rpcRequest.JSONRPC != "2.0" || strings.TrimSpace(rpcRequest.ID) == "" || strings.TrimSpace(rpcRequest.Method) == "" {
		writeRPCError(response, rpcRequest.ID, http.StatusBadRequest, -32600, "invalid JSON-RPC envelope")
		return
	}
	result, rpcErr := dispatch(service, request.Context(), rpcRequest)
	if rpcErr != nil {
		writeRPCResponse(response, http.StatusOK, RPCResponse{JSONRPC: "2.0", ID: rpcRequest.ID, Error: rpcErr})
		return
	}
	writeRPCResponse(response, http.StatusOK, RPCResponse{JSONRPC: "2.0", ID: rpcRequest.ID, Result: result})
}

func dispatch(service Service, ctx context.Context, request RPCRequest) (any, *RPCError) {
	switch request.Method {
	case "provider.describe":
		return service.Manifest, nil
	case "provider.health":
		if service.Health == nil {
			return HealthResult{Status: "ok"}, nil
		}
		result, err := service.Health(ctx)
		return validatedResult(request.Method, result, err)
	case "catalog.search":
		if !HasCapability(service.Manifest.Capabilities, "catalog.search") || service.Search == nil {
			return nil, &RPCError{Code: -32601, Message: "method not available"}
		}
		var params CatalogSearchParams
		if err := decodeStrictJSON(request.Params, &params); err != nil {
			return nil, &RPCError{Code: -32602, Message: "invalid catalog.search params"}
		}
		params.Query = strings.TrimSpace(params.Query)
		if len(params.Query) < 2 || len(params.Query) > 200 {
			return nil, &RPCError{Code: -32602, Message: "query must contain between 2 and 200 characters"}
		}
		if params.Limit <= 0 {
			params.Limit = 20
		}
		if params.Limit > 50 || (params.Mode != "" && params.Mode != "preview" && params.Mode != "complete") {
			return nil, &RPCError{Code: -32602, Message: "invalid catalog.search limit or mode"}
		}
		result, err := service.Search(ctx, params)
		return validatedResult(request.Method, result, err)
	case "collection.resolve":
		if !HasCapability(service.Manifest.Capabilities, "collection.resolve") || service.ResolveCollection == nil {
			return nil, &RPCError{Code: -32601, Message: "method not available"}
		}
		var params CollectionResolveParams
		if err := decodeStrictJSON(request.Params, &params); err != nil || strings.TrimSpace(params.Reference) == "" || len(params.Reference) > 256*1024 {
			return nil, &RPCError{Code: -32602, Message: "invalid collection.resolve params"}
		}
		result, err := service.ResolveCollection(ctx, params)
		return validatedResult(request.Method, result, err)
	case "item.resolve":
		if !HasCapability(service.Manifest.Capabilities, "item.resolve") || service.ResolveItem == nil {
			return nil, &RPCError{Code: -32601, Message: "method not available"}
		}
		var params ItemResolveParams
		if err := decodeStrictJSON(request.Params, &params); err != nil || strings.TrimSpace(params.Reference) == "" || len(params.Reference) > 256*1024 {
			return nil, &RPCError{Code: -32602, Message: "invalid item.resolve params"}
		}
		result, err := service.ResolveItem(ctx, params)
		return validatedResult(request.Method, result, err)
	default:
		return nil, &RPCError{Code: -32601, Message: "method not found"}
	}
}

func validatedResult(method string, result any, err error) (any, *RPCError) {
	if err != nil {
		return nil, &RPCError{Code: -32000, Message: "provider operation failed"}
	}
	if err := ValidateResult(method, result); err != nil {
		return nil, &RPCError{Code: -32603, Message: err.Error()}
	}
	return result, nil
}

func decodeStrictJSON(data []byte, target any) error {
	if len(data) == 0 {
		return errors.New("missing JSON value")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("more than one JSON value")
	}
	return nil
}

func writeRPCError(response http.ResponseWriter, id string, status, code int, message string) {
	writeRPCResponse(response, status, RPCResponse{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: message}})
}

func writeRPCResponse(response http.ResponseWriter, status int, value RPCResponse) {
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}
