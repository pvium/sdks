package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pvium/sdks/go-sdk/config"
	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

const inviteTestPrivateKey = "0x59c6995e998f97a5a0044976f0d7f3f6f8f53f6a2046baf4f01cb4f1f6bcb58f"

func TestInvitesCreateAndSignOAuthInviteLink(t *testing.T) {
	t.Parallel()

	service := NewInviteService(transport.NewHTTPClient(config.Config{
		BaseURL:     "http://localhost:4005/v1",
		ConsentHost: "http://localhost:3000",
		ClientID:    "app_test",
		APIKey:      "pk_test_dummy",
	}))

	bundle, err := service.CreateBundle(models.OAuthInviteBundleInput{
		Identities: []models.InviteIdentity{{Type: models.InviteIdentityEmail, Value: "Test.User@example.com"}},
		Scopes:     []string{"read:ethereum_wallet", "read:user"},
		Chain:      "ethereum",
	})
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	signed, err := service.SignBundle(bundle, models.OAuthInviteSigner{Chain: "ethereum", PrivateKey: inviteTestPrivateKey})
	if err != nil {
		t.Fatalf("sign bundle: %v", err)
	}

	if signed.ClientID != "app_test" {
		t.Fatalf("client id mismatch: %s", signed.ClientID)
	}
	if len(signed.Invites) != 1 {
		t.Fatalf("expected one invite, got %d", len(signed.Invites))
	}
	if signed.Root.SignatureType != "evm-personal-sign" || signed.Root.SignerAddress == "" {
		t.Fatalf("root signature metadata mismatch: %+v", signed.Root)
	}
	if strings.Join(signed.Scopes, " ") != "read:ethereum_wallet read:user" {
		t.Fatalf("scopes mismatch: %+v", signed.Scopes)
	}

	invite := signed.Invites[0]
	if invite.IdentityType != models.InviteIdentityEmail || invite.IdentityValue != "test.user@example.com" {
		t.Fatalf("identity mismatch: %+v", invite)
	}
	if invite.AppClientID != "app_test" || invite.LeafVersion != "2" || len(invite.SecretHash) != 64 || len(invite.Proof) != 0 {
		t.Fatalf("invite metadata mismatch: %+v", invite)
	}
	inviteURL, err := url.Parse(invite.InviteLink)
	if err != nil {
		t.Fatalf("parse invite link: %v", err)
	}
	query := inviteURL.Query()
	if inviteURL.Scheme+"://"+inviteURL.Host != "http://localhost:3000" || inviteURL.Path != "/oauth2/authorize" {
		t.Fatalf("invite url mismatch: %s", invite.InviteLink)
	}
	if query.Get("client_id") != "app_test" || query.Get("response_type") != "code" {
		t.Fatalf("invite url auth params mismatch: %s", invite.InviteLink)
	}
	if query.Get("scope") != "read:ethereum_wallet read:user" || query.Get("invite_nonce") != invite.InviteNonce || query.Get("invite_secret") != invite.InviteSecret {
		t.Fatalf("invite url invite params mismatch: %s", invite.InviteLink)
	}
	if query.Get("identity_type") != "email" || query.Get("identity_hint") != "test.user@example.com" {
		t.Fatalf("invite url identity params mismatch: %s", invite.InviteLink)
	}
}

