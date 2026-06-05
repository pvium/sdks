from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

from ...core.client import PviumHttpClient, PviumSdkConfig
from ...core.types import RequestOptions


@dataclass
class ExchangeAuthorizationCodeInput:
    code: str
    redirectUri: str
    clientId: Optional[str] = None
    apiKey: Optional[str] = None


@dataclass
class RefreshAccessTokenInput:
    refreshToken: str
    clientId: Optional[str] = None
    apiKey: Optional[str] = None


class PviumOAuth:
    def __init__(self, http: PviumHttpClient, config: PviumSdkConfig):
        self.http = http
        self.config = config

    def exchangeCodeForToken(self, input: ExchangeAuthorizationCodeInput, options: Optional[RequestOptions] = None):
        options = dict(options or {})
        options["skipApiKey"] = True
        response = self.http.request(
            "POST",
            "/v1/client-apps/oauth2/token",
            body={
                "clientId": input.clientId or self._require_client_id(),
                "apiKey": input.apiKey or options.get("apiKey") or self._require_api_key(),
                "grantType": "authorization_code",
                "code": input.code,
                "redirectUri": input.redirectUri,
            },
            options=options,
        )
        return self.http.parseResponseBody(response)

    def refreshAccessToken(self, input: RefreshAccessTokenInput, options: Optional[RequestOptions] = None):
        options = dict(options or {})
        options["skipApiKey"] = True
        response = self.http.request(
            "POST",
            "/v1/client-apps/oauth2/token",
            body={
                "clientId": input.clientId or self._require_client_id(),
                "apiKey": input.apiKey or options.get("apiKey") or self._require_api_key(),
                "grantType": "refresh_token",
                "refreshToken": input.refreshToken,
            },
            options=options,
        )
        return self.http.parseResponseBody(response)

    def getAccessTokenFromRefreshToken(self, input: RefreshAccessTokenInput, options: Optional[RequestOptions] = None):
        return self.refreshAccessToken(input, options)

    def getUserInfo(self, options: Optional[RequestOptions] = None):
        response = self.http.request("GET", "/v1/users/me", options=options)
        return self.http.parseResponseBody(response)

    def _require_client_id(self) -> str:
        if not self.config.clientId:
            raise RuntimeError("PviumSdkConfig.clientId is required for OAuth methods")
        return self.config.clientId

    def _require_api_key(self) -> str:
        if not self.config.apiKey:
            raise RuntimeError("PviumSdkConfig.apiKey is required for OAuth token exchange")
        return self.config.apiKey
