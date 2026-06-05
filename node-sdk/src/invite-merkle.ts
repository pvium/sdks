import crypto from "crypto-js";
import { verifyMessage } from "ethers";
import keccak256 from "keccak256";
import { MerkleTree } from "merkletreejs";

export interface BatchInviteMerkleInput {
  appClientId: string;
  batchId?: string;
  chain?: string;
  scopes: string[];
  invites: Array<{
    inviteId?: string;
    email: string;
    inviteNonce?: string;
    expiresAt?: string | Date;
    defaultPayoutAmount?: number;
  }>;
  createdAt?: number;
  rootNonce?: string;
}

export interface BatchInviteMerkleInvite {
  inviteId?: string;
  email: string;
  inviteNonce: string;
  loginHintHash: string;
  emailCommitment: string;
  defaultPayoutAmount?: number;
  expiresAt: number;
  leaf: string;
  leafMessage: string;
  proof: string[];
}

export interface BatchInviteMerkleData {
  version: "1";
  appClientId: string;
  batchId: string;
  chain?: string;
  scopes: string[];
  root: string;
  rootNonce: string;
  inviteCount: number;
  createdAt: number;
  expiresAt: number;
  signatureMessage: string;
  invites: BatchInviteMerkleInvite[];
}

export interface BatchInviteProofVerificationInput {
  appClientId: string;
  batchId?: string;
  email: string;
  inviteNonce: string;
  loginHintHash: string;
  emailCommitment?: string;
  defaultPayoutAmount?: number;
  expiresAt?: string | Date | number;
  leaf: string;
  proof: string[];
  root: string;
  signatureMessage?: string;
  signature?: string;
  signatureType?: string;
  signerAddress?: string;
}

export interface BatchInviteProofVerificationResult {
  valid: boolean;
  leaf: string;
  leafMessage: string;
  loginHintHash: string;
  emailCommitment: string;
  proofValid: boolean;
  signatureValid?: boolean;
  recoveredSigner?: string;
  errors: string[];
}

const normalizeScopes = (scopes: string[]): string[] => {
  return Array.from(
    new Set(scopes.map((scope) => scope.trim()).filter(Boolean)),
  ).sort((a, b) => a.localeCompare(b));
};

const DEFAULT_INVITE_TTL_SECONDS = 7 * 24 * 60 * 60;

const sha256 = (value: string): string => {
  return crypto.SHA256(value).toString();
};

const randomHex = (bytes: number): string => {
  const randomBytes = new Uint8Array(bytes);
  globalThis.crypto.getRandomValues(randomBytes);

  return Array.from(randomBytes, (value) =>
    value.toString(16).padStart(2, "0"),
  ).join("");
};

const toUnixSeconds = (value?: string | Date | number): number => {
  if (!value) return 0;
  if (typeof value === "number") return value;

  return Math.floor(new Date(value).getTime() / 1000);
};

const normalizeAmount = (value?: number): string => {
  if (value === undefined || value === null) return "";

  return Number(value).toString();
};

const buildEmailCommitment = (
  batchId: string | undefined,
  email: string,
  inviteNonce: string,
): string => {
  return sha256(
    [
      "payy.invite.email.v1",
      batchId || "",
      email.trim().toLowerCase(),
      inviteNonce,
    ].join(":"),
  );
};

const buildLoginHintHash = (email: string, inviteNonce: string): string => {
  return sha256(`${email.trim().toLowerCase()}:${inviteNonce}`).substring(
    0,
    12,
  );
};

const buildLeafMessage = (params: {
  appClientId: string;
  batchId: string;
  emailCommitment: string;
  inviteNonce: string;
  loginHintHash: string;
  defaultPayoutAmount?: number;
  expiresAt: number;
}): string => {
  return [
    'PVIUM_INVITE_LEAF_V1',
    `appClientId=${params.appClientId}`,
    `batchId=${params.batchId}`,
    `emailCommitment=${params.emailCommitment}`,
    `inviteNonce=${params.inviteNonce}`,
    `loginHintHash=${params.loginHintHash}`,
    `defaultPayoutAmount=${normalizeAmount(params.defaultPayoutAmount)}`,
    `expiresAt=${params.expiresAt}`,
  ].join('\n');
};

