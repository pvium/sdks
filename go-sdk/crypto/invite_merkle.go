package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/pvium/sdks/go-sdk/models"
)

var (
	emailRE      = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	evmAddressRE = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	solanaAddrRE = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)
	handleRE     = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9._-]{0,30}[a-z0-9])?$`)
)

func createRandomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(buf), nil
}

func CreateInviteNonce() (string, error) {
	value, err := createRandomHex(16)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(value, "0x"), nil
}

func CreateInviteSecret() (string, error) {
	hexValue, err := createRandomHex(32)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(hexValue, "0x"), nil
}

func CreateRootNonce(batchID string, scopes []string) (string, error) {
	salt, err := CreateInviteNonce()
	if err != nil {
		return "", err
	}
	scopes = normalizeInviteScopes(scopes)
	return sha256Hex(strings.Join([]string{"payy.invite.root.v1", batchID, strings.Join(scopes, " "), salt}, ":")), nil
}

func BuildInviteMasterSecretMessage(rootNonce string) string {
	return "PVIUM_INVITE_SECRET_V2:" + rootNonce
}

func DeriveMasterSecret(rawSignatureHex string) string {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(rawSignatureHex)), "0x")
	if normalized == "" {
		return ""
	}
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}

func DeriveInviteSecret(masterSecret, inviteNonce string) string {
	h := sha256.Sum256([]byte(masterSecret + ":" + inviteNonce))
	return hex.EncodeToString(h[:])
}

func BuildInviteSecretHash(inviteSecret string) string {
	return sha256Hex(inviteSecret)
}

func BuildInviteIdentityCommitment(identityType models.InviteIdentityType, value, inviteNonce string) string {
	return sha256Hex(strings.Join([]string{"pvium.invite.identity.v2", string(identityType), NormalizeIdentityValue(identityType, value), inviteNonce}, ":"))
}

func NormalizeIdentityValue(identityType models.InviteIdentityType, value string) string {
	v := strings.TrimSpace(value)
	switch identityType {
	case models.InviteIdentityEmail:
		return strings.ToLower(v)
	case models.InviteIdentityHandle, models.InviteIdentityX, models.InviteIdentityTwitter, models.InviteIdentityGitHub, models.InviteIdentityDiscord, models.InviteIdentityTelegram:
		v = strings.TrimPrefix(v, "@")
		return strings.ToLower(v)
	case models.InviteIdentityWallet:
		if evmAddressRE.MatchString(v) {
			return strings.ToLower(v)
		}
		return v
	default:
		return v
	}
}

func ValidateIdentityValue(identityType models.InviteIdentityType, value string) error {
	n := NormalizeIdentityValue(identityType, value)
	switch identityType {
	case models.InviteIdentityEmail:
		if !emailRE.MatchString(n) {
			return errors.New("invalid email identity")
		}
	case models.InviteIdentityHandle, models.InviteIdentityX, models.InviteIdentityTwitter, models.InviteIdentityGitHub, models.InviteIdentityDiscord, models.InviteIdentityTelegram:
		if !handleRE.MatchString(n) {
			return errors.New("invalid handle identity")
		}
	case models.InviteIdentityWallet:
		if !(evmAddressRE.MatchString(n) || solanaAddrRE.MatchString(n)) {
			return errors.New("invalid wallet identity")
		}
	}
	return nil
}

func DetectInviteIdentityType(value string) models.InviteIdentityType {
	v := strings.TrimSpace(value)
	switch {
	case emailRE.MatchString(v):
		return models.InviteIdentityEmail
	case evmAddressRE.MatchString(v), solanaAddrRE.MatchString(v):
		return models.InviteIdentityWallet
	default:
		return models.InviteIdentityHandle
	}
}

func generateMerkleRoot(leaves []string) string {
	if len(leaves) == 0 {
		return KeccakHex([]byte{})
	}
	layer := append([]string{}, leaves...)
	sort.Strings(layer)
	for len(layer) > 1 {
		next := make([]string, 0, (len(layer)+1)/2)
		for i := 0; i < len(layer); i += 2 {
			left := layer[i]
			right := left
			if i+1 < len(layer) {
				right = layer[i+1]
			}
			pair := left + right
			next = append(next, KeccakHex([]byte(pair)))
		}
		sort.Strings(next)
		layer = next
	}
	return layer[0]
}

type BatchInviteMerkleInputInviteV2 struct {
	InviteID            string                    `json:"inviteId,omitempty"`
	IdentityType        models.InviteIdentityType `json:"identityType"`
	IdentityValue       string                    `json:"identityValue"`
	InviteNonce         string                    `json:"inviteNonce,omitempty"`
	InviteSecret        string                    `json:"inviteSecret,omitempty"`
	ExpiresAt           any                       `json:"expiresAt,omitempty"`
	DefaultPayoutAmount float64                   `json:"defaultPayoutAmount,omitempty"`
}

type BatchInviteMerkleInputV2 struct {
	AppClientID string                           `json:"appClientId"`
	BatchID     string                           `json:"batchId,omitempty"`
	Chain       string                           `json:"chain,omitempty"`
	Scopes      []string                         `json:"scopes"`
	Invites     []BatchInviteMerkleInputInviteV2 `json:"invites"`
	CreatedAt   int64                            `json:"createdAt,omitempty"`
	RootNonce   string                           `json:"rootNonce,omitempty"`
}

func GenerateBatchInviteMerkleDataV2(input BatchInviteMerkleInputV2) (models.BatchInviteMerkleData, error) {
	if len(input.Invites) == 0 {
		return models.BatchInviteMerkleData{}, errors.New("cannot generate invite Merkle data without invites")
	}
	scopes := normalizeInviteScopes(input.Scopes)
	createdAt := input.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}
	rootNonce := input.RootNonce
	if rootNonce == "" {
		var err error
		rootNonce, err = CreateRootNonce(input.BatchID, scopes)
		if err != nil {
			return models.BatchInviteMerkleData{}, err
		}
	}
	invites := make([]models.SignedInvite, 0, len(input.Invites))
	leaves := make([][]byte, 0, len(input.Invites))
	expiresAt := int64(0)
	for _, invite := range input.Invites {
		if err := ValidateIdentityValue(invite.IdentityType, invite.IdentityValue); err != nil {
			return models.BatchInviteMerkleData{}, fmt.Errorf("invalid invite identity (%s=%s): %w", invite.IdentityType, invite.IdentityValue, err)
		}
		nonce := invite.InviteNonce
		if nonce == "" {
			var err error
			nonce, err = CreateInviteNonce()
			if err != nil {
				return models.BatchInviteMerkleData{}, err
			}
		}
		secret := invite.InviteSecret
		if secret == "" {
			var err error
			secret, err = CreateInviteSecret()
			if err != nil {
				return models.BatchInviteMerkleData{}, err
			}
		}
		identityValue := NormalizeIdentityValue(invite.IdentityType, invite.IdentityValue)
		secretHash := BuildInviteSecretHash(secret)
		identityCommitment := BuildInviteIdentityCommitment(invite.IdentityType, identityValue, nonce)
		inviteExpiresAt := toUnixSeconds(invite.ExpiresAt)
		if inviteExpiresAt == 0 {
			inviteExpiresAt = createdAt + int64(7*24*60*60)
		}
		if inviteExpiresAt > expiresAt {
			expiresAt = inviteExpiresAt
		}
		leafMessage := buildInviteLeafMessageV2(input.AppClientID, input.BatchID, invite.IdentityType, identityCommitment, nonce, secretHash, invite.DefaultPayoutAmount, inviteExpiresAt)
		leaf := KeccakHex([]byte(leafMessage))
		leafBytes, _ := hex.DecodeString(strings.TrimPrefix(leaf, "0x"))
		leaves = append(leaves, leafBytes)
		invites = append(invites, models.SignedInvite{
			IdentityType:        invite.IdentityType,
			IdentityValue:       identityValue,
			IdentityCommitment:  identityCommitment,
			SecretHash:          secretHash,
			LeafVersion:         "2",
			InviteNonce:         nonce,
			InviteSecret:        secret,
			DefaultPayoutAmount: invite.DefaultPayoutAmount,
			AppClientID:         input.AppClientID,
			Leaf:                leaf,
			Proof:               []string{},
			ExpiresAt:           time.Unix(inviteExpiresAt, 0).UTC().Format(time.RFC3339),
		})
	}
	levels := buildMerkleLevelsV2(leaves)
	root := "0x" + hex.EncodeToString(levels[len(levels)-1][0])
	for i := range invites {
		invites[i].Proof = merkleProofV2(levels, i)
	}
	signatureMessage := buildInviteRootMessageV2(input.AppClientID, input.BatchID, root, rootNonce, scopes, createdAt, expiresAt)
	return models.BatchInviteMerkleData{
		Version:          "2",
		AppClientID:      input.AppClientID,
		BatchID:          input.BatchID,
		Chain:            input.Chain,
		Scopes:           scopes,
		Root:             root,
		RootNonce:        rootNonce,
		InviteCount:      len(invites),
		CreatedAt:        createdAt,
		ExpiresAt:        expiresAt,
		SignatureMessage: signatureMessage,
		Invites:          invites,
	}, nil
}

type BatchInviteProofVerificationInputV2 struct {
	AppClientID         string                    `json:"appClientId"`
	BatchID             string                    `json:"batchId,omitempty"`
	IdentityType        models.InviteIdentityType `json:"identityType"`
	IdentityValue       string                    `json:"identityValue"`
	InviteNonce         string                    `json:"inviteNonce"`
	InviteSecret        string                    `json:"inviteSecret"`
	IdentityCommitment  string                    `json:"identityCommitment,omitempty"`
	SecretHash          string                    `json:"secretHash,omitempty"`
	DefaultPayoutAmount float64                   `json:"defaultPayoutAmount,omitempty"`
	ExpiresAt           any                       `json:"expiresAt,omitempty"`
	Leaf                string                    `json:"leaf"`
	Proof               []string                  `json:"proof"`
	Root                string                    `json:"root"`
	SignatureMessage    string                    `json:"signatureMessage,omitempty"`
	Signature           string                    `json:"signature,omitempty"`
	SignatureType       string                    `json:"signatureType,omitempty"`
	SignerAddress       string                    `json:"signerAddress,omitempty"`
}

type BatchInviteProofVerificationResultV2 struct {
	Valid              bool     `json:"valid"`
	Leaf               string   `json:"leaf"`
	LeafMessage        string   `json:"leafMessage"`
	IdentityCommitment string   `json:"identityCommitment"`
	SecretHash         string   `json:"secretHash"`
	ProofValid         bool     `json:"proofValid"`
	SignatureValid     *bool    `json:"signatureValid,omitempty"`
	RecoveredSigner    string   `json:"recoveredSigner,omitempty"`
	Errors             []string `json:"errors"`
}

func VerifyBatchInviteProofV2(input BatchInviteProofVerificationInputV2) BatchInviteProofVerificationResultV2 {
	errs := []string{}
	if err := ValidateIdentityValue(input.IdentityType, input.IdentityValue); err != nil {
		errs = append(errs, err.Error())
	}
	identityCommitment := BuildInviteIdentityCommitment(input.IdentityType, input.IdentityValue, input.InviteNonce)
	secretHash := BuildInviteSecretHash(input.InviteSecret)
	expiresAt := toUnixSeconds(input.ExpiresAt)
	leafMessage := buildInviteLeafMessageV2(input.AppClientID, input.BatchID, input.IdentityType, identityCommitment, input.InviteNonce, secretHash, input.DefaultPayoutAmount, expiresAt)
	leaf := KeccakHex([]byte(leafMessage))
	if input.IdentityCommitment != "" && strings.ToLower(input.IdentityCommitment) != strings.ToLower(identityCommitment) {
		errs = append(errs, "Identity commitment does not match signed-in user")
	}
	if input.SecretHash != "" && strings.ToLower(input.SecretHash) != strings.ToLower(secretHash) {
		errs = append(errs, "Secret hash does not match provided invite secret")
	}
	if strings.ToLower(input.Leaf) != strings.ToLower(leaf) {
		errs = append(errs, "Invite leaf does not match invite data")
	}
	proofValid := verifyMerkleProofV2(leaf, input.Proof, input.Root)
	if !proofValid {
		errs = append(errs, "Invite proof is not in the Merkle root")
	}
	var signatureValid *bool
	recoveredSigner := ""
	if input.SignatureType == "evm-personal-sign" && input.Signature != "" && input.SignatureMessage != "" {
		valid := false
		recovered, err := recoverPersonalMessageSigner(input.SignatureMessage, input.Signature)
		if err == nil {
			recoveredSigner = recovered
			valid = input.SignerAddress == "" || strings.EqualFold(recovered, input.SignerAddress)
		}
		if !valid {
			errs = append(errs, "Invite root signature is invalid")
		}
		signatureValid = &valid
	}
	if input.SignatureMessage != "" && !strings.Contains(input.SignatureMessage, input.Root) {
		errs = append(errs, "Invite root signature message does not contain root")
	}
	return BatchInviteProofVerificationResultV2{
		Valid:              len(errs) == 0,
		Leaf:               leaf,
		LeafMessage:        leafMessage,
		IdentityCommitment: identityCommitment,
		SecretHash:         secretHash,
		ProofValid:         proofValid,
		SignatureValid:     signatureValid,
		RecoveredSigner:    recoveredSigner,
		Errors:             errs,
	}
}

func verifyMerkleProofV2(leaf string, proof []string, root string) bool {
	leafBytes, err := hex.DecodeString(strings.TrimPrefix(strings.ToLower(leaf), "0x"))
	if err != nil {
		return false
	}
	h := leafBytes
	for _, p := range proof {
		proofBytes, decodeErr := hex.DecodeString(strings.TrimPrefix(strings.ToLower(p), "0x"))
		if decodeErr != nil {
			return false
		}
		a := h
		b := proofBytes
		if bytes.Compare(a, b) > 0 {
			a, b = b, a
		}
		h = ethcrypto.Keccak256(append(append([]byte{}, a...), b...))
	}
	return "0x"+hex.EncodeToString(h) == strings.ToLower(root)
}

func buildMerkleLevelsV2(leaves [][]byte) [][][]byte {
	current := make([][]byte, len(leaves))
	for i := range leaves {
		current[i] = append([]byte(nil), leaves[i]...)
	}
	levels := [][][]byte{current}
	for len(current) > 1 {
		next := make([][]byte, 0, (len(current)+1)/2)
		for i := 0; i < len(current); i += 2 {
			if i+1 == len(current) {
				next = append(next, current[i])
				continue
			}
			left, right := current[i], current[i+1]
			if bytes.Compare(left, right) > 0 {
				left, right = right, left
			}
			next = append(next, ethcrypto.Keccak256(append(append([]byte(nil), left...), right...)))
		}
		current = next
		levels = append(levels, current)
	}
	return levels
}

func merkleProofV2(levels [][][]byte, index int) []string {
	proof := []string{}
	for level := 0; level < len(levels)-1; level++ {
		nodes := levels[level]
		sibling := index ^ 1
		if sibling < len(nodes) {
			proof = append(proof, "0x"+hex.EncodeToString(nodes[sibling]))
		}
		index /= 2
	}
	return proof
}

func buildInviteLeafMessageV2(appClientID, batchID string, identityType models.InviteIdentityType, identityCommitment, inviteNonce, secretHash string, defaultPayoutAmount float64, expiresAt int64) string {
	amount := ""
	if defaultPayoutAmount != 0 {
		amount = fmt.Sprintf("%v", defaultPayoutAmount)
	}
	return strings.Join([]string{
		"PVIUM_INVITE_LEAF_V2",
		"appClientId=" + appClientID,
		"batchId=" + batchID,
		"identityType=" + string(identityType),
		"identityCommitment=" + identityCommitment,
		"inviteNonce=" + inviteNonce,
		"secretHash=" + secretHash,
		"defaultPayoutAmount=" + amount,
		fmt.Sprintf("expiresAt=%d", expiresAt),
	}, "\n")
}

func buildInviteRootMessageV2(appClientID, batchID, root, rootNonce string, scopes []string, createdAt, expiresAt int64) string {
	return strings.Join([]string{
		"PVIUM_INVITE_ROOT_V2",
		"version=2",
		"appClientId=" + appClientID,
		"batchId=" + batchID,
		"root=" + root,
		"rootNonce=" + rootNonce,
		"scopes=" + strings.Join(scopes, " "),
		fmt.Sprintf("createdAt=%d", createdAt),
		fmt.Sprintf("expiresAt=%d", expiresAt),
	}, "\n")
}

func recoverPersonalMessageSigner(message, signatureHex string) (string, error) {
	sig, err := hex.DecodeString(strings.TrimPrefix(signatureHex, "0x"))
	if err != nil {
		return "", err
	}
	if len(sig) != 65 {
		return "", errors.New("invalid signature length")
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	msgBytes := []byte(message)
	prefixed := ethcrypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(msgBytes))), msgBytes)
	pubKey, err := ethcrypto.SigToPub(prefixed, sig)
	if err != nil {
		return "", err
	}
	return ethcrypto.PubkeyToAddress(*pubKey).Hex(), nil
}

func normalizeInviteScopes(scopes []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	sort.Strings(out)
	return out
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func toUnixSeconds(value any) int64 {
	switch v := value.(type) {
	case nil:
		return 0
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		if v == "" {
			return 0
		}
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			return parsed.Unix()
		}
	default:
		if t, ok := value.(time.Time); ok {
			return t.Unix()
		}
	}
	return 0
}
