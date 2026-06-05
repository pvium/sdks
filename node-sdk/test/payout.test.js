const assert = require("node:assert/strict");
const test = require("node:test");
const {
  AbiCoder,
  Wallet,
  getAddress,
  keccak256,
  parseUnits,
  recoverAddress,
  solidityPacked,
} = require("ethers");

const {
  PviumSdk,
  PayoutCurrency,
  PayoutFinalization,
  PayoutIntent,
  computeScheduledPayoutHash,
  computeSigningKeyAuthorizationHash,
  createPayoutNonce,
  generateInstantPayoutHash,
} = require("../dist/index.js");

const ABI_CODER = AbiCoder.defaultAbiCoder();

test("PviumSdk.init exposes payout service", () => {
  const sdk = PviumSdk.init({
    baseUrl: "https://api.example.test/v1",
    apiKey: "app_key",
  });

  assert.equal(typeof sdk.payout.create, "function");
  assert.equal(typeof sdk.payout.finalize, "function");
  assert.equal(typeof sdk.payout.addPayments, "function");
  assert.equal(typeof sdk.payout.addRecipients, "function");
  assert.equal(typeof sdk.payout.resolveRecipients, "function");
  assert.equal(typeof sdk.payout.removePayments, "function");
  assert.equal(typeof sdk.payout.deletePayment, "function");
  assert.equal(typeof sdk.payout.updatePayment, "function");
  assert.equal(typeof sdk.payout.editPayment, "function");
  assert.equal(typeof sdk.payout.revokeInvite, "function");
  assert.equal(typeof sdk.payout.revokeInviteRoot, "function");
  assert.equal(typeof sdk.payout.delete, "function");
  assert.equal(typeof sdk.payout.listInvites, "function");
  assert.equal(typeof sdk.payout.listPayments, "function");
});

test("createPayoutNonce generates a 16-byte hex nonce", () => {
  const nonce = createPayoutNonce();
  assert.match(nonce, /^0x[0-9a-f]{32}$/);
});

test("payout.authorizeSigningKey signs the raw ECDSA authorization digest", async () => {
  const sdk = PviumSdk.init({
    baseUrl: "https://api.example.test/v1",
    apiKey: "app_key",
  });
  const privateKey =
    "0x59c6995e998f97a5a004497e5daaaa853d873599e62e568a0a7d3a57c5fd8d0d";
  const authorizer = new Wallet(privateKey);
  const signingKeyAddress = "0x0000000000000000000000000000000000000002";
  const escrowBatchId =
    "0x1111111111111111111111111111111111111111111111111111111111111111";

  const authorization = await sdk.payout.authorizeSigningKey(
    escrowBatchId,
    signingKeyAddress,
    "ethereum",
    {
      transactionMax: "1000000",
      totalMax: "5000000",
      expiration: 1777488000,
      timestamp: 1777487451,
    },
    privateKey,
  );

  const expectedHash = keccak256(
    solidityPacked(
      ["bytes32", "address", "uint256", "uint256", "uint256", "uint256"],
      [
        escrowBatchId,
        getAddress(signingKeyAddress),
        1000000n,
        5000000n,
        1777488000n,
        1777487451n,
      ],
    ),
  );

  assert.equal(authorization.authMessageHash, expectedHash);
  assert.equal(
    computeSigningKeyAuthorizationHash({
      batchHash: escrowBatchId,
      signingKey: signingKeyAddress,
      transactionMax: "1000000",
      totalMax: "5000000",
      expiration: 1777488000,
      timestamp: 1777487451,
    }).authMessageHash,
    expectedHash,
  );
  assert.equal(authorization.signingKey, getAddress(signingKeyAddress));
  assert.equal(authorization.timestamp, 1777487451n);
  assert.equal(
    getAddress(recoverAddress(authorization.authMessageHash, authorization.signature)),
    authorizer.address,
  );
});

function createMockSdk(responseBody = { meta: { statusCode: 200, success: true } }) {
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    clientId: "client_123",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      return new Response(JSON.stringify(responseBody), {
        status: responseBody.meta?.statusCode || 200,
        headers: { "content-type": "application/json" },
      });
    },
  });

  return { sdk, requests };
}

