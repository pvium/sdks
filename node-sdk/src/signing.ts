import {
  AbiCoder,
  Wallet,
  type AbiCoder as AbiCoderTypes,
  type Signer,
  concat,
  getBytes,
  id,
  keccak256,
  toBeHex,
  toUtf8Bytes,
} from "ethers";

export type HexString = `0x${string}`;
export type Numeric = number | bigint;

export type MessageSigner = Pick<Signer, "signMessage">;
export type SignerInput = string | MessageSigner;

const ABI_CODER: AbiCoderTypes = AbiCoder.defaultAbiCoder();

export const PVIUM_SIGNATURE_DOMAIN = id(
  "PVIUM_SIGNATURE_MESSAGE",
) as HexString;

export interface CreateProjectRequestPayload {
  app: string;
  projectId: string;
  metadata: string;
  tokenAddress: string;
  refundAddress: string;
  appFeeAddress: string;
  appAdminAddress: string;
  appFeeBps: Numeric;
  disputeWindowSeconds: Numeric;
  lockDuration: Numeric;
  minimumBalancePerVendor: Numeric;
}

export interface CreateProjectSignatureOptions {
  pviumFeeBps: Numeric;
  chainId: Numeric;
  signatureDomain?: HexString;
}

export interface CreateClaimRequestPayload {
  app: string;
  projectId: string;
  claimId: string;
  receiver: string;
  amount: Numeric;
  claimableAfter: Numeric;
  claimDeadline: Numeric;
  nonce: Numeric;
}

export interface FinalizeClaimRequestPayload {
  app: string;
  projectId: string;
  claimId: string;
}

export interface RelayedCallRequestPayload {
  appId: string;
  projectId: string;
  payload: HexString;
  nonce: Numeric;
  chainId: Numeric;
}

export interface ResolveDisputeRequestPayload {
  claimId: string;
  approved: boolean;
  chainId: Numeric;
}

export function createSignerFromPrivateKey(privateKey: string): Wallet {
  return new Wallet(privateKey);
}

function resolveSigner(signerOrPrivateKey: SignerInput): MessageSigner {
  if (typeof signerOrPrivateKey === "string") {
    return createSignerFromPrivateKey(signerOrPrivateKey);
  }

  return signerOrPrivateKey;
}

export function hashAbiEncodedPayload(
  types: readonly string[],
  values: readonly unknown[],
): HexString {
  return keccak256(ABI_CODER.encode(types, values)) as HexString;
}

export async function signMessageHash(
  messageHash: string,
  signerOrPrivateKey: SignerInput,
): Promise<string> {
  const signer = resolveSigner(signerOrPrivateKey);
  return signer.signMessage(getBytes(messageHash));
}

export function hashCreateProjectRequest(
  payload: CreateProjectRequestPayload,
  options: CreateProjectSignatureOptions,
): HexString {
  const signatureDomain = options.signatureDomain ?? PVIUM_SIGNATURE_DOMAIN;

  return hashAbiEncodedPayload(
    [
      "bytes32",
      "string",
      "string",
      "string",
      "address",
      "address",
      "address",
      "address",
      "uint256",
      "uint256",
      "uint256",
      "uint256",
      "uint256",
      "uint256",
    ],
    [
      signatureDomain,
      payload.app,
      payload.projectId,
      payload.metadata,
      payload.tokenAddress,
      payload.refundAddress,
      payload.appFeeAddress,
      payload.appAdminAddress,
      payload.appFeeBps,
      payload.disputeWindowSeconds,
      payload.lockDuration,
      payload.minimumBalancePerVendor,
      options.pviumFeeBps,
      options.chainId,
    ],
  );
}

export async function signCreateProjectRequest(
  payload: CreateProjectRequestPayload,
  signerOrPrivateKey: SignerInput,
  options: CreateProjectSignatureOptions,
): Promise<string> {
  const messageHash = hashCreateProjectRequest(payload, options);
  return signMessageHash(messageHash, signerOrPrivateKey);
}