const buildRootMessage = (params: {
  version: string;
  appClientId: string;
  batchId: string;
  root: string;
  rootNonce: string;
  scopes: string[];
  createdAt: number;
  expiresAt: number;
}): string => {
  return [
    'PVIUM_INVITE_ROOT_V1',
    `version=${params.version}`,
    `appClientId=${params.appClientId}`,
    `batchId=${params.batchId}`,
    `root=${params.root}`,
    `rootNonce=${params.rootNonce}`,
    `scopes=${params.scopes.join(' ')}`,
    `createdAt=${params.createdAt}`,
    `expiresAt=${params.expiresAt}`,
  ].join('\n');
};

/**
 * Derive a rootNonce from (batchId, scopes, salt). The previous version used
 * `createdAt` (unix seconds) as the third input, which made the nonce
 * predictable and could collide for app-level invites (no batchId) created
 * in the same second with identical scopes. Replacing the timestamp with a
 * random 16-byte salt preserves the derivation shape — useful if we ever
 * want to re-verify the nonce — while pushing collision probability down to
 * 2⁻¹²⁸ and removing the public pre-image of the nonce.
 */
export const createRootNonce = (
  batchId: string | undefined,
  scopes: string[],
  salt?: string,
): string => {
  const rootSalt = salt || randomHex(16);
  return sha256(
    [
      "payy.invite.root.v1",
      batchId || "",
      scopes.join(" "),
      rootSalt,
    ].join(":"),
  );
};

export const createInviteNonce = (): string => randomHex(16);

// ============================================================================
// V2: typed identity + per-invite hashed secret
// ============================================================================

export type InviteIdentityType =
  | 'email'
  | 'handle'
  | 'wallet'
  | 'x'
  | 'github'
  | 'twitter'
  | 'discord'
  | 'telegram';

export const SUPPORTED_INVITE_IDENTITY_TYPES: InviteIdentityType[] = [
  'email',
  'handle',
  'wallet',
  'x',
  'github',
  'twitter',
  'discord',
  'telegram',
];

/**
 * Best-effort type detection for a raw identity string. Returns null when the
 * input is empty or doesn't match any supported shape — callers should surface
 * a dropdown in that case so the user picks a type explicitly. The Solana /
 * handle overlap (32-char base58 that is also alphanumeric) resolves as
 * "wallet" because Solana has the stricter length lower bound, but the UI
 * should still render the dropdown so the user can override.
 */
export const detectInviteIdentityType = (
  raw: string,
): { type: InviteIdentityType; ambiguous: boolean } | null => {
  const trimmed = (raw ?? "").trim();
  if (!trimmed) return null;

  // Email: contains `@` not at the start, followed by a `.` after the @
  if (EMAIL_RE.test(trimmed)) {
    return { type: "email", ambiguous: false };
  }

  // Explicit handle prefix
  if (trimmed.startsWith("@")) {
    const rest = trimmed.slice(1).toLowerCase();
    if (HANDLE_RE.test(rest)) {
      return { type: "handle", ambiguous: false };
    }
  }

  // EVM wallet
  if (EVM_ADDRESS_RE.test(trimmed)) {
    return { type: "wallet", ambiguous: false };
  }

  // Solana wallet — base58, 32-44 chars, case-sensitive
  if (SOLANA_ADDRESS_RE.test(trimmed)) {
    // Ambiguous with handle when length is exactly 32 and lowercase happens
    // to also satisfy the handle regex.
    const lowered = trimmed.toLowerCase();
    const alsoValidHandle =
      trimmed.length <= 32 && HANDLE_RE.test(lowered);
    return { type: "wallet", ambiguous: alsoValidHandle };
  }

  // Handle fallback (short alphanumeric with dot/dash/underscore)
  if (HANDLE_RE.test(trimmed.toLowerCase())) {
    return { type: "handle", ambiguous: false };
  }

  return null;
};

