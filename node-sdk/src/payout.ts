import {
  AbiCoder,
  Wallet,
  getAddress,
  getBytes,
  hexlify,
  keccak256,
  parseUnits,
  solidityPacked,
} from 'ethers';
import { MerkleTree } from "merkletreejs";
import {
  PviumHttpClient,
  PviumSdkConfig,
  resolvePviumConsentHost,
} from "./client";
import { ApiMeta, RequestOptions } from "./types";

export type PayoutType = "Instant" | "Scheduled" | "Milestone" | "Escrow";
export type PayoutComplianceMode = "Open" | "Strict";
export type HexString = `0x${string}`;
export type PayoutChain =
  | 'base'
  | 'bsc'
  | 'solana'
  | 'base-testnet'
  | 'solana-testnet'
  | 'localhost';

export enum PayoutCurrency {
  USDC = 'USDC',
  USDT = 'USDT',
}

export type PayoutCurrencyInput =
  | PayoutCurrency
  | 'usdc'
  | 'usdt'
  | 'USDC'
  | 'USDT';

type SupportedPayoutCurrency = {
  contractAddress: string;
  decimals: number;
};

export interface PayoutPayment {
  receiver: string;
  amount: number | string;
  token?: string;
  tokenSymbol?: PayoutCurrencyInput;
  decimals?: number;
  memo?: string;
  publicId?: string;
  claimDate?: number;
  claimEnd?: number;
}

export type PayoutPaymentWithoutToken = Omit<
  PayoutPayment,
  'token' | 'tokenSymbol'
> & {
  token?: never;
  tokenSymbol?: never;
};

interface CreatePayoutBaseInput {
  id?: string;
  chain: PayoutChain;
  nonce?: string;
  escrowBatch?: string | PayoutRecord;
  lockDuration?: number;
  label?: string;
  name?: string;
  description?: string;
  complianceMode?: PayoutComplianceMode;
  scheduleDate?: number;
  metadata?: Record<string, unknown>;
}

export type CreatePayoutInput =
  | (CreatePayoutBaseInput & {
      type: 'Scheduled' | 'Milestone' | 'Escrow';
      payoutCurrency: PayoutCurrencyInput;
      payments?: PayoutPaymentWithoutToken[];
    })
  | (CreatePayoutBaseInput & {
      type?: PayoutType;
      payoutCurrency?: undefined;
      payments?: PayoutPayment[];
    })
  | (CreatePayoutBaseInput & {
      type?: 'Instant';
      payoutCurrency?: PayoutCurrencyInput;
      payments?: PayoutPayment[];
    });

export interface AddPayoutPaymentsInput {
  payments: PayoutPayment[];
  /**
   * Required when adding payments to an escrow payout. The SDK uses this to
   * create and finalize the linked scheduled child payout in one call.
   */
  signer?: PayoutSignerInput;
  finalizeOptions?: PayoutFinalizeOptions & {
    id?: string;
    name?: string;
    description?: string;
    metadata?: Record<string, unknown>;
  };
  requestOptions?: RequestOptions;
}

export interface PayoutRecipient {
  identityType: string;
  identityValue: string;
  defaultPayoutAmount?: number;
  memo?: string;
}

export interface ResolvePayoutRecipient {
  identityType: string;
  identityValue: string;
}

export interface AddPayoutRecipientsInput {
  recipients: PayoutRecipient[];
}

export interface ResolvePayoutRecipientsInput {
  recipients: ResolvePayoutRecipient[];
}

export interface RemovePayoutPaymentsInput {
  paymentIds: Array<number | string>;
}

export interface UpdatePayoutPaymentInput {
  amount?: number;
  memo?: string;
  claimDate?: number;
}

export interface PayoutPaymentsListQuery {
  page?: number;
  perPage?: number;
  limit?: number;
}

export interface PayoutRecipientResult {
  identity?: string;
  identityType?: string;
  identityValue?: string;
  userId?: string;
  email?: string;
  handle?: string;
  ethereumWallet?: string;
  solanaWallet?: string;
  receiver?: string;
  [key: string]: unknown;
}

export interface PayoutRecipientError {
  identity?: string;
  identityType?: string;
  identityValue?: string;
  reason?: string;
  [key: string]: unknown;
}

export interface AddPayoutRecipientsResult {
  added: PayoutRecipientResult[];
  errors: PayoutRecipientError[];
}

export interface ResolvePayoutRecipientsResult {
  resolved: PayoutRecipientResult[];
  errors: PayoutRecipientError[];
}

export interface PayoutListQuery {
  page?: number;
  limit?: number;
  paymentType?: "Instant" | "Scheduled" | "Escrow";
  isCommitment?: boolean;
  status?: string;
}

export interface PayoutFinalizeOptions {
  clientId?: string;
  chain?: string;
  chainId?: number;
  escrowBatch?: string | PayoutRecord;
  fundingToken?: string;
  payments?: PayoutPayment[];
  complianceMode?: PayoutComplianceMode;
  gracePeriod?: number;
  disapprovalDeadline?: number;
  claimDate?: number;
  lockDuration?: number;
  timestamp?: number;
  signerAddress?: string;
}

export interface PayoutMessageSignature {
  signature: string | Uint8Array;
  signatureType?: "evm-personal-sign" | "solana-message" | string;
  signerAddress?: string;
}

export interface PayoutProviderSigner {
  chain?: "ethereum" | "solana" | string;
  address?: string;
  getAddress?: () => Promise<string> | string;
  signMessage?: (
    message: string | Uint8Array,
  ) => Promise<string | Uint8Array | PayoutMessageSignature> | string | Uint8Array | PayoutMessageSignature;
  signFinalize?: (
    message: string | Uint8Array,
  ) => Promise<string | Uint8Array | PayoutMessageSignature> | string | Uint8Array | PayoutMessageSignature;
  signFunding?: (
    digest: HexString,
  ) => Promise<string | PayoutMessageSignature> | string | PayoutMessageSignature;
  signDigest?: (
    digest: HexString,
  ) => Promise<string | PayoutMessageSignature> | string | PayoutMessageSignature;
  request?: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
  signerAddress?: string;
}

export type PayoutSignatureResult = {
  signature: string;
  signerAddress?: string;
};
export type PayoutSigningKeyNetworkType = 'ethereum' | 'solana';
export type PayoutMessageSigner = (
  message: string,
) => Promise<string | PayoutMessageSignature> | string | PayoutMessageSignature;
export type PayoutSignerInput =
  | string
  | {
      chain: "ethereum";
      privateKey: string;
    }
  | PayoutProviderSigner
  | PayoutMessageSigner;

export interface PayoutSigningKeyAuthorizationData {
  transactionMax: number | string | bigint;
  totalMax: number | string | bigint;
  expiration: number | string | bigint;
  timestamp?: number | string | bigint;
}

export interface PayoutSigningKeyAuthorization {
  batchHash: HexString;
  signingKey: string;
  networkType: PayoutSigningKeyNetworkType;
  transactionMax: string | bigint;
  totalMax: string | bigint;
  expiration: string | bigint;
  timestamp: string | bigint;
  authMessageHash: HexString;
  signature: string;
}

export type PayoutSigningKeySignatureResult =
  | string
  | Uint8Array
  | { signature: string | Uint8Array };

export type PayoutSigningKeySignerInput =
  | string
  | {
      privateKey: string;
    }
  | {
      signAuthorization?: (
        digest: HexString,
      ) =>
        | Promise<PayoutSigningKeySignatureResult>
        | PayoutSigningKeySignatureResult;
      signDigest?: (
        digest: HexString,
      ) =>
        | Promise<PayoutSigningKeySignatureResult>
        | PayoutSigningKeySignatureResult;
      request?: (args: {
        method: string;
        params?: unknown[];
      }) => Promise<unknown>;
    }
  | ((
      digest: HexString,
    ) =>
      | Promise<PayoutSigningKeySignatureResult>
      | PayoutSigningKeySignatureResult);

export interface PayoutApiResponse<T> {
  meta: ApiMeta;
  data: T;
}

export type PayoutReference = string | PayoutRecord | PayoutIntent;

export interface PayoutRecord {
  id: string;
  chain: string;
  paymentType: "Instant" | "Scheduled" | "Escrow";
  escrowBatch?: string;
  complianceMode?: PayoutComplianceMode;
  isCommitment?: boolean;
  status?: string;
  nonce?: string;
  batchDataHash?: string;
  batchHash?: string;
  merkleRoot?: string;
  batchSignature?: string;
  fundingSignature?: string;
  lockDuration?: number;
  metadata?: Record<string, any>;
  payments?: PayoutPayment[];
  app?: {
    _id?: string;
    clientId?: string;
    name?: string;
    [key: string]: unknown;
  } | null;
  [key: string]: unknown;
}

