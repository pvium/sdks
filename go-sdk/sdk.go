package pvium

import (
	"github.com/pvium/sdks/go-sdk/config"
	"github.com/pvium/sdks/go-sdk/services"
	"github.com/pvium/sdks/go-sdk/transport"
)

type SDK struct {
	Client    *transport.HTTPClient
	Endpoints *services.EndpointsService
	OAuth     *services.OAuthService
	Invites   *services.InviteService
	Payouts   *services.PayoutService
}

func New(cfg config.Config) *SDK {
	client := transport.NewHTTPClient(cfg)
	return &SDK{
		Client:    client,
		Endpoints: services.NewEndpointsService(client),
		OAuth:     services.NewOAuthService(client),
		Invites:   services.NewInviteService(client),
		Payouts:   services.NewPayoutService(client),
	}
}

func Init(cfg config.Config) *SDK {
	return New(cfg)
}
