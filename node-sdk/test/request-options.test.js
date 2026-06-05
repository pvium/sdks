const assert = require("node:assert/strict");
const test = require("node:test");
const { PviumSdk } = require("../dist/index.js");

test("accessToken request option uses Bearer auth and suppresses configured api key", async () => {
  let capturedRequest;
  const sdk = PviumSdk.init({
    baseUrl: "https://api.example.test/v1",
    apiKey: "app_key",
    fetchFn: async (url, init) => {
      capturedRequest = { url, init };
      return new Response(
        JSON.stringify({
          meta: { statusCode: 201, success: true },
          data: { id: 123, code: "INV-123", url: "https://pay.example.test" },
        }),
        { headers: { "content-type": "application/json" } },
      );
    },
  });

  await sdk.endpoints.createInvoice(
    {
      name: "Reward",
      description: "Merged PR reward",
      amount: 20,
      dueDate: new Date("2026-01-01T00:00:00.000Z").toISOString(),
      paymentChannels: [{ chain: "base", currency: "USDC" }],
      redirectUri: "https://github.com",
    },
    { accessToken: "access_user" },
  );

  assert.equal(
    capturedRequest.init.headers.Authorization,
    "Bearer access_user",
  );
  assert.equal(capturedRequest.init.headers["x-api-key"], undefined);
});

test("configured api key is used when accessToken is not provided", async () => {
  let capturedRequest;
  const sdk = PviumSdk.init({
    baseUrl: "https://api.example.test/v1",
    apiKey: "app_key",
    fetchFn: async (url, init) => {
      capturedRequest = { url, init };
      return new Response(
        JSON.stringify({
          meta: { statusCode: 200, success: true },
          data: [],
        }),
        { headers: { "content-type": "application/json" } },
      );
    },
  });

  await sdk.endpoints.listInvoices();

  assert.equal(capturedRequest.init.headers["x-api-key"], "app_key");
  assert.equal(capturedRequest.init.headers.Authorization, undefined);
});