export interface BatchInviteMerkleInputV2 {
  appClientId: string;
  batchId?: string;
  chain?: string;
  scopes: string[];
  invites: Array<{
    inviteId?: string;
    identityType: InviteIdentityType;
    identityValue: string;
    inviteNonce?: string;
    inviteSecret?: string;
    expiresAt?: string | Date;
    defaultPayoutAmount?: number;
  }>;
  createdAt?: number;
  rootNonce?: string;
}

export interface BatchInviteMerkleInviteV2 {
  inviteId?: string;
  identityType: InviteIdentityType;
  identityValue: string;
  identityValueRaw: string;
  identityCommitment: string;
  inviteNonce: string;
  inviteSecret: string;
  secretHash: string;
  defaultPayoutAmount?: number;
  expiresAt: number;
  leaf: string;
  leafMessage: string;
  proof: string[];
}

export interface BatchInviteMerkleDataV2 {
  version: "2";
  appClientId: string;
  batchId: string;
  chain?: string;
  scopes: string[];
  root: string;
  rootNonce: string;
  inviteCount: number;
  createdAt: number;
  expiresAt: number;
  signatureMessage: string;
  invites: BatchInviteMerkleInviteV2[];
}

export interface BatchInviteProofVerificationInputV2 {
  appClientId: string;
  batchId?: string;
  identityType: InviteIdentityType;
  identityValue: string;
  inviteNonce: string;
  inviteSecret: string;
  identityCommitment?: string;
  secretHash?: string;
  defaultPayoutAmount?: number;
  expiresAt?: string | Date | number;
  leaf: string;
  proof: string[];
  root: string;
  signatureMessage?: string;
  signature?: string;
  signatureType?: string;
  signerAddress?: string;
}

export interface BatchInviteProofVerificationResultV2 {
  valid: boolean;
  leaf: string;
  leafMessage: string;
  identityCommitment: string;
  secretHash: string;
  proofValid: boolean;
  signatureValid?: boolean;
  recoveredSigner?: string;
  errors: string[];
}

const EVM_ADDRESS_RE = /^0x[0-9a-fA-F]{40}$/;
const EVM_ADDRESS_LOWER_RE = /^0x[0-9a-f]{40}$/;
const SOLANA_ADDRESS_RE = /^[1-9A-HJ-NP-Za-km-z]{32,44}$/;
const HANDLE_RE = /^[a-z0-9](?:[a-z0-9._-]{0,30}[a-z0-9])?$/;
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export const normalizeIdentityValue = (
  type: InviteIdentityType,
  value: string,
): string => {
  const trimmed = (value ?? "").trim();
  switch (type) {
    case 'email':
      return trimmed.toLowerCase();
    case 'handle':
      return trimmed.toLowerCase().replace(/^@/, '');
    case 'wallet':
      return EVM_ADDRESS_RE.test(trimmed) ? trimmed.toLowerCase() : trimmed;
    case 'x':
    case 'twitter':
    case 'github':
    case 'discord':
    case 'telegram':
      return trimmed.toLowerCase().replace(/^@/, '');
    default:
      throw new Error(`Unsupported identity type: ${type}`);
  }
};

