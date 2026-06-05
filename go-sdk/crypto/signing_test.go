package crypto

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/pvium/sdks/go-sdk/models"
)

const testPrivateKey = "0x59c6995e998f97a5a0044976f0d7f3f6f8f53f6a2046baf4f01cb4f1f6bcb58f"

func TestCreateSignerFromPrivateKeyReturnsUsableSigner(t *testing.T) {
	t.Parallel()

	pk, err := ethcrypto.HexToECDSA(strings.TrimPrefix(testPrivateKey, "0x"))
	if err != nil {
		t.Fatalf("ecdsa from key: %v", err)
	}
	expectedAddress := ethcrypto.PubkeyToAddress(pk.PublicKey).Hex()

	messageHash := KeccakHex([]byte("hello-pvium"))
	hashBytes, err := hex.DecodeString(strings.TrimPrefix(messageHash, "0x"))
	if err != nil {
		t.Fatalf("decode hash: %v", err)
	}

	signer := CreateSignerFromPrivateKey(testPrivateKey)
	signature, err := signer(hashBytes)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}

	recovered, err := recoverAddressFromSignedMessageHash(messageHash, signature)
	if err != nil {
		t.Fatalf("recover address: %v", err)
	}
	if !strings.EqualFold(recovered, expectedAddress) {
		t.Fatalf("recovered address mismatch: got %s want %s", recovered, expectedAddress)
	}
}

func TestSignCreateProjectRequestMatchesNodeEncoding(t *testing.T) {
	t.Parallel()

	payload := CreateProjectRequestPayload{
		"app":                     "test-app",
		"projectId":               "project-001",
		"metadata":                "ipfs://QmTest",
		"tokenAddress":            "0x0000000000000000000000000000000000000001",
		"refundAddress":           "0x0000000000000000000000000000000000000002",
		"appFeeAddress":           "0x0000000000000000000000000000000000000003",
		"appAdminAddress":         "0x0000000000000000000000000000000000000004",
		"appFeeBps":               200,
		"disputeWindowSeconds":    259200,
		"lockDuration":            7776000,
		"minimumBalancePerVendor": "100000000",
	}
	options := CreateProjectSignatureOptions{PviumFeeBps: 100, ChainID: 84532}
	hashHex, err := HashCreateProjectRequest(payload, options)
	if err != nil {
		t.Fatalf("hash create project: %v", err)
	}
	const expectedHash = "0x5bbbd173c7dac509b085ecb03b03e723b6dcab05239131c1977f310bc9082c22"
	if hashHex != expectedHash {
		t.Fatalf("project hash mismatch: got %s want %s", hashHex, expectedHash)
	}
	sig, err := SignCreateProjectRequest(payload, SignerInput{PrivateKey: testPrivateKey}, options)
	if err != nil {
		t.Fatalf("sign create project: %v", err)
	}
	assertRecoveredAddress(t, hashHex, sig)
}

func TestSignCreateClaimRequestMatchesNodeEncoding(t *testing.T) {
	t.Parallel()

	payload := CreateClaimRequestPayload{
		"app":            "test-app",
		"projectId":      "project-001",
		"claimId":        "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"receiver":       "0x0000000000000000000000000000000000000005",
		"amount":         "100000000",
		"claimableAfter": 1700000000,
		"claimDeadline":  0,
		"nonce":          1,
	}

	hashHex, err := HashCreateClaimRequest(payload)
	if err != nil {
		t.Fatalf("hash create claim: %v", err)
	}
	const expectedHash = "0xad04b5811dfc72641da4abae1e86f33e043342c69ae091d894b52b5401b9cc21"
	if hashHex != expectedHash {
		t.Fatalf("claim hash mismatch: got %s want %s", hashHex, expectedHash)
	}
	sig, err := SignCreateClaimRequest(payload, SignerInput{PrivateKey: testPrivateKey})
	if err != nil {
		t.Fatalf("sign create claim: %v", err)
	}
	assertRecoveredAddress(t, hashHex, sig)
}