test("payout.create returns a payout intent with envelope compatibility", async () => {
  const privateKey =
    "0x59c6995e998f97a5a004497e5daaaa853d873599e62e568a0a7d3a57c5fd8d0d";
  const responses = [
    {
      meta: { statusCode: 201, success: true },
      data: {
        id: "batch_1",
        chain: "base",
        paymentType: "Instant",
        nonce: "0x11111111111111111111111111111111",
        complianceMode: "Open",
        payments: [
          {
            receiver: "0x0000000000000000000000000000000000000001",
            amount: 25,
            token: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
            decimals: 6,
          },
        ],
      },
    },
    {
      meta: { statusCode: 200, success: true },
      data: {
        id: "batch_1",
        chain: "base",
        paymentType: "Instant",
        nonce: "0x11111111111111111111111111111111",
        batchDataHash:
          "0x2222222222222222222222222222222222222222222222222222222222222222",
      },
    },
  ];
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    clientId: "client_123",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });

  const payout = await sdk.payout.create({
    type: "Instant",
    chain: "base",
    name: "Creator payroll",
    payments: [
      {
        receiver: "0x0000000000000000000000000000000000000001",
        amount: 25,
        token: "usdc",
      },
    ],
  });

  assert.ok(payout instanceof PayoutIntent);
  assert.equal(payout.id, "batch_1");
  assert.equal(payout.data.id, "batch_1");
  assert.equal(payout.meta.success, true);
  assert.equal(typeof payout.finalize, "function");

  const finalized = await payout.finalize(privateKey, {
    timestamp: 1777487451,
  });

  assert.ok(finalized instanceof PayoutFinalization);
  assert.equal(finalized.payout.id, "batch_1");
  assert.equal(finalized.data.payout.id, "batch_1");
  assert.equal(finalized.meta.success, true);
  assert.equal(requests[1].init.method, "PATCH");
});

test("payout intent addPayments proxies through the payout service", async () => {
  const responses = [
    {
      meta: { statusCode: 201, success: true },
      data: { id: "batch_1", chain: "base", paymentType: "Instant" },
    },
    {
      meta: { statusCode: 200, success: true },
      data: { id: "batch_1", chain: "base", paymentType: "Instant" },
    },
  ];
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });

  const payoutIntent = await sdk.payout.create({
    type: "Instant",
    chain: "base",
    name: "Creator payroll",
    payments: [],
  });

  const updated = await payoutIntent.addPayments([
    {
      receiver: "0x0000000000000000000000000000000000000001",
      amount: 25,
      token: "usdc",
    },
  ]);

  assert.ok(updated instanceof PayoutIntent);
  assert.equal(requests[1].url, "http://localhost:4005/v1/batch-payments/batch_1/payments");
  assert.equal(requests[1].init.method, "POST");
});

test("payout intent proxies batch lifecycle endpoints", async () => {
  const responses = [
    {
      meta: { statusCode: 201, success: true },
      data: { id: "batch_1", chain: "base", paymentType: "Instant" },
    },
    { meta: { statusCode: 200, success: true }, data: { id: 7 } },
    { meta: { statusCode: 200, success: true } },
    { meta: { statusCode: 200, success: true } },
    { meta: { statusCode: 200, success: true } },
  ];
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });

  const payoutIntent = await sdk.payout.create({
    type: "Instant",
    chain: "base",
    name: "Creator payroll",
    payments: [],
  });

  await payoutIntent.editPayment(7, { amount: 50, memo: "updated" });
  await payoutIntent.revokeInvite("invite_1");
  await payoutIntent.revokeInviteRoot("root_1");
  await payoutIntent.delete();

  assert.equal(requests[1].url, "http://localhost:4005/v1/batch-payments/batch_1/payments/7");
  assert.equal(requests[1].init.method, "PATCH");
  assert.deepEqual(JSON.parse(requests[1].init.body), {
    amount: 50,
    memo: "updated",
  });
  assert.equal(requests[2].url, "http://localhost:4005/v1/batch-payments/batch_1/invites/invite_1");
  assert.equal(requests[2].init.method, "DELETE");
  assert.equal(requests[3].url, "http://localhost:4005/v1/batch-payments/batch_1/invite-roots/root_1");
  assert.equal(requests[3].init.method, "DELETE");
  assert.equal(requests[4].url, "http://localhost:4005/v1/batch-payments/batch_1");
  assert.equal(requests[4].init.method, "DELETE");
});

