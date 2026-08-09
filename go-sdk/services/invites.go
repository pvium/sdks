package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	pvcrypto "github.com/pvium/sdks/go-sdk/crypto"
	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

const defaultInviteTTLSeconds = int64(7 * 24 * 60 * 60)

type InviteService struct {
	client *transport.HTTPClient
}

func NewInviteService(client *transport.HTTPClient) *InviteService {
	return &InviteService{client: client}
}

func (s *InviteService) CreateBundle(input models.OAuthInviteBundleInput) (models.OAuthInviteBundleDraft, error) {
	cfg := s.client.Config()
	if strings.TrimSpace(cfg.ClientID) == "" {
		return models.OAuthInviteBundleDraft{}, errors.New("clientId is required for invite methods")
	}
	if strings.TrimSpace(cfg.ConsentHost) == "" {
		return models.OAuthInviteBundleDraft{}, errors.New("consentHost is required for invite methods")
	}
	if len(input.Identities) == 0 {
		return models.OAuthInviteBundleDraft{}, errors.New("at least one invite identity is required")
	}

	batchID := input.BatchID
	if input.BatchInvite != nil {
		if strings.TrimSpace(input.BatchInvite.BatchID) == "" {
			return models.OAuthInviteBundleDraft{}, errors.New("batchInvite.batchId is required for batch invite bundles")
		}
		batchID = input.BatchInvite.BatchID
	}
	for _, identity := range input.Identities {
		if err := pvcrypto.ValidateIdentityValue(identity.Type, identity.Value); err != nil {
			return models.OAuthInviteBundleDraft{}, fmt.Errorf("invalid invite identity (%s=%s): %w", identity.Type, identity.Value, err)
		}
	}
	orgReferenceID, err := pvcrypto.NormalizeOrgReferenceID(input.OrgReferenceID)
	if err != nil {
		return models.OAuthInviteBundleDraft{}, err
	}

	stateParams := map[string]any{}
	if input.BatchInvite != nil {
		for k, v := range input.BatchInvite.StateParams {
			stateParams[k] = v
		}
	}
	for k, v := range input.StateParams {
		stateParams[k] = v
	}
	var batchInvite *models.OAuthInviteBatchOptions
	if input.BatchInvite != nil {
		copy := *input.BatchInvite
		batchInvite = &copy
	} else if batchID != "" {
		batchInvite = &models.OAuthInviteBatchOptions{BatchID: batchID}
	}

	return models.OAuthInviteBundleDraft{
		ClientID:       cfg.ClientID,
		ConsentHost:    strings.TrimRight(cfg.ConsentHost, "/"),
		Identities:     input.Identities,
		Scopes:         normalizeScopes(input.Scopes, input.Chain),
		BatchID:        batchID,
		BatchInvite:    batchInvite,
		Chain:          input.Chain,
		State:          input.State,
		StateParams:    stateParams,
		RedirectURI:    input.RedirectURI,
		CreatedAt:      input.CreatedAt,
		RootNonce:      input.RootNonce,
		SigningKey:     input.SigningKey,
		OrgReferenceID: orgReferenceID,
	}, nil
}