export const validateIdentityValue = (
  type: InviteIdentityType,
  value: string,
): string | null => {
  if (!value || !value.trim()) return "Identity value is required";
  if (!SUPPORTED_INVITE_IDENTITY_TYPES.includes(type)) {
    return `Identity type not yet supported: ${type}`;
  }

  const normalized = normalizeIdentityValue(type, value);
  switch (type) {
    case 'email':
      return EMAIL_RE.test(normalized) ? null : 'Invalid email format';
    case 'handle':
      return HANDLE_RE.test(normalized) ? null : 'Invalid handle format';
    case 'wallet':
      if (
        EVM_ADDRESS_LOWER_RE.test(normalized) ||
        SOLANA_ADDRESS_RE.test(normalized)
      ) {
        return null;
      }
      return 'Invalid wallet address format';
    case 'github':
    case 'x':
    case 'twitter':
    case 'discord':
    case 'telegram':
      return HANDLE_RE.test(normalized)
        ? null
        : `Invalid ${type} handle format`;
    default:
      return `Identity type not yet supported: ${type}`;
  }
};

export const createInviteSecret = (): string => randomHex(32);

export const buildSecretHash = (inviteSecret: string): string => {
  return sha256(inviteSecret);
};

// Deterministic per-root master-secret derivation. The sender's root-signing
// wallet produces a signature over this canonical message; that signature is
// hashed to form a 32-byte master secret that never leaves the browser. Any
// device the same wallet later signs in on can re-derive the same master
// secret — so we don't need to persist plaintext invite secrets to IDB.
export const INVITE_SECRET_DOMAIN_V2 = "PVIUM_INVITE_SECRET_V2";

export const buildInviteMasterSecretMessage = (rootNonce: string): string => {
  return `${INVITE_SECRET_DOMAIN_V2}:${rootNonce}`;
};

/**
 * Hash the raw signature bytes (hex-encoded, `0x` prefix optional) to produce
 * the master secret. sha256 both fixes the length to 32 bytes and hides the
 * raw signature — the master secret is what ends up mixed per invite.
 */
export const deriveMasterSecret = (rawSignatureHex: string): string => {
  const normalized = rawSignatureHex.replace(/^0x/, "").toLowerCase();
  if (!normalized) throw new Error("Cannot derive master secret from empty signature");
  return sha256(normalized);
};

/**
 * Per-invite secret derived from the master secret and the invite's public
 * nonce. This is the plaintext value placed in the invite URL; the server
 * only ever sees sha256(inviteSecret) in the signed Merkle leaf.
 */
export const deriveInviteSecret = (
  masterSecret: string,
  inviteNonce: string,
): string => {
  return sha256(`${masterSecret}:${inviteNonce}`);
};

export const buildIdentityCommitment = (
  type: InviteIdentityType,
  value: string,
  inviteNonce: string,
): string => {
  console.log('BUILDINGCommitment', type, value, inviteNonce);
  return sha256(
    [
      'pvium.invite.identity.v2',
      type,
      normalizeIdentityValue(type, value),
      inviteNonce,
    ].join(':'),
  );
};

const buildLeafMessageV2 = (params: {
  appClientId: string;
  batchId: string;
  identityType: InviteIdentityType;
  identityCommitment: string;
  inviteNonce: string;
  secretHash: string;
  defaultPayoutAmount?: number;
  expiresAt: number;
}): string => {
  return [
    'PVIUM_INVITE_LEAF_V2',
    `appClientId=${params.appClientId}`,
    `batchId=${params.batchId}`,
    `identityType=${params.identityType}`,
    `identityCommitment=${params.identityCommitment}`,
    `inviteNonce=${params.inviteNonce}`,
    `secretHash=${params.secretHash}`,
    `defaultPayoutAmount=${normalizeAmount(params.defaultPayoutAmount)}`,
    `expiresAt=${params.expiresAt}`,
  ].join('\n');
};

