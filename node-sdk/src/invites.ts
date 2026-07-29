import { Wallet } from "ethers";
import crypto from "crypto-js";
import {
  PviumHttpClient,
  PviumSdkConfig,
  resolvePviumConsentHost,
} from "./client";
import { RequestOptions } from "./types";
import {
  buildInviteMasterSecretMessage,
  createInviteNonce,
  createInviteSecret,
  createRootNonce,
  deriveInviteSecret,
  deriveMasterSecret,
  generateBatchInviteMerkleDataV2,
  normalizeIdentityValue,
  validateIdentityValue,
  type BatchInviteMerkleDataV2,
  type InviteIdentityType,
} from "./invite-merkle";

export type InviteSigningChain = "ethereum" | "solana";

export type OAuthScope =
  | "read:user"
  | "read:invoice"
  | "write:invoice"
  | "read:accepted_invoice"
  | "write:accepted_invoice"
  | "read:business_profile"
  | "write:business_profile"
  | "read:ethereum_wallet"
  | "read:solana_wallet"
  | "read:kyc_status"
  | "read:aml_status"
  | "read:legal_id"
  /**
   * @deprecated Use read:legal_id instead.
   */
  | "read:kyc"
  | "read:tax_certificates"
  | "read:kyc_legal_name"
  | "read:kyc_country"
  | "read:kyc_tax_id"
  | "read:kyc_dob"
  | "read:kyc_address"
  | "read:kyc_document_metadata"
  | "read:batch_payment"
  | "write:batch_payment"
  | "write:escrow_claim"
  | "read:escrow_earnings"
  | "write:escrow_account"
  | "read:escrow_account"
  | "write:escrow_dispute"
  | (string & {});

export interface OAuthInviteIdentity {
  type: InviteIdentityType;
  value: string;
  defaultPayoutAmount?: number;
  expiresAt?: string | Date;
}

export type OAuthInviteStateParams = Record<
  string,
  string | number | boolean | null | undefined
>;

export interface OAuthInviteBatchData {
  batchId: string;
  stateParams?: OAuthInviteStateParams;
}

export interface OAuthInviteBundleInput {
  identities: OAuthInviteIdentity[];
  scopes?: OAuthScope[];
  /**
   * @deprecated Prefer batchInvite.batchId for batch payment invite bundles.
   * Kept for backwards compatibility.
   */
  batchId?: string;
  batchInvite?: OAuthInviteBatchData;
  chain?: InviteSigningChain | string;
  state?: string;
  stateParams?: OAuthInviteStateParams;
  redirectUri?: string;
  createdAt?: number;
  rootNonce?: string;
}

export interface OAuthInviteBundleDraft {
  clientId: string;
  consentHost: string;
  identities: OAuthInviteIdentity[];
  scopes: string[];
  batchId?: string;
  batchInvite?: OAuthInviteBatchData;
  chain?: InviteSigningChain | string;
  state?: string;
  stateParams?: OAuthInviteStateParams;
  redirectUri?: string;
  createdAt?: number;
  rootNonce?: string;
}

export interface InviteMessageSignature {
  signature: string;
  signatureType?: "evm-personal-sign" | "solana-message" | string;
  signerAddress?: string;
}

export type OAuthInviteSigner =
  | {
      chain: "ethereum";
      privateKey: string;
    }
  | {
      chain: "ethereum";
      signMessage: (
        message: string,
      ) => Promise<string | InviteMessageSignature>;
      signMasterSecret?: (
        message: string,
      ) => Promise<string | InviteMessageSignature>;
      signInviteRoot?: (
        message: string,
      ) => Promise<string | InviteMessageSignature>;
      signerAddress?: string;
    }
  | {
      chain: "solana";
      signMessage: (
        message: Uint8Array,
      ) => Promise<Uint8Array | string | InviteMessageSignature>;
      signMasterSecret?: (
        message: Uint8Array,
      ) => Promise<Uint8Array | string | InviteMessageSignature>;
      signInviteRoot?: (
        message: Uint8Array,
      ) => Promise<Uint8Array | string | InviteMessageSignature>;
      signerAddress?: string;
    };

