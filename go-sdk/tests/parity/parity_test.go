package parity_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	pvium "github.com/pvium/sdks/go-sdk"
)

type fixture struct {
	Signing struct {
		SignatureDomain string `json:"signatureDomain"`
		DisputeHash     string `json:"disputeHash"`
	} `json:"signing"`
	Payouts struct {
		ScheduledHash string `json:"scheduledHash"`
	} `json:"payouts"`
	Invites struct {
		Identity struct {
			Email  string `json:"email"`
			Wallet string `json:"wallet"`
			Detect struct {
				Type string `json:"type"`
			} `json:"detect"`
		} `json:"identity"`
		ProofInput struct {
			Leaf  string   `json:"leaf"`
			Proof []string `json:"proof"`
			Root  string   `json:"root"`
		} `json:"proofInput"`
		ProofValid    bool   `json:"proofValid"`
		MasterMessage string `json:"masterMessage"`
		DerivedMaster string `json:"derivedMaster"`
		DerivedInvite string `json:"derivedInvite"`
	} `json:"invites"`
}

func loadFixture(t *testing.T) fixture {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "fixtures", "ts_parity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return f
}

func TestSigningParity(t *testing.T) {
	f := loadFixture(t)

	if got := pvium.SignatureDomainFromText("PVIUM_SIGNATURE_MESSAGE"); got != f.Signing.SignatureDomain {
		t.Fatalf("signature domain mismatch: got %s want %s", got, f.Signing.SignatureDomain)
	}

	gotDispute, err := pvium.HashDisputeRequest("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 84532)
	if err != nil {
		t.Fatalf("hash dispute: %v", err)
	}
	if gotDispute != f.Signing.DisputeHash {
		t.Fatalf("dispute hash mismatch: got %s want %s", gotDispute, f.Signing.DisputeHash)
	}
}

func TestPayoutHashParity(t *testing.T) {
	f := loadFixture(t)

	got, err := pvium.ComputeScheduledPayoutHash(pvium.ScheduledPayoutHashParams{
		PayoutID:            "123e4567-e89b-12d3-a456-426614174000",
		FundingToken:        "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		GracePeriod:         3600,
		DisapprovalDeadline: 1715645800,
		Timestamp:           1715644800,
		ChainID:             84532,
	})
	if err != nil {
		t.Fatalf("compute scheduled hash: %v", err)
	}
	if got != f.Payouts.ScheduledHash {
		t.Fatalf("scheduled hash mismatch: got %s want %s", got, f.Payouts.ScheduledHash)
	}
}

func TestInviteUtilityParity(t *testing.T) {
	f := loadFixture(t)

	if got := pvium.NormalizeIdentityValue(pvium.InviteIdentityEmail, " Test@Example.com "); got != f.Invites.Identity.Email {
		t.Fatalf("normalize email mismatch: got %s want %s", got, f.Invites.Identity.Email)
	}
	if got := pvium.NormalizeIdentityValue(pvium.InviteIdentityWallet, "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"); got != f.Invites.Identity.Wallet {
		t.Fatalf("normalize wallet mismatch: got %s want %s", got, f.Invites.Identity.Wallet)
	}
	if got := string(pvium.DetectInviteIdentityType("payee@example.com")); got != f.Invites.Identity.Detect.Type {
		t.Fatalf("detect type mismatch: got %s want %s", got, f.Invites.Identity.Detect.Type)
	}

	if got := pvium.BuildInviteMasterSecretMessage("abcdef1234567890abcdef1234567890"); got != f.Invites.MasterMessage {
		t.Fatalf("master message mismatch: got %s want %s", got, f.Invites.MasterMessage)
	}
	if got := pvium.DeriveMasterSecret("0xabcdef"); got != f.Invites.DerivedMaster {
		t.Fatalf("derived master mismatch: got %s want %s", got, f.Invites.DerivedMaster)
	}
	if got := pvium.DeriveInviteSecret(pvium.DeriveMasterSecret("0xabcdef"), "11111111111111111111111111111111"); got != f.Invites.DerivedInvite {
		t.Fatalf("derived invite mismatch: got %s want %s", got, f.Invites.DerivedInvite)
	}

	merkle, err := pvium.GenerateBatchInviteMerkleDataV2(pvium.BatchInviteMerkleInputV2{
		AppClientID: "app_test",
		BatchID:     "batch_123",
		Scopes:      []string{"read:user", "read:ethereum_wallet"},
		CreatedAt:   1700000000,
		RootNonce:   "abcdef1234567890abcdef1234567890",
		Invites: []pvium.BatchInviteMerkleInputInviteV2{{
			IdentityType:  pvium.InviteIdentityEmail,
			IdentityValue: "payee@example.com",
			InviteNonce:   "11111111111111111111111111111111",
			InviteSecret:  "2222222222222222222222222222222222222222222222222222222222222222",
		}},
	})
	if err != nil {
		t.Fatalf("generate invite merkle: %v", err)
	}
	invite := merkle.Invites[0]
	proofResult := pvium.VerifyBatchInviteProofV2(pvium.BatchInviteProofVerificationInputV2{
		AppClientID:        "app_test",
		BatchID:            "batch_123",
		IdentityType:       invite.IdentityType,
		IdentityValue:      invite.IdentityValue,
		InviteNonce:        invite.InviteNonce,
		InviteSecret:       invite.InviteSecret,
		IdentityCommitment: invite.IdentityCommitment,
		SecretHash:         invite.SecretHash,
		ExpiresAt:          invite.ExpiresAt,
		Leaf:               invite.Leaf,
		Proof:              invite.Proof,
		Root:               merkle.Root,
	})
	if !proofResult.Valid {
		t.Fatalf("proof valid mismatch: %+v", proofResult.Errors)
	}
}
