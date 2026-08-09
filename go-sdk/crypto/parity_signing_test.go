package crypto

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func recoverAddressFromDigest(t *testing.T, digest []byte, signatureHex string) string {
	t.Helper()
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

func recoverPersonalSign(t *testing.T, messageHashHex, signatureHex string) string {
	t.Helper()
	hashBytes, err := hex.DecodeString(strings.TrimPrefix(messageHashHex, "0x"))
	if err != nil {
		t.Fatalf("decode message hash: %v", err)
	}
	prefixed := ethcrypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(hashBytes))), hashBytes)
	return recoverAddressFromDigest(t, prefixed, signatureHex)
}

func TestHashFinalizeClaimRequestMatchesParityFixture(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../parity-fixtures/finalize-claim-request.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture struct {
		PrivateKey string `json:"privateKey"`
		Signer     string `json:"signer"`
		ChainID    int64  `json:"chainId"`
		Claims     []struct {
			App       string `json:"app"`
			ProjectID string `json:"projectId"`
			ClaimID   string `json:"claimId"`
		} `json:"claims"`
		MessageHash string `json:"messageHash"`
		Signature   string `json:"signature"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	claims := make(FinalizeClaimRequestPayload, 0, len(fixture.Claims))
	for _, c := range fixture.Claims {
		claims = append(claims, map[string]any{
			"app":       c.App,
			"projectId": c.ProjectID,
			"claimId":   c.ClaimID,
		})
	}

	hash, err := HashFinalizeClaimRequest(claims, fixture.ChainID)
	if err != nil {
		t.Fatalf("hash finalize claim: %v", err)
	}
	if !strings.EqualFold(hash, fixture.MessageHash) {
		t.Fatalf("message hash mismatch:\n got  %s\n want %s", hash, fixture.MessageHash)
	}

	signature, err := SignFinalizeClaimRequest(claims, SignerInput{PrivateKey: fixture.PrivateKey}, fixture.ChainID)
	if err != nil {
		t.Fatalf("sign finalize claim: %v", err)
	}
	if !strings.EqualFold(signature, fixture.Signature) {
		t.Fatalf("signature mismatch:\n got  %s\n want %s", signature, fixture.Signature)
	}

	recovered := recoverPersonalSign(t, fixture.MessageHash, fixture.Signature)
	if !strings.EqualFold(recovered, fixture.Signer) {
		t.Fatalf("recovered signer mismatch:\n got  %s\n want %s", recovered, fixture.Signer)
	}
}

func TestHashFinalizeClaimRequestIsCollisionResistant(t *testing.T) {
	t.Parallel()

	claimID := "0x1111111111111111111111111111111111111111111111111111111111111111"
	a := FinalizeClaimRequestPayload{{"app": "app", "projectId": "roj", "claimId": claimID}}
	b := FinalizeClaimRequestPayload{{"app": "ap", "projectId": "proj", "claimId": claimID}}

	hashA, err := HashFinalizeClaimRequest(a, 8453)
	if err != nil {
		t.Fatalf("hash a: %v", err)
	}
	hashB, err := HashFinalizeClaimRequest(b, 8453)
	if err != nil {
		t.Fatalf("hash b: %v", err)
	}
	if hashA == hashB {
		t.Fatalf("expected distinct hashes for length-delimited encoding, both were %s", hashA)
	}
}

func TestComputeSigningKeyAuthorizationHashMatchesParityFixture(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../parity-fixtures/signing-key-authorization.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture struct {
		Input struct {
			BatchHash      string `json:"batchHash"`
			SigningKey     string `json:"signingKey"`
			TransactionMax string `json:"transactionMax"`
			TotalMax       string `json:"totalMax"`
			Expiration     string `json:"expiration"`
			Timestamp      string `json:"timestamp"`
		} `json:"input"`
		AuthMessageHash string `json:"authMessageHash"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	result, err := ComputeSigningKeyAuthorizationHash(SigningKeyAuthorizationHashParams{
		BatchHash:      fixture.Input.BatchHash,
		SigningKey:     fixture.Input.SigningKey,
		TransactionMax: fixture.Input.TransactionMax,
		TotalMax:       fixture.Input.TotalMax,
		Expiration:     fixture.Input.Expiration,
		Timestamp:      fixture.Input.Timestamp,
	})
	if err != nil {
		t.Fatalf("compute signing key authorization hash: %v", err)
	}
	if !strings.EqualFold(result.AuthMessageHash, fixture.AuthMessageHash) {
		t.Fatalf("auth message hash mismatch:\n got  %s\n want %s", result.AuthMessageHash, fixture.AuthMessageHash)
	}
}

func TestComputeSigningKeyAuthorizationHashRejectsNegative(t *testing.T) {
	t.Parallel()

	_, err := ComputeSigningKeyAuthorizationHash(SigningKeyAuthorizationHashParams{
		BatchHash:      "0x1111111111111111111111111111111111111111111111111111111111111111",
		SigningKey:     "0x0000000000000000000000000000000000000002",
		TransactionMax: "-1",
		TotalMax:       "5000000",
		Expiration:     "1777488000",
		Timestamp:      "1777487451",
	})
	if err == nil {
		t.Fatal("expected error for negative transactionMax")
	}
}