test("payout intent deletePayment deletes a single payment id", async () => {
  const responses = [
    {
      meta: { statusCode: 201, success: true },
      data: { id: "batch_1", chain: "base", paymentType: "Instant" },
    },
    { meta: { statusCode: 200, success: true } },
  ];
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });

  const payoutIntent = await sdk.payout.create({
    type: "Instant",
    chain: "base",
    name: "Creator payroll",
    payments: [],
  });

  await payoutIntent.deletePayment(7);

  assert.equal(requests[1].url, "http://localhost:4005/v1/batch-payments/batch_1/payments");
  assert.equal(requests[1].init.method, "DELETE");
  assert.deepEqual(JSON.parse(requests[1].init.body), { paymentIds: [7] });
});

test("payout intent listPayments calls paginated payments endpoint", async () => {
  const responses = [
    {
      meta: { statusCode: 201, success: true },
      data: { id: "batch_1", chain: "base", paymentType: "Instant" },
    },
    {
      meta: {
        statusCode: 200,
        success: true,
        pagination: { totalCount: 1, perPage: 50, current: 1 },
      },
      data: [{ id: 7, receiver: "0x0000000000000000000000000000000000000001" }],
    },
  ];
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });

  const payoutIntent = await sdk.payout.create({
    type: "Instant",
    chain: "base",
    name: "Creator payroll",
    payments: [],
  });

  const payments = await payoutIntent.listPayments({ page: 1, perPage: 50 });

  assert.equal(
    requests[1].url,
    "http://localhost:4005/v1/batch-payments/batch_1/payments?page=1&perPage=50",
  );
  assert.equal(requests[1].init.method, "GET");
  assert.equal(payments.data[0].id, 7);
});

test("payout.addRecipients posts recipient identities to open payees endpoint", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 201, success: true },
    data: { added: [], errors: [] },
  });

  await sdk.payout.addRecipients("batch 1", [
    {
      identityType: "github",
      identityValue: "@feminefa",
      defaultPayoutAmount: 25,
      memo: "github payout",
    },
  ]);

  assert.equal(
    requests[0].url,
    "http://localhost:4005/v1/batch-payments/batch%201/open-payees",
  );
  assert.equal(requests[0].init.method, "POST");
  assert.deepEqual(JSON.parse(requests[0].init.body), {
    recipients: [
      {
        identityType: "github",
        identityValue: "@feminefa",
        defaultPayoutAmount: 25,
        memo: "github payout",
      },
    ],
  });
});

test("payout.resolveRecipients posts recipient identities to resolver endpoint", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 200, success: true },
    data: { resolved: [], errors: [] },
  });

  await sdk.payout.resolveRecipients("batch_1", [
    { identityType: "email", identityValue: "payee@example.com" },
  ]);

  assert.equal(
    requests[0].url,
    "http://localhost:4005/v1/batch-payments/batch_1/resolve-recipients",
  );
  assert.equal(requests[0].init.method, "POST");
  assert.deepEqual(JSON.parse(requests[0].init.body), {
    recipients: [{ identityType: "email", identityValue: "payee@example.com" }],
  });
});

test("payout.removePayments deletes payment ids from payout", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 200, success: true },
  });

  await sdk.payout.removePayments("batch_1", ["1", 2]);

  assert.equal(
    requests[0].url,
    "http://localhost:4005/v1/batch-payments/batch_1/payments",
  );
  assert.equal(requests[0].init.method, "DELETE");
  assert.deepEqual(JSON.parse(requests[0].init.body), {
    paymentIds: [1, 2],
  });
});