func TestSignFinalizeClaimRequestMatchesNodeEncoding(t *testing.T) {
	t.Parallel()

	claims := FinalizeClaimRequestPayload{
		{
			"app":       "test-app",
			"projectId": "usdc-project",
			"claimId":   "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		{
			"app":       "test-app",
			"projectId": "usdt-project",
			"claimId":   "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		},
	}
	hashHex, err := HashFinalizeClaimRequest(claims, 84532)
	if err != nil {
		t.Fatalf("hash finalize claim: %v", err)
	}
	const expectedHash = "0x968419473994171a9ca2099d6307d9151e044b929039f7ecb0feb1610d5d6e8a"
	if hashHex != expectedHash {
		t.Fatalf("finalize hash mismatch: got %s want %s", hashHex, expectedHash)
	}
	sig, err := SignFinalizeClaimRequest(claims, SignerInput{PrivateKey: testPrivateKey}, 84532)
	if err != nil {
		t.Fatalf("sign finalize claim: %v", err)
	}
	assertRecoveredAddress(t, hashHex, sig)
}

func TestRelayedDisputeResolveHelpersMatchNodeEncoding(t *testing.T) {
	t.Parallel()

	relayedPayloadHex := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000a61646456656e646f7273000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	relayed := RelayedCallRequestPayload{
		"appId":     "test-app",
		"projectId": "project-001",
		"payload":   relayedPayloadHex,
		"nonce":     2,
		"chainId":   84532,
	}
	relayedHash, err := HashRelayedCallRequest(relayed)
	if err != nil {
		t.Fatalf("hash relayed: %v", err)
	}
	if relayedHash != "0x00b426c901b062511ad9f52f1cb7e71ee1a3d81c66c2263a48881728db9caaf1" {
		t.Fatalf("relayed hash mismatch: %s", relayedHash)
	}
	relayedSig, err := SignRelayedCallRequest(relayed, SignerInput{PrivateKey: testPrivateKey})
	if err != nil {
		t.Fatalf("sign relayed: %v", err)
	}
	assertRecoveredAddress(t, relayedHash, relayedSig)

	claimID := "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	disputeHash, err := HashDisputeRequest(claimID, 84532)
	if err != nil {
		t.Fatalf("hash dispute: %v", err)
	}
	if disputeHash != "0x9c9040837d80b9869bc24271d24c950747a1f7adb3e5e46469ad02bc8af709fb" {
		t.Fatalf("dispute hash mismatch: %s", disputeHash)
	}
	disputeSig, err := SignDisputeRequest(claimID, SignerInput{PrivateKey: testPrivateKey}, 84532)
	if err != nil {
		t.Fatalf("sign dispute: %v", err)
	}
	assertRecoveredAddress(t, disputeHash, disputeSig)

	resolve := ResolveDisputeRequestPayload{ClaimID: claimID, Approved: true, ChainID: 84532}
	resolveHash, err := HashResolveDisputeRequest(resolve)
	if err != nil {
		t.Fatalf("hash resolve: %v", err)
	}
	if resolveHash != "0x49f4a1de9692ba231ab1492da44780ce6b31f71a4f4e10772783ebbb055fa2c3" {
		t.Fatalf("resolve hash mismatch: %s", resolveHash)
	}
	resolveSig, err := SignResolveDisputeRequest(resolve, SignerInput{PrivateKey: testPrivateKey})
	if err != nil {
		t.Fatalf("sign resolve: %v", err)
	}
	assertRecoveredAddress(t, resolveHash, resolveSig)
}

func TestCreateProjectAttestationMatchesNodeEncoding(t *testing.T) {
	t.Parallel()

	hashHex, err := HashCreateProjectAttestation("0x1234", 84532)
	if err != nil {
		t.Fatalf("hash attestation: %v", err)
	}
	const expectedHash = "0x785a7bc092e1a9b71dad95d5063e84de9479e5fbe7f3258178cd83e8272d7893"
	if hashHex != expectedHash {
		t.Fatalf("attestation hash mismatch: got %s want %s", hashHex, expectedHash)
	}
	sig, err := SignCreateProjectAttestation("0x1234", SignerInput{PrivateKey: testPrivateKey}, 84532)
	if err != nil {
		t.Fatalf("sign attestation: %v", err)
	}
	assertRecoveredAddress(t, hashHex, sig)
}

func TestCreatePayoutNonceGenerates16ByteHex(t *testing.T) {
	t.Parallel()

	nonce, err := CreatePayoutNonce()
	if err != nil {
		t.Fatalf("create payout nonce: %v", err)
	}
	if !strings.HasPrefix(nonce, "0x") || len(nonce) != 34 {
		t.Fatalf("unexpected nonce format: %s", nonce)
	}
}

func TestGenerateInstantPayoutHashMatchesNodeSDK(t *testing.T) {
	t.Parallel()

	decimals := 6
	got, err := GenerateInstantPayoutHash([]models.PayoutPayment{
		{
			Receiver: "0x0000000000000000000000000000000000000001",
			Amount:   "12.5",
			Token:    "0x0000000000000000000000000000000000000002",
			Decimals: &decimals,
			Memo:     "first",
		},
		{
			Receiver: "0x0000000000000000000000000000000000000003",
			Amount:   1,
			Token:    "0x0000000000000000000000000000000000000002",
			Decimals: &decimals,
		},
	}, "0x1234abcd")
	if err != nil {
		t.Fatalf("generate instant payout hash: %v", err)
	}
	const expected = "0x9bc903fb74d064b2fc7d3dd278c83877926de88df55e8d2193249c1656754681"
	if got != expected {
		t.Fatalf("instant payout hash mismatch: got %s want %s", got, expected)
	}
}

func recoverAddressFromSignedMessageHash(hashHex, sigHex string) (string, error) {
	hashBytes, err := hex.DecodeString(strings.TrimPrefix(hashHex, "0x"))
	if err != nil {
		return "", err
	}
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
	if err != nil {
		return "", err
	}
	if len(sigBytes) != 65 {
		return "", fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}
	prefixed := ethcrypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(hashBytes))), hashBytes)
	pubKey, err := ethcrypto.SigToPub(prefixed, sigBytes)
	if err != nil {
		return "", err
	}
	return ethcrypto.PubkeyToAddress(*pubKey).Hex(), nil
}

func assertRecoveredAddress(t *testing.T, hashHex, sigHex string) {
	t.Helper()
	recovered, err := recoverAddressFromSignedMessageHash(hashHex, sigHex)
	if err != nil {
		t.Fatalf("recover address: %v", err)
	}
	pk, _ := ethcrypto.HexToECDSA(strings.TrimPrefix(testPrivateKey, "0x"))
	expectedAddress := ethcrypto.PubkeyToAddress(pk.PublicKey).Hex()
	if !strings.EqualFold(recovered, expectedAddress) {
		t.Fatalf("recovered address mismatch: got %s want %s", recovered, expectedAddress)
	}
}