func (s *InviteService) SignBundle(bundle models.OAuthInviteBundleDraft, signer models.OAuthInviteSigner) (models.SignedOAuthInviteBundle, error) {
	scopes := normalizeScopes(bundle.Scopes, bundle.Chain)
	createdAt := bundle.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}
	batchID := bundle.BatchID
	rootNonce := bundle.RootNonce
	if rootNonce == "" {
		rootNonce = createRootNonce(batchID, scopes)
	}
	derivationSalt := rootNonce
	if batchID != "" {
		derivationSalt = batchID
	}
	masterMessage := pvcrypto.BuildInviteMasterSecretMessage(derivationSalt)
	masterSignature, masterSignerAddress, err := signInviteMessage(signer, masterMessage, true)
	if err != nil {
		return models.SignedOAuthInviteBundle{}, err
	}
	_ = masterSignerAddress
	masterSecret := pvcrypto.DeriveMasterSecret(masterSignature)

	merkle, err := generateInviteMerkle(models.BatchInviteMerkleData{
		AppClientID:    bundle.ClientID,
		BatchID:        batchID,
		Chain:          bundle.Chain,
		Scopes:         scopes,
		RootNonce:      rootNonce,
		CreatedAt:      createdAt,
		OrgReferenceID: bundle.OrgReferenceID,
	}, bundle.Identities, masterSecret)
	if err != nil {
		return models.SignedOAuthInviteBundle{}, err
	}
	merkle.SigningKey, merkle.SigningKeyType, err = pvcrypto.NormalizeSigningKeyRequest(bundle.SigningKey)
	if err != nil {
		return models.SignedOAuthInviteBundle{}, err
	}
	if merkle.OrgReferenceID != "" {
		merkle.DerivedOrgClientID, err = pvcrypto.DeriveOrgClientID(bundle.ClientID, merkle.OrgReferenceID)
		if err != nil {
			return models.SignedOAuthInviteBundle{}, err
		}
		merkle.Version = "4"
		merkle.SignatureMessage = buildRootMessageV4(merkle.AppClientID, merkle.BatchID, merkle.Root, merkle.RootNonce, merkle.Scopes, merkle.SigningKey, merkle.SigningKeyType, merkle.OrgReferenceID, merkle.CreatedAt, merkle.ExpiresAt)
	} else if merkle.SigningKey != "" {
		merkle.Version = "3"
		merkle.SignatureMessage = buildRootMessageV3(merkle.AppClientID, merkle.BatchID, merkle.Root, merkle.RootNonce, merkle.Scopes, merkle.SigningKey, merkle.SigningKeyType, merkle.CreatedAt, merkle.ExpiresAt)
	}

	rootSignature, signerAddress, err := signInviteMessage(signer, merkle.SignatureMessage, false)
	if err != nil {
		return models.SignedOAuthInviteBundle{}, err
	}
	signatureType := "evm-personal-sign"
	if strings.Contains(strings.ToLower(signer.Chain), "solana") {
		signatureType = "solana-message"
	}
	state := buildStateParam(bundle.State, bundle.StateParams, batchID)
	invites := make([]models.SignedInvite, 0, len(merkle.Invites))
	for _, invite := range merkle.Invites {
		invite.InviteLink = generateInviteLink(bundle.ConsentHost, bundle.ClientID, scopes, state, bundle.RedirectURI, batchID, invite.InviteNonce, invite.InviteSecret, invite.IdentityType, invite.IdentityValue)
		invites = append(invites, invite)
	}
	groupInviteLink := generateGroupInviteLink(bundle.ConsentHost, bundle.ClientID, scopes, state, bundle.RedirectURI, batchID, masterSecret)
	inviteLinks := make([]string, 0, len(invites))
	for _, invite := range invites {
		inviteLinks = append(inviteLinks, invite.InviteLink)
	}

	return models.SignedOAuthInviteBundle{
		ClientID:     bundle.ClientID,
		ConsentHost:  bundle.ConsentHost,
		BatchID:      batchID,
		BatchInvite:  bundle.BatchInvite,
		Scopes:       merkle.Scopes,
		Chain:        bundle.Chain,
		MasterSecret: masterSecret,
		Root: models.InviteRootSignature{
			Root:               merkle.Root,
			Nonce:              merkle.RootNonce,
			Signature:          rootSignature,
			SignatureType:      signatureType,
			Scopes:             merkle.Scopes,
			SigningKey:         merkle.SigningKey,
			SigningKeyType:     merkle.SigningKeyType,
			OrgReferenceID:     merkle.OrgReferenceID,
			DerivedOrgClientID: merkle.DerivedOrgClientID,
			SignatureMessage:   merkle.SignatureMessage,
			SignatureTimestamp: merkle.CreatedAt,
			SignerAddress:      signerAddress,
			InviteCount:        merkle.InviteCount,
			ExpiresAt:          time.Unix(merkle.ExpiresAt, 0).UTC().Format(time.RFC3339),
			Metadata: map[string]any{
				"version":      merkle.Version,
				"leafEncoding": "PVIUM_INVITE_LEAF_V2",
				"signingChain": firstNonEmpty(signer.Chain, bundle.Chain),
			},
		},
		Invites:         invites,
		InviteLinks:     inviteLinks,
		GroupInviteLink: groupInviteLink,
		Merkle:          merkle,
	}, nil
}

