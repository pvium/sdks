import json

from pvium_sdk import PviumSdk, PviumSdkConfig
from pvium_sdk.oauth import ExchangeAuthorizationCodeInput, RefreshAccessTokenInput


def make_fetch(requests):
    def _fetch(method, url, headers, payload, timeout):
        requests.append({"method": method, "url": url, "headers": headers, "payload": payload})
        body = {
            "meta": {"statusCode": 200, "success": True},
            "data": {
                "accessToken": "access_token",
                "refreshToken": "refresh_token",
                "expiresIn": 3600,
            },
        }
        return 200, {"content-type": "application/json"}, json.dumps(body)

    return _fetch


def test_exchange_code_sends_api_key_in_body_not_header():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="https://api.example.test/v1",
            apiKey="pk_test_dummy",
            clientId="app_test",
            fetchFn=make_fetch(requests),
        )
    )

    sdk.oauth.exchangeCodeForToken(
        ExchangeAuthorizationCodeInput(code="oauth_code", redirectUri="https://example.test/callback")
    )

    assert requests[0]["url"] == "https://api.example.test/v1/client-apps/oauth2/token"
    assert requests[0]["method"] == "POST"
    assert requests[0]["headers"].get("x-api-key") is None
    assert json.loads(requests[0]["payload"]) == {
        "clientId": "app_test",
        "apiKey": "pk_test_dummy",
        "grantType": "authorization_code",
        "code": "oauth_code",
        "redirectUri": "https://example.test/callback",
    }


def test_refresh_access_token_flow():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="https://api.example.test/v1",
            apiKey="pk_test_dummy",
            clientId="app_test",
            fetchFn=make_fetch(requests),
        )
    )

    sdk.oauth.refreshAccessToken(RefreshAccessTokenInput(refreshToken="refresh_token"))

    assert requests[0]["headers"].get("x-api-key") is None
    assert json.loads(requests[0]["payload"]) == {
        "clientId": "app_test",
        "apiKey": "pk_test_dummy",
        "grantType": "refresh_token",
        "refreshToken": "refresh_token",
    }


def test_get_access_token_from_refresh_token_uses_oauth_token_endpoint():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="https://api.example.test/v1",
            apiKey="pk_test_dummy",
            clientId="app_test",
            fetchFn=make_fetch(requests),
        )
    )

    sdk.oauth.getAccessTokenFromRefreshToken(RefreshAccessTokenInput(refreshToken="refresh_token"))

    assert requests[0]["url"] == "https://api.example.test/v1/client-apps/oauth2/token"
    assert requests[0]["method"] == "POST"
    assert requests[0]["headers"].get("x-api-key") is None
    assert json.loads(requests[0]["payload"]) == {
        "clientId": "app_test",
        "apiKey": "pk_test_dummy",
        "grantType": "refresh_token",
        "refreshToken": "refresh_token",
    }