func TestInvitesBatchBundleLinksAndCommit(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	service := NewInviteService(transport.NewHTTPClient(config.Config{
		BaseURL:     ts.URL,
		ConsentHost: "http://localhost:3000",
		ClientID:    "app_test",
		APIKey:      "pk_test_dummy",
	}))
	bundle, err := service.CreateBundle(models.OAuthInviteBundleInput{
		Identities: []models.InviteIdentity{{Type: models.InviteIdentityEmail, Value: "Batch.User@example.com"}},
		Scopes:     []string{"read:user", "read:ethereum_wallet"},
		Chain:      "ethereum",
		BatchInvite: &models.OAuthInviteBatchOptions{
			BatchID:     "batch_123",
			StateParams: map[string]any{"source": "sdk-test"},
		},
		StateParams: map[string]any{"returnTo": "/admin/bulk-payments/batch_123"},
	})
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	signed, err := service.SignBundle(bundle, models.OAuthInviteSigner{Chain: "ethereum", PrivateKey: inviteTestPrivateKey})
	if err != nil {
		t.Fatalf("sign bundle: %v", err)
	}
	if signed.BatchID != "batch_123" || signed.BatchInvite.BatchID != "batch_123" {
		t.Fatalf("batch id mismatch: %+v", signed)
	}
	inviteURL, _ := url.Parse(signed.Invites[0].InviteLink)
	if inviteURL.Query().Get("batchId") != "batch_123" {
		t.Fatalf("invite batchId mismatch: %s", signed.Invites[0].InviteLink)
	}
	state, err := url.ParseQuery(inviteURL.Query().Get("state"))
	if err != nil {
		t.Fatalf("parse nested state: %v", err)
	}
	if state.Get("batchId") != "batch_123" || state.Get("source") != "sdk-test" || state.Get("returnTo") != "/admin/bulk-payments/batch_123" {
		t.Fatalf("state mismatch: %s", inviteURL.Query().Get("state"))
	}
	groupURL, _ := url.Parse(signed.GroupInviteLink)
	if groupURL.Query().Get("batchId") != "batch_123" {
		t.Fatalf("group batchId mismatch: %s", signed.GroupInviteLink)
	}

	_, err = service.CommitBundle(context.Background(), signed, nil)
	if err != nil {
		t.Fatalf("commit bundle: %v", err)
	}
	if gotPath != "/batch-payments/batch_123/invites" {
		t.Fatalf("commit path mismatch: %s", gotPath)
	}
	if _, ok := gotBody["root"].(map[string]any); !ok {
		t.Fatalf("commit root missing: %+v", gotBody)
	}
	invites := gotBody["invites"].([]any)
	invite := invites[0].(map[string]any)
	if _, ok := invite["inviteSecret"]; ok {
		t.Fatalf("commit body must omit inviteSecret: %+v", invite)
	}
	if _, ok := invite["inviteLink"]; ok {
		t.Fatalf("commit body must omit inviteLink: %+v", invite)
	}
}

func TestInvitesSupportsSeparateMasterAndRootSigners(t *testing.T) {
	t.Parallel()

	service := NewInviteService(transport.NewHTTPClient(config.Config{
		BaseURL:     "http://localhost:4005/v1",
		ConsentHost: "http://localhost:3000",
		ClientID:    "app_test",
		APIKey:      "pk_test_dummy",
	}))
	bundle, err := service.CreateBundle(models.OAuthInviteBundleInput{
		Identities: []models.InviteIdentity{{Type: models.InviteIdentityEmail, Value: "Split.Signer@example.com"}},
		Scopes:     []string{"read:user", "read:ethereum_wallet"},
		Chain:      "ethereum",
	})
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	calls := []string{}
	signed, err := service.SignBundle(bundle, models.OAuthInviteSigner{
		Chain:         "ethereum",
		SignerAddress: "0x0000000000000000000000000000000000000003",
		SignMessage: func(string) (string, error) {
			t.Fatal("fallback signMessage should not be called")
			return "", nil
		},
		SignMasterSecret: func(message string) (string, error) {
			calls = append(calls, "master:"+message)
			return "0xmaster", nil
		},
		SignInviteRoot: func(message string) (string, error) {
			calls = append(calls, "root:"+message)
			return "0xroot", nil
		},
	})
	if err != nil {
		t.Fatalf("sign bundle: %v", err)
	}
	if len(calls) != 2 || !strings.HasPrefix(calls[0], "master:PVIUM_INVITE_SECRET_V2:") || !strings.HasPrefix(calls[1], "root:PVIUM_INVITE_ROOT_V2") {
		t.Fatalf("unexpected signer calls: %+v", calls)
	}
	if signed.Root.SignerAddress != "0x0000000000000000000000000000000000000003" || signed.Root.Signature != "0xroot" {
		t.Fatalf("root signer mismatch: %+v", signed.Root)
	}
}

func TestCreateBundleRequiresIdentitiesAndBatchIDWhenBatchInviteProvided(t *testing.T) {
	t.Parallel()

	service := NewInviteService(transport.NewHTTPClient(config.Config{
		BaseURL:     "https://api-sandbox.pvium.com/v1",
		ConsentHost: "https://sandbox.pvium.com",
		ClientID:    "client_123",
	}))

	_, err := service.CreateBundle(models.OAuthInviteBundleInput{})
	if err == nil {
		t.Fatal("expected error when identities are empty")
	}
	_, err = service.CreateBundle(models.OAuthInviteBundleInput{
		Identities:  []models.InviteIdentity{{Type: models.InviteIdentityEmail, Value: "payee@example.com"}},
		BatchInvite: &models.OAuthInviteBatchOptions{BatchID: "   "},
	})
	if err == nil {
		t.Fatal("expected error when batchInvite.batchId is empty")
	}
}
