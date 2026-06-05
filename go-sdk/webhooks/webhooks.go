package webhooks

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pvium/sdks/go-sdk/models"
)

func VerifyPviumWebhookToken(token string, secret string, options *models.VerifyPviumWebhookTokenOptions) (models.PviumWebhookTokenPayload, error) {
	if secret == "" {
		return models.PviumWebhookTokenPayload{}, errors.New("webhook secret is required")
	}

	candidates := []string{secret}
	allowFallback := true
	if options != nil && options.AllowHashedSecretFallback != nil {
		allowFallback = *options.AllowHashedSecretFallback
	}
	if allowFallback {
		hashed := sha256.Sum256([]byte(secret))
		hashedSecret := fmt.Sprintf("%x", hashed)
		if hashedSecret != secret {
			candidates = append(candidates, hashedSecret)
		}
	}

	var claims jwt.MapClaims
	parsed := false
	var lastErr error
	for _, candidate := range candidates {
		tryClaims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(token, tryClaims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(candidate), nil
		}, jwt.WithoutClaimsValidation())
		if err == nil {
			claims = tryClaims
			parsed = true
			break
		}
		lastErr = err
	}

	if !parsed {
		if lastErr != nil {
			return models.PviumWebhookTokenPayload{}, lastErr
		}
		return models.PviumWebhookTokenPayload{}, errors.New("invalid webhook token")
	}

	now := time.Now()
	if options != nil && !options.Now.IsZero() {
		now = options.Now
	}
	if exp, ok := claims["exp"].(float64); ok {
		if now.Unix() >= int64(exp) {
			return models.PviumWebhookTokenPayload{}, errors.New("Expired Pvium webhook token")
		}
	}
	if iat, ok := claims["iat"].(float64); ok {
		if now.Unix() < int64(iat) {
			return models.PviumWebhookTokenPayload{}, errors.New("webhook token iat check failed")
		}
	}

	payload := models.PviumWebhookTokenPayload{}
	if event, ok := claims["event"].(string); ok {
		payload.Event = event
	}
	if options != nil && options.ExpectedEvent != "" && payload.Event != "" && payload.Event != options.ExpectedEvent {
		return models.PviumWebhookTokenPayload{}, errors.New("webhook token event mismatch")
	}
	if data, ok := claims["data"].(map[string]any); ok {
		payload.Data = data
	} else {
		payload.Data = map[string]any{}
	}
	if iat, ok := claims["iat"].(float64); ok {
		payload.Iat = int64(iat)
	}
	if exp, ok := claims["exp"].(float64); ok {
		payload.Exp = int64(exp)
	}
	return payload, nil
}

func ResolvePviumWebhookPayload(body map[string]any, secret string, options ...*models.VerifyPviumWebhookTokenOptions) (models.PviumWebhookTokenPayload, error) {
	var opts *models.VerifyPviumWebhookTokenOptions
	if len(options) > 0 {
		opts = options[0]
	}

	event := ""
	if v, ok := body["event"].(string); ok {
		event = v
	} else if v, ok := body["type"].(string); ok {
		event = v
	}

	tokenAny, ok := body["token"]
	if !ok {
		payload := models.PviumWebhookTokenPayload{Event: event, Data: map[string]any{}}
		if data, ok := body["data"].(map[string]any); ok {
			payload.Data = data
		}
		return payload, nil
	}
	token, ok := tokenAny.(string)
	if !ok || token == "" {
		return models.PviumWebhookTokenPayload{}, errors.New("invalid token field")
	}

	if opts == nil {
		opts = &models.VerifyPviumWebhookTokenOptions{}
	}
	if opts.ExpectedEvent == "" {
		optsCopy := *opts
		optsCopy.ExpectedEvent = event
		opts = &optsCopy
	}

	tokenPayload, err := VerifyPviumWebhookToken(token, secret, opts)
	if err != nil {
		return models.PviumWebhookTokenPayload{}, err
	}
	if tokenPayload.Event == "" {
		tokenPayload.Event = event
	}
	if tokenPayload.Data == nil {
		tokenPayload.Data = map[string]any{}
	}

	return tokenPayload, nil
}
