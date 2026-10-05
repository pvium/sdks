const assert = require("node:assert/strict");
const test = require("node:test");

const { PviumSdk } = require("../dist/index.js");
const { p2idAddress } = require("@pvium/p2id-core");

const UNREGISTERED = "Recipient is not a Pvium user";

function mockSdk(responses) {
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    clientId: "client_123",
    fetchFn: async (url, init) => {
      requests.push({ url: String(url), init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta?.statusCode || 200,
        headers: { "content-type": "application/json" },
      });
    },
  });
  return { sdk, requests };
}

const batch = (complianceMode = "Open", chain = "base-testnet") => ({
  meta: { statusCode: 200, success: true },
  data: { id: "batch_1", chain, paymentType: "Instant", complianceMode },
});

const resolveResponse = (resolved, errors) => ({
  meta: { statusCode: 200, success: true },
  data: { resolved, errors },
});

test("resolveRecipients injects P2ID only for unregistered identities; returns { resolved, errors } unwrapped", async () => {
  const { sdk } = mockSdk([
    batch("Open", "base-testnet"),
    resolveResponse(
      [{ identityType: "email", identityValue: "bob@example.com", receiver: "0x0000000000000000000000000000000000000001" }],
      [
        { identity: "email=alice@example.com", identityType: "email", identityValue: "alice@example.com", reason: UNREGISTERED },
        { identity: "github=octocat", identityType: "github", identityValue: "octocat", reason: UNREGISTERED },
        // registered but unpayable on this chain — must NOT be injected, stays an error
        { identity: "email=carol@example.com", identityType: "email", identityValue: "carol@example.com", reason: "User has no base-testnet wallet linked" },
      ],
    ),
  ]);

  const payout = await sdk.payout.get("batch_1");
  const res = await payout.resolveRecipients(
    [
      { identityType: "email", identityValue: "bob@example.com" },
      { identityType: "email", identityValue: "alice@example.com" },
      { identityType: "github", identityValue: "octocat" },
      { identityType: "email", identityValue: "carol@example.com" },
    ],
    { unregisteredIdentity: "p2id" },
  );

  // Unwrapped: no meta/data wrapper.
  assert.equal(res.data, undefined);
  assert.equal(res.resolved.length, 3); // bob + injected alice + injected octocat
  assert.equal(res.errors.length, 1); // carol (no wallet) stays

  const alice = res.resolved.find((r) => r.identityValue === "alice@example.com");
  assert.equal(alice.p2id, true);
  assert.equal(
    alice.receiver,
    p2idAddress({ identityType: "email", identityValue: "alice@example.com", environment: "sandbox" }),
  );

  const octocat = res.resolved.find((r) => r.identityValue === "octocat");
  assert.equal(octocat.p2id, true);
  assert.equal(
    octocat.receiver,
    p2idAddress({ identityType: "github", identityValue: "octocat", environment: "sandbox" }),
  );

  const carol = res.errors.find((e) => e.identityValue === "carol@example.com");
  assert.ok(carol, "the no-wallet identity remains an error");
  assert.equal(res.resolved.some((r) => r.identityValue === "carol@example.com"), false);
});

test("resolveRecipients rejects unresolved identities by default", async () => {
  const { sdk } = mockSdk([
    batch("Open", "base-testnet"),
    resolveResponse([], [
      { identity: "email=ghost@example.com", identityType: "email", identityValue: "ghost@example.com", reason: UNREGISTERED },
    ]),
  ]);
  const payout = await sdk.payout.get("batch_1");
  await assert.rejects(
    () => payout.resolveRecipients([{ identityType: "email", identityValue: "ghost@example.com" }]),
    /could not be resolved/,
  );
});

test("resolveRecipients p2id is refused for non-Open compliance", async () => {
  const { sdk } = mockSdk([
    batch("Strict", "base-testnet"),
    resolveResponse([], [
      { identity: "email=x@example.com", identityType: "email", identityValue: "x@example.com", reason: UNREGISTERED },
    ]),
  ]);
  const payout = await sdk.payout.get("batch_1");
  await assert.rejects(
    () =>
      payout.resolveRecipients(
        [{ identityType: "email", identityValue: "x@example.com" }],
        { unregisteredIdentity: "p2id" },
      ),
    /Open compliance/,
  );
});

test("resolveRecipients runs the injected verifier (following the attestation url) and throws on a bad mapping", async () => {
  const { sdk, requests } = mockSdk([
    batch("Open", "base-testnet"),
    resolveResponse(
      [
        {
          identityType: "github",
          identityValue: "octocat",
          receiver: "0x0000000000000000000000000000000000000001",
          attestation: {
            publicId: "p_1",
            url: "http://localhost:4005/v1/proofs/p_1",
            identityType: "github",
            issuedAt: 1,
            circuitVersion: 1,
            vkHash: "0x00",
          },
        },
      ],
      [],
    ),
    {
      meta: { statusCode: 200, success: true },
      data: {
        attestation: {
          proof: "0x",
          publicInputs: "0x",
          wallet: "0x0000000000000000000000000000000000000001",
          identityType: "github",
          issuedAt: 1,
          circuitVersion: 1,
          vkHash: "0x00",
        },
      },
    },
  ]);

  const payout = await sdk.payout.get("batch_1");
  const seen = [];
  await assert.rejects(
    () =>
      payout.resolveRecipients(
        [{ identityType: "github", identityValue: "octocat" }],
        {
          verify: (input) => {
            seen.push({ identityType: input.identityType, identityValue: input.identityValue });
            return false;
          },
        },
      ),
    /verification failed/,
  );
  assert.deepEqual(seen, [{ identityType: "github", identityValue: "octocat" }]);
  // getAttestation followed the summary url path.
  assert.ok(requests.some((r) => r.url.includes("/proofs/p_1")));
});
