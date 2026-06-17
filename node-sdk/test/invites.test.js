const test = require("node:test");
const assert = require("node:assert/strict");
const { Wallet, verifyMessage } = require("ethers");

const { PviumSdk } = require("../dist/index.js");

const TEST_PRIVATE_KEY =
  "0x59c6995e998f97a5a0044976f0d7f3f6f8f53f6a2046baf4f01cb4f1f6bcb58f";
const TEST_ADDRESS = new Wallet(TEST_PRIVATE_KEY).address;

test("creates and signs an OAuth invite link with a dummy EVM key", async () => {
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    consentHost: "http://localhost:3000",
    clientId: "app_test",
    apiKey: "pk_test_dummy",
  });

  const bundle = sdk.invites.createBundle({
    identities: [{ type: "email", value: "Test.User@example.com" }],
    scopes: ["read:ethereum_wallet", "read:user"],
    chain: "ethereum",
  });

  const signed = await sdk.invites.signBundle(bundle, {
    chain: "ethereum",
    privateKey: TEST_PRIVATE_KEY,
  });

  assert.equal(signed.clientId, "app_test");
  assert.equal(signed.invites.length, 1);
  assert.equal(signed.root.signatureType, "evm-personal-sign");
  assert.equal(signed.root.signerAddress, TEST_ADDRESS);
  assert.deepEqual(signed.scopes, ["read:ethereum_wallet", "read:user"]);

  const recovered = verifyMessage(
    signed.root.signatureMessage,
    signed.root.signature,
  );
  assert.equal(recovered, TEST_ADDRESS);

  const invite = signed.invites[0];
  assert.equal(invite.identityType, "email");
  assert.equal(invite.identityValue, "test.user@example.com");
  assert.equal(invite.appClientId, "app_test");
  assert.equal(invite.leafVersion, "2");
  assert.equal(typeof invite.secretHash, "string");
  assert.equal(invite.secretHash.length, 64);
  assert.equal(invite.proof.length, 0);

  const inviteUrl = new URL(invite.inviteLink);
  assert.equal(inviteUrl.origin, "http://localhost:3000");
  assert.equal(inviteUrl.pathname, "/oauth2/authorize");
  assert.equal(inviteUrl.searchParams.get("client_id"), "app_test");
  assert.equal(inviteUrl.searchParams.get("response_type"), "code");
  assert.equal(
    inviteUrl.searchParams.get("scope"),
    "read:ethereum_wallet read:user",
  );
  assert.equal(inviteUrl.searchParams.get("invite_nonce"), invite.inviteNonce);
  assert.equal(
    inviteUrl.searchParams.get("invite_secret"),
    invite.inviteSecret,
  );
  assert.equal(inviteUrl.searchParams.get("identity_type"), "email");
  assert.equal(
    inviteUrl.searchParams.get("identity_hint"),
    "test.user@example.com",
  );
});

test("creates batch invite bundle links with explicit batchId and custom state", async () => {
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    consentHost: "http://localhost:3000",
    clientId: "app_test",
    apiKey: "pk_test_dummy",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = JSON.parse(init.body);
      return new Response(JSON.stringify({ data: body.invites }), {
        status: 201,
        headers: { "content-type": "application/json" },
      });
    },
  });

  const bundle = sdk.invites.createBundle({
    identities: [{ type: "email", value: "Batch.User@example.com" }],
    scopes: ["read:user", "read:ethereum_wallet"],
    chain: "ethereum",
    batchInvite: {
      batchId: "batch_123",
      stateParams: { source: "sdk-test" },
    },
    stateParams: { returnTo: "/admin/bulk-payments/batch_123" },
  });

  const signed = await sdk.invites.signBundle(bundle, {
    chain: "ethereum",
    privateKey: TEST_PRIVATE_KEY,
  });

  assert.equal(signed.batchId, "batch_123");
  assert.equal(signed.batchInvite.batchId, "batch_123");

  const inviteUrl = new URL(signed.invites[0].inviteLink);
  assert.equal(inviteUrl.searchParams.get("batchId"), "batch_123");

  const state = new URLSearchParams(inviteUrl.searchParams.get("state"));
  assert.equal(state.get("batchId"), "batch_123");
  assert.equal(state.get("source"), "sdk-test");
  assert.equal(state.get("returnTo"), "/admin/bulk-payments/batch_123");

  const groupUrl = new URL(signed.groupInviteLink);
  assert.equal(groupUrl.searchParams.get("batchId"), "batch_123");

  const committed = await sdk.invites.commitBundle(signed);

  assert.equal(requests.length, 1);
  assert.equal(
    requests[0].url,
    "http://localhost:4005/v1/batch-payments/batch_123/invites",
  );
  assert.equal(committed.inviteCommitted, true);
  assert.equal(committed.alreadyAccepted, false);
  assert.equal(
    committed.committedInvites[0].inviteNonce,
    signed.invites[0].inviteNonce,
  );
});

