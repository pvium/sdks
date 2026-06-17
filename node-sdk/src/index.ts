import { PviumEndpoints } from "./endpoints";
import { PviumHttpClient, PviumSdkConfig } from "./client";
import { PviumInviteService } from "./invites";
import { PviumOAuth } from "./oauth";
import { PviumPayoutService } from "./payout";
export { PviumApiError } from "./client";
export type { PviumSdkConfig } from "./client";
export type {
  CreateInvoiceData,
  CreateInvoiceRequest,
  CreateInvoiceResponse,
  CancelInvoiceResponse,
  InvoiceListItem,
  InstallmentPayment,
  InvoiceStatusResponse,
  ListInvoicesResponse,
  OAuthLinkedAccount,
  OAuthSocialHandle,
  OAuthTokenData,
  OAuthTokenResponse,
  OAuthUserInfo,
  OAuthUserInfoResponse,
  RequestOptions,
} from "./types";
export { PviumOAuth } from "./oauth";
export type {
  ExchangeAuthorizationCodeInput,
  RefreshAccessTokenInput,
} from "./oauth";
export {
  PVIUM_SIGNATURE_DOMAIN,
  createSignerFromPrivateKey,
  hashAbiEncodedPayload,
  hashCreateClaimRequest,
  hashCreateProjectAttestation,
  hashCreateProjectRequest,
  hashDisputeRequest,
  hashFinalizeClaimRequest,
  hashRelayedCallRequest,
  hashResolveDisputeRequest,
  signCreateClaimRequest,
  signCreateProjectAttestation,
  signCreateProjectRequest,
  signDisputeRequest,
  signFinalizeClaimRequest,
  signMessageHash,
  signRelayedCallRequest,
  signResolveDisputeRequest,
  signatureDomainFromText,
} from "./signing";
export type {
  CreateClaimRequestPayload,
  CreateProjectRequestPayload,
  CreateProjectSignatureOptions,
  FinalizeClaimRequestPayload,
  HexString,
  MessageSigner,
  Numeric,
  RelayedCallRequestPayload,
  ResolveDisputeRequestPayload,
  SignerInput,
} from "./signing";
export {
  buildInviteMasterSecretMessage,
  createInviteNonce,
  createInviteSecret,
  createRootNonce,
  deriveInviteSecret,
  deriveMasterSecret,
  detectInviteIdentityType,
  generateBatchInviteMerkleDataV2,
  normalizeIdentityValue,
  SUPPORTED_INVITE_IDENTITY_TYPES,
  validateIdentityValue,
  verifyBatchInviteProofV2,
} from "./invite-merkle";
export {
  resolvePviumWebhookPayload,
  verifyPviumWebhookToken,
} from "./webhooks";
export type {
  PviumWebhookTokenPayload,
  VerifyPviumWebhookTokenOptions,
} from "./webhooks";
export type {
  BatchInviteMerkleDataV2,
  BatchInviteMerkleInputV2,
  BatchInviteMerkleInviteV2,
  BatchInviteProofVerificationInputV2,
  BatchInviteProofVerificationResultV2,
  InviteIdentityType,
} from "./invite-merkle";
export { PviumInviteService } from "./invites";
export {
  computeSigningKeyAuthorizationHash,
  computeScheduledPayoutHash,
  createPayoutNonce,
  generateInstantPayoutHash,
  PayoutFinalization,
  PayoutIntent,
  PayoutCurrency,
  PviumPayoutService,
} from "./payout";
export type {
  AppInviteRecord,
  FindAppInviteByIdentityInput,
  FindAppInviteByIdentityResult,
  InviteMessageSignature,
  InviteSigningChain,
  OAuthInviteBatchData,
  OAuthInviteBundleDraft,
  OAuthInviteBundleInput,
  OAuthInviteCommitResult,
  OAuthInviteIdentity,
  OAuthInviteSigner,
  OAuthInviteStateParams,
  SignedOAuthInviteBundle,
} from "./invites";
export type {
  AddPayoutPaymentsInput,
  AddPayoutRecipientsInput,
  AddPayoutRecipientsResult,
  CreatePayoutInput,
  FinalizePayoutData,
  PayoutApiResponse,
  PayoutChain,
  PayoutComplianceMode,
  PayoutCurrencyInput,
  PayoutFinalizeOptions,
  PayoutFundingCall,
  PayoutListQuery,
  PayoutMessageSignature,
  PayoutMessageSigner,
  PayoutPayment,
  PayoutPaymentsListQuery,
  PayoutProviderSigner,
  PayoutRecipient,
  PayoutRecipientError,
  PayoutRecipientResult,
  PayoutRecord,
  PayoutSigningKeyAuthorization,
  PayoutSigningKeyAuthorizationData,
  PayoutSigningKeyNetworkType,
  PayoutSigningKeySignatureResult,
  PayoutSigningKeySignerInput,
  PayoutSignerInput,
  PayoutType,
  RemovePayoutPaymentsInput,
  ResolvePayoutRecipient,
  ResolvePayoutRecipientsInput,
  ResolvePayoutRecipientsResult,
  UpdatePayoutPaymentInput,
} from "./payout";

export class PviumSdk {
  readonly http: PviumHttpClient;
  readonly endpoints: PviumEndpoints;
  readonly invites: PviumInviteService;
  readonly oauth: PviumOAuth;
  readonly payout: PviumPayoutService;

  constructor(config: PviumSdkConfig) {
    this.http = new PviumHttpClient(config);
    this.endpoints = new PviumEndpoints(this.http);
    this.invites = new PviumInviteService(this.http, config);
    this.oauth = new PviumOAuth(this.http, config);
    this.payout = new PviumPayoutService(this.http, config);
  }

  static init(config: PviumSdkConfig): PviumSdk {
    return new PviumSdk(config);
  }
}