export interface SignedOAuthInviteBundle {
  clientId: string;
  consentHost: string;
  batchId: string;
  batchInvite?: OAuthInviteBatchData;
  scopes: string[];
  chain?: InviteSigningChain | string;
  masterSecret: string;
  root: {
    root: string;
    nonce: string;
    signature: string;
    signatureType: string;
    scopes: string[];
    signatureMessage: string;
    signatureTimestamp: number;
    signerAddress?: string;
    inviteCount: number;
    expiresAt?: string;
    metadata: {
      version: "2";
      leafEncoding: "PVIUM_INVITE_LEAF_V2";
      signingChain?: string;
    };
  };
  invites: Array<{
    identityType: InviteIdentityType;
    identityValue: string;
    identityCommitment: string;
    secretHash: string;
    leafVersion: "2";
    inviteNonce: string;
    inviteSecret: string;
    inviteLink: string;
    defaultPayoutAmount?: number;
    appClientId: string;
    leaf: string;
    proof: string[];
    expiresAt?: string;
  }>;
  inviteLinks: string[];
  groupInviteLink?: string;
  merkle: BatchInviteMerkleDataV2;
}

export interface AppInviteRecord {
  id?: string;
  _id?: string;
  identityType?: string;
  identityValue?: string;
  status?: string;
  inviteNonce?: string;
  acceptedAt?: string;
  expiresAt?: string;
  createdAt?: string;
  updatedAt?: string;
  [key: string]: unknown;
}

export interface FindAppInviteByIdentityInput {
  identityType: InviteIdentityType;
  identityValue: string;
  status?: string;
}

export interface FindAppInviteByIdentityResult {
  invites: AppInviteRecord[];
  invite: AppInviteRecord | null;
  accepted: boolean;
  raw: unknown;
}

export interface OAuthInviteCommitResult {
  raw: unknown;
  invites: AppInviteRecord[];
  committedInvites: AppInviteRecord[];
  existingInvites: AppInviteRecord[];
  inviteCommitted: boolean;
  alreadyAccepted: boolean;
}

export interface OpenOrganizationInviteInput {
  label?: string;
  scopes?: OAuthScope[];
  allowedIdentityTypes?: string[];
  allowedEmailDomains?: string[];
  requireKyc?: boolean;
  requireTaxProfile?: boolean;
  maxUses?: number;
  expiresAt?: string | Date;
  redirectUri?: string;
  state?: string;
  stateParams?: OAuthInviteStateParams;
  createdAt?: number;
  inviteNonce?: string;
  inviteSecret?: string;
}

export interface OpenOrganizationInviteDraft {
  clientId: string;
  consentHost: string;
  label?: string;
  scopes: string[];
  allowedIdentityTypes: string[];
  allowedEmailDomains: string[];
  requireKyc: boolean;
  requireTaxProfile: boolean;
  maxUses?: number;
  expiresAt?: string;
  redirectUri?: string;
  state?: string;
  stateParams?: OAuthInviteStateParams;
  createdAt: number;
  inviteNonce: string;
  inviteSecret: string;
}

export interface SignedOpenOrganizationInvite {
  clientId: string;
  consentHost: string;
  label?: string;
  inviteNonce: string;
  inviteSecret: string;
  secretHash: string;
  policyHash: string;
  signature: string;
  signatureType: string;
  signatureMessage: string;
  signatureTimestamp: number;
  signerAddress?: string;
  scopes: string[];
  allowedIdentityTypes: string[];
  allowedEmailDomains: string[];
  requireKyc: boolean;
  requireTaxProfile: boolean;
  maxUses?: number;
  expiresAt?: string;
  redirectUri?: string;
  state?: string;
  stateParams?: OAuthInviteStateParams;
  metadata: {
    version: "1";
    encoding: "PVIUM_OPEN_ORGANIZATION_INVITE_V1";
  };
}

export interface OpenOrganizationInviteCommitResult {
  raw: unknown;
  invite: AppInviteRecord | null;
  inviteLink?: string;
}

const normalizeScopes = (scopes: string[]): string[] => {
  return Array.from(
    new Set(scopes.map((scope) => scope.trim()).filter(Boolean)),
  ).sort((a, b) => a.localeCompare(b));
};

