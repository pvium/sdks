package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pvium/sdks/go-sdk/config"
	"github.com/pvium/sdks/go-sdk/models"
)

func TestRequestOptionsAccessTokenUsesBearerAndSuppressesAPIKey(t *testing.T) {
	t.Parallel()

	var authHeader, apiKeyHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		apiKeyHeader = r.Header.Get("x-api-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":201,"success":true},"data":{"id":123}}`))
	}))
	defer ts.Close()

	client := NewHTTPClient(config.Config{BaseURL: ts.URL, APIKey: "app_key"})
	_, _, err := client.Do(context.Background(), Request{
		Method: "POST",
		Path:   "/invoices",
		Body:   map[string]any{"name": "Reward"},
		Options: &models.RequestOptions{
			AccessToken: "access_user",
		},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if authHeader != "Bearer access_user" {
		t.Fatalf("authorization mismatch: %s", authHeader)
	}
	if apiKeyHeader != "" {
		t.Fatalf("x-api-key must be suppressed when access token is provided")
	}
}

func TestConfiguredAPIKeyUsedWhenNoAccessToken(t *testing.T) {
	t.Parallel()

	var authHeader, apiKeyHeader, contentType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		apiKeyHeader = r.Header.Get("x-api-key")
		contentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":[]}`))
	}))
	defer ts.Close()

	client := NewHTTPClient(config.Config{BaseURL: ts.URL, APIKey: "app_key"})
	_, _, err := client.Do(context.Background(), Request{Method: "GET", Path: "/invoices"})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if apiKeyHeader != "app_key" {
		t.Fatalf("x-api-key mismatch: %s", apiKeyHeader)
	}
	if authHeader != "" {
		t.Fatalf("authorization header should be empty")
	}
	if contentType != "" {
		t.Fatalf("content-type should be empty for bodyless requests, got %q", contentType)
	}
}
