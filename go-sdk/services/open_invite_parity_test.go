package services

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/pvium/sdks/go-sdk/config"
	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

func recoverRawDigestSigner(t *testing.T, digestHex, signatureHex string) string {
	t.Helper()
	digest, err := hex.DecodeString(strings.TrimPrefix(digestHex, "0x"))
	if err != nil {
		t.Fatalf("decode digest: %v", err)
	}
	sig, err := hex.DecodeString(strings.TrimPrefix(signatureHex, "0x"))
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	if len(sig) != 65 {
		t.Fatalf("unexpected signature length %d", len(sig))
	}
	normalized := make([]byte, 65)
	copy(normalized, sig)
	if normalized[64] >= 27 {
		normalized[64] -= 27
	}
	pub, err := ethcrypto.SigToPub(digest, normalized)
	if err != nil {
		t.Fatalf("recover pubkey: %v", err)
	}
	return ethcrypto.PubkeyToAddress(*pub).Hex()
}

func TestAuthorizeSigningKeyMatchesParityFixture(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../parity-fixtures/signing-key-authorization.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture struct {
		PrivateKey    string `json:"privateKey"`
		FundingSigner string `json:"fundingSigner"`
		NetworkType   string `json:"networkType"`
		Input         struct {
			BatchHash      string `json:"batchHash"`
			SigningKey     string `json:"signingKey"`
			TransactionMax string `json:"transactionMax"`
			TotalMax       string `json:"totalMax"`
			Expiration     string `json:"expiration"`
			Timestamp      string `json:"timestamp"`
		} `json:"input"`
		AuthMessageHash string `json:"authMessageHash"`
		Signature       string `json:"signature"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: "https://api.example.test"}))
	auth, err := service.AuthorizeSigningKey(
		fixture.Input.BatchHash,
		fixture.Input.SigningKey,
		models.PayoutSigningKeyNetworkType(fixture.NetworkType),
		models.PayoutSigningKeyAuthorizationData{
			TransactionMax: fixture.Input.TransactionMax,
			TotalMax:       fixture.Input.TotalMax,
			Expiration:     fixture.Input.Expiration,
			Timestamp:      fixture.Input.Timestamp,
		},
		models.PayoutSignerInput{PrivateKey: fixture.PrivateKey},
	)
	if err != nil {
		t.Fatalf("authorize signing key: %v", err)
	}

	if !strings.EqualFold(auth.AuthMessageHash, fixture.AuthMessageHash) {
		t.Fatalf("auth message hash mismatch:\n got  %s\n want %s", auth.AuthMessageHash, fixture.AuthMessageHash)
	}
	if !strings.EqualFold(auth.Signature, fixture.Signature) {
		t.Fatalf("signature mismatch:\n got  %s\n want %s", auth.Signature, fixture.Signature)
	}
	if auth.TransactionMax != fixture.Input.TransactionMax || auth.TotalMax != fixture.Input.TotalMax ||
		auth.Expiration != fixture.Input.Expiration || auth.Timestamp != fixture.Input.Timestamp {
		t.Fatalf("normalized authorization fields mismatch: %+v", auth)
	}
	if string(auth.NetworkType) != fixture.NetworkType {
		t.Fatalf("network type mismatch: %s", auth.NetworkType)
	}

	recovered := recoverRawDigestSigner(t, fixture.AuthMessageHash, fixture.Signature)
	if !strings.EqualFold(recovered, fixture.FundingSigner) {
		t.Fatalf("recovered signer mismatch:\n got  %s\n want %s", recovered, fixture.FundingSigner)
	}
}

func TestAuthorizeSigningKeyRejectsUnknownNetwork(t *testing.T) {
	t.Parallel()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: "https://api.example.test"}))
	_, err := service.AuthorizeSigningKey(
		"0x1111111111111111111111111111111111111111111111111111111111111111",
		"0x0000000000000000000000000000000000000002",
		models.PayoutSigningKeyNetworkType("polkadot"),
		models.PayoutSigningKeyAuthorizationData{TransactionMax: "1", TotalMax: "1", Expiration: "1", Timestamp: "1"},
		models.PayoutSignerInput{PrivateKey: "0x59c6995e998f97a5a004497e5daaaa853d873599e62e568a0a7d3a57c5fd8d0d"},
	)
	if err == nil || !strings.Contains(err.Error(), "networkType must be ethereum or solana") {
		t.Fatalf("expected network type error, got %v", err)
	}
}

func TestCreateSignedOpenOrganizationInviteMatchesParityFixture(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../parity-fixtures/open-organization-invite.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture struct {
		Config struct {
			ClientID    string `json:"clientId"`
			ConsentHost string `json:"consentHost"`
			BaseURL     string `json:"baseUrl"`
			APIKey      string `json:"apiKey"`
		} `json:"config"`
		Signer struct {
			Chain      string `json:"chain"`
			PrivateKey string `json:"privateKey"`
		} `json:"signer"`
		Input struct {
			Label                string         `json:"label"`
			Scopes               []string       `json:"scopes"`
			AllowedIdentityTypes []string       `json:"allowedIdentityTypes"`
			AllowedEmailDomains  []string       `json:"allowedEmailDomains"`
			RequireKyc           bool           `json:"requireKyc"`
			RequireTaxProfile    bool           `json:"requireTaxProfile"`
			MaxUses              int64          `json:"maxUses"`
			ExpiresAt            string         `json:"expiresAt"`
			CreatedAt            int64          `json:"createdAt"`
			InviteNonce          string         `json:"inviteNonce"`
			InviteSecret         string         `json:"inviteSecret"`
			RedirectURI          string         `json:"redirectUri"`
			State                string         `json:"state"`
			StateParams          map[string]any `json:"stateParams"`
		} `json:"input"`
		Expected struct {
			SecretHash           string   `json:"secretHash"`
			PolicyHash           string   `json:"policyHash"`
			Signature            string   `json:"signature"`
			SignatureType        string   `json:"signatureType"`
			SignatureMessage     string   `json:"signatureMessage"`
			SignatureTimestamp   int64    `json:"signatureTimestamp"`
			SignerAddress        string   `json:"signerAddress"`
			Scopes               []string `json:"scopes"`
			AllowedIdentityTypes []string `json:"allowedIdentityTypes"`
			AllowedEmailDomains  []string `json:"allowedEmailDomains"`
		} `json:"expected"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	service := NewInviteService(transport.NewHTTPClient(config.Config{
		ClientID:    fixture.Config.ClientID,
		ConsentHost: fixture.Config.ConsentHost,
		BaseURL:     fixture.Config.BaseURL,
		APIKey:      fixture.Config.APIKey,
	}))

	signed, err := service.CreateSignedOpenOrganizationInvite(models.OpenOrganizationInviteInput{
		Label:                fixture.Input.Label,
		Scopes:               fixture.Input.Scopes,
		AllowedIdentityTypes: fixture.Input.AllowedIdentityTypes,
		AllowedEmailDomains:  fixture.Input.AllowedEmailDomains,
		RequireKyc:           fixture.Input.RequireKyc,
		RequireTaxProfile:    fixture.Input.RequireTaxProfile,
		MaxUses:              fixture.Input.MaxUses,
		ExpiresAt:            fixture.Input.ExpiresAt,
		RedirectURI:          fixture.Input.RedirectURI,
		State:                fixture.Input.State,
		StateParams:          fixture.Input.StateParams,
		CreatedAt:            fixture.Input.CreatedAt,
		InviteNonce:          fixture.Input.InviteNonce,
		InviteSecret:         fixture.Input.InviteSecret,
	}, models.OAuthInviteSigner{Chain: fixture.Signer.Chain, PrivateKey: fixture.Signer.PrivateKey})
	if err != nil {
		t.Fatalf("create signed open organization invite: %v", err)
	}

	if signed.SignatureMessage != fixture.Expected.SignatureMessage {
		t.Fatalf("signature message mismatch:\n got  %q\n want %q", signed.SignatureMessage, fixture.Expected.SignatureMessage)
	}
	if signed.PolicyHash != fixture.Expected.PolicyHash {
		t.Fatalf("policy hash mismatch:\n got  %s\n want %s", signed.PolicyHash, fixture.Expected.PolicyHash)
	}
	if signed.SecretHash != fixture.Expected.SecretHash {
		t.Fatalf("secret hash mismatch:\n got  %s\n want %s", signed.SecretHash, fixture.Expected.SecretHash)
	}
	if !strings.EqualFold(signed.Signature, fixture.Expected.Signature) {
		t.Fatalf("signature mismatch:\n got  %s\n want %s", signed.Signature, fixture.Expected.Signature)
	}
	if signed.SignatureType != fixture.Expected.SignatureType {
		t.Fatalf("signature type mismatch: %s", signed.SignatureType)
	}
	if !strings.EqualFold(signed.SignerAddress, fixture.Expected.SignerAddress) {
		t.Fatalf("signer address mismatch:\n got  %s\n want %s", signed.SignerAddress, fixture.Expected.SignerAddress)
	}
	if signed.SignatureTimestamp != fixture.Expected.SignatureTimestamp {
		t.Fatalf("signature timestamp mismatch: %d", signed.SignatureTimestamp)
	}
	if !reflect.DeepEqual(signed.Scopes, fixture.Expected.Scopes) {
		t.Fatalf("scopes mismatch:\n got  %+v\n want %+v", signed.Scopes, fixture.Expected.Scopes)
	}
	if !reflect.DeepEqual(signed.AllowedIdentityTypes, fixture.Expected.AllowedIdentityTypes) {
		t.Fatalf("allowed identity types mismatch:\n got  %+v\n want %+v", signed.AllowedIdentityTypes, fixture.Expected.AllowedIdentityTypes)
	}
	if !reflect.DeepEqual(signed.AllowedEmailDomains, fixture.Expected.AllowedEmailDomains) {
		t.Fatalf("allowed email domains mismatch:\n got  %+v\n want %+v", signed.AllowedEmailDomains, fixture.Expected.AllowedEmailDomains)
	}
}