test("payout.create maps direct scheduled payout currency and date into metadata", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 201, success: true },
    data: { id: "batch_1", paymentType: "Scheduled" },
  });

  await sdk.payout.create({
    type: "Scheduled",
    chain: "base",
    name: "March creator payouts",
    payoutCurrency: PayoutCurrency.USDC,
    scheduleDate: 1777488000,
    metadata: { campaign: "march" },
    payments: [
      {
        receiver: "0x0000000000000000000000000000000000000001",
        amount: "25",
      },
    ],
  });

  const payload = JSON.parse(requests[0].init.body);
  assert.equal(
    payload.metadata.payoutCurrency,
    "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
  );
  assert.equal(payload.metadata.scheduledDate, 1777488000);
  assert.equal(payload.metadata.campaign, "march");
  assert.equal(
    payload.payments[0].token,
    "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
  );
  assert.equal(payload.payments[0].decimals, 6);
});

test("payout.create maps direct currency from supported network config", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 201, success: true },
    data: { id: "batch_1", paymentType: "Scheduled" },
  });

  await sdk.payout.create({
    type: "Scheduled",
    chain: "solana-testnet",
    payoutCurrency: "usdt",
    payments: [
      {
        receiver: "recipient",
        amount: "25",
      },
    ],
  });

  const payload = JSON.parse(requests[0].init.body);
  assert.equal(
    payload.metadata.payoutCurrency,
    "SPFPKg9zeE7ReqW3j9QU6p7XhPP8JDU5Dx4fgrTwVyF",
  );
  assert.equal(
    payload.payments[0].token,
    "SPFPKg9zeE7ReqW3j9QU6p7XhPP8JDU5Dx4fgrTwVyF",
  );
  assert.equal(payload.payments[0].decimals, 6);
});

test("payout.create resolves payment token symbols into configured addresses", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 201, success: true },
    data: { id: "batch_1", paymentType: "Scheduled" },
  });

  await sdk.payout.create({
    type: "Scheduled",
    chain: "base",
    payments: [
      {
        receiver: "0x0000000000000000000000000000000000000001",
        amount: "25",
        token: "usdt",
      },
      {
        receiver: "0x0000000000000000000000000000000000000002",
        amount: "10",
        tokenSymbol: "usdc",
      },
    ],
  });

  const payload = JSON.parse(requests[0].init.body);
  assert.equal(
    payload.payments[0].token,
    "0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2",
  );
  assert.equal(payload.payments[0].decimals, 6);
  assert.equal(
    payload.payments[1].token,
    "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
  );
  assert.equal(payload.payments[1].decimals, 6);
  assert.equal(payload.payments[1].tokenSymbol, undefined);
});

test("payout.create rejects payment token mismatches when payoutCurrency is provided", async () => {
  const { sdk } = createMockSdk({
    meta: { statusCode: 201, success: true },
    data: { id: "batch_1", paymentType: "Scheduled" },
  });

  await assert.rejects(
    () =>
      sdk.payout.create({
        type: "Scheduled",
        chain: "base",
        payoutCurrency: "usdc",
        payments: [
          {
            receiver: "0x0000000000000000000000000000000000000001",
            amount: "25",
            token: "usdt",
          },
        ],
      }),
    /Payment token must match payoutCurrency/,
  );
});

test("payout.create validates explicit payment token addresses against supported config", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 201, success: true },
    data: { id: "batch_1", paymentType: "Scheduled" },
  });

  await sdk.payout.create({
    type: "Scheduled",
    chain: "base",
    payments: [
      {
        receiver: "0x0000000000000000000000000000000000000001",
        amount: "25",
        token: "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
      },
    ],
  });

  let payload = JSON.parse(requests[0].init.body);
  assert.equal(
    payload.payments[0].token,
    "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
  );
  assert.equal(payload.payments[0].decimals, 6);

  await assert.rejects(
    () =>
      sdk.payout.create({
        type: "Scheduled",
        chain: "base",
        payments: [
          {
            receiver: "0x0000000000000000000000000000000000000001",
            amount: "25",
            token: "0x0000000000000000000000000000000000000009",
          },
        ],
      }),
    /Payment token .* is not supported on chain base/,
  );
});