const buildRootMessageV2 = (params: {
  appClientId: string;
  batchId: string;
  root: string;
  rootNonce: string;
  scopes: string[];
  createdAt: number;
  expiresAt: number;
}): string => {
  return [
    'PVIUM_INVITE_ROOT_V2',
    'version=2',
    `appClientId=${params.appClientId}`,
    `batchId=${params.batchId}`,
    `root=${params.root}`,
    `rootNonce=${params.rootNonce}`,
    `scopes=${params.scopes.join(' ')}`,
    `createdAt=${params.createdAt}`,
    `expiresAt=${params.expiresAt}`,
  ].join('\n');
};

export const generateBatchInviteMerkleDataV2 = (
  input: BatchInviteMerkleInputV2,
): BatchInviteMerkleDataV2 => {
  if (!input.invites.length) {
    throw new Error("Cannot generate invite Merkle data without invites");
  }

  for (const invite of input.invites) {
    const err = validateIdentityValue(invite.identityType, invite.identityValue);
    if (err) {
      throw new Error(
        `Invalid invite identity (${invite.identityType}=${invite.identityValue}): ${err}`,
      );
    }
  }

  const scopes = normalizeScopes(input.scopes);
  const batchId = input.batchId || "";
  const createdAt = input.createdAt || Math.floor(Date.now() / 1000);
  const rootNonce =
    input.rootNonce || createRootNonce(batchId, scopes);

  const invitesWithoutProofs = input.invites.map((invite) => {
    const inviteNonce = invite.inviteNonce || createInviteNonce();
    const inviteSecret = invite.inviteSecret || createInviteSecret();
    const secretHash = buildSecretHash(inviteSecret);
    const identityValue = normalizeIdentityValue(
      invite.identityType,
      invite.identityValue,
    );
    const identityCommitment = buildIdentityCommitment(
      invite.identityType,
      identityValue,
      inviteNonce,
    );
    const expiresAt =
      toUnixSeconds(invite.expiresAt) || createdAt + DEFAULT_INVITE_TTL_SECONDS;
    const leafMessage = buildLeafMessageV2({
      appClientId: input.appClientId,
      batchId,
      identityType: invite.identityType,
      identityCommitment,
      inviteNonce,
      secretHash,
      defaultPayoutAmount: invite.defaultPayoutAmount,
      expiresAt,
    });
    const leafBuffer = keccak256(Buffer.from(leafMessage, "utf8"));

    return {
      inviteId: invite.inviteId,
      identityType: invite.identityType,
      identityValue,
      identityValueRaw: invite.identityValue,
      identityCommitment,
      inviteNonce,
      inviteSecret,
      secretHash,
      defaultPayoutAmount: invite.defaultPayoutAmount,
      expiresAt,
      leaf: `0x${leafBuffer.toString("hex")}`,
      leafBuffer,
      leafMessage,
    };
  });

  const tree = new MerkleTree(
    invitesWithoutProofs.map((invite) => invite.leafBuffer),
    keccak256,
    { sortPairs: true },
  );
  const root = tree.getHexRoot();
  const expiresAt = Math.max(
    ...invitesWithoutProofs.map((invite) => invite.expiresAt),
  );

  const invites = invitesWithoutProofs.map(({ leafBuffer, ...invite }) => ({
    ...invite,
    proof: tree.getHexProof(leafBuffer),
  }));

  const signatureMessage = buildRootMessageV2({
    appClientId: input.appClientId,
    batchId,
    root,
    rootNonce,
    scopes,
    createdAt,
    expiresAt,
  });

  return {
    version: "2",
    appClientId: input.appClientId,
    batchId,
    chain: input.chain,
    scopes,
    root,
    rootNonce,
    inviteCount: invites.length,
    createdAt,
    expiresAt,
    signatureMessage,
    invites,
  };
};

