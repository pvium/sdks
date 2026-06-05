package pvium

import (
	"testing"

	"github.com/pvium/sdks/go-sdk/config"
)

func TestInitExposesPayoutService(t *testing.T) {
	t.Parallel()

	sdk := Init(config.Config{BaseURL: "https://api.example.test/v1", APIKey: "app_key"})
	if sdk == nil {
		t.Fatal("sdk should not be nil")
	}
	if sdk.Payouts == nil {
		t.Fatal("sdk payout service should not be nil")
	}
	if sdk.Endpoints == nil || sdk.OAuth == nil || sdk.Invites == nil || sdk.Client == nil {
		t.Fatal("sdk services/client should be initialized")
	}
}