test("payout.create injects milestone commitment metadata", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 201, success: true },
    data: { id: "batch_1", paymentType: "Scheduled", isCommitment: true },
  });

  await sdk.payout.create({
    type: "Milestone",
    chain: "base",
    name: "Website build",
    payoutCurrency: "usdc",
    metadata: {
      milestones: [
        {
          name: "Design approval",
          amount: 500,
          dueDate: "2026-07-01T00:00:00.000Z",
          status: "pending",
        },
      ],
    },
  });

  const payload = JSON.parse(requests[0].init.body);
  assert.equal(payload.paymentType, "Scheduled");
  assert.equal(payload.isCommitment, true);
  assert.equal(payload.metadata.commitmentType, "milestone");
});

test("payout.finalize accepts a message signing function for instant payouts", async () => {
  const requests = [];
  const responses = [
    {
      meta: { statusCode: 200, success: true },
      data: {
        id: "120bdabb-5790-415c-ae75-c2fca1cc5232",
        chain: "base",
        paymentType: "Instant",
        complianceMode: "Open",
        nonce: "0x1234",
        app: { clientId: "app_test" },
        payments: [
          {
            receiver: "0x0000000000000000000000000000000000000001",
            amount: "1",
            token: "0x0000000000000000000000000000000000000002",
            decimals: 6,
            memo: "",
          },
        ],
      },
    },
    {
      meta: { statusCode: 200, success: true },
      data: {
        id: "120bdabb-5790-415c-ae75-c2fca1cc5232",
        chain: "base",
        paymentType: "Instant",
        batchDataHash: "0xabc",
      },
    },
  ];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });
  const signedMessages = [];

  await sdk.payout.finalize(
    "120bdabb-5790-415c-ae75-c2fca1cc5232",
    async (message) => {
      signedMessages.push(message);
      return "0xsigned";
    },
    {
      signerAddress: "0x0000000000000000000000000000000000000003",
      timestamp: 123,
    },
  );

  assert.equal(signedMessages.length, 1);
  assert.ok(signedMessages[0].startsWith("PVIUM_SIGNED_BATCH:app_test:"));
  assert.equal(requests[1].init.method, "PATCH");
  const payload = JSON.parse(requests[1].init.body);
  assert.equal(
    payload.signer,
    "0x0000000000000000000000000000000000000003",
  );
  assert.equal(
    payload.batchSignature,
    "123:0x0000000000000000000000000000000000000003:0xsigned",
  );
});

test("payout.finalize chain override can skip scheduled funding signature for solana", async () => {
  const requests = [];
  const responses = [
    {
      meta: { statusCode: 200, success: true },
      data: {
        id: "120bdabb-5790-415c-ae75-c2fca1cc5232",
        chain: "base",
        paymentType: "Scheduled",
        complianceMode: "Open",
        metadata: {
          payoutCurrency: "0x0000000000000000000000000000000000000002",
          gracePeriod: 0,
          disapprovalDeadline: 0,
          scheduledDate: 0,
        },
        app: { clientId: "app_test" },
        payments: [
          {
            receiver: "0x0000000000000000000000000000000000000001",
            amount: "1",
            token: "0x0000000000000000000000000000000000000002",
            decimals: 6,
            memo: "",
          },
        ],
      },
    },
    {
      meta: { statusCode: 200, success: true },
      data: {
        id: "batch_1",
        chain: "solana",
        paymentType: "Scheduled",
        merkleRoot: "0xabc",
      },
    },
  ];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });

  await sdk.payout.finalize(
    "120bdabb-5790-415c-ae75-c2fca1cc5232",
    async () => "signed",
    {
      chain: "solana",
      chainId: 1,
      signerAddress: "0x0000000000000000000000000000000000000003",
      timestamp: 123,
    },
  );

  const payload = JSON.parse(requests[1].init.body);
  assert.equal(payload.fundingSignature, undefined);
  assert.equal(payload.batchSignature.endsWith(":signed"), true);
});

