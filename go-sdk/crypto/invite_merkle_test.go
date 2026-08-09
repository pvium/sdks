package crypto

import (
	"strings"
	"testing"

	"github.com/pvium/sdks/go-sdk/models"
)

func TestGenerateBatchInviteMerkleDataV2EmitsV4ForOrgReferenceID(t *testing.T) {
	t.Parallel()

	merkle, err := GenerateBatchInviteMerkleDataV2(BatchInviteMerkleInputV2{
		AppClientID:    "app_test",
		Scopes:         []string{"read:user", "read:github"},
		CreatedAt:      1700000000,
		RootNonce:      "abcdef1234567890abcdef1234567890",
		OrgReferenceID: "maintainer-ref-1",
		SigningKey: &models.InviteSigningKeyRequest{
			PublicKey: "0x00000000000000000000000000000000000000AA",
			KeyType:   "ethereum",
		},
		Invites: []BatchInviteMerkleInputInviteV2{{
			IdentityType:  models.InviteIdentityGitHub,
			IdentityValue: "octocat",
			InviteNonce:   "11111111111111111111111111111111",
			InviteSecret:  "2222222222222222222222222222222222222222222222222222222222222222",
			ExpiresAt:     int64(1716250000),
		}},
	})
	if err != nil {
		t.Fatalf("generate merkle: %v", err)
	}
	derived, err := DeriveOrgClientID("app_test", "maintainer-ref-1")
	if err != nil {
		t.Fatalf("derive org client id: %v", err)
	}

	if merkle.Version != "4" || merkle.OrgReferenceID != "maintainer-ref-1" || merkle.DerivedOrgClientID != derived {
		t.Fatalf("V4 metadata mismatch: %+v", merkle)
	}
	if merkle.SigningKey != "0x00000000000000000000000000000000000000aa" || merkle.SigningKeyType != "ethereum" {
		t.Fatalf("signing key mismatch: %+v", merkle)
	}
	for _, expected := range []string{
		"PVIUM_INVITE_ROOT_V4",
		"version=4",
		"orgReferenceId=maintainer-ref-1",
		"signingKey=0x00000000000000000000000000000000000000aa",
		"signingKeyType=ethereum",
	} {
		if !strings.Contains(merkle.SignatureMessage, expected) {
			t.Fatalf("signature message missing %q:\n%s", expected, merkle.SignatureMessage)
		}
	}
}

func TestGenerateBatchInviteMerkleDataV2EmitsV3ForSigningKeyOnly(t *testing.T) {
	t.Parallel()

	merkle, err := GenerateBatchInviteMerkleDataV2(BatchInviteMerkleInputV2{
		AppClientID: "app_test",
		Scopes:      []string{"read:user"},
		CreatedAt:   1700000000,
		RootNonce:   "abcdef1234567890abcdef1234567890",
		SigningKey: &models.InviteSigningKeyRequest{
			PublicKey: "0x00000000000000000000000000000000000000aa",
			KeyType:   "ethereum",
		},
		Invites: []BatchInviteMerkleInputInviteV2{{
			IdentityType:  models.InviteIdentityGitHub,
			IdentityValue: "octocat",
			InviteNonce:   "11111111111111111111111111111111",
			InviteSecret:  "2222222222222222222222222222222222222222222222222222222222222222",
			ExpiresAt:     int64(1716250000),
		}},
	})
	if err != nil {
		t.Fatalf("generate merkle: %v", err)
	}
	if merkle.Version != "3" || !strings.HasPrefix(merkle.SignatureMessage, "PVIUM_INVITE_ROOT_V3") {
		t.Fatalf("expected V3 root, got %+v", merkle)
	}
}