func (s *InviteService) CommitBundle(ctx context.Context, bundle models.SignedOAuthInviteBundle, options *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	batchID := bundle.BatchID
	if bundle.BatchInvite != nil && bundle.BatchInvite.BatchID != "" {
		batchID = bundle.BatchInvite.BatchID
	}
	path := fmt.Sprintf("/client-apps/%s/invites", bundle.ClientID)
	if batchID != "" {
		path = fmt.Sprintf("/batch-payments/%s/invites", batchID)
	} else if options != nil && options.CommitClientAppID != "" {
		path = fmt.Sprintf("/client-apps/%s/invites", options.CommitClientAppID)
	}
	invites := make([]map[string]any, 0, len(bundle.Invites))
	for _, invite := range bundle.Invites {
		item := map[string]any{
			"identityType":        invite.IdentityType,
			"identityValue":       invite.IdentityValue,
			"identityCommitment":  invite.IdentityCommitment,
			"secretHash":          invite.SecretHash,
			"leafVersion":         invite.LeafVersion,
			"inviteNonce":         invite.InviteNonce,
			"defaultPayoutAmount": invite.DefaultPayoutAmount,
			"appClientId":         invite.AppClientID,
			"leaf":                invite.Leaf,
			"proof":               invite.Proof,
		}
		if invite.DefaultPayoutAmount == 0 {
			delete(item, "defaultPayoutAmount")
		}
		if invite.ExpiresAt != "" {
			item["expiresAt"] = invite.ExpiresAt
		}
		invites = append(invites, item)
	}
	body := map[string]any{"root": bundle.Root, "invites": invites}
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: path, Body: body, Options: options})
	if err != nil {
		return models.APIResponse[map[string]any]{}, err
	}
	return transport.Decode[models.APIResponse[map[string]any]](raw)
}

func (s *InviteService) CreateSignedBundle(input models.OAuthInviteBundleInput, signer models.OAuthInviteSigner) (models.SignedOAuthInviteBundle, error) {
	bundle, err := s.CreateBundle(input)
	if err != nil {
		return models.SignedOAuthInviteBundle{}, err
	}
	return s.SignBundle(bundle, signer)
}

func (s *InviteService) CreateSignedAndCommit(ctx context.Context, input models.OAuthInviteBundleInput, signer models.OAuthInviteSigner, options *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	signed, err := s.CreateSignedBundle(input, signer)
	if err != nil {
		return models.APIResponse[map[string]any]{}, err
	}
	return s.CommitBundle(ctx, signed, options)
}

// openOrganizationInvitePolicy is marshalled with fields in EXACT insertion order to
// match the node SDK's JSON.stringify output used for the signed message and policyHash.
type openOrganizationInvitePolicy struct {
	AppClientID          string   `json:"appClientId"`
	AllowedEmailDomains  []string `json:"allowedEmailDomains"`
	AllowedIdentityTypes []string `json:"allowedIdentityTypes"`
	CreatedAt            int64    `json:"createdAt"`
	ExpiresAt            string   `json:"expiresAt"`
	InviteNonce          string   `json:"inviteNonce"`
	MaxUses              int64    `json:"maxUses"`
	RequireKyc           bool     `json:"requireKyc"`
	RequireTaxProfile    bool     `json:"requireTaxProfile"`
	Scopes               []string `json:"scopes"`
}

const openOrganizationInviteEncoding = "PVIUM_OPEN_ORGANIZATION_INVITE_V1"

func (s *InviteService) CreateOpenOrganizationInvite(input models.OpenOrganizationInviteInput) (models.OpenOrganizationInviteDraft, error) {
	cfg := s.client.Config()
	if strings.TrimSpace(cfg.ClientID) == "" {
		return models.OpenOrganizationInviteDraft{}, errors.New("clientId is required for invite methods")
	}
	consentHost := strings.TrimRight(cfg.ConsentHost, "/")
	if consentHost == "" {
		return models.OpenOrganizationInviteDraft{}, errors.New("consentHost is required for invite methods")
	}

	createdAt := input.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}
	scopes := input.Scopes
	if len(scopes) == 0 {
		scopes = defaultScopesForChain("")
	}
	expiresAt := ""
	if strings.TrimSpace(input.ExpiresAt) != "" {
		expiresAt = normalizeISOTimestamp(input.ExpiresAt)
	}
	inviteNonce := input.InviteNonce
	if inviteNonce == "" {
		generated, err := pvcrypto.CreateInviteNonce()
		if err != nil {
			return models.OpenOrganizationInviteDraft{}, err
		}
		inviteNonce = generated
	}
	inviteSecret := input.InviteSecret
	if inviteSecret == "" {
		generated, err := pvcrypto.CreateInviteSecret()
		if err != nil {
			return models.OpenOrganizationInviteDraft{}, err
		}
		inviteSecret = generated
	}

	return models.OpenOrganizationInviteDraft{
		ClientID:             cfg.ClientID,
		ConsentHost:          consentHost,
		Label:                input.Label,
		Scopes:               normalizeStringSet(scopes),
		AllowedIdentityTypes: normalizeStringSet(input.AllowedIdentityTypes),
		AllowedEmailDomains:  normalizeEmailDomains(input.AllowedEmailDomains),
		RequireKyc:           input.RequireKyc,
		RequireTaxProfile:    input.RequireTaxProfile,
		MaxUses:              input.MaxUses,
		ExpiresAt:            expiresAt,
		RedirectURI:          input.RedirectURI,
		State:                input.State,
		StateParams:          input.StateParams,
		CreatedAt:            createdAt,
		InviteNonce:          inviteNonce,
		InviteSecret:         inviteSecret,
	}, nil
}

