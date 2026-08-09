const assert = require("node:assert/strict");
const test = require("node:test");
const fs = require("node:fs");
const path = require("node:path");

const { PviumSdk } = require("../dist/index.js");

function loadFixture() {
  return JSON.parse(
    fs.readFileSync(
      path.join(__dirname, "../../parity-fixtures/open-organization-invite.json"),
      "utf8",
    ),
  );
}

test("createSignedOpenOrganizationInvite matches the shared parity fixture", async () => {
  const fx = loadFixture();
  const sdk = PviumSdk.init(fx.config);

  const signed = await sdk.invites.createSignedOpenOrganizationInvite(
    fx.input,
    fx.signer,
  );

  const e = fx.expected;
  assert.equal(signed.signatureMessage, e.signatureMessage);
  assert.equal(signed.policyHash, e.policyHash);
  assert.equal(signed.secretHash, e.secretHash);
  assert.equal(signed.signature, e.signature);
  assert.equal(signed.signatureType, e.signatureType);
  assert.equal(signed.signatureTimestamp, e.signatureTimestamp);
  assert.equal(signed.signerAddress, e.signerAddress);
  assert.deepEqual(signed.scopes, e.scopes);
  assert.deepEqual(signed.allowedIdentityTypes, e.allowedIdentityTypes);
  assert.deepEqual(signed.allowedEmailDomains, e.allowedEmailDomains);
});
