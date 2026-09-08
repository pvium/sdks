package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/pvium/sdks/go-sdk/config"
	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

func TestPayabilitySharedFixture(t *testing.T) {
	raw, err := os.ReadFile("../../parity-fixtures/payability.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		PayoutID string `json:"payoutId"`
		Method   string `json:"method"`
		Path     string `json:"path"`
		Body     struct {
			Identities []models.PayoutPayabilityIdentity `json:"identities"`
		} `json:"body"`
		Responses []json.RawMessage `json:"responses"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, response := range fixture.Responses {
		for _, intent := range []bool{false, true} {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != fixture.Method || r.URL.EscapedPath() != fixture.Path || r.URL.RawQuery != "" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
				}
				if r.Header.Get("Authorization") != "Bearer parity-token" || r.Header.Get("x-api-key") != "" {
					t.Error("request options were not forwarded correctly")
				}
				var body struct {
					Identities []models.PayoutPayabilityIdentity `json:"identities"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(body, fixture.Body) {
					t.Errorf("unexpected body: %+v", body)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(response)
			}))
			service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL + "/v1", APIKey: "app-key"}))
			options := &models.RequestOptions{AccessToken: "parity-token"}
			var result models.APIResponse[models.PayoutPayabilityResult]
			if intent {
				payout := &PayoutIntent{PayoutRecord: models.PayoutRecord{ID: fixture.PayoutID}, service: service}
				result, err = payout.IsPayable(context.Background(), fixture.Body.Identities, options)
			} else {
				result, err = service.IsPayable(context.Background(), fixture.PayoutID, fixture.Body.Identities, options)
			}
			ts.Close()
			if err != nil {
				t.Fatal(err)
			}
			// JSON round-trip checks every field and preserves false versus null.
			encoded, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(response, &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("response mismatch: got %s want %s", encoded, response)
			}
		}
	}
}

func TestPayabilityUnboundIntent(t *testing.T) {
	var intent *PayoutIntent
	if _, err := intent.IsPayable(context.Background(), nil, nil); err == nil {
		t.Fatal("expected unbound intent error")
	}
}

func TestPayabilityPropagatesAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"meta":{"success":false,"message":"Batch not found"}}`))
	}))
	defer ts.Close()
	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	if _, err := service.IsPayable(context.Background(), "missing", nil, nil); err == nil {
		t.Fatal("expected API error")
	}
}
