package config

import "time"

const (
	SandboxBaseURL        = "https://api-sandbox.pvium.com/v1"
	ProductionBaseURL     = "https://api.pvium.com/v1"
	SandboxConsentHost    = "https://sandbox.pvium.com"
	ProductionConsentHost = "https://pvium.com"
	DefaultTimeout        = 30 * time.Second
	DefaultInviteTTL      = 7 * 24 * time.Hour
)

type Environment string

const (
	EnvironmentSandbox    Environment = "sandbox"
	EnvironmentProduction Environment = "production"
)

type Config struct {
	ClientID      string
	APIKey        string
	Environment   Environment
	BaseURL       string
	ConsentHost   string
	Timeout       time.Duration
	DefaultHeader map[string]string
}

func (c Config) WithDefaults() Config {
	out := c
	if out.Environment == "" {
		out.Environment = EnvironmentProduction
	}
	if out.BaseURL == "" {
		if out.Environment == EnvironmentSandbox {
			out.BaseURL = SandboxBaseURL
		} else {
			out.BaseURL = ProductionBaseURL
		}
	}
	if out.ConsentHost == "" {
		if out.Environment == EnvironmentSandbox {
			out.ConsentHost = SandboxConsentHost
		} else {
			out.ConsentHost = ProductionConsentHost
		}
	}
	if out.Timeout <= 0 {
		out.Timeout = DefaultTimeout
	}
	if out.DefaultHeader == nil {
		out.DefaultHeader = map[string]string{}
	}
	return out
}
