package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/pvium/sdks/go-sdk/config"
	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

type capturedOAuthRequest struct {
	Path     string
	Method   string
	APIKey   string
	Body     map[string]any
	Requests int
}

func createMockOAuthService(t *testing.T) (*OAuthService, *capturedOAuthRequest, func()) {
	t.Helper()

	captured := &capturedOAuthRequest{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Requests++
		captured.Path = r.URL.Path
		captured.Method = r.Method
		captured.APIKey = r.Header.Get("x-api-key")
		if err := json.NewDecoder(r.Body).Decode(&captured.Body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":{"accessToken":"access_token","refreshToken":"refresh_token","expiresIn":3600}}`))
	}))

	client := transport.NewHTTPClient(config.Config{
		BaseURL:  ts.URL,
		APIKey:   "pk_test_dummy",
		ClientID: "app_test",
	})

	return NewOAuthService(client), captured, ts.Close
}

func TestExchangeCodeForTokenSendsAPIKeyInTokenRequestBody(t *testing.T) {
	t.Parallel()

	service, captured, cleanup := createMockOAuthService(t)
	defer cleanup()

	_, err := service.ExchangeCodeForToken(context.Background(), models.ExchangeAuthorizationCodeInput{
		Code:        "oauth_code",
		RedirectURI: "https://example.test/callback",
	}, nil)
	if err != nil {
		t.Fatalf("exchange code for token: %v", err)
	}

	if captured.Path != "/client-apps/oauth2/token" {
		t.Fatalf("url path mismatch: got %s want %s", captured.Path, "/client-apps/oauth2/token")
	}
	if captured.Method != http.MethodPost {
		t.Fatalf("method mismatch: got %s want %s", captured.Method, http.MethodPost)
	}
	if captured.APIKey != "" {
		t.Fatalf("expected x-api-key header to be omitted, got %q", captured.APIKey)
	}

	wantBody := map[string]any{
		"clientId":    "app_test",
		"apiKey":      "pk_test_dummy",
		"grantType":   "authorization_code",
		"code":        "oauth_code",
		"redirectUri": "https://example.test/callback",
	}
	if !reflect.DeepEqual(captured.Body, wantBody) {
		t.Fatalf("body mismatch:\ngot  %#v\nwant %#v", captured.Body, wantBody)
	}
}

func TestRefreshAccessTokenSendsAPIKeyInTokenRequestBody(t *testing.T) {
	t.Parallel()

	service, captured, cleanup := createMockOAuthService(t)
	defer cleanup()

	_, err := service.RefreshAccessToken(context.Background(), models.RefreshAccessTokenInput{
		RefreshToken: "refresh_token",
	}, nil)
	if err != nil {
		t.Fatalf("refresh access token: %v", err)
	}

	if captured.APIKey != "" {
		t.Fatalf("expected x-api-key header to be omitted, got %q", captured.APIKey)
	}

	wantBody := map[string]any{
		"clientId":     "app_test",
		"apiKey":       "pk_test_dummy",
		"grantType":    "refresh_token",
		"refreshToken": "refresh_token",
	}
	if !reflect.DeepEqual(captured.Body, wantBody) {
		t.Fatalf("body mismatch:\ngot  %#v\nwant %#v", captured.Body, wantBody)
	}
}

func TestGetAccessTokenFromRefreshTokenRefreshesThroughOAuthTokenEndpoint(t *testing.T) {
	t.Parallel()

	service, captured, cleanup := createMockOAuthService(t)
	defer cleanup()

	_, err := service.GetAccessTokenFromRefreshToken(context.Background(), models.RefreshAccessTokenInput{
		RefreshToken: "refresh_token",
	}, nil)
	if err != nil {
		t.Fatalf("get access token from refresh token: %v", err)
	}

	if captured.Path != "/client-apps/oauth2/token" {
		t.Fatalf("url path mismatch: got %s want %s", captured.Path, "/client-apps/oauth2/token")
	}
	if captured.Method != http.MethodPost {
		t.Fatalf("method mismatch: got %s want %s", captured.Method, http.MethodPost)
	}
	if captured.APIKey != "" {
		t.Fatalf("expected x-api-key header to be omitted, got %q", captured.APIKey)
	}

	wantBody := map[string]any{
		"clientId":     "app_test",
		"apiKey":       "pk_test_dummy",
		"grantType":    "refresh_token",
		"refreshToken": "refresh_token",
	}
	if !reflect.DeepEqual(captured.Body, wantBody) {
		t.Fatalf("body mismatch:\ngot  %#v\nwant %#v", captured.Body, wantBody)
	}
}

func TestExchangeCodeForTokenRequiresClientIDAndAPIKey(t *testing.T) {
	t.Parallel()

	client := transport.NewHTTPClient(config.Config{BaseURL: "https://example.com"})
	service := NewOAuthService(client)

	_, err := service.ExchangeCodeForToken(context.Background(), models.ExchangeAuthorizationCodeInput{
		Code:        "code-1",
		RedirectURI: "https://example.com/callback",
	}, nil)
	if err == nil {
		t.Fatal("expected error when clientId and apiKey are missing")
	}
}
