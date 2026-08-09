const test = require("node:test");
const assert = require("node:assert/strict");
const crypto = require("node:crypto");

const {
  PviumSdk,
  deriveOrgClientId,
  normalizeOrgReferenceId,
  DERIVED_ORG_CLIENT_ID_PREFIX,
} = require("../dist/index.js");

const TEST_PRIVATE_KEY =
  "0x59c6995e998f97a5a0044976f0d7f3f6f8f53f6a2046baf4f01cb4f1f6bcb58f";

function initSdk() {
  return PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    consentHost: "http://localhost:3000",
    clientId: "app_test",
    apiKey: "pk_test_dummy",
  });
}

test("deriveOrgClientId is deterministic and matches the documented formula", () => {
  const derived = deriveOrgClientId("app_test", "ref_abc-123");
  const expectedDigest = crypto
    .createHash("sha256")
    .update("PVIUM_DERIVED_CLIENT_ID_V2:app_test:ref_abc-123")
    .digest("hex")
    .substring(0, 32);

  assert.equal(derived, `subcli_${expectedDigest}`);
  assert.equal(derived, deriveOrgClientId("app_test", "ref_abc-123"));
  assert.ok(derived.startsWith(DERIVED_ORG_CLIENT_ID_PREFIX));
  // Different inviter or referenceId must derive a different org id.
  assert.notEqual(derived, deriveOrgClientId("app_other", "ref_abc-123"));
  assert.notEqual(derived, deriveOrgClientId("app_test", "ref_abc-124"));
});

test("normalizeOrgReferenceId enforces charset and length", () => {
  assert.equal(normalizeOrgReferenceId(" ref_1 "), "ref_1");
  assert.equal(normalizeOrgReferenceId(undefined), undefined);
  assert.throws(() => normalizeOrgReferenceId("bad:ref"));
  assert.throws(() => normalizeOrgReferenceId(""));
  assert.throws(() => normalizeOrgReferenceId("x".repeat(129)));
});

test("bundle with orgReferenceId signs a V4 root carrying the org line", async () => {
  const sdk = initSdk();
  const bundle = sdk.invites.createBundle({
    identities: [{ type: "github", value: "octocat" }],
    scopes: ["read:user", "read:github"],
    chain: "base",
    signingKey: {
      publicKey: "0x00000000000000000000000000000000000000aa",
      keyType: "ethereum",
    },
    orgReferenceId: "maintainer-ref-1",
  });

  const signed = await sdk.invites.signBundle(bundle, {
    chain: "base",
    privateKey: TEST_PRIVATE_KEY,
  });

  assert.equal(signed.root.metadata.version, "4");
  assert.equal(signed.root.orgReferenceId, "maintainer-ref-1");
  assert.equal(
    signed.root.derivedOrgClientId,
    deriveOrgClientId("app_test", "maintainer-ref-1"),
  );

  const lines = signed.root.signatureMessage.split("\n");
  assert.equal(lines[0], "PVIUM_INVITE_ROOT_V4");
  assert.ok(lines.includes("version=4"));
  assert.ok(lines.includes("orgReferenceId=maintainer-ref-1"));
  assert.ok(
    lines.includes("signingKey=0x00000000000000000000000000000000000000aa"),
  );
  // Leaf encoding is unchanged — only the root message gained lines.
  assert.equal(signed.invites[0].leafVersion, "2");
});

test("bundle without orgReferenceId keeps emitting V3/V2 roots", async () => {
  const sdk = initSdk();

  const withKey = await sdk.invites.signBundle(
    sdk.invites.createBundle({
      identities: [{ type: "github", value: "octocat" }],
      scopes: ["read:user"],
      chain: "base",
      signingKey: {
        publicKey: "0x00000000000000000000000000000000000000aa",
        keyType: "ethereum",
      },
    }),
    { chain: "base", privateKey: TEST_PRIVATE_KEY },
  );
  assert.equal(withKey.root.metadata.version, "3");
  assert.ok(withKey.root.signatureMessage.startsWith("PVIUM_INVITE_ROOT_V3"));

  const plain = await sdk.invites.signBundle(
    sdk.invites.createBundle({
      identities: [{ type: "github", value: "octocat" }],
      scopes: ["read:user"],
      chain: "base",
    }),
    { chain: "base", privateKey: TEST_PRIVATE_KEY },
  );
  assert.equal(plain.root.metadata.version, "2");
  assert.ok(plain.root.signatureMessage.startsWith("PVIUM_INVITE_ROOT_V2"));
});

test("createBundle rejects malformed orgReferenceId", () => {
  const sdk = initSdk();
  assert.throws(() =>
    sdk.invites.createBundle({
      identities: [{ type: "github", value: "octocat" }],
      scopes: ["read:user"],
      chain: "base",
      orgReferenceId: "not valid!",
    }),
  );
});
