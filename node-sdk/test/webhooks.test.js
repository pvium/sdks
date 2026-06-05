const test = require("node:test");
const assert = require("node:assert/strict");
const { createHmac, createHash } = require("node:crypto");

const {
  resolvePviumWebhookPayload,
  verifyPviumWebhookToken,
} = require("../dist/index.js");

function signJwt(payload, secret) {
  const encodedHeader = base64UrlJson({ alg: "HS256", typ: "JWT" });
  const encodedPayload = base64UrlJson(payload);
  const signature = createHmac("sha256", secret)
    .update(`${encodedHeader}.${encodedPayload}`)
    .digest("base64url");

  return `${encodedHeader}.${encodedPayload}.${signature}`;
}

function base64UrlJson(value) {
  return Buffer.from(JSON.stringify(value)).toString("base64url");
}

test("verifyPviumWebhookToken verifies backend HS256 webhook JWTs", () => {
  const token = signJwt(
    {
      event: "oauth.invite.accepted",
      data: { githubLogin: "octocat" },
      iat: 1_700_000_000,
      exp: 4_000_000_000,
    },
    "webhook_secret",
  );

  const payload = verifyPviumWebhookToken(token, "webhook_secret", {
    expectedEvent: "oauth.invite.accepted",
  });

  assert.equal(payload.event, "oauth.invite.accepted");
  assert.deepEqual(payload.data, { githubLogin: "octocat" });
});

test("verifyPviumWebhookToken supports backend tokens signed with sha256(secret)", () => {
  const secret = "secret_abc123";
  const hashedSecret = createHash("sha256").update(secret).digest("hex");
  const token = signJwt(
    {
      event: "batch.payee.added",
      data: { batch: { id: "batch_123" } },
      exp: 4_000_000_000,
    },
    hashedSecret,
  );

  const payload = verifyPviumWebhookToken(token, secret);

  assert.equal(payload.event, "batch.payee.added");
  assert.deepEqual(payload.data, { batch: { id: "batch_123" } });
});

test("resolvePviumWebhookPayload returns token data and checks body event", () => {
  const token = signJwt(
    {
      event: "invoice.paid",
      data: { invoiceId: "inv_123" },
      exp: 4_000_000_000,
    },
    "webhook_secret",
  );

  const resolved = resolvePviumWebhookPayload(
    { event: "invoice.paid", token },
    "webhook_secret",
  );

  assert.equal(resolved.event, "invoice.paid");
  assert.deepEqual(resolved.data, { invoiceId: "inv_123" });
});

test("verifyPviumWebhookToken rejects expired tokens", () => {
  const token = signJwt(
    {
      event: "invoice.paid",
      data: {},
      exp: 1_700_000_000,
    },
    "webhook_secret",
  );

  assert.throws(
    () =>
      verifyPviumWebhookToken(token, "webhook_secret", {
        now: 1_800_000_000_000,
      }),
    /Expired Pvium webhook token/,
  );
});