const defaultScopesForChain = (chain?: string): string[] => {
  const chainLower = chain?.toLowerCase() || "";
  const scopes = ["read:user"];

  if (chainLower.includes("solana")) {
    scopes.push("read:solana_wallet");
  } else if (chainLower) {
    scopes.push("read:ethereum_wallet");
  }

  return normalizeScopes(scopes);
};

const toHexString = (bytes: Uint8Array): string =>
  Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");

const bytesToBase64 = (bytes: Uint8Array): string => {
  if (typeof Buffer !== "undefined") {
    return Buffer.from(bytes).toString("base64");
  }

  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary);
};

const normalizeConsentHost = (host: string): string => host.replace(/\/$/, "");

const normalizeStateParamValue = (
  value: string | number | boolean | null | undefined,
): string | undefined => {
  if (value === undefined || value === null) return undefined;
  return String(value);
};

const buildInviteState = (params: {
  state?: string;
  stateParams?: OAuthInviteStateParams;
  batchId?: string;
}): string | undefined => {
  const entries = Object.entries(params.stateParams ?? {}).filter(
    ([, value]) => value !== undefined && value !== null,
  );

  if (entries.length === 0) {
    return params.state ?? (params.batchId ? `b_${params.batchId}` : undefined);
  }

  const state = new URLSearchParams();

  if (params.state) {
    state.set("state", params.state);
  }

  if (params.batchId) {
    state.set("batchId", params.batchId);
  }

  for (const [key, value] of entries) {
    const normalized = normalizeStateParamValue(value);
    if (normalized !== undefined) state.set(key, normalized);
  }

  return state.toString();
};

const getResponseItems = (response: unknown): unknown[] => {
  if (!response || typeof response !== "object") return [];

  const record = response as Record<string, unknown>;
  if (Array.isArray(record.data)) return record.data;
  if (Array.isArray(record.value)) return record.value;

  const data = record.data;
  if (data && typeof data === "object") {
    const dataRecord = data as Record<string, unknown>;
    if (Array.isArray(dataRecord.value)) return dataRecord.value;
    if (Array.isArray(dataRecord.items)) return dataRecord.items;
  }

  return [];
};

const sha256 = (value: string): string => crypto.SHA256(value).toString();

const normalizeOpenInviteIdentityTypes = (identityTypes: string[] = []) =>
  Array.from(
    new Set(identityTypes.map((type) => type.trim()).filter(Boolean)),
  ).sort((a, b) => a.localeCompare(b));

const normalizeOpenInviteEmailDomains = (domains: string[] = []) =>
  Array.from(
    new Set(
      domains.map((domain) => domain.trim().toLowerCase()).filter(Boolean),
    ),
  ).sort((a, b) => a.localeCompare(b));

export class PviumInviteService {
  constructor(
    private readonly http: PviumHttpClient,
    private readonly config: PviumSdkConfig,
  ) {}

  createBundle(input: OAuthInviteBundleInput): OAuthInviteBundleDraft {
    const clientId = this.requireClientId();
    const consentHost = this.requireConsentHost();
    const batchId = input.batchInvite?.batchId ?? input.batchId;

    if (!input.identities.length) {
      throw new Error("At least one invite identity is required");
    }

    if (input.batchInvite && !input.batchInvite.batchId.trim()) {
      throw new Error(
        "batchInvite.batchId is required for batch invite bundles",
      );
    }

    for (const identity of input.identities) {
      const err = validateIdentityValue(identity.type, identity.value);
      if (err) {
        throw new Error(
          `Invalid invite identity (${identity.type}=${identity.value}): ${err}`,
        );
      }
    }

    return {
      clientId,
      consentHost,
      identities: input.identities,
      scopes: normalizeScopes(
        input.scopes ?? defaultScopesForChain(input.chain),
      ),
      batchId,
      batchInvite: input.batchInvite
        ? {
            ...input.batchInvite,
            batchId: input.batchInvite.batchId,
          }
        : batchId
          ? { batchId }
          : undefined,
      chain: input.chain,
      state: input.state,
      stateParams: {
        ...(input.batchInvite?.stateParams ?? {}),
        ...(input.stateParams ?? {}),
      },
      redirectUri: input.redirectUri,
      createdAt: input.createdAt,
      rootNonce: input.rootNonce,
    };
  }