export function hashCreateProjectAttestation(
  appSignature: string,
  chainId: Numeric,
  signatureDomain: HexString = PVIUM_SIGNATURE_DOMAIN,
): HexString {
  return hashAbiEncodedPayload(
    ["bytes32", "bytes", "uint256"],
    [signatureDomain, appSignature, chainId],
  );
}

export async function signCreateProjectAttestation(
  appSignature: string,
  signerOrPrivateKey: SignerInput,
  chainId: Numeric,
  signatureDomain: HexString = PVIUM_SIGNATURE_DOMAIN,
): Promise<string> {
  const messageHash = hashCreateProjectAttestation(
    appSignature,
    chainId,
    signatureDomain,
  );
  return signMessageHash(messageHash, signerOrPrivateKey);
}

export function hashCreateClaimRequest(
  payload: CreateClaimRequestPayload,
): HexString {
  return hashAbiEncodedPayload(
    [
      "string",
      "string",
      "bytes32",
      "address",
      "uint256",
      "uint256",
      "uint256",
      "uint256",
    ],
    [
      payload.app,
      payload.projectId,
      payload.claimId,
      payload.receiver,
      payload.amount,
      payload.claimableAfter,
      payload.claimDeadline,
      payload.nonce,
    ],
  );
}

export async function signCreateClaimRequest(
  payload: CreateClaimRequestPayload,
  signerOrPrivateKey: SignerInput,
): Promise<string> {
  const messageHash = hashCreateClaimRequest(payload);
  return signMessageHash(messageHash, signerOrPrivateKey);
}

export function hashFinalizeClaimRequest(
  claims: readonly FinalizeClaimRequestPayload[],
  chainId: Numeric,
): HexString {
  let dataPacked: HexString = "0x";

  for (const claim of claims) {
    dataPacked = concat([
      dataPacked,
      toUtf8Bytes(claim.app),
      toUtf8Bytes(claim.projectId),
      claim.claimId,
    ]) as HexString;
  }

  return keccak256(concat([dataPacked, toBeHex(chainId, 32)])) as HexString;
}

export async function signFinalizeClaimRequest(
  claims: readonly FinalizeClaimRequestPayload[],
  signerOrPrivateKey: SignerInput,
  chainId: Numeric,
): Promise<string> {
  const messageHash = hashFinalizeClaimRequest(claims, chainId);
  return signMessageHash(messageHash, signerOrPrivateKey);
}

export function hashRelayedCallRequest(
  payload: RelayedCallRequestPayload,
): HexString {
  return hashAbiEncodedPayload(
    ["string", "string", "bytes", "uint256", "uint256"],
    [
      payload.appId,
      payload.projectId,
      payload.payload,
      payload.nonce,
      payload.chainId,
    ],
  );
}

export async function signRelayedCallRequest(
  payload: RelayedCallRequestPayload,
  signerOrPrivateKey: SignerInput,
): Promise<string> {
  const messageHash = hashRelayedCallRequest(payload);
  return signMessageHash(messageHash, signerOrPrivateKey);
}

export function hashDisputeRequest(
  claimId: string,
  chainId: Numeric,
): HexString {
  return hashAbiEncodedPayload(["bytes32", "uint256"], [claimId, chainId]);
}

export async function signDisputeRequest(
  claimId: string,
  signerOrPrivateKey: SignerInput,
  chainId: Numeric,
): Promise<string> {
  const messageHash = hashDisputeRequest(claimId, chainId);
  return signMessageHash(messageHash, signerOrPrivateKey);
}

export function hashResolveDisputeRequest(
  payload: ResolveDisputeRequestPayload,
): HexString {
  return hashAbiEncodedPayload(
    ["bytes32", "bool", "uint256"],
    [payload.claimId, payload.approved, payload.chainId],
  );
}

export async function signResolveDisputeRequest(
  payload: ResolveDisputeRequestPayload,
  signerOrPrivateKey: SignerInput,
): Promise<string> {
  const messageHash = hashResolveDisputeRequest(payload);
  return signMessageHash(messageHash, signerOrPrivateKey);
}

export function signatureDomainFromText(message: string): HexString {
  return id(message) as HexString;
}