func (s *InviteService) SignOpenOrganizationInvite(draft models.OpenOrganizationInviteDraft, signer models.OAuthInviteSigner) (models.SignedOpenOrganizationInvite, error) {
	policy := buildOpenOrganizationInvitePolicy(draft)
	policyJSON, err := compactJSON(policy)
	if err != nil {
		return models.SignedOpenOrganizationInvite{}, err
	}
	message := openOrganizationInviteEncoding + "\n" + policyJSON

	signature, signerAddress, err := signInviteMessage(signer, message, false)
	if err != nil {
		return models.SignedOpenOrganizationInvite{}, err
	}
	signatureType := "evm-personal-sign"
	if strings.Contains(strings.ToLower(signer.Chain), "solana") {
		signatureType = "solana-message"
	}

	maxUses := int64(0)
	if policy.MaxUses > 0 {
		maxUses = policy.MaxUses
	}

	return models.SignedOpenOrganizationInvite{
		ClientID:             draft.ClientID,
		ConsentHost:          draft.ConsentHost,
		Label:                draft.Label,
		InviteNonce:          draft.InviteNonce,
		InviteSecret:         draft.InviteSecret,
		SecretHash:           sha256Hex(draft.InviteSecret),
		PolicyHash:           sha256Hex(policyJSON),
		Signature:            signature,
		SignatureType:        signatureType,
		SignatureMessage:     message,
		SignatureTimestamp:   draft.CreatedAt,
		SignerAddress:        signerAddress,
		Scopes:               policy.Scopes,
		AllowedIdentityTypes: policy.AllowedIdentityTypes,
		AllowedEmailDomains:  policy.AllowedEmailDomains,
		RequireKyc:           policy.RequireKyc,
		RequireTaxProfile:    policy.RequireTaxProfile,
		MaxUses:              maxUses,
		ExpiresAt:            policy.ExpiresAt,
		RedirectURI:          draft.RedirectURI,
		State:                draft.State,
		StateParams:          draft.StateParams,
		Metadata: map[string]string{
			"version":  "1",
			"encoding": openOrganizationInviteEncoding,
		},
	}, nil
}

func (s *InviteService) CommitOpenOrganizationInvite(ctx context.Context, invite models.SignedOpenOrganizationInvite, options *models.RequestOptions) (models.OpenOrganizationInviteCommitResult, error) {
	body, err := openInviteCommitBody(invite)
	if err != nil {
		return models.OpenOrganizationInviteCommitResult{}, err
	}
	path := fmt.Sprintf("/client-apps/%s/open-invites", url.PathEscape(invite.ClientID))
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: path, Body: body, Options: options})
	if err != nil {
		return models.OpenOrganizationInviteCommitResult{}, err
	}
	parsed, err := transport.Decode[map[string]any](raw)
	if err != nil {
		return models.OpenOrganizationInviteCommitResult{}, err
	}

	record := getResponseValue(parsed)
	inviteID := ""
	if record != nil {
		inviteID = firstNonEmpty(fmt.Sprint(record["id"]), fmt.Sprint(record["_id"]))
		if inviteID == "<nil>" {
			inviteID = ""
		}
	}

	inviteLink := ""
	if inviteID != "" {
		inviteLink = generateOpenOrganizationInviteLink(
			invite.ConsentHost,
			invite.ClientID,
			inviteID,
			invite.InviteSecret,
			invite.Scopes,
			invite.RedirectURI,
			buildStateParam(invite.State, invite.StateParams, ""),
		)
	}

	result := models.OpenOrganizationInviteCommitResult{Raw: parsed, InviteLink: inviteLink}
	if record != nil {
		merged := map[string]any{}
		for k, v := range record {
			merged[k] = v
		}
		if inviteLink != "" {
			merged["inviteLink"] = inviteLink
		}
		result.Invite = merged
	}
	return result, nil
}