test("payout.finalize supports separate finalize and funding signers", async () => {
  const requests = [];
  const calls = [];
  const responses = [
    {
      meta: { statusCode: 200, success: true },
      data: {
        id: "120bdabb-5790-415c-ae75-c2fca1cc5232",
        chain: "base",
        paymentType: "Scheduled",
        complianceMode: "Open",
        metadata: {
          payoutCurrency: "0x0000000000000000000000000000000000000002",
          gracePeriod: 0,
          disapprovalDeadline: 0,
          scheduledDate: 0,
        },
        app: { clientId: "app_test" },
        payments: [
          {
            receiver: "0x0000000000000000000000000000000000000001",
            amount: "1",
            token: "0x0000000000000000000000000000000000000002",
            decimals: 6,
            memo: "",
          },
        ],
      },
    },
    {
      meta: { statusCode: 200, success: true },
      data: {
        id: "120bdabb-5790-415c-ae75-c2fca1cc5232",
        chain: "base",
        paymentType: "Scheduled",
        merkleRoot: "0xabc",
      },
    },
  ];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });

  await sdk.payout.finalize(
    "120bdabb-5790-415c-ae75-c2fca1cc5232",
    {
      chain: "ethereum",
      signerAddress: "0x0000000000000000000000000000000000000003",
      signMessage: async () => {
        throw new Error("fallback signMessage should not be called");
      },
      signFinalize: async (message) => {
        calls.push(`finalize:${message}`);
        return "finalize-signature";
      },
      signFunding: async (digest) => {
        calls.push(`funding:${digest}`);
        return "funding-signature";
      },
    },
    {
      chain: "base",
      chainId: 8453,
      timestamp: 123,
    },
  );

  assert.equal(calls.length, 2);
  assert.ok(calls[0].startsWith("finalize:PVIUM_SIGNED_SCHEDULE:"));
  assert.ok(calls[1].startsWith("funding:0x"));
  const payload = JSON.parse(requests[1].init.body);
  assert.equal(payload.batchSignature.endsWith(":finalize-signature"), true);
  assert.equal(payload.fundingSignature, "funding-signature");
});

test("payout.addPayments creates and signs an escrow child scheduled payout when given an escrow payout and private key", async () => {
  const requests = [];
  const privateKey =
    "0x59c6995e998f97a5a004497e5daaaa853d873599e62e568a0a7d3a57c5fd8d0d";
  const escrowBatch = {
    id: "7a6ca76d-77f7-4c0e-9da9-c64f1cb18a1f",
    chain: "base",
    paymentType: "Escrow",
    status: "funded",
    complianceMode: "Open",
    name: "Creator escrow",
    batchHash:
      "0x1111111111111111111111111111111111111111111111111111111111111111",
    metadata: {
      payoutCurrency: "0x0000000000000000000000000000000000000002",
    },
    app: { clientId: "app_test" },
  };
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      return new Response(
        JSON.stringify({
          meta: { statusCode: 201, success: true },
          data: {
            id: "22222222-2222-4222-8222-222222222222",
            chain: "base",
            paymentType: "Scheduled",
            escrowBatch: escrowBatch.id,
            merkleRoot:
              "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
          },
        }),
        {
          status: 201,
          headers: { "content-type": "application/json" },
        },
      );
    },
  });

  await sdk.payout.addPayments(escrowBatch, {
    payments: [
      {
        receiver: "0x0000000000000000000000000000000000000001",
        amount: "25",
        decimals: 6,
        memo: "escrow work",
      },
    ],
    signer: privateKey,
    finalizeOptions: {
      id: "22222222-2222-4222-8222-222222222222",
      timestamp: 1777487451,
      claimDate: 1777488000,
    },
  });

  assert.equal(requests.length, 1);
  assert.equal(requests[0].url, "http://localhost:4005/v1/batch-payments");
  assert.equal(requests[0].init.method, "POST");

  const payload = JSON.parse(requests[0].init.body);
  assert.equal(payload.id, "22222222-2222-4222-8222-222222222222");
  assert.equal(payload.paymentType, "Scheduled");
  assert.equal(payload.escrowBatch, escrowBatch.id);
  assert.match(payload.batchSignature, /^1777487451:0x[0-9a-f]{40}:0x/i);
  assert.match(payload.fundingSignature, /^0x[0-9a-f]+$/i);
  assert.match(payload.batchHash, /^0x[0-9a-f]{64}$/i);
  assert.match(payload.batchDataHash, /^0x[0-9a-f]{64}$/i);
  assert.match(payload.merkleRoot, /^0x[0-9a-f]{64}$/i);
  assert.deepEqual(
    payload.proofs[0].receiver,
    "0x0000000000000000000000000000000000000001",
  );
  assert.equal(payload.metadata.escrowBatch, escrowBatch.id);
  assert.equal(payload.metadata.escrowBatchHash, escrowBatch.batchHash);
  assert.equal(payload.metadata.scheduledDate, 1777488000);
  assert.equal(payload.payments[0].claimDate, 1777488000);
  assert.equal(
    payload.payments[0].token,
    "0x0000000000000000000000000000000000000002",
  );
});