  async signBundle(
    bundle: OAuthInviteBundleDraft,
    signer: OAuthInviteSigner,
  ): Promise<SignedOAuthInviteBundle> {
    const scopes = normalizeScopes(bundle.scopes);
    const createdAt = bundle.createdAt ?? Math.floor(Date.now() / 1000);
    const batchId = bundle.batchId || "";
    const rootNonce = bundle.rootNonce || createRootNonce(batchId, scopes);
    const derivationSalt = batchId || rootNonce;
    const masterMessage = buildInviteMasterSecretMessage(derivationSalt);
    const masterSignature = await this.signMessageForMasterSecret(
      masterMessage,
      signer,
    );
    const masterSecret = deriveMasterSecret(masterSignature.signatureHex);

    const inviteEntries = bundle.identities.map((identity) => {
      const inviteNonce = createInviteNonce();
      return {
        identityType: identity.type,
        identityValue: identity.value,
        inviteNonce,
        inviteSecret: deriveInviteSecret(masterSecret, inviteNonce),
        defaultPayoutAmount: identity.defaultPayoutAmount,
        expiresAt: identity.expiresAt,
      };
    });

    const merkle = generateBatchInviteMerkleDataV2({
      appClientId: bundle.clientId,
      batchId: batchId || undefined,
      chain: bundle.chain,
      scopes,
      createdAt,
      rootNonce,
      invites: inviteEntries,
    });

    const rootSignature = await this.signRootMessage(
      merkle.signatureMessage,
      signer,
    );
    const signingChain = signer.chain || bundle.chain;
    const state = buildInviteState({
      state: bundle.state,
      stateParams: bundle.stateParams,
      batchId,
    });

    const invites = merkle.invites.map((invite) => {
      const expiresAt = invite.expiresAt
        ? new Date(invite.expiresAt * 1000).toISOString()
        : undefined;
      const inviteLink = this.generateInviteLink({
        consentHost: bundle.consentHost,
        clientId: bundle.clientId,
        scopes: merkle.scopes,
        state,
        redirectUri: bundle.redirectUri,
        batchId: batchId || undefined,
        inviteNonce: invite.inviteNonce,
        inviteSecret: invite.inviteSecret,
        identityType: invite.identityType,
        identityHint: invite.identityValue,
      });

      return {
        identityType: invite.identityType,
        identityValue: invite.identityValue,
        identityCommitment: invite.identityCommitment,
        secretHash: invite.secretHash,
        leafVersion: merkle.version,
        inviteNonce: invite.inviteNonce,
        inviteSecret: invite.inviteSecret,
        inviteLink,
        defaultPayoutAmount: invite.defaultPayoutAmount,
        appClientId: bundle.clientId,
        leaf: invite.leaf,
        proof: invite.proof,
        expiresAt,
      };
    });

    const groupInviteLink = this.generateGroupInviteLink({
      consentHost: bundle.consentHost,
      clientId: bundle.clientId,
      scopes: merkle.scopes,
      state,
      redirectUri: bundle.redirectUri,
      batchId: batchId || undefined,
      masterSecret,
    });

    return {
      clientId: bundle.clientId,
      consentHost: bundle.consentHost,
      batchId,
      batchInvite: bundle.batchInvite,
      scopes: merkle.scopes,
      chain: bundle.chain,
      masterSecret,
      root: {
        root: merkle.root,
        nonce: merkle.rootNonce,
        signature: rootSignature.signature,
        signatureType: rootSignature.signatureType,
        scopes: merkle.scopes,
        signatureMessage: merkle.signatureMessage,
        signatureTimestamp: merkle.createdAt,
        signerAddress: rootSignature.signerAddress,
        inviteCount: merkle.inviteCount,
        expiresAt: merkle.expiresAt
          ? new Date(merkle.expiresAt * 1000).toISOString()
          : undefined,
        metadata: {
          version: merkle.version,
          leafEncoding: "PVIUM_INVITE_LEAF_V2",
          signingChain,
        },
      },
      invites,
      inviteLinks: invites.map((invite) => invite.inviteLink),
      groupInviteLink,
      merkle,
    };
  }

