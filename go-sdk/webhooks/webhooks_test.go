package webhooks

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pvium/sdks/go-sdk/models"
)

func signWebhookToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestVerifyPviumWebhookTokenVerifiesBackendHS256WebhookJWTs(t *testing.T) {
	t.Parallel()

	token := signWebhookToken(t, "webhook_secret", jwt.MapClaims{
		"event": "oauth.invite.accepted",
		"data":  map[string]any{"githubLogin": "octocat"},
		"iat":   1700000000,
		"exp":   4000000000,
	})

	payload, err := VerifyPviumWebhookToken(token, "webhook_secret", &models.VerifyPviumWebhookTokenOptions{
		ExpectedEvent: "oauth.invite.accepted",
	})
	if err != nil {
		t.Fatalf("verify webhook token: %v", err)
	}

	if payload.Event != "oauth.invite.accepted" {
		t.Fatalf("event mismatch: got %s want %s", payload.Event, "oauth.invite.accepted")
	}
	wantData := map[string]any{"githubLogin": "octocat"}
	if !reflect.DeepEqual(payload.Data, wantData) {
		t.Fatalf("data mismatch:\ngot  %#v\nwant %#v", payload.Data, wantData)
	}
}

func TestVerifyPviumWebhookTokenSupportsBackendTokensSignedWithSHA256Secret(t *testing.T) {
	t.Parallel()

	secret := "secret_abc123"
	hashed := sha256.Sum256([]byte(secret))
	token := signWebhookToken(t, fmt.Sprintf("%x", hashed), jwt.MapClaims{
		"event": "batch.payee.added",
		"data":  map[string]any{"batch": map[string]any{"id": "batch_123"}},
		"exp":   4000000000,
	})

	payload, err := VerifyPviumWebhookToken(token, secret, nil)
	if err != nil {
		t.Fatalf("verify token with hashed-secret fallback: %v", err)
	}

	if payload.Event != "batch.payee.added" {
		t.Fatalf("event mismatch: got %s want %s", payload.Event, "batch.payee.added")
	}
	wantData := map[string]any{"batch": map[string]any{"id": "batch_123"}}
	if !reflect.DeepEqual(payload.Data, wantData) {
		t.Fatalf("data mismatch:\ngot  %#v\nwant %#v", payload.Data, wantData)
	}
}

func TestResolvePviumWebhookPayloadReturnsTokenDataAndChecksBodyEvent(t *testing.T) {
	t.Parallel()

	token := signWebhookToken(t, "webhook_secret", jwt.MapClaims{
		"event": "invoice.paid",
		"data":  map[string]any{"invoiceId": "inv_123"},
		"exp":   4000000000,
	})

	resolved, err := ResolvePviumWebhookPayload(map[string]any{
		"event": "invoice.paid",
		"token": token,
	}, "webhook_secret")
	if err != nil {
		t.Fatalf("resolve webhook payload: %v", err)
	}

	if resolved.Event != "invoice.paid" {
		t.Fatalf("event mismatch: got %s want %s", resolved.Event, "invoice.paid")
	}
	wantData := map[string]any{"invoiceId": "inv_123"}
	if !reflect.DeepEqual(resolved.Data, wantData) {
		t.Fatalf("data mismatch:\ngot  %#v\nwant %#v", resolved.Data, wantData)
	}
}

func TestVerifyPviumWebhookTokenRejectsExpiredTokens(t *testing.T) {
	t.Parallel()

	token := signWebhookToken(t, "webhook_secret", jwt.MapClaims{
		"event": "invoice.paid",
		"data":  map[string]any{},
		"exp":   1700000000,
	})

	_, err := VerifyPviumWebhookToken(token, "webhook_secret", &models.VerifyPviumWebhookTokenOptions{
		Now: time.Unix(1800000000, 0),
	})
	if err == nil {
		t.Fatal("expected expired token error")
	}
	if !strings.Contains(err.Error(), "Expired Pvium webhook token") {
		t.Fatalf("error mismatch: got %q", err.Error())
	}
}

func TestVerifyPviumWebhookTokenCanDisableHashedSecretFallback(t *testing.T) {
	t.Parallel()

	secret := "secret_abc123"
	hashed := sha256.Sum256([]byte(secret))
	token := signWebhookToken(t, fmt.Sprintf("%x", hashed), jwt.MapClaims{
		"event": "oauth.invite.accepted",
		"exp":   4000000000,
	})

	allowFallback := false
	_, err := VerifyPviumWebhookToken(token, secret, &models.VerifyPviumWebhookTokenOptions{
		AllowHashedSecretFallback: &allowFallback,
	})
	if err == nil {
		t.Fatal("expected verification to fail when hashed-secret fallback is disabled")
	}
}

func TestResolvePviumWebhookPayloadWithoutTokenUsesBodyEventAndData(t *testing.T) {
	t.Parallel()

	payload, err := ResolvePviumWebhookPayload(map[string]any{
		"event": "invoice.created",
		"data":  map[string]any{"id": "evt_1"},
	}, "unused-secret")
	if err != nil {
		t.Fatalf("resolve tokenless webhook payload: %v", err)
	}
	if payload.Event != "invoice.created" {
		t.Fatalf("event mismatch: got %s want %s", payload.Event, "invoice.created")
	}
	if payload.Data["id"] != "evt_1" {
		t.Fatalf("data id mismatch: got %v want %s", payload.Data["id"], "evt_1")
	}
}