export const verifyBatchInviteProofV2 = (
  input: BatchInviteProofVerificationInputV2,
): BatchInviteProofVerificationResultV2 => {
  const errors: string[] = [];
  const batchId = input.batchId || "";

  const identityErr = validateIdentityValue(
    input.identityType,
    input.identityValue,
  );
  if (identityErr) errors.push(identityErr);

  const identityCommitment = buildIdentityCommitment(
    input.identityType,
    input.identityValue,
    input.inviteNonce,
  );
  const secretHash = buildSecretHash(input.inviteSecret);
  const expiresAt = toUnixSeconds(input.expiresAt);
  const leafMessage = buildLeafMessageV2({
    appClientId: input.appClientId,
    batchId,
    identityType: input.identityType,
    identityCommitment,
    inviteNonce: input.inviteNonce,
    secretHash,
    defaultPayoutAmount: input.defaultPayoutAmount,
    expiresAt,
  });
  const leafBuffer = keccak256(Buffer.from(leafMessage, "utf8"));
  const leaf = `0x${leafBuffer.toString("hex")}`;

  if (
    input.identityCommitment &&
    input.identityCommitment.toLowerCase() !== identityCommitment.toLowerCase()
  ) {
    errors.push("Identity commitment does not match signed-in user");
  }

  if (
    input.secretHash &&
    input.secretHash.toLowerCase() !== secretHash.toLowerCase()
  ) {
    errors.push("Secret hash does not match provided invite secret");
  }

  if (input.leaf.toLowerCase() !== leaf.toLowerCase()) {
    errors.push("Invite leaf does not match invite data");
  }

  const tree = new MerkleTree([], keccak256, { sortPairs: true });
  const proofValid = tree.verify(input.proof || [], leafBuffer, input.root);

  if (!proofValid) {
    errors.push("Invite proof is not in the Merkle root");
  }

  let signatureValid: boolean | undefined;
  let recoveredSigner: string | undefined;

  if (
    input.signatureType === "evm-personal-sign" &&
    input.signature &&
    input.signatureMessage
  ) {
    try {
      recoveredSigner = verifyMessage(input.signatureMessage, input.signature);
      signatureValid =
        !input.signerAddress ||
        recoveredSigner.toLowerCase() === input.signerAddress.toLowerCase();

      if (!signatureValid) {
        errors.push("Invite root signature signer does not match");
      }
    } catch (_error) {
      signatureValid = false;
      errors.push("Invite root signature is invalid");
    }
  }

  if (input.signatureMessage && !input.signatureMessage.includes(input.root)) {
    errors.push("Invite root signature message does not contain root");
  }

  return {
    valid: errors.length === 0,
    leaf,
    leafMessage,
    identityCommitment,
    secretHash,
    proofValid,
    signatureValid,
    recoveredSigner,
    errors,
  };
};

// ============================================================================
// V1 (kept for backward compatibility — do not modify)
// ============================================================================

export const generateBatchInviteMerkleData = (
  input: BatchInviteMerkleInput,
): BatchInviteMerkleData => {
  if (!input.invites.length) {
    throw new Error("Cannot generate invite Merkle data without invites");
  }

  const scopes = normalizeScopes(input.scopes);
  const batchId = input.batchId || "";
  const createdAt = input.createdAt || Math.floor(Date.now() / 1000);
  const rootNonce =
    input.rootNonce || createRootNonce(batchId, scopes);

  const invitesWithoutProofs = input.invites.map((invite) => {
    const inviteNonce = invite.inviteNonce || createInviteNonce();
    const loginHintHash = buildLoginHintHash(invite.email, inviteNonce);
    const emailCommitment = buildEmailCommitment(
      input.batchId,
      invite.email,
      inviteNonce,
    );
    const expiresAt =
      toUnixSeconds(invite.expiresAt) || createdAt + DEFAULT_INVITE_TTL_SECONDS;
    const leafMessage = buildLeafMessage({
      appClientId: input.appClientId,
      batchId,
      emailCommitment,
      inviteNonce,
      loginHintHash,
      defaultPayoutAmount: invite.defaultPayoutAmount,
      expiresAt,
    });
    const leafBuffer = keccak256(Buffer.from(leafMessage, "utf8"));

    return {
      inviteId: invite.inviteId,
      email: invite.email,
      inviteNonce,
      loginHintHash,
      emailCommitment,
      defaultPayoutAmount: invite.defaultPayoutAmount,
      expiresAt,
      leaf: `0x${leafBuffer.toString("hex")}`,
      leafBuffer,
      leafMessage,
    };
  });

  const tree = new MerkleTree(
    invitesWithoutProofs.map((invite) => invite.leafBuffer),
    keccak256,
    { sortPairs: true },
  );
  const root = tree.getHexRoot();
  const expiresAt = Math.max(
    ...invitesWithoutProofs.map((invite) => invite.expiresAt),
  );

  const invites = invitesWithoutProofs.map(({ leafBuffer, ...invite }) => ({
    ...invite,
    proof: tree.getHexProof(leafBuffer),
  }));

  const signatureMessage = buildRootMessage({
    version: "1",
    appClientId: input.appClientId,
    batchId,
    root,
    rootNonce,
    scopes,
    createdAt,
    expiresAt,
  });

  return {
    version: "1",
    appClientId: input.appClientId,
    batchId,
    chain: input.chain,
    scopes,
    root,
    rootNonce,
    inviteCount: invites.length,
    createdAt,
    expiresAt,
    signatureMessage,
    invites,
  };
};

