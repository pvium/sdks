package pvium

import (
	"github.com/pvium/sdks/go-sdk/config"
	pvcrypto "github.com/pvium/sdks/go-sdk/crypto"
	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/services"
	"github.com/pvium/sdks/go-sdk/transport"
	pvwebhooks "github.com/pvium/sdks/go-sdk/webhooks"
)

// Re-export config symbols for backwards compatibility.
const (
	SandboxBaseURL        = config.SandboxBaseURL
	ProductionBaseURL     = config.ProductionBaseURL
	SandboxConsentHost    = config.SandboxConsentHost
	ProductionConsentHost = config.ProductionConsentHost
	DefaultTimeout        = config.DefaultTimeout
	DefaultInviteTTL      = config.DefaultInviteTTL
)

const (
	EnvironmentSandbox    = config.EnvironmentSandbox
	EnvironmentProduction = config.EnvironmentProduction
)

type Environment = config.Environment
type Config = config.Config

type RequestOptions = models.RequestOptions
type APIMeta = models.APIMeta
type PaginationMeta = models.PaginationMeta
type APIResponse[T any] = models.APIResponse[T]

type InvoicePaymentChannel = models.InvoicePaymentChannel
type CreateInvoiceRequest = models.CreateInvoiceRequest
type InstallmentPlanItem = models.InstallmentPlanItem
type InvoiceItem = models.InvoiceItem
type InvoiceListItem = models.InvoiceListItem
type CreateInvoiceData = models.CreateInvoiceData
type InvoiceStatusInstallment = models.InvoiceStatusInstallment
type InvoiceStatusData = models.InvoiceStatusData
type InstallmentPayment = models.InstallmentPayment

type OAuthTokenData = models.OAuthTokenData
type OAuthSocialHandle = models.OAuthSocialHandle
type OAuthLinkedAccount = models.OAuthLinkedAccount
type OAuthUserInfo = models.OAuthUserInfo
type ExchangeAuthorizationCodeInput = models.ExchangeAuthorizationCodeInput
type RefreshAccessTokenInput = models.RefreshAccessTokenInput

type InviteIdentityType = models.InviteIdentityType

const (
	InviteIdentityEmail    = models.InviteIdentityEmail
	InviteIdentityHandle   = models.InviteIdentityHandle
	InviteIdentityWallet   = models.InviteIdentityWallet
	InviteIdentityX        = models.InviteIdentityX
	InviteIdentityGitHub   = models.InviteIdentityGitHub
	InviteIdentityTwitter  = models.InviteIdentityTwitter
	InviteIdentityDiscord  = models.InviteIdentityDiscord
	InviteIdentityTelegram = models.InviteIdentityTelegram
)

type InviteIdentity = models.InviteIdentity
type OAuthInviteBatchOptions = models.OAuthInviteBatchOptions
type OAuthInviteBundleInput = models.OAuthInviteBundleInput
type OAuthInviteSigner = models.OAuthInviteSigner
type OAuthInviteBundleDraft = models.OAuthInviteBundleDraft
type SignedOAuthInviteBundle = models.SignedOAuthInviteBundle
type SignedInvite = models.SignedInvite
type InviteRootSignature = models.InviteRootSignature
type BatchInviteMerkleData = models.BatchInviteMerkleData

type PayoutType = models.PayoutType

const (
	PayoutTypeInstant   = models.PayoutTypeInstant
	PayoutTypeScheduled = models.PayoutTypeScheduled
	PayoutTypeEscrow    = models.PayoutTypeEscrow
	PayoutTypeMilestone = models.PayoutTypeMilestone
)

type PayoutChain = models.PayoutChain

const (
	PayoutChainBase          = models.PayoutChainBase
	PayoutChainBSC           = models.PayoutChainBSC
	PayoutChainSolana        = models.PayoutChainSolana
	PayoutChainBaseTestnet   = models.PayoutChainBaseTestnet
	PayoutChainSolanaTestnet = models.PayoutChainSolanaTestnet
	PayoutChainLocalhost     = models.PayoutChainLocalhost
)

type PayoutCurrency = models.PayoutCurrency

const (
	PayoutCurrencyUSDC = models.PayoutCurrencyUSDC
	PayoutCurrencyUSDT = models.PayoutCurrencyUSDT
)

type PayoutComplianceMode = models.PayoutComplianceMode

const (
	PayoutComplianceOpen   = models.PayoutComplianceOpen
	PayoutComplianceStrict = models.PayoutComplianceStrict
)