  async commitBundle(
    bundle: SignedOAuthInviteBundle,
    options?: RequestOptions,
  ): Promise<OAuthInviteCommitResult> {
    const batchId = bundle.batchInvite?.batchId || bundle.batchId;
    const path = batchId
      ? `/v1/batch-payments/${encodeURIComponent(batchId)}/invites`
      : `/v1/client-apps/${encodeURIComponent(bundle.clientId)}/invites`;

    const response = await this.http.request({
      method: "POST",
      path,
      body: {
        root: bundle.root,
        invites: bundle.invites.map(
          ({ inviteSecret, inviteLink, ...invite }) => invite,
        ),
      },
      options,
    });

    const body = await this.http.parseResponseBody<unknown>(response);
    const inviteSecretByNonce = new Map(
      bundle.invites.map((invite) => [invite.inviteNonce, invite.inviteSecret]),
    );
    const invites = (getResponseItems(body) as AppInviteRecord[]).map(
      (invite) => {
        const inviteId = String(invite.id || invite._id || "");
        const inviteSecret =
          typeof invite.inviteNonce === "string"
            ? inviteSecretByNonce.get(invite.inviteNonce)
            : undefined;

        if (!inviteId || !inviteSecret) return invite;

        return {
          ...invite,
          inviteLink: this.generateShortInviteLink({
            consentHost: bundle.consentHost,
            inviteId,
            inviteSecret,
          }),
        };
      },
    );
    const requestedNonces = new Set(
      bundle.invites.map((invite) => invite.inviteNonce),
    );
    const committedInvites = invites.filter(
      (invite) =>
        typeof invite.inviteNonce === "string" &&
        requestedNonces.has(invite.inviteNonce),
    );
    const existingInvites = invites.filter(
      (invite) =>
        typeof invite.inviteNonce === "string" &&
        !requestedNonces.has(invite.inviteNonce),
    );

    return {
      raw: body,
      invites,
      committedInvites,
      existingInvites,
      inviteCommitted: committedInvites.length > 0,
      alreadyAccepted: existingInvites.some(
        (invite) => invite.status === "accepted",
      ),
    };
  }

  async findAppInviteByIdentity(
    input: FindAppInviteByIdentityInput,
    options?: RequestOptions,
  ): Promise<FindAppInviteByIdentityResult> {
    const response = await this.http.request({
      method: "GET",
      path: "/v1/batch-payments/app-invites",
      query: {
        identityType: input.identityType,
        identityValue: input.identityValue,
        status: input.status,
      },
      options,
    });
    const body = await this.http.parseResponseBody<unknown>(response);
    const invites = getResponseItems(body) as AppInviteRecord[];
    const normalizedIdentityValue = normalizeIdentityValue(
      input.identityType,
      input.identityValue,
    );
    const invite =
      invites.find(
        (item) =>
          item.identityType === input.identityType &&
          item.identityValue === normalizedIdentityValue &&
          (!input.status || item.status === input.status),
      ) ?? null;

    return {
      invites,
      invite,
      accepted: invite?.status === "accepted",
      raw: body,
    };
  }

  async createSignedBundle(
    input: OAuthInviteBundleInput,
    signer: OAuthInviteSigner,
  ): Promise<SignedOAuthInviteBundle> {
    return this.signBundle(this.createBundle(input), signer);
  }

  async createSignedAndCommit(
    input: OAuthInviteBundleInput,
    signer: OAuthInviteSigner,
    options?: RequestOptions,
  ): Promise<unknown> {
    const bundle = await this.createSignedBundle(input, signer);
    return this.commitBundle(bundle, options);
  }