export interface PayoutFundingCall {
  chainId?: number;
  to?: string;
  data?: HexString;
  value?: string;
  functionName?: string;
  abi?: unknown[];
  args?: unknown[];
}

export interface FinalizePayoutData {
  payout: PayoutRecord;
  fundingUrl?: string;
  fundingCall?: PayoutFundingCall;
  batchDataHash: HexString;
  batchHash?: HexString;
  merkleRoot?: HexString;
}

export class PayoutIntent implements PayoutRecord {
  [key: string]: unknown;

  readonly meta!: ApiMeta;
  readonly data!: PayoutRecord;
  private readonly service!: PviumPayoutService;

  id!: string;
  chain!: string;
  paymentType!: "Instant" | "Scheduled" | "Escrow";
  escrowBatch?: string;
  complianceMode?: PayoutComplianceMode;
  isCommitment?: boolean;
  status?: string;
  nonce?: string;
  batchDataHash?: string;
  batchHash?: string;
  merkleRoot?: string;
  batchSignature?: string;
  fundingSignature?: string;
  lockDuration?: number;
  metadata?: Record<string, any>;
  payments?: PayoutPayment[];
  app?: {
    _id?: string;
    clientId?: string;
    name?: string;
    [key: string]: unknown;
  } | null;

  constructor(
    service: PviumPayoutService,
    meta: ApiMeta,
    data: PayoutRecord,
  ) {
    Object.assign(this, data);
    Object.defineProperty(this, 'service', {
      value: service,
      enumerable: false,
    });
    Object.defineProperty(this, 'meta', {
      value: meta,
      enumerable: true,
    });
    Object.defineProperty(this, 'data', {
      value: data,
      enumerable: true,
    });
  }

  async finalize(
    signer: PayoutSignerInput,
    options: PayoutFinalizeOptions = {},
    requestOptions?: RequestOptions,
  ): Promise<PayoutFinalization> {
    return this.service.finalize(this.data, signer, options, requestOptions);
  }

  async addPayments(
    input: AddPayoutPaymentsInput | PayoutPayment[],
    options?: RequestOptions,
  ): Promise<PayoutIntent | PayoutFinalization> {
    return this.service.addPayments(this.data, input, options);
  }