type PayoutPayment = models.PayoutPayment
type PayoutRecipient = models.PayoutRecipient
type PayoutRecord = models.PayoutRecord
type CreatePayoutInput = models.CreatePayoutInput
type PayoutListQuery = models.PayoutListQuery
type PayoutPaymentsListQuery = models.PayoutPaymentsListQuery
type FinalizePayoutData = models.FinalizePayoutData
type AddPayoutPaymentsInput = models.AddPayoutPaymentsInput
type RemovePayoutPaymentsInput = models.RemovePayoutPaymentsInput
type UpdatePayoutPaymentInput = models.UpdatePayoutPaymentInput
type AddPayoutRecipientsInput = models.AddPayoutRecipientsInput
type ResolvePayoutRecipient = models.ResolvePayoutRecipient
type ResolvePayoutRecipientsInput = models.ResolvePayoutRecipientsInput
type PayoutRecipientResult = models.PayoutRecipientResult
type PayoutRecipientError = models.PayoutRecipientError
type AddPayoutRecipientsResult = models.AddPayoutRecipientsResult
type ResolvePayoutRecipientsResult = models.ResolvePayoutRecipientsResult
type PayoutSignerInput = models.PayoutSignerInput
type PayoutFinalizeOptions = models.PayoutFinalizeOptions
type PayoutIntent = services.PayoutIntent
type PayoutFinalization = services.PayoutFinalization

type PviumWebhookTokenPayload = models.PviumWebhookTokenPayload
type VerifyPviumWebhookTokenOptions = models.VerifyPviumWebhookTokenOptions

type SignerInput = pvcrypto.SignerInput
type MessageSigner = pvcrypto.MessageSigner
type CreateProjectRequestPayload = pvcrypto.CreateProjectRequestPayload
type CreateProjectSignatureOptions = pvcrypto.CreateProjectSignatureOptions
type CreateClaimRequestPayload = pvcrypto.CreateClaimRequestPayload
type FinalizeClaimRequestPayload = pvcrypto.FinalizeClaimRequestPayload
type ResolveDisputeRequestPayload = pvcrypto.ResolveDisputeRequestPayload
type RelayedCallRequestPayload = pvcrypto.RelayedCallRequestPayload
type ScheduledPayoutHashParams = pvcrypto.ScheduledPayoutHashParams
type EscrowPayoutHashParams = pvcrypto.EscrowPayoutHashParams
type BatchInviteMerkleInputV2 = pvcrypto.BatchInviteMerkleInputV2
type BatchInviteMerkleInputInviteV2 = pvcrypto.BatchInviteMerkleInputInviteV2
type BatchInviteProofVerificationInputV2 = pvcrypto.BatchInviteProofVerificationInputV2
type BatchInviteProofVerificationResultV2 = pvcrypto.BatchInviteProofVerificationResultV2

type HTTPClient = transport.HTTPClient
type EndpointsService = services.EndpointsService
type OAuthService = services.OAuthService
type InviteService = services.InviteService
type PayoutService = services.PayoutService

func NewHTTPClient(cfg Config) *HTTPClient { return transport.NewHTTPClient(cfg) }
func NewEndpointsService(client *HTTPClient) *EndpointsService {
	return services.NewEndpointsService(client)
}
func NewOAuthService(client *HTTPClient) *OAuthService   { return services.NewOAuthService(client) }
func NewInviteService(client *HTTPClient) *InviteService { return services.NewInviteService(client) }
func NewPayoutService(client *HTTPClient) *PayoutService { return services.NewPayoutService(client) }