  createOpenOrganizationInvite(
    input: OpenOrganizationInviteInput = {},
  ): OpenOrganizationInviteDraft {
    const clientId = this.requireClientId();
    const consentHost = this.requireConsentHost();
    const createdAt = input.createdAt ?? Math.floor(Date.now() / 1000);

    return {
      clientId,
      consentHost,
      label: input.label,
      scopes: normalizeScopes(input.scopes ?? defaultScopesForChain()),
      allowedIdentityTypes: normalizeOpenInviteIdentityTypes(
        input.allowedIdentityTypes ?? [],
      ),
      allowedEmailDomains: normalizeOpenInviteEmailDomains(
        input.allowedEmailDomains ?? [],
      ),
      requireKyc: !!input.requireKyc,
      requireTaxProfile: !!input.requireTaxProfile,
      maxUses: input.maxUses,
      expiresAt: input.expiresAt
        ? new Date(input.expiresAt).toISOString()
        : undefined,
      redirectUri: input.redirectUri,
      state: input.state,
      stateParams: input.stateParams,
      createdAt,
      inviteNonce: input.inviteNonce || createInviteNonce(),
      inviteSecret: input.inviteSecret || createInviteSecret(),
    };
  }

  async signOpenOrganizationInvite(
    draft: OpenOrganizationInviteDraft,
    signer: OAuthInviteSigner,
  ): Promise<SignedOpenOrganizationInvite> {
    const policy = this.buildOpenOrganizationInvitePolicy(draft);
    const signatureMessage =
      this.buildOpenOrganizationInvitePolicyMessage(policy);
    const signature = await this.signRootMessage(signatureMessage, signer);

    return {
      clientId: draft.clientId,
      consentHost: draft.consentHost,
      label: draft.label,
      inviteNonce: draft.inviteNonce,
      inviteSecret: draft.inviteSecret,
      secretHash: sha256(draft.inviteSecret),
      policyHash: sha256(JSON.stringify(policy)),
      signature: signature.signature,
      signatureType: signature.signatureType,
      signatureMessage,
      signatureTimestamp: draft.createdAt,
      signerAddress: signature.signerAddress,
      scopes: policy.scopes,
      allowedIdentityTypes: policy.allowedIdentityTypes,
      allowedEmailDomains: policy.allowedEmailDomains,
      requireKyc: policy.requireKyc,
      requireTaxProfile: policy.requireTaxProfile,
      maxUses: policy.maxUses > 0 ? policy.maxUses : undefined,
      expiresAt: policy.expiresAt || undefined,
      redirectUri: draft.redirectUri,
      state: draft.state,
      stateParams: draft.stateParams,
      metadata: {
        version: "1",
        encoding: "PVIUM_OPEN_ORGANIZATION_INVITE_V1",
      },
    };
  }

  async commitOpenOrganizationInvite(
    invite: SignedOpenOrganizationInvite,
    options?: RequestOptions,
  ): Promise<OpenOrganizationInviteCommitResult> {
    const { inviteSecret, consentHost, redirectUri, state, stateParams, ...body } =
      invite;
    const response = await this.http.request({
      method: "POST",
      path: `/v1/client-apps/${encodeURIComponent(invite.clientId)}/open-invites`,
      body,
      options,
    });

    const raw = await this.http.parseResponseBody<unknown>(response);
    const record = this.getResponseValue(raw) as AppInviteRecord | null;
    const inviteId = String(record?.id || record?._id || "");
    const inviteLink = inviteId
      ? this.generateOpenOrganizationInviteLink({
          consentHost,
          clientId: invite.clientId,
          inviteId,
          inviteSecret,
          scopes: invite.scopes,
          redirectUri,
          state: buildInviteState({ state, stateParams }),
        })
      : undefined;

    return {
      raw,
      invite: record ? { ...record, inviteLink } : null,
      inviteLink,
    };
  }

  async createSignedOpenOrganizationInvite(
    input: OpenOrganizationInviteInput,
    signer: OAuthInviteSigner,
  ): Promise<SignedOpenOrganizationInvite> {
    return this.signOpenOrganizationInvite(
      this.createOpenOrganizationInvite(input),
      signer,
    );
  }

  async createSignedAndCommitOpenOrganizationInvite(
    input: OpenOrganizationInviteInput,
    signer: OAuthInviteSigner,
    options?: RequestOptions,
  ): Promise<OpenOrganizationInviteCommitResult> {
    const invite = await this.createSignedOpenOrganizationInvite(input, signer);
    return this.commitOpenOrganizationInvite(invite, options);
  }

