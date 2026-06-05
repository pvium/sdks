import { PviumHttpClient, PviumSdkConfig } from "./client";
import {
  OAuthTokenResponse,
  OAuthUserInfoResponse,
  RequestOptions,
} from "./types";

export interface ExchangeAuthorizationCodeInput {
  code: string;
  redirectUri: string;
  clientId?: string;
  apiKey?: string;
}

export interface RefreshAccessTokenInput {
  refreshToken: string;
  clientId?: string;
  apiKey?: string;
}

export class PviumOAuth {
  constructor(
    private readonly http: PviumHttpClient,
    private readonly config: PviumSdkConfig,
  ) {}

  async exchangeCodeForToken(
    input: ExchangeAuthorizationCodeInput,
    options?: RequestOptions,
  ): Promise<OAuthTokenResponse> {
    const response = await this.http.request({
      method: "POST",
      path: "/v1/client-apps/oauth2/token",
      body: {
        clientId: input.clientId ?? this.requireClientId(),
        apiKey: input.apiKey ?? options?.apiKey ?? this.requireApiKey(),
        grantType: "authorization_code",
        code: input.code,
        redirectUri: input.redirectUri,
      },
      options: {
        ...options,
        skipApiKey: true,
      },
    });

    return this.http.parseResponseBody<OAuthTokenResponse>(response);
  }

  async refreshAccessToken(
    input: RefreshAccessTokenInput,
    options?: RequestOptions,
  ): Promise<OAuthTokenResponse> {
    const response = await this.http.request({
      method: "POST",
      path: "/v1/client-apps/oauth2/token",
      body: {
        clientId: input.clientId ?? this.requireClientId(),
        apiKey: input.apiKey ?? options?.apiKey ?? this.requireApiKey(),
        grantType: "refresh_token",
        refreshToken: input.refreshToken,
      },
      options: {
        ...options,
        skipApiKey: true,
      },
    });

    return this.http.parseResponseBody<OAuthTokenResponse>(response);
  }

  async getAccessTokenFromRefreshToken(
    input: RefreshAccessTokenInput,
    options?: RequestOptions,
  ): Promise<OAuthTokenResponse> {
    return this.refreshAccessToken(input, options);
  }

  async getUserInfo(options?: RequestOptions): Promise<OAuthUserInfoResponse> {
    const response = await this.http.request({
      method: "GET",
      path: "/v1/users/me",
      options: {
        ...options,
      },
    });

    return this.http.parseResponseBody<OAuthUserInfoResponse>(response);
  }

  private requireClientId(): string {
    if (!this.config.clientId) {
      throw new Error("PviumSdkConfig.clientId is required for OAuth methods");
    }

    return this.config.clientId;
  }

  private requireApiKey(): string {
    if (!this.config.apiKey) {
      throw new Error(
        "PviumSdkConfig.apiKey is required for OAuth token exchange",
      );
    }

    return this.config.apiKey;
  }
}