func SignatureDomainFromText(text string) string { return pvcrypto.SignatureDomainFromText(text) }
func KeccakHex(input []byte) string              { return pvcrypto.KeccakHex(input) }
func HashABIEncodedPayload(types []string, values []any) (string, error) {
	return pvcrypto.HashABIEncodedPayload(types, values)
}
func SignMessageHash(hashHex string, privateKey string) (string, error) {
	return pvcrypto.SignMessageHash(hashHex, privateKey)
}
func CreateSignerFromPrivateKey(privateKey string) MessageSigner {
	return pvcrypto.CreateSignerFromPrivateKey(privateKey)
}
func HashCreateProjectRequest(payload CreateProjectRequestPayload, options CreateProjectSignatureOptions) (string, error) {
	return pvcrypto.HashCreateProjectRequest(payload, options)
}
func HashCreateClaimRequest(payload CreateClaimRequestPayload) (string, error) {
	return pvcrypto.HashCreateClaimRequest(payload)
}
func HashFinalizeClaimRequest(payload FinalizeClaimRequestPayload, chainID any) (string, error) {
	return pvcrypto.HashFinalizeClaimRequest(payload, chainID)
}
func HashDisputeRequest(claimID string, chainID uint64) (string, error) {
	return pvcrypto.HashDisputeRequest(claimID, chainID)
}
func HashResolveDisputeRequest(payload ResolveDisputeRequestPayload) (string, error) {
	return pvcrypto.HashResolveDisputeRequest(payload)
}
func HashRelayedCallRequest(payload RelayedCallRequestPayload) (string, error) {
	return pvcrypto.HashRelayedCallRequest(payload)
}
func SignCreateProjectRequest(payload CreateProjectRequestPayload, signer SignerInput, options CreateProjectSignatureOptions) (string, error) {
	return pvcrypto.SignCreateProjectRequest(payload, signer, options)
}
func HashCreateProjectAttestation(appSignature string, chainID any, signatureDomain ...string) (string, error) {
	return pvcrypto.HashCreateProjectAttestation(appSignature, chainID, signatureDomain...)
}
func SignCreateProjectAttestation(appSignature string, signer SignerInput, chainID any, signatureDomain ...string) (string, error) {
	return pvcrypto.SignCreateProjectAttestation(appSignature, signer, chainID, signatureDomain...)
}
func SignCreateClaimRequest(payload CreateClaimRequestPayload, signer SignerInput) (string, error) {
	return pvcrypto.SignCreateClaimRequest(payload, signer)
}
func SignFinalizeClaimRequest(payload FinalizeClaimRequestPayload, signer SignerInput, chainID any) (string, error) {
	return pvcrypto.SignFinalizeClaimRequest(payload, signer, chainID)
}
func SignDisputeRequest(claimID string, signer SignerInput, chainID uint64) (string, error) {
	return pvcrypto.SignDisputeRequest(claimID, signer, chainID)
}
func SignResolveDisputeRequest(payload ResolveDisputeRequestPayload, signer SignerInput) (string, error) {
	return pvcrypto.SignResolveDisputeRequest(payload, signer)
}
func SignRelayedCallRequest(payload RelayedCallRequestPayload, signer SignerInput) (string, error) {
	return pvcrypto.SignRelayedCallRequest(payload, signer)
}

func CreatePayoutNonce() (string, error) { return pvcrypto.CreatePayoutNonce() }
func GenerateInstantPayoutHash(payments []PayoutPayment, nonce string) (string, error) {
	return pvcrypto.GenerateInstantPayoutHash(payments, nonce)
}
func ComputeScheduledPayoutHash(params ScheduledPayoutHashParams) (string, error) {
	return pvcrypto.ComputeScheduledPayoutHash(params)
}
func ComputeEscrowPayoutHash(params EscrowPayoutHashParams) (string, error) {
	return pvcrypto.ComputeEscrowPayoutHash(params)
}
func ComputeEscrowFundingDigest(escrowBatchHash, withdrawalWallet string) string {
	return pvcrypto.ComputeEscrowFundingDigest(escrowBatchHash, withdrawalWallet)
}
func ComputeEscrowScheduledFundingDigest(escrowBatchHash, merkleRoot string) string {
	return pvcrypto.ComputeEscrowScheduledFundingDigest(escrowBatchHash, merkleRoot)
}

func CreateInviteNonce() (string, error)  { return pvcrypto.CreateInviteNonce() }
func CreateInviteSecret() (string, error) { return pvcrypto.CreateInviteSecret() }
func CreateRootNonce(batchID string, scopes []string) (string, error) {
	return pvcrypto.CreateRootNonce(batchID, scopes)
}
func BuildInviteMasterSecretMessage(rootNonce string) string {
	return pvcrypto.BuildInviteMasterSecretMessage(rootNonce)
}
func DeriveMasterSecret(rawSignatureHex string) string {
	return pvcrypto.DeriveMasterSecret(rawSignatureHex)
}
func DeriveInviteSecret(masterSecret, inviteNonce string) string {
	return pvcrypto.DeriveInviteSecret(masterSecret, inviteNonce)
}
func NormalizeIdentityValue(identityType InviteIdentityType, value string) string {
	return pvcrypto.NormalizeIdentityValue(identityType, value)
}
func ValidateIdentityValue(identityType InviteIdentityType, value string) error {
	return pvcrypto.ValidateIdentityValue(identityType, value)
}
func DetectInviteIdentityType(value string) InviteIdentityType {
	return pvcrypto.DetectInviteIdentityType(value)
}
func GenerateBatchInviteMerkleDataV2(input BatchInviteMerkleInputV2) (BatchInviteMerkleData, error) {
	return pvcrypto.GenerateBatchInviteMerkleDataV2(input)
}
func VerifyBatchInviteProofV2(input BatchInviteProofVerificationInputV2) BatchInviteProofVerificationResultV2 {
	return pvcrypto.VerifyBatchInviteProofV2(input)
}

func VerifyPviumWebhookToken(token, secret string, options *VerifyPviumWebhookTokenOptions) (PviumWebhookTokenPayload, error) {
	return pvwebhooks.VerifyPviumWebhookToken(token, secret, options)
}
func ResolvePviumWebhookPayload(body map[string]any, secret string) (PviumWebhookTokenPayload, error) {
	return pvwebhooks.ResolvePviumWebhookPayload(body, secret)
}