test("payout.addPayments rejects escrow payouts without a signer", async () => {
  const { sdk } = createMockSdk();

  await assert.rejects(
    () =>
      sdk.payout.addPayments(
        {
          id: "7a6ca76d-77f7-4c0e-9da9-c64f1cb18a1f",
          chain: "base",
          paymentType: "Escrow",
          status: "funded",
          batchHash:
            "0x1111111111111111111111111111111111111111111111111111111111111111",
          metadata: {
            payoutCurrency: "0x0000000000000000000000000000000000000002",
          },
          app: { clientId: "app_test" },
        },
        [
          {
            receiver: "0x0000000000000000000000000000000000000001",
            amount: "25",
            decimals: 6,
          },
        ],
      ),
    /signer or private key is required/,
  );
});

test("generateInstantPayoutHash matches manual ABI encoding", () => {
  const nonce = "0x1234abcd";
  const payments = [
    {
      receiver: "0x0000000000000000000000000000000000000001",
      amount: "12.5",
      token: "0x0000000000000000000000000000000000000002",
      decimals: 6,
      memo: "first",
    },
    {
      receiver: "0x0000000000000000000000000000000000000003",
      amount: 1,
      token: "0x0000000000000000000000000000000000000002",
      decimals: 6,
    },
  ];

  const expected = keccak256(
    ABI_CODER.encode(
      [
        "bytes",
        "tuple(address receiver, uint256 amount, address token, string memo)[]",
      ],
      [
        nonce,
        [
          {
            receiver: payments[0].receiver,
            amount: parseUnits("12.5", 6),
            token: payments[0].token,
            memo: "first",
          },
          {
            receiver: payments[1].receiver,
            amount: parseUnits("1", 6),
            token: payments[1].token,
            memo: "",
          },
        ],
      ],
    ),
  );

  assert.equal(generateInstantPayoutHash(payments, nonce), expected);
});

test("computeScheduledPayoutHash matches backend scheduled hash formula", () => {
  const payoutId = "120bdabb-5790-415c-ae75-c2fca1cc5232";
  const fundingToken = "0x0000000000000000000000000000000000000002";
  const gracePeriod = 86400;
  const disapprovalDeadline = 3600;
  const timestamp = 1777487451;
  const chainId = 8453;
  const payoutIdBytes32 = `0x${payoutId.replace(/-/g, "").padEnd(64, "0")}`;

  const expected = keccak256(
    ABI_CODER.encode(
      ["bytes32", "address", "uint256", "uint256", "uint256", "uint256"],
      [
        payoutIdBytes32,
        fundingToken,
        gracePeriod,
        disapprovalDeadline,
        timestamp,
        chainId,
      ],
    ),
  );

  assert.equal(
    computeScheduledPayoutHash({
      payoutId,
      fundingToken,
      gracePeriod,
      disapprovalDeadline,
      timestamp,
      chainId,
    }),
    expected,
  );
});