test("commitBundle detects returned accepted invites with different nonces", async () => {
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    consentHost: "http://localhost:3000",
    clientId: "app_test",
    apiKey: "pk_test_dummy",
    fetchFn: async () =>
      new Response(
        JSON.stringify({
          data: [
            {
              id: "invite_existing",
              identityType: "github",
              identityValue: "feminefa",
              inviteNonce: "existing_nonce",
              status: "accepted",
            },
          ],
        }),
        {
          status: 201,
          headers: { "content-type": "application/json" },
        },
      ),
  });

  const signed = await sdk.invites.createSignedBundle(
    {
      identities: [{ type: "github", value: "feminefa" }],
      scopes: ["read:user"],
      chain: "ethereum",
    },
    {
      chain: "ethereum",
      privateKey: TEST_PRIVATE_KEY,
    },
  );
  const committed = await sdk.invites.commitBundle(signed);

  assert.equal(committed.inviteCommitted, false);
  assert.equal(committed.alreadyAccepted, true);
  assert.equal(committed.existingInvites[0].inviteNonce, "existing_nonce");
  assert.notEqual(
    committed.existingInvites[0].inviteNonce,
    signed.invites[0].inviteNonce,
  );
});

test("supports separate master-secret and invite-root signers", async () => {
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    consentHost: "http://localhost:3000",
    clientId: "app_test",
    apiKey: "pk_test_dummy",
  });
  const wallet = new Wallet(TEST_PRIVATE_KEY);
  const calls = [];

  const bundle = sdk.invites.createBundle({
    identities: [{ type: "email", value: "Split.Signer@example.com" }],
    scopes: ["read:user", "read:ethereum_wallet"],
    chain: "ethereum",
  });

  const signed = await sdk.invites.signBundle(bundle, {
    chain: "ethereum",
    signerAddress: wallet.address,
    signMessage: async () => {
      throw new Error("fallback signMessage should not be called");
    },
    signMasterSecret: async (message) => {
      calls.push(`master:${message}`);
      return wallet.signMessage(message);
    },
    signInviteRoot: async (message) => {
      calls.push(`root:${message}`);
      return wallet.signMessage(message);
    },
  });

  assert.equal(calls.length, 2);
  assert.ok(calls[0].startsWith("master:PVIUM_INVITE_SECRET_V2:"));
  assert.ok(calls[1].startsWith('root:PVIUM_INVITE_ROOT_V2'));
  assert.equal(signed.root.signerAddress, wallet.address);
  assert.equal(
    verifyMessage(signed.root.signatureMessage, signed.root.signature),
    wallet.address,
  );
});

test("finds app invites by identity through the app-invites endpoint", async () => {
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    consentHost: "http://localhost:3000",
    clientId: "app_test",
    apiKey: "pk_test_dummy",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      return new Response(
        JSON.stringify({
          data: [
            {
              id: "invite_1",
              identityType: "github",
              identityValue: "feminefa",
              status: "accepted",
            },
          ],
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      );
    },
  });

  const result = await sdk.invites.findAppInviteByIdentity({
    identityType: "github",
    identityValue: "@FemInefa",
    status: "accepted",
  });

  assert.equal(requests.length, 1);
  const url = new URL(requests[0].url);
  assert.equal(url.pathname, "/v1/batch-payments/app-invites");
  assert.equal(url.searchParams.get("identityType"), "github");
  assert.equal(url.searchParams.get("identityValue"), "@FemInefa");
  assert.equal(url.searchParams.get("status"), "accepted");
  assert.equal(requests[0].init.headers["x-api-key"], "pk_test_dummy");
  assert.equal(result.accepted, true);
  assert.equal(result.invite.id, "invite_1");
});
