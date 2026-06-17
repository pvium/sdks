const test = require("node:test");
const assert = require("node:assert/strict");

const { PviumApiError, PviumSdk } = require("../dist/index.js");

function createMockSdk(config = {}) {
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "https://api.example.test/v1",
    apiKey: "pk_test_dummy",
    clientId: "app_test",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      return new Response(
        JSON.stringify({
          meta: { statusCode: 200, success: true },
          data: {
            accessToken: "access_token",
            refreshToken: "refresh_token",
            expiresIn: 3600,
          },
        }),
        { headers: { "content-type": "application/json" } },
      );
    },
    ...config,
  });

  return { sdk, requests };
}

test("exchangeCodeForToken sends apiKey in the token request body", async () => {
  const { sdk, requests } = createMockSdk();

  await sdk.oauth.exchangeCodeForToken({
    code: "oauth_code",
    redirectUri: "https://example.test/callback",
  });

  assert.equal(requests[0].url, "https://api.example.test/v1/client-apps/oauth2/token");
  assert.equal(requests[0].init.method, "POST");
  assert.equal(requests[0].init.headers["x-api-key"], undefined);
  assert.deepEqual(JSON.parse(requests[0].init.body), {
    clientId: "app_test",
    apiKey: "pk_test_dummy",
    grantType: "authorization_code",
    code: "oauth_code",
    redirectUri: "https://example.test/callback",
  });
});

test("refreshAccessToken sends apiKey in the token request body", async () => {
  const { sdk, requests } = createMockSdk();

  await sdk.oauth.refreshAccessToken({
    refreshToken: "refresh_token",
  });

  assert.equal(requests[0].init.headers["x-api-key"], undefined);
  assert.deepEqual(JSON.parse(requests[0].init.body), {
    clientId: "app_test",
    apiKey: "pk_test_dummy",
    grantType: "refresh_token",
    refreshToken: "refresh_token",
  });
});

test("getAccessTokenFromRefreshToken refreshes through the OAuth token endpoint", async () => {
  const { sdk, requests } = createMockSdk();

  await sdk.oauth.getAccessTokenFromRefreshToken({
    refreshToken: "refresh_token",
  });

  assert.equal(requests[0].url, "https://api.example.test/v1/client-apps/oauth2/token");
  assert.equal(requests[0].init.method, "POST");
  assert.equal(requests[0].init.headers["x-api-key"], undefined);
  assert.deepEqual(JSON.parse(requests[0].init.body), {
    clientId: "app_test",
    apiKey: "pk_test_dummy",
    grantType: "refresh_token",
    refreshToken: "refresh_token",
  });
});

test("refreshAccessToken rejects non-2xx responses with PviumApiError", async () => {
  const { sdk } = createMockSdk({
    fetchFn: async () =>
      new Response(
        JSON.stringify({
          meta: {
            statusCode: 401,
            success: false,
            message: "Unauthorized",
            developerMessage: "Refresh token expired",
          },
        }),
        {
          status: 401,
          statusText: "Unauthorized",
          headers: { "content-type": "application/json" },
        },
      ),
  });

  await assert.rejects(
    () => sdk.oauth.refreshAccessToken({ refreshToken: "expired_refresh" }),
    (error) => {
      assert.ok(error instanceof PviumApiError);
      assert.equal(error.status, 401);
      assert.equal(error.message, "Pvium API request failed with 401: Refresh token expired");
      assert.deepEqual(error.body, {
        meta: {
          statusCode: 401,
          success: false,
          message: "Unauthorized",
          developerMessage: "Refresh token expired",
        },
      });
      return true;
    },
  );
});