  private buildOpenOrganizationInvitePolicy(
    draft: OpenOrganizationInviteDraft,
  ) {
    return {
      appClientId: draft.clientId,
      allowedEmailDomains: normalizeOpenInviteEmailDomains(
        draft.allowedEmailDomains,
      ),
      allowedIdentityTypes: normalizeOpenInviteIdentityTypes(
        draft.allowedIdentityTypes,
      ),
      createdAt: Number(draft.createdAt || 0),
      expiresAt: draft.expiresAt ? new Date(draft.expiresAt).toISOString() : "",
      inviteNonce: draft.inviteNonce,
      maxUses:
        draft.maxUses === undefined || draft.maxUses === null
          ? 0
          : Number(draft.maxUses),
      requireKyc: !!draft.requireKyc,
      requireTaxProfile: !!draft.requireTaxProfile,
      scopes: normalizeScopes(draft.scopes || []),
    };
  }

  private buildOpenOrganizationInvitePolicyMessage(
    policy: Record<string, unknown>,
  ): string {
    return ["PVIUM_OPEN_ORGANIZATION_INVITE_V1", JSON.stringify(policy)].join(
      "\n",
    );
  }

  private getResponseValue(response: unknown): unknown {
    if (!response || typeof response !== "object") return null;
    const record = response as Record<string, unknown>;
    if (record.data !== undefined) return record.data;
    if (record.value !== undefined) return record.value;
    return response;
  }

  private async signMessageForMasterSecret(
    message: string,
    signer: OAuthInviteSigner,
  ): Promise<{ signatureHex: string; signerAddress?: string }> {
    if (signer.chain === "ethereum" && "privateKey" in signer) {
      const wallet = new Wallet(signer.privateKey);
      const signature = await wallet.signMessage(message);
      return {
        signatureHex: signature.replace(/^0x/, "").toLowerCase(),
        signerAddress: wallet.address,
      };
    }

    if (signer.chain === "ethereum") {
      const result = await (signer.signMasterSecret ?? signer.signMessage)(
        message,
      );
      const signature = typeof result === "string" ? result : result.signature;
      return {
        signatureHex: signature.replace(/^0x/, "").toLowerCase(),
        signerAddress:
          typeof result === "string"
            ? signer.signerAddress
            : result.signerAddress,
      };
    }

    const encoded = new TextEncoder().encode(message);
    const result = await (signer.signMasterSecret ?? signer.signMessage)(
      encoded,
    );
    const signature =
      typeof result === "string" || result instanceof Uint8Array
        ? result
        : result.signature;
    const signatureHex =
      signature instanceof Uint8Array
        ? toHexString(signature)
        : Buffer.from(signature, "base64").toString("hex");

    return {
      signatureHex,
      signerAddress:
        typeof result === "string" || result instanceof Uint8Array
          ? signer.signerAddress
          : result.signerAddress,
    };
  }

  private async signRootMessage(
    message: string,
    signer: OAuthInviteSigner,
  ): Promise<{
    signature: string;
    signatureType: string;
    signerAddress?: string;
  }> {
    if (signer.chain === "ethereum" && "privateKey" in signer) {
      const wallet = new Wallet(signer.privateKey);
      return {
        signature: await wallet.signMessage(message),
        signatureType: "evm-personal-sign",
        signerAddress: wallet.address,
      };
    }

    if (signer.chain === "ethereum") {
      const result = await (signer.signInviteRoot ?? signer.signMessage)(
        message,
      );
      if (typeof result === "string") {
        return {
          signature: result,
          signatureType: "evm-personal-sign",
          signerAddress: signer.signerAddress,
        };
      }

      return {
        signature: result.signature,
        signatureType: result.signatureType || "evm-personal-sign",
        signerAddress: result.signerAddress || signer.signerAddress,
      };
    }

    const encoded = new TextEncoder().encode(message);
    const result = await (signer.signInviteRoot ?? signer.signMessage)(encoded);

    if (result instanceof Uint8Array) {
      return {
        signature: bytesToBase64(result),
        signatureType: "solana-message",
        signerAddress: signer.signerAddress,
      };
    }

    if (typeof result === "string") {
      return {
        signature: result,
        signatureType: "solana-message",
        signerAddress: signer.signerAddress,
      };
    }

    return {
      signature: result.signature,
      signatureType: result.signatureType || "solana-message",
      signerAddress: result.signerAddress || signer.signerAddress,
    };
  }