  async addRecipients(
    input: AddPayoutRecipientsInput | PayoutRecipient[],
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<AddPayoutRecipientsResult>> {
    return this.service.addRecipients(this.id, input, options);
  }

  async resolveRecipients(
    input: ResolvePayoutRecipientsInput | ResolvePayoutRecipient[],
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<ResolvePayoutRecipientsResult>> {
    return this.service.resolveRecipients(this.id, input, options);
  }

  async removePayments(
    input: RemovePayoutPaymentsInput | Array<number | string>,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<undefined>> {
    return this.service.removePayments(this.id, input, options);
  }

  async deletePayment(
    paymentId: number | string,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<undefined>> {
    return this.service.deletePayment(this.id, paymentId, options);
  }

  async updatePayment(
    paymentId: number | string,
    input: UpdatePayoutPaymentInput,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<unknown>> {
    return this.service.updatePayment(this.id, paymentId, input, options);
  }

  async editPayment(
    paymentId: number | string,
    input: UpdatePayoutPaymentInput,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<unknown>> {
    return this.service.editPayment(this.id, paymentId, input, options);
  }

  async revokeInvite(
    inviteId: string,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<undefined>> {
    return this.service.revokeInvite(this.id, inviteId, options);
  }

  async revokeInviteRoot(
    inviteRootId: string,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<undefined>> {
    return this.service.revokeInviteRoot(this.id, inviteRootId, options);
  }

  async delete(options?: RequestOptions): Promise<PayoutApiResponse<undefined>> {
    return this.service.delete(this.id, options);
  }

  async listInvites(options?: RequestOptions): Promise<PayoutApiResponse<unknown[]>> {
    return this.service.listInvites(this.id, options);
  }

  async listPayments(
    query?: PayoutPaymentsListQuery,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<unknown[]>> {
    return this.service.listPayments(this.id, query, options);
  }
}

export class PayoutFinalization implements PayoutApiResponse<FinalizePayoutData> {
  readonly meta: ApiMeta;
  readonly data: FinalizePayoutData;
  readonly payout: PayoutIntent;
  readonly fundingUrl?: string;
  readonly fundingCall?: PayoutFundingCall;
  readonly batchDataHash: HexString;
  readonly batchHash?: HexString;
  readonly merkleRoot?: HexString;

  constructor(
    service: PviumPayoutService,
    meta: ApiMeta,
    data: FinalizePayoutData,
  ) {
    this.meta = meta;
    this.data = data;
    this.payout = new PayoutIntent(service, meta, data.payout);
    this.fundingUrl = data.fundingUrl;
    this.fundingCall = data.fundingCall;
    this.batchDataHash = data.batchDataHash;
    this.batchHash = data.batchHash;
    this.merkleRoot = data.merkleRoot;
  }
}

interface MerkleProofData {
  receiver: string;
  proof: string[];
  leaf: string;
}

interface PaymentEntry {
  receiverAddress: string;
  amount: string;
  claimableDate: number;
  memo: string;
}

const ZERO_HEX = /^0x[0-9a-fA-F]*$/;

export function createPayoutNonce(): HexString {
  const cryptoImpl = globalThis.crypto;
  if (!cryptoImpl?.getRandomValues) {
    throw new Error("Secure random number generation is unavailable");
  }

  const bytes = new Uint8Array(16);
  cryptoImpl.getRandomValues(bytes);
  return `0x${Array.from(bytes, (byte) =>
    byte.toString(16).padStart(2, "0"),
  ).join("")}`;
}

function createPayoutId(): string {
  const cryptoImpl = globalThis.crypto;
  if (cryptoImpl?.randomUUID) {
    return cryptoImpl.randomUUID();
  }
  if (!cryptoImpl?.getRandomValues) {
    throw new Error("Secure random UUID generation is unavailable");
  }

  const bytes = new Uint8Array(16);
  cryptoImpl.getRandomValues(bytes);
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (byte) =>
    byte.toString(16).padStart(2, "0"),
  ).join("");

  return [
    hex.slice(0, 8),
    hex.slice(8, 12),
    hex.slice(12, 16),
    hex.slice(16, 20),
    hex.slice(20),
  ].join("-");
}

function mapPayoutType(type: PayoutType | undefined): {
  paymentType: "Instant" | "Scheduled" | "Escrow";
  isCommitment: boolean;
} {
  if (type === "Milestone") {
    return { paymentType: "Scheduled", isCommitment: true };
  }

  if (type === "Escrow") {
    return { paymentType: "Escrow", isCommitment: false };
  }

  if (type === "Scheduled") {
    return { paymentType: "Scheduled", isCommitment: false };
  }

  return { paymentType: "Instant", isCommitment: false };
}

function normalizeHexAddress(value: string): `0x${string}` {
  return (value.toLowerCase().startsWith("0x")
    ? value
    : `0x${value}`) as `0x${string}`;
}

function normalizeInstantNonce(nonce: string): `0x${string}` {
  const trimmed = nonce.trim();

  if (!trimmed) {
    throw new Error("Payout nonce is required");
  }

  const hexBody = trimmed.startsWith("0x") ? trimmed.slice(2) : trimmed;
  if (!ZERO_HEX.test(`0x${hexBody}`)) {
    throw new Error(`Payout nonce must be hex-compatible: ${nonce}`);
  }

  return `0x${hexBody.length % 2 === 0 ? hexBody : `0${hexBody}`}`;
}

export function generateInstantPayoutHash(
  payments: PayoutPayment[],
  nonce: string,
): HexString {
  const payouts = payments.map((payment) => {
    if (payment.decimals === undefined) {
      throw new Error("Payment decimals are required to hash instant payouts");
    }
    if (!payment.token) {
      throw new Error("Payment token is required to hash instant payouts");
    }

    return {
      receiver: normalizeHexAddress(payment.receiver),
      amount: parseUnits(payment.amount.toString(), payment.decimals),
      token: normalizeHexAddress(payment.token),
      memo: payment.memo || "",
    };
  });

  const encoded = AbiCoder.defaultAbiCoder().encode(
    [
      "bytes",
      "tuple(address receiver, uint256 amount, address token, string memo)[]",
    ],
    [normalizeInstantNonce(nonce), payouts],
  );

  return keccak256(encoded) as HexString;
}

function getPayoutIdBytes32(payoutId: string): HexString {
  const payoutIdHex = payoutId.replace(/-/g, "");
  return `0x${payoutIdHex.padEnd(64, "0")}` as HexString;
}

function normalizeBytes32Id(value: string, context: string): HexString {
  const trimmed = value.trim();
  const hexBody = trimmed.startsWith('0x')
    ? trimmed.slice(2)
    : trimmed.replace(/-/g, '');

  if (!hexBody || hexBody.length > 64 || !/^[0-9a-fA-F]+$/.test(hexBody)) {
    throw new Error(
      `${context} must be a bytes32 hex value or hex-compatible id`,
    );
  }

  return `0x${hexBody.padEnd(64, '0').toLowerCase()}` as HexString;
}

function normalizeAuthorizationUint(
  value: number | string | bigint,
  context: string,
): bigint {
  const normalized =
    typeof value === 'bigint'
      ? value
      : typeof value === 'number'
        ? BigInt(value)
        : BigInt(value);

  if (normalized < 0n) {
    throw new Error(`${context} must be greater than or equal to zero`);
  }

  return normalized;
}

export interface SigningKeyAuthorizationHash {
  normalizedInput: {
    batchHash: HexString;
    signingKey: string;
    transactionMax: bigint;
    totalMax: bigint;
    expiration: bigint;
    timestamp: bigint;
  };
  authMessageHash: HexString;
}

export function computeSigningKeyAuthorizationHash(params: {
  batchHash: string;
  signingKey: string;
  transactionMax: number | string | bigint;
  totalMax: number | string | bigint;
  expiration: number | string | bigint;
  timestamp: number | string | bigint;
}): SigningKeyAuthorizationHash {
  const batchHash = normalizeBytes32Id(params.batchHash, 'batchHash');
  const signingKey = getAddress(params.signingKey);
  const transactionMax = normalizeAuthorizationUint(
    params.transactionMax,
    'transactionMax',
  );
  const totalMax = normalizeAuthorizationUint(params.totalMax, 'totalMax');
  const expiration = normalizeAuthorizationUint(
    params.expiration,
    'expiration',
  );
  const timestamp = normalizeAuthorizationUint(params.timestamp, 'timestamp');
  const authMessageHash = keccak256(
    solidityPacked(
      ['bytes32', 'address', 'uint256', 'uint256', 'uint256', 'uint256'],
      [batchHash, signingKey, transactionMax, totalMax, expiration, timestamp],
    ),
  ) as HexString;

  return {
    normalizedInput: {
      batchHash,
      signingKey,
      transactionMax,
      totalMax,
      expiration,
      timestamp,
    },
    authMessageHash,
  };
}

export function computeScheduledPayoutHash(params: {
  payoutId: string;
  fundingToken: string;
  gracePeriod: number;
  disapprovalDeadline: number;
  timestamp: number;
  chainId: number;
}): HexString {
  const payoutIdBytes32 = getPayoutIdBytes32(params.payoutId);

  return keccak256(
    AbiCoder.defaultAbiCoder().encode(
      ["bytes32", "address", "uint256", "uint256", "uint256", "uint256"],
      [
        payoutIdBytes32,
        params.fundingToken,
        params.gracePeriod,
        params.disapprovalDeadline,
        params.timestamp,
        params.chainId,
      ],
    ),
  ) as HexString;
}

export function computeEscrowPayoutHash(params: {
  payoutId: string;
  fundingToken: string;
  lockDuration: number | bigint;
  timestamp: number;
  chainId: number;
}): HexString {
  const externalBatchId = getPayoutIdBytes32(params.payoutId);
  return keccak256(
    AbiCoder.defaultAbiCoder().encode(
      ["bytes32", "address", "uint256", "uint256", "uint256"],
      [
        externalBatchId,
        getAddress(params.fundingToken),
        BigInt(params.lockDuration),
        params.timestamp,
        params.chainId,
      ],
    ),
  ) as HexString;
}

export function computeEscrowFundingDigest(params: {
  escrowBatchHash: HexString;
  withdrawalWallet: string;
}): HexString {
  return keccak256(
    solidityPacked(
      ["bytes32", "address"],
      [params.escrowBatchHash, getAddress(params.withdrawalWallet)],
    ),
  ) as HexString;
}

export function computeEscrowScheduledFundingDigest(params: {
  escrowBatchHash: HexString;
  merkleRoot: HexString;
}): HexString {
  return keccak256(
    solidityPacked(
      ["bytes32", "bytes32"],
      [params.escrowBatchHash, params.merkleRoot],
    ),
  ) as HexString;
}

function generateLeafHash(batchHash: string, entry: PaymentEntry): Buffer {
  const encoded = solidityPacked(
    ["bytes32", "address", "uint256", "uint256", "string"],
    [
      batchHash,
      entry.receiverAddress,
      entry.amount,
      entry.claimableDate,
      entry.memo,
    ],
  );

  return Buffer.from(getBytes(keccak256(encoded)));
}

function hashMerkleNode(value: Buffer): Buffer {
  return Buffer.from(getBytes(keccak256(value)));
}

function bytesToBase64(bytes: Uint8Array): string {
  if (typeof Buffer !== "undefined") {
    return Buffer.from(bytes).toString("base64");
  }

  let binary = "";
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary);
}

function normalizePayoutSignature(
  result: string | Uint8Array | PayoutMessageSignature,
): string {
  if (typeof result === "string") return result;
  if (result instanceof Uint8Array) return bytesToBase64(result);
  return result.signature instanceof Uint8Array
    ? bytesToBase64(result.signature)
    : result.signature;
}

function normalizePayoutSignatureResult(
  result: string | Uint8Array | PayoutMessageSignature,
): PayoutSignatureResult {
  if (typeof result === "string") {
    return { signature: result };
  }
  if (result instanceof Uint8Array) {
    return { signature: bytesToBase64(result) };
  }
  return {
    signature:
      result.signature instanceof Uint8Array
        ? bytesToBase64(result.signature)
        : result.signature,
    signerAddress: result.signerAddress?.toLowerCase(),
  };
}

function generateMerkleTreeForPayout(
  batchHash: string,
  payments: PayoutPayment[],
  defaultClaimDate?: number,
): { merkleRoot: HexString; proofs: MerkleProofData[] } {
  if (payments.length === 0) {
    throw new Error("Cannot finalize scheduled payouts without payments");
  }

  const entries: PaymentEntry[] = payments.map((payment) => {
    if (payment.decimals === undefined) {
      throw new Error("Payment decimals are required to hash scheduled payouts");
    }

    return {
      receiverAddress: payment.receiver.toLowerCase(),
      amount: parseUnits(payment.amount.toString(), payment.decimals).toString(),
      claimableDate: payment.claimDate || defaultClaimDate || 0,
      memo: payment.memo || "",
    };
  });

  const leaves = entries.map((entry) => generateLeafHash(batchHash, entry));
  const tree = new MerkleTree(leaves, hashMerkleNode, { sortPairs: true });
  const merkleRoot = `0x${tree.getRoot().toString("hex")}` as HexString;

  const proofs = entries.map((entry) => {
    const leaf = generateLeafHash(batchHash, entry);
    return {
      receiver: entry.receiverAddress,
      proof: tree.getHexProof(leaf),
      leaf: `0x${leaf.toString("hex")}`,
    };
  });

  return { merkleRoot, proofs };
}

async function resolveSignerAddress(
  signer: PayoutSignerInput,
  fallback?: string,
): Promise<string | undefined> {
  if (typeof signer === "string" || "privateKey" in signer) {
    const privateKey = typeof signer === "string" ? signer : signer.privateKey;
    return new Wallet(privateKey).address.toLowerCase();
  }

  if (typeof signer === "function") {
    return fallback?.toLowerCase();
  }

  const address =
    fallback ||
    signer.signerAddress ||
    signer.address ||
    (await signer.getAddress?.());
  return address?.toLowerCase();
}

async function signPayoutFinalizeMessage(
  signer: PayoutSignerInput,
  message: string,
  chain?: string,
): Promise<PayoutSignatureResult> {
  if (typeof signer === "string" || "privateKey" in signer) {
    const privateKey = typeof signer === "string" ? signer : signer.privateKey;
    const wallet = new Wallet(privateKey);
    return {
      signature: await wallet.signMessage(message),
      signerAddress: wallet.address.toLowerCase(),
    };
  }

  if (typeof signer === "function") {
    return normalizePayoutSignatureResult(await signer(message));
  }

  const sign = signer.signFinalize ?? signer.signMessage;
  if (!sign) {
    throw new Error("Signer must provide signMessage(message) or signFinalize(message)");
  }

  const isSolana = (chain || signer.chain || "").toLowerCase().includes("solana");
  const payload = isSolana ? new TextEncoder().encode(message) : message;
  return normalizePayoutSignatureResult(await sign(payload));
}

function requireSignerAddress(
  signerAddress: string | undefined,
  context: string,
): string {
  if (!signerAddress) {
    throw new Error(
      `${context} requires signerAddress, or signMessage/signFinalize must return { signature, signerAddress }`,
    );
  }

  return signerAddress.toLowerCase();
}

async function signFundingDigest(
  signer: PayoutSignerInput,
  digest: HexString,
): Promise<string> {
  if (typeof signer === "string" || "privateKey" in signer) {
    const privateKey = typeof signer === "string" ? signer : signer.privateKey;
    return new Wallet(privateKey).signingKey.sign(digest).serialized;
  }

  if (typeof signer === "function") {
    throw new Error(
      "EVM payout finalization requires signFunding(digest), signDigest(digest), provider.request({ method: 'secp256k1_sign' }), or a private key for the funding signature",
    );
  }

  const sign = signer.signFunding ?? signer.signDigest;
  if (sign) {
    return normalizePayoutSignature(await sign(digest));
  }

  if (signer.request) {
    const result = await signer.request({
      method: "secp256k1_sign",
      params: [digest],
    });
    return String(result);
  }

  throw new Error(
    "EVM payout finalization requires signFunding(digest), signDigest(digest), provider.request({ method: 'secp256k1_sign' }), or a private key for the funding signature",
  );
}

function normalizeSigningKeySignature(
  result: PayoutSigningKeySignatureResult,
): string {
  if (typeof result === 'string') return result;
  if (result instanceof Uint8Array) return hexlify(result);
  return result.signature instanceof Uint8Array
    ? hexlify(result.signature)
    : result.signature;
}

async function signSigningKeyAuthorizationDigest(
  signer: PayoutSigningKeySignerInput,
  digest: HexString,
): Promise<string> {
  if (
    typeof signer === 'string' ||
    (typeof signer === 'object' && 'privateKey' in signer)
  ) {
    const privateKey = typeof signer === 'string' ? signer : signer.privateKey;
    return new Wallet(privateKey).signingKey.sign(digest).serialized;
  }

  if (typeof signer === 'function') {
    return normalizeSigningKeySignature(await signer(digest));
  }

  const sign = signer.signAuthorization ?? signer.signDigest;
  if (sign) {
    return normalizeSigningKeySignature(await sign(digest));
  }

  if (signer.request) {
    const result = await signer.request({
      method: 'secp256k1_sign',
      params: [digest],
    });
    return String(result);
  }

  throw new Error(
    "Signing key authorization requires signAuthorization(digest), signDigest(digest), provider.request({ method: 'secp256k1_sign' }), or a private key",
  );
}

function getPayoutPayments(payout: PayoutRecord): PayoutPayment[] {
  if (!Array.isArray(payout.payments)) {
    throw new Error("Payout response does not include payments");
  }

  return payout.payments;
}

function resolveClientId(payout: PayoutRecord, fallback?: string): string {
  const clientId = fallback || payout.app?.clientId;
  if (!clientId) {
    throw new Error("clientId is required to finalize this payout");
  }

  return clientId;
}

function normalizeTokenAddress(value: unknown): string | undefined {
  if (typeof value === "string") {
    return value.startsWith("0x") ? getAddress(value) : undefined;
  }

  if (value && typeof value === "object") {
    const record = value as Record<string, unknown>;
    return (
      normalizeTokenAddress(record.contractAddress) ||
      normalizeTokenAddress(record.address) ||
      normalizeTokenAddress(record.token) ||
      normalizeTokenAddress(record.payoutToken) ||
      normalizeTokenAddress(record.fundingToken) ||
      normalizeTokenAddress(record.current)
    );
  }

  return undefined;
}

const STABLECOIN_TOKEN_ADDRESSES: Record<
  string,
  Partial<Record<PayoutCurrency, SupportedPayoutCurrency>>
> = {
  base: {
    [PayoutCurrency.USDC]: {
      contractAddress: '0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913',
      decimals: 6,
    },
    [PayoutCurrency.USDT]: {
      contractAddress: '0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2',
      decimals: 6,
    },
  },
  bsc: {
    [PayoutCurrency.USDT]: {
      contractAddress: '0x55d398326f99059fF775485246999027B3197955',
      decimals: 18,
    },
    [PayoutCurrency.USDC]: {
      contractAddress: '0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d',
      decimals: 18,
    },
  },
  solana: {
    [PayoutCurrency.USDC]: {
      contractAddress: 'EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v',
      decimals: 6,
    },
    [PayoutCurrency.USDT]: {
      contractAddress: 'Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB',
      decimals: 6,
    },
  },
  'base-testnet': {
    [PayoutCurrency.USDC]: {
      contractAddress: '0x7dCEd3bFcC97948a665BB665a5D7eEfdfce39C3A',
      decimals: 18,
    },
    [PayoutCurrency.USDT]: {
      contractAddress: '0x9d0C28036AC12d2150a23DE40Bc4A92f7Aa1A79E',
      decimals: 18,
    },
  },
  'solana-testnet': {
    [PayoutCurrency.USDC]: {
      contractAddress: 'CmBGSxKZtv22ZiVpKGMP1oMPZfc5rsgr3pEGBDRcjiAy',
      decimals: 6,
    },
    [PayoutCurrency.USDT]: {
      contractAddress: 'SPFPKg9zeE7ReqW3j9QU6p7XhPP8JDU5Dx4fgrTwVyF',
      decimals: 6,
    },
  },
  localhost: {
    [PayoutCurrency.USDT]: {
      contractAddress: '0x5FbDB2315678afecb367f032d93F642f64180aa3',
      decimals: 18,
    },
  },
};

const CHAIN_ALIASES: Record<string, keyof typeof STABLECOIN_TOKEN_ADDRESSES> = {
  '8453': 'base',
  base: 'base',
  basemainnet: 'base',
  'base-mainnet': 'base',
  '56': 'bsc',
  bsc: 'bsc',
  binance: 'bsc',
  binancesmartchain: 'bsc',
  'binance-smart-chain': 'bsc',
  '101': 'solana',
  solana: 'solana',
  '84532': 'base-testnet',
  basetestnet: 'base-testnet',
  'base-testnet': 'base-testnet',
  basesepolia: 'base-testnet',
  'base-sepolia': 'base-testnet',
  '1012': 'solana-testnet',
  solanatestnet: 'solana-testnet',
  'solana-testnet': 'solana-testnet',
  '31337': 'localhost',
  localhost: 'localhost',
};

const PAYOUT_CHAIN_IDS: Record<keyof typeof STABLECOIN_TOKEN_ADDRESSES, number> = {
  base: 8453,
  bsc: 56,
  solana: 101,
  'base-testnet': 84532,
  'solana-testnet': 1012,
  localhost: 31337,
};

function normalizePayoutCurrency(
  value?: string,
): PayoutCurrency | undefined {
  if (!value) return undefined;
  const normalized = value.toLowerCase();
  if (normalized === 'usdc') return PayoutCurrency.USDC;
  if (normalized === 'usdt') return PayoutCurrency.USDT;
  return undefined;
}

function resolvePayoutCurrencyAddress(
  chain: string,
  currency?: PayoutCurrencyInput,
): string | undefined {
  const config = resolvePayoutCurrencyConfig(chain, currency);
  if (!config) return undefined;

  return config.contractAddress.startsWith('0x')
    ? getAddress(config.contractAddress)
    : config.contractAddress;
}

function resolvePayoutCurrencyConfig(
  chain: string,
  currency?: PayoutCurrencyInput,
): SupportedPayoutCurrency | undefined {
  const normalizedCurrency = normalizePayoutCurrency(currency);
  if (!normalizedCurrency) return undefined;

  const chainKey =
    CHAIN_ALIASES[chain.toLowerCase().replace(/\s+/g, '')] ||
    CHAIN_ALIASES[chain.toLowerCase()];
  const config = chainKey
    ? STABLECOIN_TOKEN_ADDRESSES[chainKey]?.[normalizedCurrency]
    : undefined;

  if (!config) {
    throw new Error(
      `payoutCurrency ${normalizedCurrency} is not supported on chain ${chain}`,
    );
  }

  return config;
}

function resolvePayoutCurrencyByToken(
  chain: string,
  token?: string,
): SupportedPayoutCurrency | undefined {
  if (!token) return undefined;
  const chainKey =
    CHAIN_ALIASES[chain.toLowerCase().replace(/\s+/g, '')] ||
    CHAIN_ALIASES[chain.toLowerCase()];
  if (!chainKey) return undefined;

  const normalizedToken = normalizeTokenAddress(token) || token;
  return Object.values(STABLECOIN_TOKEN_ADDRESSES[chainKey] || {}).find(
    (currency) => {
      if (!currency) return false;
      const normalizedAddress =
        normalizeTokenAddress(currency.contractAddress) ||
        currency.contractAddress;
      return normalizedAddress === normalizedToken;
    },
  );
}

function formatConfiguredToken(currency: SupportedPayoutCurrency): string {
  return currency.contractAddress.startsWith('0x')
    ? getAddress(currency.contractAddress)
    : currency.contractAddress;
}

function normalizeTokenValue(token?: string): string | undefined {
  return (
    normalizeTokenAddress(token) ||
    (typeof token === 'string' && token.length > 10 ? token : undefined)
  );
}

function resolvePaymentToken(
  chain: string,
  payment: Pick<PayoutPayment, 'token' | 'tokenSymbol'>,
  tokenFallback?: string,
): {
  token?: string;
  currency?: SupportedPayoutCurrency;
} {
  const symbol = payment.tokenSymbol || normalizePayoutCurrency(payment.token);
  if (symbol) {
    const currency = resolvePayoutCurrencyConfig(chain, symbol);
    if (!currency) return {};
    return {
      token: formatConfiguredToken(currency),
      currency,
    };
  }

  if (payment.token) {
    const token = normalizeTokenValue(payment.token);
    const currency = resolvePayoutCurrencyByToken(chain, token);
    if (!token || !currency) {
      const fallbackToken = normalizeTokenValue(tokenFallback);
      if (token && fallbackToken === token) {
        return { token };
      }

      throw new Error(
        `Payment token ${payment.token} is not supported on chain ${chain}`,
      );
    }

    return {
      token: formatConfiguredToken(currency),
      currency,
    };
  }

  const token = normalizeTokenValue(tokenFallback);
  const currency = resolvePayoutCurrencyByToken(chain, token);
  return {
    token: currency ? formatConfiguredToken(currency) : token,
    currency,
  };
}

function buildPayoutMetadata(
  input: CreatePayoutInput,
): Record<string, unknown> {
  const metadata = { ...(input.metadata || {}) };
  const payoutCurrencyConfig = resolvePayoutCurrencyConfig(
    input.chain,
    input.payoutCurrency,
  );

  if (payoutCurrencyConfig) {
    metadata.payoutCurrency = payoutCurrencyConfig.contractAddress.startsWith('0x')
      ? getAddress(payoutCurrencyConfig.contractAddress)
      : payoutCurrencyConfig.contractAddress;
    metadata.payoutCurrencyDecimals = payoutCurrencyConfig.decimals;
  }
  if (input.scheduleDate != null) {
    metadata.scheduledDate = input.scheduleDate;
  }
  if (input.lockDuration && metadata.lockDuration == null) {
    metadata.lockDuration = input.lockDuration;
  }
  if (input.type === "Milestone") {
    metadata.commitmentType = "milestone";
  }

  return metadata;
}

function resolvePayoutFundingTokenCandidate(payout?: PayoutRecord): string | undefined {
  if (!payout) return undefined;

  return (
    normalizeTokenAddress(payout.payoutToken) ||
    normalizeTokenAddress(payout.token) ||
    normalizeTokenAddress(payout.fundingToken) ||
    normalizeTokenAddress(payout.metadata?.payoutToken) ||
    normalizeTokenAddress(payout.metadata?.payoutCurrency) ||
    normalizeTokenAddress(payout.metadata?.fundingToken) ||
    normalizeTokenAddress(payout.payments?.[0]?.token)
  );
}

function resolveFundingToken(
  payout: PayoutRecord,
  options: PayoutFinalizeOptions,
): string {
  const linkedEscrow =
    typeof options.escrowBatch === "object" ? options.escrowBatch : undefined;
  const token =
    normalizeTokenAddress(options.fundingToken) ||
    resolvePayoutFundingTokenCandidate(payout) ||
    resolvePayoutFundingTokenCandidate(linkedEscrow);

  if (!token) {
    throw new Error(
      "fundingToken must be provided as an address to finalize scheduled payouts",
    );
  }

  return token;
}

function normalizePaymentsForCreate(
  chain: string,
  payments?: PayoutPayment[],
  tokenFallback?: string,
  decimalsFallback?: number,
  expectedToken?: string,
): PayoutPayment[] | undefined {
  if (!payments) return undefined;
  const normalizedExpectedToken = normalizeTokenValue(expectedToken);

  return payments.map((payment) => {
    const { tokenSymbol: _tokenSymbol, ...rest } = payment;
    const resolved = resolvePaymentToken(chain, payment, tokenFallback);
    const normalizedPaymentToken = normalizeTokenValue(resolved.token);
    if (
      normalizedExpectedToken &&
      normalizedPaymentToken &&
      normalizedPaymentToken !== normalizedExpectedToken
    ) {
      throw new Error(
        "Payment token must match payoutCurrency when payoutCurrency is provided",
      );
    }

    return {
      ...rest,
      token: resolved.token,
      decimals: payment.decimals ?? resolved.currency?.decimals ?? decimalsFallback,
      amount:
        typeof payment.amount === "string"
          ? Number(payment.amount)
          : payment.amount,
    };
  });
}

function resolvePayoutId(reference?: string | PayoutRecord): string | undefined {
  if (!reference) return undefined;
  return typeof reference === "string" ? reference : reference.id;
}

function assertPayoutChain(chain: string): PayoutChain {
  const chainKey = CHAIN_ALIASES[
    chain.toLowerCase().replace(/\s+/g, '')
  ] as PayoutChain | undefined;
  if (!chainKey) {
    throw new Error(`Unsupported payout chain ${chain}`);
  }

  return chainKey;
}

function resolvePayoutChainId(
  chain: string,
  chainId?: number,
  context = 'payout finalization',
): number {
  if (chainId) return chainId;

  const chainKey = CHAIN_ALIASES[chain.toLowerCase().replace(/\s+/g, '')];
  const resolvedChainId = chainKey ? PAYOUT_CHAIN_IDS[chainKey] : undefined;
  if (!resolvedChainId) {
    throw new Error(`chainId is required for ${context}`);
  }

  return resolvedChainId;
}

function normalizePaymentsForSigning(
  payments: PayoutPayment[],
  chain: string,
  tokenFallback?: string,
): PayoutPayment[] {
  const tokenDecimals = resolvePayoutCurrencyByToken(chain, tokenFallback)?.decimals;
  return payments.map((payment) => {
    const { tokenSymbol: _tokenSymbol, ...rest } = payment;
    const resolved = resolvePaymentToken(chain, payment, tokenFallback);
    const decimals =
      payment.decimals ??
      resolved.currency?.decimals ??
      tokenDecimals;

    return {
      ...rest,
      token: resolved.token,
      decimals,
    };
  });
}

export class PviumPayoutService {
  private readonly consentHost: string;
  private readonly clientId?: string;

  constructor(
    private readonly http: PviumHttpClient,
    config: PviumSdkConfig,
  ) {
    this.consentHost = resolvePviumConsentHost(config);
    this.clientId = config.clientId;
  }

  async authorizeSigningKey(
    batchHash: HexString,
    signingKey: string,
    networkType: PayoutSigningKeyNetworkType,
    authorizationData: PayoutSigningKeyAuthorizationData,
    signer: PayoutSigningKeySignerInput,
  ): Promise<PayoutSigningKeyAuthorization> {
    if (networkType !== 'ethereum' && networkType !== 'solana') {
      throw new Error('networkType must be ethereum or solana');
    }

    const { normalizedInput, authMessageHash } =
      computeSigningKeyAuthorizationHash({
        batchHash,
        signingKey: signingKey,
        transactionMax: authorizationData.transactionMax,
        totalMax: authorizationData.totalMax,
        expiration: authorizationData.expiration,
        timestamp: authorizationData.timestamp ?? Math.floor(Date.now() / 1000),
      });
    const signature = await signSigningKeyAuthorizationDigest(
      signer,
      authMessageHash,
    );

    return {
      batchHash,
      signingKey: normalizedInput.signingKey,
      networkType,
      transactionMax: normalizedInput.transactionMax,
      totalMax: normalizedInput.totalMax,
      expiration: normalizedInput.expiration,
      timestamp: normalizedInput.timestamp,
      authMessageHash,
      signature,
    };
  }

  async create(
    input: CreatePayoutInput,
    options?: RequestOptions,
  ): Promise<PayoutIntent> {
    const mapped = mapPayoutType(input.type);
    const escrowBatchId = resolvePayoutId(input.escrowBatch);
    const metadata = buildPayoutMetadata(input);
    const tokenFallback = String(
      normalizeTokenAddress(metadata.payoutToken) ||
        normalizeTokenAddress(metadata.payoutCurrency) ||
        normalizeTokenAddress(metadata.fundingToken) ||
        metadata.payoutCurrency ||
        metadata.fundingToken ||
        '',
    );
    const decimalsFallback =
      typeof metadata.payoutCurrencyDecimals === 'number'
        ? metadata.payoutCurrencyDecimals
        : resolvePayoutCurrencyByToken(input.chain, tokenFallback)?.decimals;

    const response = await this.http.request({
      method: 'POST',
      path: '/v1/batch-payments',
      body: {
        id: input.id,
        chain: input.chain,
        nonce: input.nonce || createPayoutNonce(),
        paymentType: mapped.paymentType,
        isCommitment: mapped.isCommitment,
        escrowBatch: escrowBatchId,
        payments: normalizePaymentsForCreate(
          input.chain,
          input.payments,
          tokenFallback,
          decimalsFallback,
          input.payoutCurrency ? tokenFallback : undefined,
        ),
        lockDuration: input.lockDuration,
        label: input.label || input.name,
        name: input.name,
        description: input.description,
        complianceMode: input.complianceMode || 'Open',
        metadata,
      },
      options,
    });

    return this.wrapPayoutResponse(
      await this.parse<PayoutApiResponse<PayoutRecord>>(response),
    );
  }

  async createFinalized(
    input: CreatePayoutInput,
    signer: PayoutSignerInput,
    options: PayoutFinalizeOptions = {},
    requestOptions?: RequestOptions,
  ): Promise<PayoutFinalization> {
    if (!input.id) {
      throw new Error('id is required to create finalized payouts');
    }

    const mapped = mapPayoutType(input.type);
    if (mapped.paymentType !== 'Scheduled' && !mapped.isCommitment) {
      throw new Error('createFinalized currently supports scheduled payouts');
    }

    const escrowBatchId = resolvePayoutId(input.escrowBatch);
    const metadata = buildPayoutMetadata(input);
    const tokenFallback =
      normalizeTokenAddress(metadata.payoutToken) ||
      normalizeTokenAddress(metadata.payoutCurrency) ||
      normalizeTokenAddress(metadata.fundingToken) ||
      (typeof metadata.payoutCurrency === 'string'
        ? metadata.payoutCurrency
        : undefined) ||
      (typeof metadata.fundingToken === 'string'
        ? metadata.fundingToken
        : undefined);
    const decimalsFallback =
      typeof metadata.payoutCurrencyDecimals === 'number'
        ? metadata.payoutCurrencyDecimals
        : resolvePayoutCurrencyByToken(input.chain, tokenFallback)?.decimals;
    const payments = normalizePaymentsForCreate(
      input.chain,
      input.payments,
      tokenFallback,
      decimalsFallback,
      input.payoutCurrency ? tokenFallback : undefined,
    );
    const payout: PayoutRecord = {
      id: input.id,
      chain: input.chain,
      nonce: input.nonce || createPayoutNonce(),
      paymentType: mapped.paymentType,
      isCommitment: mapped.isCommitment,
      complianceMode: input.complianceMode || 'Open',
      escrowBatch: escrowBatchId,
      metadata,
      payments,
    };

    const finalized = await this.buildScheduledFinalizePayload(
      payout,
      signer,
      options,
      requestOptions,
    );

    const response = await this.http.request({
      method: 'POST',
      path: '/v1/batch-payments',
      body: {
        id: input.id,
        chain: input.chain,
        nonce: payout.nonce,
        paymentType: mapped.paymentType,
        isCommitment: mapped.isCommitment,
        escrowBatch: escrowBatchId,
        payments: normalizePaymentsForCreate(
          input.chain,
          payments,
          finalized.fundingToken,
          resolvePayoutCurrencyByToken(input.chain, finalized.fundingToken)
            ?.decimals,
          input.payoutCurrency ? finalized.fundingToken : undefined,
        ),
        label: input.label || input.name,
        name: input.name,
        description: input.description,
        complianceMode: input.complianceMode || 'Open',
        metadata,
        ...finalized.updatePayload,
      },
      options: requestOptions,
    });

    const created = await this.parse<PayoutApiResponse<PayoutRecord>>(response);
    const identifier = created.data.merkleRoot || finalized.merkleRoot;

    return this.wrapFinalizationResponse({
      meta: created.meta,
      data: {
        payout: created.data,
        fundingUrl: identifier
          ? `${this.consentHost}/batch/${identifier}`
          : undefined,
        batchDataHash: finalized.batchDataHash,
        batchHash: finalized.batchHash,
        merkleRoot: finalized.merkleRoot,
      },
    });
  }

  async list(
    query?: PayoutListQuery,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<PayoutRecord[]>> {
    const response = await this.http.request({
      method: 'GET',
      path: '/v1/batch-payments',
      query: query ? { ...query } : undefined,
      options,
    });

    return this.parse<PayoutApiResponse<PayoutRecord[]>>(response);
  }

  async get(
    payoutId: string,
    options?: RequestOptions,
  ): Promise<PayoutIntent> {
    const response = await this.http.request({
      method: 'GET',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}`,
      options,
    });

    return this.wrapPayoutResponse(
      await this.parse<PayoutApiResponse<PayoutRecord>>(response),
    );
  }

  async addPayments(
    payout: string | PayoutRecord,
    input: AddPayoutPaymentsInput | PayoutPayment[],
    options?: RequestOptions,
  ): Promise<PayoutIntent | PayoutFinalization> {
    const payments = Array.isArray(input) ? input : input.payments;
    const requestOptions =
      !Array.isArray(input) && input.requestOptions
        ? input.requestOptions
        : options;

    if (typeof payout !== 'string' && payout.paymentType === 'Escrow') {
      const signer = Array.isArray(input) ? undefined : input.signer;
      if (!signer) {
        throw new Error(
          'A signer or private key is required to add payments to escrow payouts',
        );
      }

      return this.addEscrowPayees(
        payout,
        payments,
        signer,
        Array.isArray(input) ? {} : input.finalizeOptions || {},
        requestOptions,
      );
    }

    const payoutId = typeof payout === 'string' ? payout : payout.id;
    const response = await this.http.request({
      method: 'POST',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}/payments`,
      body: {
        payments: normalizePaymentsForCreate(
          typeof payout === 'string' ? '' : payout.chain,
          payments,
        ),
      },
      options: requestOptions,
    });

    return this.wrapPayoutResponse(
      await this.parse<PayoutApiResponse<PayoutRecord>>(response),
    );
  }

  private async addEscrowPayees(
    escrowBatch: string | PayoutRecord,
    payments: PayoutPayment[],
    signer: PayoutSignerInput,
    options: PayoutFinalizeOptions & {
      id?: string;
      name?: string;
      description?: string;
      metadata?: Record<string, unknown>;
    } = {},
    requestOptions?: RequestOptions,
  ): Promise<PayoutFinalization> {
    const escrowPayout =
      typeof escrowBatch === 'string'
        ? (await this.get(escrowBatch, requestOptions)).data
        : escrowBatch;

    if (escrowPayout.paymentType !== 'Escrow') {
      throw new Error('addEscrowPayees requires an escrow payout');
    }
    if (!escrowPayout.batchHash) {
      throw new Error('Escrow payout must be finalized before adding payees');
    }
    if (escrowPayout.status && escrowPayout.status !== 'funded') {
      throw new Error('Escrow payout must be funded before adding payees');
    }
    if (!payments.length) {
      throw new Error('At least one payee is required');
    }

    const claimDate = options.claimDate || Math.floor(Date.now() / 1000);
    const fundingToken =
      normalizeTokenAddress(options.fundingToken) ||
      resolvePayoutFundingTokenCandidate(escrowPayout);
    const scheduledPayments = payments.map((payment) => ({
      ...payment,
      token: normalizeTokenAddress(payment.token) || fundingToken,
      claimDate: payment.claimDate || claimDate,
    }));

    return this.createFinalized(
      {
        id: options.id || createPayoutId(),
        type: 'Scheduled',
        chain: assertPayoutChain(options.chain || escrowPayout.chain),
        name:
          options.name ||
          `${String(escrowPayout.name || 'Escrow payout')} Payees`,
        description: options.description,
        complianceMode:
          options.complianceMode || escrowPayout.complianceMode || 'Open',
        escrowBatch: escrowPayout,
        payments: scheduledPayments,
        metadata: {
          ...(options.metadata || {}),
          payoutCurrency:
            (options.metadata || {}).payoutCurrency ?? fundingToken,
          escrowBatch: escrowPayout.id,
          escrowBatchHash: escrowPayout.batchHash,
          scheduledDate: claimDate,
        },
      },
      signer,
      {
        ...options,
        clientId: options.clientId || escrowPayout.app?.clientId,
        chain: options.chain || escrowPayout.chain,
        escrowBatch: escrowPayout,
        fundingToken,
        payments: scheduledPayments,
        claimDate,
      },
      requestOptions,
    );
  }

  async addRecipients(
    payoutId: string,
    input: AddPayoutRecipientsInput | PayoutRecipient[],
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<AddPayoutRecipientsResult>> {
    const recipients = Array.isArray(input) ? input : input.recipients;
    const response = await this.http.request({
      method: 'POST',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}/open-payees`,
      body: { recipients },
      options,
    });

    return this.parse<PayoutApiResponse<AddPayoutRecipientsResult>>(response);
  }

  async resolveRecipients(
    payoutId: string,
    input: ResolvePayoutRecipientsInput | ResolvePayoutRecipient[],
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<ResolvePayoutRecipientsResult>> {
    const recipients = Array.isArray(input) ? input : input.recipients;
    const response = await this.http.request({
      method: 'POST',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}/resolve-recipients`,
      body: { recipients },
      options,
    });

    return this.parse<PayoutApiResponse<ResolvePayoutRecipientsResult>>(
      response,
    );
  }

  async removePayments(
    payoutId: string,
    input: RemovePayoutPaymentsInput | Array<number | string>,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<undefined>> {
    const paymentIds = Array.isArray(input) ? input : input.paymentIds;
    const response = await this.http.request({
      method: 'DELETE',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}/payments`,
      body: { paymentIds: paymentIds.map((id) => Number(id)) },
      options,
    });

    return this.parse<PayoutApiResponse<undefined>>(response);
  }

  async deletePayment(
    payoutId: string,
    paymentId: number | string,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<undefined>> {
    return this.removePayments(payoutId, [paymentId], options);
  }

  async updatePayment(
    payoutId: string,
    paymentId: number | string,
    input: UpdatePayoutPaymentInput,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<unknown>> {
    const response = await this.http.request({
      method: 'PATCH',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}/payments/${encodeURIComponent(String(paymentId))}`,
      body: input,
      options,
    });

    return this.parse<PayoutApiResponse<unknown>>(response);
  }

  async editPayment(
    payoutId: string,
    paymentId: number | string,
    input: UpdatePayoutPaymentInput,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<unknown>> {
    return this.updatePayment(payoutId, paymentId, input, options);
  }

  async revokeInvite(
    payoutId: string,
    inviteId: string,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<undefined>> {
    const response = await this.http.request({
      method: 'DELETE',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}/invites/${encodeURIComponent(inviteId)}`,
      options,
    });

    return this.parse<PayoutApiResponse<undefined>>(response);
  }

  async revokeInviteRoot(
    payoutId: string,
    inviteRootId: string,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<undefined>> {
    const response = await this.http.request({
      method: 'DELETE',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}/invite-roots/${encodeURIComponent(inviteRootId)}`,
      options,
    });

    return this.parse<PayoutApiResponse<undefined>>(response);
  }

  async delete(
    payoutId: string,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<undefined>> {
    const response = await this.http.request({
      method: 'DELETE',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}`,
      options,
    });

    return this.parse<PayoutApiResponse<undefined>>(response);
  }

  async listInvites(
    payoutId: string,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<unknown[]>> {
    const response = await this.http.request({
      method: 'GET',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}/invites`,
      options,
    });

    return this.parse<PayoutApiResponse<unknown[]>>(response);
  }

  async listPayments(
    payoutId: string,
    query?: PayoutPaymentsListQuery,
    options?: RequestOptions,
  ): Promise<PayoutApiResponse<unknown[]>> {
    const response = await this.http.request({
      method: 'GET',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}/payments`,
      query: query ? { ...query } : undefined,
      options,
    });

    return this.parse<PayoutApiResponse<unknown[]>>(response);
  }

  private async buildScheduledFinalizePayload(
    payout: PayoutRecord,
    signer: PayoutSignerInput,
    options: PayoutFinalizeOptions,
    requestOptions?: RequestOptions,
  ): Promise<{
    updatePayload: Record<string, unknown>;
    batchDataHash: HexString;
    batchHash: HexString;
    merkleRoot: HexString;
    fundingToken: string;
  }> {
    const payoutId = payout.id;
    const timestamp = options.timestamp || Math.floor(Date.now() / 1000);
    let signerAddress = await resolveSignerAddress(
      signer,
      options.signerAddress,
    );
    const complianceMode =
      options.complianceMode || payout.complianceMode || 'Open';
    const clientId = resolveClientId(payout, options.clientId || this.clientId);
    const chain = options.chain || payout.chain || '';
    const chainId = resolvePayoutChainId(
      chain,
      options.chainId,
      'scheduled payout finalization',
    );
    const gracePeriod =
      options.gracePeriod ?? Number(payout.metadata?.gracePeriod || 0);
    const disapprovalDeadline =
      options.disapprovalDeadline ??
      Number(payout.metadata?.disapprovalDeadline || 0);
    const fundingToken = resolveFundingToken(payout, options);
    const payments = normalizePaymentsForSigning(
      options.payments || getPayoutPayments(payout),
      chain,
      fundingToken,
    );
    const batchHash = computeScheduledPayoutHash({
      payoutId,
      fundingToken,
      gracePeriod,
      disapprovalDeadline,
      timestamp,
      chainId,
    });

    const backendMessage = `PVIUM_SIGNED_SCHEDULE:${clientId}:${batchHash}:${complianceMode}:${timestamp}`;
    const backendSignature = await signPayoutFinalizeMessage(
      signer,
      backendMessage,
      chain,
    );
    signerAddress = requireSignerAddress(
      signerAddress || backendSignature.signerAddress,
      'Scheduled payout finalization',
    );

    const merkle = generateMerkleTreeForPayout(
      batchHash,
      payments,
      options.claimDate ||
        Number(
          payout.metadata?.scheduledDate || payout.metadata?.claimableDate || 0,
        ),
    );
    const merkleRoot = merkle.merkleRoot;
    const batchDataHash = keccak256(
      solidityPacked(
        ['bytes32', 'bytes32', 'address'],
        [batchHash, merkleRoot, signerAddress],
      ),
    ) as HexString;

    const updatePayload: Record<string, unknown> = {
      signer: signerAddress,
      batchSignature: `${timestamp}:${signerAddress}:${backendSignature.signature}`,
      batchHash,
      merkleRoot,
      batchDataHash,
      proofs: merkle.proofs.map((proof) => ({
        receiver: proof.receiver,
        proof: proof.proof,
      })),
      gracePeriod,
      disapprovalDeadline,
    };

    if (!chain.toLowerCase().includes('solana')) {
      let fundingDigest = batchDataHash;
      const linkedEscrow =
        options.escrowBatch ||
        payout.escrowBatch ||
        (typeof payout.metadata?.escrowBatch === 'string'
          ? payout.metadata.escrowBatch
          : undefined);

      if (linkedEscrow) {
        const escrowPayout =
          typeof linkedEscrow === 'string'
            ? (await this.get(linkedEscrow, requestOptions)).data
            : linkedEscrow;

        if (!escrowPayout.batchHash) {
          throw new Error(
            'Linked escrow payout must be finalized before finalizing scheduled payouts',
          );
        }

        fundingDigest = computeEscrowScheduledFundingDigest({
          escrowBatchHash: escrowPayout.batchHash as HexString,
          merkleRoot,
        });
      }

      updatePayload.fundingSignature = await signFundingDigest(
        signer,
        fundingDigest,
      );
    }

    return {
      updatePayload,
      batchDataHash,
      batchHash,
      merkleRoot,
      fundingToken,
    };
  }

  async finalize(
    payoutInput: PayoutReference,
    signer: PayoutSignerInput,
    options: PayoutFinalizeOptions = {},
    requestOptions?: RequestOptions,
  ): Promise<PayoutFinalization> {
    const payout =
      typeof payoutInput === 'string'
        ? (await this.get(payoutInput, requestOptions)).data
        : payoutInput;
    const payoutId = payout.id;
    const timestamp = options.timestamp || Math.floor(Date.now() / 1000);
    let signerAddress = await resolveSignerAddress(
      signer,
      options.signerAddress,
    );
    const complianceMode =
      options.complianceMode || payout.complianceMode || 'Open';
    const clientId = resolveClientId(payout, options.clientId || this.clientId);
    const chain = options.chain || payout.chain || '';

    const updatePayload: Record<string, unknown> = {};

    let batchDataHash: HexString;
    let batchHash: HexString | undefined;
    let merkleRoot: HexString | undefined;

    if (payout.paymentType === 'Scheduled' || payout.isCommitment) {
      const chainId = resolvePayoutChainId(
        chain,
        options.chainId,
        'scheduled payout finalization',
      );

      const gracePeriod =
        options.gracePeriod ?? Number(payout.metadata?.gracePeriod || 0);
      const disapprovalDeadline =
        options.disapprovalDeadline ??
        Number(payout.metadata?.disapprovalDeadline || 0);
      const fundingToken = resolveFundingToken(payout, options);
      const payments = normalizePaymentsForSigning(
        options.payments || getPayoutPayments(payout),
        chain,
        fundingToken,
      );
      batchHash = computeScheduledPayoutHash({
        payoutId,
        fundingToken,
        gracePeriod,
        disapprovalDeadline,
        timestamp,
        chainId,
      });

      const backendMessage = `PVIUM_SIGNED_SCHEDULE:${clientId}:${batchHash}:${complianceMode}:${timestamp}`;
      const backendSignature = await signPayoutFinalizeMessage(
        signer,
        backendMessage,
        chain,
      );
      signerAddress = requireSignerAddress(
        signerAddress || backendSignature.signerAddress,
        'Scheduled payout finalization',
      );

      const merkle = generateMerkleTreeForPayout(
        batchHash,
        payments,
        options.claimDate ||
          Number(
            payout.metadata?.scheduledDate ||
              payout.metadata?.claimableDate ||
              0,
          ),
      );
      merkleRoot = merkle.merkleRoot;
      batchDataHash = keccak256(
        solidityPacked(
          ['bytes32', 'bytes32', 'address'],
          [batchHash, merkleRoot, signerAddress],
        ),
      ) as HexString;

      updatePayload.signer = signerAddress;
      updatePayload.batchSignature = `${timestamp}:${signerAddress}:${backendSignature.signature}`;
      updatePayload.batchHash = batchHash;
      updatePayload.merkleRoot = merkleRoot;
      updatePayload.batchDataHash = batchDataHash;
      updatePayload.proofs = merkle.proofs.map((proof) => ({
        receiver: proof.receiver,
        proof: proof.proof,
      }));
      updatePayload.gracePeriod = gracePeriod;
      updatePayload.disapprovalDeadline = disapprovalDeadline;

      if (!chain.toLowerCase().includes('solana')) {
        let fundingDigest = batchDataHash;

        const linkedEscrow =
          options.escrowBatch ||
          payout.escrowBatch ||
          (typeof payout.metadata?.escrowBatch === 'string'
            ? payout.metadata.escrowBatch
            : undefined);

        if (linkedEscrow) {
          const escrowPayout =
            typeof linkedEscrow === 'string'
              ? (await this.get(linkedEscrow, requestOptions)).data
              : linkedEscrow;

          if (!escrowPayout.batchHash) {
            throw new Error(
              'Linked escrow payout must be finalized before finalizing scheduled payouts',
            );
          }

          fundingDigest = computeEscrowScheduledFundingDigest({
            escrowBatchHash: escrowPayout.batchHash as HexString,
            merkleRoot,
          });
        }

        updatePayload.fundingSignature = await signFundingDigest(
          signer,
          fundingDigest,
        );
      }
    } else if (payout.paymentType === 'Escrow') {
      const chainId = resolvePayoutChainId(
        chain,
        options.chainId,
        'escrow payout finalization',
      );

      const nonce = payout.nonce || options.timestamp?.toString();
      if (!nonce) {
        throw new Error('Payout nonce is required to finalize escrow payouts');
      }

      const fundingToken = resolveFundingToken(payout, options);
      const payments = normalizePaymentsForSigning(
        options.payments || getPayoutPayments(payout),
        chain,
        fundingToken,
      );
      const lockDuration = Number(
        options.lockDuration ??
          payout.metadata?.lockDuration ??
          payout.lockDuration ??
          0,
      );
      if (!Number.isFinite(lockDuration) || lockDuration <= 0) {
        throw new Error('lockDuration is required to finalize escrow payouts');
      }

      batchDataHash = generateInstantPayoutHash(payments, nonce);
      const message = `PVIUM_SIGNED_BATCH:${clientId}:${batchDataHash}:${complianceMode}:${timestamp}`;
      const signature = await signPayoutFinalizeMessage(signer, message, chain);
      signerAddress = requireSignerAddress(
        signerAddress || signature.signerAddress,
        'Escrow payout finalization',
      );

      const escrowBatchHash = computeEscrowPayoutHash({
        payoutId,
        fundingToken,
        lockDuration,
        timestamp,
        chainId,
      });
      const escrowFundingDigest = computeEscrowFundingDigest({
        escrowBatchHash,
        withdrawalWallet: signerAddress,
      });

      updatePayload.signer = signerAddress;
      updatePayload.batchSignature = `${timestamp}:${signerAddress}:${signature.signature}`;
      updatePayload.fundingSignature = `${timestamp}:${signerAddress}:${await signFundingDigest(
        signer,
        escrowFundingDigest,
      )}`;
      updatePayload.batchHash = escrowBatchHash;
      updatePayload.batchDataHash = batchDataHash;
      updatePayload.metadata = {
        ...(payout.metadata || {}),
        lockDuration,
      };
    } else {
      const nonce = payout.nonce || options.timestamp?.toString();
      if (!nonce) {
        throw new Error('Payout nonce is required to finalize instant payouts');
      }

      const payments = options.payments || getPayoutPayments(payout);
      batchDataHash = generateInstantPayoutHash(payments, nonce);
      const message = `PVIUM_SIGNED_BATCH:${clientId}:${batchDataHash}:${complianceMode}:${timestamp}`;
      const signature = await signPayoutFinalizeMessage(signer, message, chain);
      signerAddress = requireSignerAddress(
        signerAddress || signature.signerAddress,
        'Instant payout finalization',
      );

      updatePayload.signer = signerAddress;
      updatePayload.batchSignature = `${timestamp}:${signerAddress}:${signature.signature}`;
      updatePayload.batchDataHash = batchDataHash;
    }

    const response = await this.http.request({
      method: 'PATCH',
      path: `/v1/batch-payments/${encodeURIComponent(payoutId)}`,
      body: updatePayload,
      options: requestOptions,
    });
    const finalized =
      await this.parse<PayoutApiResponse<PayoutRecord>>(response);
    const finalizedPayout = finalized.data;
    const identifier =
      finalizedPayout.paymentType === 'Scheduled'
        ? finalizedPayout.merkleRoot || merkleRoot
        : finalizedPayout.batchDataHash || batchDataHash;

    return this.wrapFinalizationResponse({
      meta: finalized.meta,
      data: {
        payout: finalizedPayout,
        fundingUrl: identifier
          ? `${this.consentHost}/batch/${identifier}`
          : undefined,
        batchDataHash,
        batchHash,
        merkleRoot,
      },
    });
  }

  private wrapPayoutResponse(response: PayoutApiResponse<PayoutRecord>): PayoutIntent {
    return new PayoutIntent(this, response.meta, response.data);
  }

  private wrapFinalizationResponse(
    response: PayoutApiResponse<FinalizePayoutData>,
  ): PayoutFinalization {
    return new PayoutFinalization(this, response.meta, response.data);
  }

  private async parse<T>(response: Response): Promise<T> {
    const body = await this.http.parseResponseBody<any>(response);

    if (!response.ok) {
      const message =
        body?.meta?.message ||
        body?.message ||
        body?.error ||
        `Pvium API request failed with status ${response.status}`;
      throw new Error(message);
    }

    return body as T;
  }
}