func (s *InviteService) CreateSignedOpenOrganizationInvite(input models.OpenOrganizationInviteInput, signer models.OAuthInviteSigner) (models.SignedOpenOrganizationInvite, error) {
	draft, err := s.CreateOpenOrganizationInvite(input)
	if err != nil {
		return models.SignedOpenOrganizationInvite{}, err
	}
	return s.SignOpenOrganizationInvite(draft, signer)
}

func (s *InviteService) CreateSignedAndCommitOpenOrganizationInvite(ctx context.Context, input models.OpenOrganizationInviteInput, signer models.OAuthInviteSigner, options *models.RequestOptions) (models.OpenOrganizationInviteCommitResult, error) {
	invite, err := s.CreateSignedOpenOrganizationInvite(input, signer)
	if err != nil {
		return models.OpenOrganizationInviteCommitResult{}, err
	}
	return s.CommitOpenOrganizationInvite(ctx, invite, options)
}

func buildOpenOrganizationInvitePolicy(draft models.OpenOrganizationInviteDraft) openOrganizationInvitePolicy {
	expiresAt := ""
	if strings.TrimSpace(draft.ExpiresAt) != "" {
		expiresAt = normalizeISOTimestamp(draft.ExpiresAt)
	}
	return openOrganizationInvitePolicy{
		AppClientID:          draft.ClientID,
		AllowedEmailDomains:  normalizeEmailDomains(draft.AllowedEmailDomains),
		AllowedIdentityTypes: normalizeStringSet(draft.AllowedIdentityTypes),
		CreatedAt:            draft.CreatedAt,
		ExpiresAt:            expiresAt,
		InviteNonce:          draft.InviteNonce,
		MaxUses:              draft.MaxUses,
		RequireKyc:           draft.RequireKyc,
		RequireTaxProfile:    draft.RequireTaxProfile,
		Scopes:               normalizeStringSet(draft.Scopes),
	}
}

