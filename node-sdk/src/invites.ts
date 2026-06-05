import { Wallet } from "ethers";
import {
  PviumHttpClient,
  PviumSdkConfig,
  resolvePviumConsentHost,
} from "./client";
import { RequestOptions } from "./types";
import {
  buildInviteMasterSecretMessage,
  createInviteNonce,
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
  scopes?: string[];
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
  ): Promise<unknown> {
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

    return this.http.parseResponseBody<unknown>(response);
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

  private generateGroupInviteLink(params: {
    consentHost: string;
    clientId: string;
    scopes: string[];
    state?: string;
    redirectUri?: string;
    batchId?: string;
    masterSecret: string;
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
