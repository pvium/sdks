const test = require("node:test");
const assert = require("node:assert/strict");
const {
  AbiCoder,
  Wallet,
  concat,
  getBytes,
  keccak256,
  toBeHex,
  toUtf8Bytes,
  verifyMessage,
} = require("ethers");

const {
  PVIUM_SIGNATURE_DOMAIN,
  createSignerFromPrivateKey,
  hashCreateClaimRequest,
  hashCreateProjectRequest,
  hashDisputeRequest,
  hashFinalizeClaimRequest,
  hashRelayedCallRequest,
  hashResolveDisputeRequest,
  signCreateClaimRequest,
  signCreateProjectRequest,
  signDisputeRequest,
  signFinalizeClaimRequest,
  signRelayedCallRequest,
  signResolveDisputeRequest,
} = require("../dist/index.js");

const ABI_CODER = AbiCoder.defaultAbiCoder();

const TEST_PRIVATE_KEY =
  "0x59c6995e998f97a5a0044976f0d7f3f6f8f53f6a2046baf4f01cb4f1f6bcb58f";
const TEST_ADDRESS = new Wallet(TEST_PRIVATE_KEY).address;
const CHAIN_ID = 84532n;

test("createSignerFromPrivateKey returns a usable signer", async () => {
  const signer = createSignerFromPrivateKey(TEST_PRIVATE_KEY);
  assert.equal(signer.address, TEST_ADDRESS);

  const messageHash = keccak256(toUtf8Bytes("hello-pvium"));
  const signature = await signer.signMessage(getBytes(messageHash));
  const recovered = verifyMessage(getBytes(messageHash), signature);

  assert.equal(recovered, TEST_ADDRESS);
});

test("signCreateProjectRequest matches manual hash encoding", async () => {
  const payload = {
    app: "test-app",
    projectId: "project-001",
    metadata: "ipfs://QmTest",
    tokenAddress: "0x0000000000000000000000000000000000000001",
    refundAddress: "0x0000000000000000000000000000000000000002",
    appFeeAddress: "0x0000000000000000000000000000000000000003",
    appAdminAddress: "0x0000000000000000000000000000000000000004",
    appFeeBps: 200,
    disputeWindowSeconds: 259200,
    lockDuration: 7776000,
    minimumBalancePerVendor: 100000000n,
  };

  const options = {
    pviumFeeBps: 100,
    chainId: CHAIN_ID,
  };

  const expectedHash = keccak256(
    ABI_CODER.encode(
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
        PVIUM_SIGNATURE_DOMAIN,
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
    ),
  );

  const helperHash = hashCreateProjectRequest(payload, options);
  assert.equal(helperHash, expectedHash);

  const signature = await signCreateProjectRequest(
    payload,
    TEST_PRIVATE_KEY,
    options,
  );
  const recovered = verifyMessage(getBytes(helperHash), signature);

  assert.equal(recovered, TEST_ADDRESS);
});

test("signCreateClaimRequest matches manual hash encoding", async () => {
  const payload = {
    app: "test-app",
    projectId: "project-001",
    claimId:
      "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    receiver: "0x0000000000000000000000000000000000000005",
    amount: 100000000n,
    claimableAfter: 1700000000,
    claimDeadline: 0,
    nonce: 1,
  };

  const expectedHash = keccak256(
    ABI_CODER.encode(
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
    ),
  );

  const helperHash = hashCreateClaimRequest(payload);
  assert.equal(helperHash, expectedHash);

  const signature = await signCreateClaimRequest(payload, TEST_PRIVATE_KEY);
  const recovered = verifyMessage(getBytes(helperHash), signature);

  assert.equal(recovered, TEST_ADDRESS);
});

test("signFinalizeClaimRequest hashes packed batch payload like contract tests", async () => {
  const claims = [
    {
      app: "test-app",
      projectId: "usdc-project",
      claimId:
        "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    },
    {
      app: "test-app",
      projectId: "usdt-project",
      claimId:
        "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    },
  ];

  let dataPacked = "0x";
  for (const claim of claims) {
    dataPacked = concat([
      dataPacked,
      toUtf8Bytes(claim.app),
      toUtf8Bytes(claim.projectId),
      claim.claimId,
    ]);
  }

  const expectedHash = keccak256(concat([dataPacked, toBeHex(CHAIN_ID, 32)]));

  const helperHash = hashFinalizeClaimRequest(claims, CHAIN_ID);
  assert.equal(helperHash, expectedHash);

  const signature = await signFinalizeClaimRequest(
    claims,
    TEST_PRIVATE_KEY,
    CHAIN_ID,
  );
  const recovered = verifyMessage(getBytes(helperHash), signature);

  assert.equal(recovered, TEST_ADDRESS);
});

test("relayed/dispute/resolve helpers match manual encoding and accept signer instance", async () => {
  const signer = createSignerFromPrivateKey(TEST_PRIVATE_KEY);

  const relayedPayload = {
    appId: "test-app",
    projectId: "project-001",
    payload: ABI_CODER.encode(["string", "address[]"], ["addVendors", []]),
    nonce: 2,
    chainId: CHAIN_ID,
  };

  const expectedRelayedHash = keccak256(
    ABI_CODER.encode(
      ["string", "string", "bytes", "uint256", "uint256"],
      [
        relayedPayload.appId,
        relayedPayload.projectId,
        relayedPayload.payload,
        relayedPayload.nonce,
        relayedPayload.chainId,
      ],
    ),
  );

  const relayedHash = hashRelayedCallRequest(relayedPayload);
  assert.equal(relayedHash, expectedRelayedHash);

  const relayedSig = await signRelayedCallRequest(relayedPayload, signer);
  assert.equal(verifyMessage(getBytes(relayedHash), relayedSig), TEST_ADDRESS);

  const claimId =
    "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd";

  const expectedDisputeHash = keccak256(
    ABI_CODER.encode(["bytes32", "uint256"], [claimId, CHAIN_ID]),
  );
  const disputeHash = hashDisputeRequest(claimId, CHAIN_ID);
  assert.equal(disputeHash, expectedDisputeHash);

  const disputeSig = await signDisputeRequest(claimId, signer, CHAIN_ID);
  assert.equal(verifyMessage(getBytes(disputeHash), disputeSig), TEST_ADDRESS);

  const resolvePayload = {
    claimId,
    approved: true,
    chainId: CHAIN_ID,
  };

  const expectedResolveHash = keccak256(
    ABI_CODER.encode(["bytes32", "bool", "uint256"], [claimId, true, CHAIN_ID]),
  );
  const resolveHash = hashResolveDisputeRequest(resolvePayload);
  assert.equal(resolveHash, expectedResolveHash);

  const resolveSig = await signResolveDisputeRequest(resolvePayload, signer);
  assert.equal(verifyMessage(getBytes(resolveHash), resolveSig), TEST_ADDRESS);
});