// compactJSON matches JS JSON.stringify: no HTML escaping and no trailing newline.
func compactJSON(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func openInviteCommitBody(invite models.SignedOpenOrganizationInvite) (map[string]any, error) {
	raw, err := json.Marshal(invite)
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	for _, key := range []string{"inviteSecret", "consentHost", "redirectUri", "state", "stateParams"} {
		delete(body, key)
	}
	return body, nil
}

func getResponseValue(response map[string]any) map[string]any {
	if response == nil {
		return nil
	}
	if data, ok := response["data"].(map[string]any); ok {
		return data
	}
	if value, ok := response["value"].(map[string]any); ok {
		return value
	}
	return response
}

func defaultScopesForChain(chain string) []string {
	scopes := []string{"read:user"}
	lc := strings.ToLower(chain)
	if strings.Contains(lc, "solana") {
		scopes = append(scopes, "read:solana_wallet")
	} else if lc != "" {
		scopes = append(scopes, "read:ethereum_wallet")
	}
	return normalizeStringSet(scopes)
}

func normalizeStringSet(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func normalizeEmailDomains(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// normalizeISOTimestamp mirrors JS new Date(x).toISOString(): millisecond precision, Z suffix.
func normalizeISOTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	layouts := []string{
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05.999999999Z07:00",
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	return value
}

func generateOpenOrganizationInviteLink(consentHost, clientID, inviteID, inviteSecret string, scopes []string, redirectURI, state string) string {
	u, _ := url.Parse(strings.TrimRight(consentHost, "/") + "/o/" + url.PathEscape(inviteID))
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(normalizeStringSet(scopes), " "))
	if redirectURI != "" {
		q.Set("redirect_uri", redirectURI)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String() + "#s=" + url.QueryEscape(inviteSecret)
}

func generateInviteMerkle(base models.BatchInviteMerkleData, identities []models.InviteIdentity, masterSecret string) (models.BatchInviteMerkleData, error) {
	if len(identities) == 0 {
		return models.BatchInviteMerkleData{}, errors.New("cannot generate invite Merkle data without invites")
	}
	expiresAt := int64(0)
	invites := make([]models.SignedInvite, 0, len(identities))
	leafBuffers := make([][]byte, 0, len(identities))
	for _, identity := range identities {
		inviteNonce := createInviteNonceNoPrefix()
		inviteSecret := pvcrypto.DeriveInviteSecret(masterSecret, inviteNonce)
		secretHash := sha256Hex(inviteSecret)
		identityValue := pvcrypto.NormalizeIdentityValue(identity.Type, identity.Value)
		identityCommitment := buildIdentityCommitment(identity.Type, identityValue, inviteNonce)
		inviteExpiresAt := base.CreatedAt + defaultInviteTTLSeconds
		if identity.ExpiresAt != "" {
			if parsed, err := time.Parse(time.RFC3339, identity.ExpiresAt); err == nil {
				inviteExpiresAt = parsed.Unix()
			}
		}
		if inviteExpiresAt > expiresAt {
			expiresAt = inviteExpiresAt
		}
		leafMessage := buildLeafMessage(base.AppClientID, base.BatchID, identity.Type, identityCommitment, inviteNonce, secretHash, identity.DefaultPayoutAmount, inviteExpiresAt)
		leaf := pvcrypto.KeccakHex([]byte(leafMessage))
		leafBytes, _ := hex.DecodeString(strings.TrimPrefix(leaf, "0x"))
		leafBuffers = append(leafBuffers, leafBytes)
		expiresAtISO := time.Unix(inviteExpiresAt, 0).UTC().Format(time.RFC3339)
		invites = append(invites, models.SignedInvite{
			IdentityType:        identity.Type,
			IdentityValue:       identityValue,
			IdentityCommitment:  identityCommitment,
			SecretHash:          secretHash,
			LeafVersion:         "2",
			InviteNonce:         inviteNonce,
			InviteSecret:        inviteSecret,
			DefaultPayoutAmount: identity.DefaultPayoutAmount,
			AppClientID:         base.AppClientID,
			Leaf:                leaf,
			Proof:               []string{},
			ExpiresAt:           expiresAtISO,
		})
	}
	levels := buildMerkleLevels(leafBuffers)
	root := "0x" + hex.EncodeToString(levels[len(levels)-1][0])
	for i := range invites {
		invites[i].Proof = merkleProof(levels, i)
	}
	base.Version = "2"
	base.Scopes = normalizeScopes(base.Scopes, base.Chain)
	base.Root = root
	base.InviteCount = len(invites)
	base.ExpiresAt = expiresAt
	base.SignatureMessage = buildRootMessage(base.AppClientID, base.BatchID, root, base.RootNonce, base.Scopes, base.CreatedAt, expiresAt)
	base.Invites = invites
	return base, nil
}

func signInviteMessage(signer models.OAuthInviteSigner, message string, master bool) (string, string, error) {
	if signer.PrivateKey != "" {
		signature, err := signPersonalMessage(message, signer.PrivateKey)
		if err != nil {
			return "", "", err
		}
		privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(signer.PrivateKey, "0x"))
		if err != nil {
			return "", "", err
		}
		return signature, crypto.PubkeyToAddress(privateKey.PublicKey).Hex(), nil
	}
	sign := signer.SignMessage
	if master && signer.SignMasterSecret != nil {
		sign = signer.SignMasterSecret
	}
	if !master && signer.SignInviteRoot != nil {
		sign = signer.SignInviteRoot
	}
	if sign == nil {
		return "", "", errors.New("invite signer must provide privateKey or signMessage")
	}
	signature, err := sign(message)
	return signature, signer.SignerAddress, err
}

func buildStateParam(state string, stateParams map[string]any, batchID string) string {
	entries := map[string]any{}
	for k, v := range stateParams {
		if v != nil {
			entries[k] = v
		}
	}
	if len(entries) == 0 {
		if state != "" {
			return state
		}
		if batchID != "" {
			return "b_" + batchID
		}
		return ""
	}
	params := url.Values{}
	if state != "" {
		params.Set("state", state)
	}
	if batchID != "" {
		params.Set("batchId", batchID)
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		params.Set(k, fmt.Sprint(entries[k]))
	}
	return params.Encode()
}

func normalizeScopes(scopes []string, chain string) []string {
	if len(scopes) == 0 {
		chainLower := strings.ToLower(chain)
		scopes = []string{"read:user"}
		if strings.Contains(chainLower, "solana") {
			scopes = append(scopes, "read:solana_wallet")
		} else if chainLower != "" {
			scopes = append(scopes, "read:ethereum_wallet")
		}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(scopes))
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

func generateInviteLink(consentHost, clientID string, scopes []string, state, redirectURI, batchID, inviteNonce, inviteSecret string, identityType models.InviteIdentityType, identityHint string) string {
	u, _ := url.Parse(strings.TrimRight(consentHost, "/") + "/oauth2/authorize")
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(normalizeScopes(scopes, ""), " "))
	if redirectURI != "" {
		q.Set("redirect_uri", redirectURI)
	}
	if state != "" {
		q.Set("state", state)
	}
	if batchID != "" {
		q.Set("batchId", batchID)
	}
	q.Set("invite_nonce", inviteNonce)
	q.Set("invite_secret", inviteSecret)
	q.Set("identity_type", string(identityType))
	q.Set("identity_hint", pvcrypto.NormalizeIdentityValue(identityType, identityHint))
	u.RawQuery = q.Encode()
	return u.String()
}

func generateGroupInviteLink(consentHost, clientID string, scopes []string, state, redirectURI, batchID, masterSecret string) string {
	u, _ := url.Parse(strings.TrimRight(consentHost, "/") + "/oauth2/authorize")
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(normalizeScopes(scopes, ""), " "))
	if redirectURI != "" {
		q.Set("redirect_uri", redirectURI)
	}
	if state != "" {
		q.Set("state", state)
	}
	if batchID != "" {
		q.Set("batchId", batchID)
	}
	q.Set("batch_link_secret", masterSecret)
	u.RawQuery = q.Encode()
	return u.String()
}

func createRootNonce(batchID string, scopes []string) string {
	return sha256Hex(strings.Join([]string{"payy.invite.root.v1", batchID, strings.Join(scopes, " "), randomHex(16)}, ":"))
}

func createInviteNonceNoPrefix() string {
	return randomHex(16)
}

func randomHex(bytesLen int) string {
	buf := make([]byte, bytesLen)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func buildIdentityCommitment(identityType models.InviteIdentityType, value, inviteNonce string) string {
	return sha256Hex(strings.Join([]string{"pvium.invite.identity.v2", string(identityType), pvcrypto.NormalizeIdentityValue(identityType, value), inviteNonce}, ":"))
}

func buildLeafMessage(appClientID, batchID string, identityType models.InviteIdentityType, identityCommitment, inviteNonce, secretHash string, defaultPayoutAmount float64, expiresAt int64) string {
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

func buildRootMessage(appClientID, batchID, root, rootNonce string, scopes []string, createdAt, expiresAt int64) string {
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

func buildRootMessageV3(appClientID, batchID, root, rootNonce string, scopes []string, signingKey, signingKeyType string, createdAt, expiresAt int64) string {
	return strings.Join([]string{
		"PVIUM_INVITE_ROOT_V3",
		"version=3",
		"appClientId=" + appClientID,
		"batchId=" + batchID,
		"root=" + root,
		"rootNonce=" + rootNonce,
		"scopes=" + strings.Join(scopes, " "),
		"signingKey=" + signingKey,
		"signingKeyType=" + signingKeyType,
		fmt.Sprintf("createdAt=%d", createdAt),
		fmt.Sprintf("expiresAt=%d", expiresAt),
	}, "\n")
}

func buildRootMessageV4(appClientID, batchID, root, rootNonce string, scopes []string, signingKey, signingKeyType, orgReferenceID string, createdAt, expiresAt int64) string {
	return strings.Join([]string{
		"PVIUM_INVITE_ROOT_V4",
		"version=4",
		"appClientId=" + appClientID,
		"batchId=" + batchID,
		"root=" + root,
		"rootNonce=" + rootNonce,
		"scopes=" + strings.Join(scopes, " "),
		"signingKey=" + signingKey,
		"signingKeyType=" + signingKeyType,
		"orgReferenceId=" + orgReferenceID,
		fmt.Sprintf("createdAt=%d", createdAt),
		fmt.Sprintf("expiresAt=%d", expiresAt),
	}, "\n")
}