  private generateInviteLink(params: {
    consentHost: string;
    clientId: string;
    scopes: string[];
    state?: string;
    redirectUri?: string;
    batchId?: string;
    inviteNonce: string;
    inviteSecret: string;
    identityType: InviteIdentityType;
    identityHint?: string;
  }): string {
    const authUrl = new URL(
      "/oauth2/authorize",
      normalizeConsentHost(params.consentHost),
    );
    authUrl.searchParams.set("client_id", params.clientId);
    authUrl.searchParams.set("response_type", "code");
    authUrl.searchParams.set("scope", normalizeScopes(params.scopes).join(" "));
    if (params.redirectUri) {
      authUrl.searchParams.set("redirect_uri", params.redirectUri);
    }
    if (params.state) authUrl.searchParams.set("state", params.state);
    if (params.batchId) authUrl.searchParams.set("batchId", params.batchId);
    authUrl.searchParams.set("invite_nonce", params.inviteNonce);
    authUrl.searchParams.set("invite_secret", params.inviteSecret);
    authUrl.searchParams.set("identity_type", params.identityType);
    if (params.identityHint) {
      authUrl.searchParams.set(
        "identity_hint",
        normalizeIdentityValue(params.identityType, params.identityHint),
      );
    }
    return authUrl.toString();
  }

  private generateShortInviteLink(params: {
    consentHost: string;
    inviteId: string;
    inviteSecret: string;
  }): string {
    const authUrl = new URL(
      `/i/${encodeURIComponent(params.inviteId)}`,
      normalizeConsentHost(params.consentHost),
    );
    authUrl.hash = `s=${encodeURIComponent(params.inviteSecret)}`;
    return authUrl.toString();
  }

  private generateOpenOrganizationInviteLink(params: {
    consentHost: string;
    clientId: string;
    inviteId: string;
    inviteSecret: string;
    scopes: string[];
    redirectUri?: string;
    state?: string;
  }): string {
    const authUrl = new URL(
      `/o/${encodeURIComponent(params.inviteId)}`,
      normalizeConsentHost(params.consentHost),
    );
    authUrl.searchParams.set("client_id", params.clientId);
    authUrl.searchParams.set("response_type", "code");
    authUrl.searchParams.set("scope", normalizeScopes(params.scopes).join(" "));
    if (params.redirectUri) {
      authUrl.searchParams.set("redirect_uri", params.redirectUri);
    }
    if (params.state) authUrl.searchParams.set("state", params.state);
    authUrl.hash = `s=${encodeURIComponent(params.inviteSecret)}`;
    return authUrl.toString();
  }

  private generateGroupInviteLink(params: {
    consentHost: string;
    clientId: string;
    scopes: string[];
    state?: string;
    redirectUri?: string;
    batchId?: string;
    masterSecret: string;
  }): string {
    if (params.batchId && !params.redirectUri && !params.state) {
      const authUrl = new URL(
        `/b/${encodeURIComponent(params.batchId)}`,
        normalizeConsentHost(params.consentHost),
      );
      authUrl.hash = `s=${encodeURIComponent(params.masterSecret)}`;
      return authUrl.toString();
    }

    const authUrl = new URL(
      "/oauth2/authorize",
      normalizeConsentHost(params.consentHost),
    );
    authUrl.searchParams.set("client_id", params.clientId);
    authUrl.searchParams.set("response_type", "code");
    authUrl.searchParams.set("scope", normalizeScopes(params.scopes).join(" "));
    if (params.redirectUri) {
      authUrl.searchParams.set("redirect_uri", params.redirectUri);
    }
    if (params.state) authUrl.searchParams.set("state", params.state);
    if (params.batchId) authUrl.searchParams.set("batchId", params.batchId);
    authUrl.searchParams.set("batch_link_secret", params.masterSecret);
    return authUrl.toString();
  }

  private requireClientId(): string {
    if (!this.config.clientId) {
      throw new Error("PviumSdkConfig.clientId is required for invite methods");
    }
    return this.config.clientId;
  }

  private requireConsentHost(): string {
    return resolvePviumConsentHost(this.config);
  }
}
