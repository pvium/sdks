package services

import (
	"context"
	"errors"

	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

type OAuthService struct {
	client *transport.HTTPClient
}

func NewOAuthService(client *transport.HTTPClient) *OAuthService {
	return &OAuthService{client: client}
}

func (s *OAuthService) ExchangeCodeForToken(ctx context.Context, input models.ExchangeAuthorizationCodeInput, options *models.RequestOptions) (models.APIResponse[models.OAuthTokenData], error) {
	clientID, err := s.resolveClientID(input.ClientID)
	if err != nil {
		return models.APIResponse[models.OAuthTokenData]{}, err
	}
	apiKey, err := s.resolveOAuthAPIKey(input.APIKey, options)
	if err != nil {
		return models.APIResponse[models.OAuthTokenData]{}, err
	}
	body := map[string]any{
		"clientId":    clientID,
		"apiKey":      apiKey,
		"grantType":   "authorization_code",
		"code":        input.Code,
		"redirectUri": input.RedirectURI,
	}
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: "/client-apps/oauth2/token", Body: body, Options: s.oauthRequestOptions(options)})
	if err != nil {
		return models.APIResponse[models.OAuthTokenData]{}, err
	}
	return transport.Decode[models.APIResponse[models.OAuthTokenData]](raw)
}

func (s *OAuthService) RefreshAccessToken(ctx context.Context, input models.RefreshAccessTokenInput, options *models.RequestOptions) (models.APIResponse[models.OAuthTokenData], error) {
	clientID, err := s.resolveClientID(input.ClientID)
	if err != nil {
		return models.APIResponse[models.OAuthTokenData]{}, err
	}
	apiKey, err := s.resolveOAuthAPIKey(input.APIKey, options)
	if err != nil {
		return models.APIResponse[models.OAuthTokenData]{}, err
	}
	body := map[string]any{
		"clientId":     clientID,
		"apiKey":       apiKey,
		"grantType":    "refresh_token",
		"refreshToken": input.RefreshToken,
	}
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: "/client-apps/oauth2/token", Body: body, Options: s.oauthRequestOptions(options)})
	if err != nil {
		return models.APIResponse[models.OAuthTokenData]{}, err
	}
	return transport.Decode[models.APIResponse[models.OAuthTokenData]](raw)
}

func (s *OAuthService) GetAccessTokenFromRefreshToken(ctx context.Context, input models.RefreshAccessTokenInput, options *models.RequestOptions) (models.APIResponse[models.OAuthTokenData], error) {
	return s.RefreshAccessToken(ctx, input, options)
}

func (s *OAuthService) GetUserInfo(ctx context.Context, options *models.RequestOptions) (models.APIResponse[models.OAuthUserInfo], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "GET", Path: "/users/me", Options: options})
	if err != nil {
		return models.APIResponse[models.OAuthUserInfo]{}, err
	}
	return transport.Decode[models.APIResponse[models.OAuthUserInfo]](raw)
}

func (s *OAuthService) resolveClientID(inputClientID string) (string, error) {
	if inputClientID != "" {
		return inputClientID, nil
	}

	clientID := s.client.Config().ClientID
	if clientID == "" {
		return "", errors.New("clientId is required for OAuth token exchange")
	}

	return clientID, nil
}

func (s *OAuthService) resolveOAuthAPIKey(inputAPIKey string, options *models.RequestOptions) (string, error) {
	if inputAPIKey != "" {
		return inputAPIKey, nil
	}

	if options != nil && options.APIKey != "" {
		return options.APIKey, nil
	}

	apiKey := s.client.Config().APIKey
	if apiKey == "" {
		return "", errors.New("apiKey is required for OAuth token exchange")
	}

	return apiKey, nil
}

func (s *OAuthService) oauthRequestOptions(options *models.RequestOptions) *models.RequestOptions {
	if options == nil {
		return &models.RequestOptions{SkipAPIKey: true}
	}

	copy := *options
	copy.SkipAPIKey = true
	return &copy
}