export const verifyBatchInviteProof = (
  input: BatchInviteProofVerificationInput,
): BatchInviteProofVerificationResult => {
  const errors: string[] = [];
  const batchId = input.batchId || "";
  const loginHintHash = buildLoginHintHash(input.email, input.inviteNonce);
  const emailCommitment = buildEmailCommitment(
    batchId,
    input.email,
    input.inviteNonce,
  );
  const expiresAt = toUnixSeconds(input.expiresAt);
  const leafMessage = buildLeafMessage({
    appClientId: input.appClientId,
    batchId,
    emailCommitment,
    inviteNonce: input.inviteNonce,
    loginHintHash,
    defaultPayoutAmount: input.defaultPayoutAmount,
    expiresAt,
  });
  const leafBuffer = keccak256(Buffer.from(leafMessage, "utf8"));
  const leaf = `0x${leafBuffer.toString("hex")}`;

  if (input.loginHintHash !== loginHintHash) {
    errors.push("Login hint hash does not match signed-in user");
  }

  if (
    input.emailCommitment &&
    input.emailCommitment.toLowerCase() !== emailCommitment.toLowerCase()
  ) {
    errors.push("Email commitment does not match signed-in user");
  }

  if (input.leaf.toLowerCase() !== leaf.toLowerCase()) {
    errors.push("Invite leaf does not match invite data");
  }

  const tree = new MerkleTree([], keccak256, { sortPairs: true });
  const proofValid = tree.verify(input.proof || [], leafBuffer, input.root);

  if (!proofValid) {
    errors.push("Invite proof is not in the Merkle root");
  }

  let signatureValid: boolean | undefined;
  let recoveredSigner: string | undefined;

  if (
    input.signatureType === "evm-personal-sign" &&
    input.signature &&
    input.signatureMessage
  ) {
    try {
      recoveredSigner = verifyMessage(input.signatureMessage, input.signature);
      signatureValid =
        !input.signerAddress ||
        recoveredSigner.toLowerCase() === input.signerAddress.toLowerCase();

      if (!signatureValid) {
        errors.push("Invite root signature signer does not match");
      }
    } catch (_error) {
      signatureValid = false;
      errors.push("Invite root signature is invalid");
    }
  }

  if (input.signatureMessage && !input.signatureMessage.includes(input.root)) {
    errors.push("Invite root signature message does not contain root");
  }

  return {
    valid: errors.length === 0,
    leaf,
    leafMessage,
    loginHintHash,
    emailCommitment,
    proofValid,
    signatureValid,
    recoveredSigner,
    errors,
  };
};
