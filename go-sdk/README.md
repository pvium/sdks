# Pvium Go SDK

Pvium provides programmable zero-custody stablecoin payments for the global gig economy.

This SDK provides Go access to Pvium APIs and helpers for generating signed OAuth invite links, including Merkle-root-backed invite bundles for invite and batch payment workflows.

- Developer documentation: https://pvium.gitbook.io/documentation
- Pvium website: https://pvium.com
- Pvium sandbox: https://sandbox.pvium.com
- TypeScript SDK README: [../node-sdk/README.md](../node-sdk/README.md)

The TypeScript SDK README is the source of truth for shared protocol details such as webhook event catalogs, OAuth scope meanings, payload shape examples, and other data-type explanations. This README keeps Go-specific API usage and examples only.

## Install

```bash
go get github.com/pvium/sdks/go-sdk
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	pvium "github.com/pvium/sdks/go-sdk"
)

func main() {
	ctx := context.Background()

	sdk := pvium.Init(pvium.Config{
		Environment: pvium.EnvironmentSandbox,
		APIKey:      os.Getenv("PVIUM_API_KEY"),
	})

	invoices, err := sdk.Endpoints.ListInvoices(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(invoices.Meta.Success, len(invoices.Data))
}
```

See [examples/quickstart/main.go](examples/quickstart/main.go) for a runnable example.

## Base URLs

- Sandbox API: `https://api-sandbox.pvium.com/v1`
- Production API: `https://api.pvium.com/v1`
- Sandbox app: `https://sandbox.pvium.com`
- Production app: `https://pvium.com`

## Configuration

`pvium.Init(config)` supports:

- `ClientID` for OAuth invite link generation.
- `APIKey` for authenticated API requests and OAuth code exchange.
- `Environment`, either `pvium.EnvironmentSandbox` or `pvium.EnvironmentProduction`.
- `BaseURL` to override the API base URL.
- `ConsentHost` to override the OAuth consent host.
- `Timeout`
- `DefaultHeader`

## API Endpoints

The `sdk.Endpoints` service exposes Pvium API operations:

- `CreateInvoice(ctx, body, options)`
- `ListInvoices(ctx, options)`
- `GetInvoiceStatus(ctx, code, options)`
- `CancelInvoice(ctx, id, options)`
- `GetInstallmentPayments(ctx, id, options)`

All API methods accept `*pvium.RequestOptions`. Pass `AccessToken` to make a Bearer-token request on behalf of an authorized user. When `AccessToken` is present, the SDK suppresses the configured `APIKey`.

```go
invoice, err := sdk.Endpoints.CreateInvoice(ctx, pvium.CreateInvoiceRequest{
	"name": "Invoice #1042",
	"items": []map[string]any{
		{"name": "Design work", "quantity": 1, "amount": 250},
	},
}, &pvium.RequestOptions{AccessToken: accessToken})
if err != nil {
	log.Fatal(err)
}

invoices, err := sdk.Endpoints.ListInvoices(ctx, &pvium.RequestOptions{
	AccessToken: accessToken,
})
if err != nil {
	log.Fatal(err)
}

fmt.Println(invoice.Data, len(invoices.Data))
```

## OAuth

The `sdk.OAuth` service exposes OAuth helper operations:

- `ExchangeCodeForToken(ctx, input, options)`
- `RefreshAccessToken(ctx, input, options)`
- `GetAccessTokenFromRefreshToken(ctx, input, options)`
- `GetUserInfo(ctx, options)`

When creating invite bundles for an OAuth flow, pass `RedirectURI` so generated invite links include the registered callback URL:

```go
signed, err := sdk.Invites.CreateSignedBundle(
	pvium.OAuthInviteBundleInput{
		Identities: []pvium.InviteIdentity{
			{Type: pvium.InviteIdentityGitHub, Value: "octocat"},
		},
		Scopes:      []string{"read:user", "read:github", "write:invoice"},
		RedirectURI: "https://example.com/api/pvium/oauth/callback",
		Chain:       "ethereum",
	},
	pvium.OAuthInviteSigner{
		Chain:      "ethereum",
		PrivateKey: os.Getenv("PVIUM_INVITE_SIGNER_PRIVATE_KEY"),
	},
)
if err != nil {
	log.Fatal(err)
}

fmt.Println(signed.InviteLinks)
```

Exchange the returned OAuth code on your server:

```go
tokens, err := sdk.OAuth.ExchangeCodeForToken(ctx, pvium.ExchangeAuthorizationCodeInput{
	Code:        code,
	RedirectURI: "https://example.com/api/pvium/oauth/callback",
}, nil)
if err != nil {
	log.Fatal(err)
}

invoice, err := sdk.Endpoints.CreateInvoice(ctx, invoiceBody, &pvium.RequestOptions{
	AccessToken: tokens.Data.AccessToken,
})
if err != nil {
	log.Fatal(err)
}
```

Cancel an invoice by marking the underlying invoice contract inactive:

```go
_, err = sdk.Endpoints.CancelInvoice(ctx, invoiceID, &pvium.RequestOptions{
	AccessToken: tokens.Data.AccessToken,
})
if err != nil {
	log.Fatal(err)
}
```

Refresh an expired access token on your server with the refresh token returned by the code exchange. This calls `POST /v1/client-apps/oauth2/token` with `grantType: "refresh_token"` and returns a new access token:

```go
refreshed, err := sdk.OAuth.GetAccessTokenFromRefreshToken(ctx, pvium.RefreshAccessTokenInput{
	RefreshToken: tokens.Data.RefreshToken,
}, nil)
if err != nil {
	log.Fatal(err)
}

_, err = sdk.Endpoints.ListInvoices(ctx, &pvium.RequestOptions{
	AccessToken: refreshed.Data.AccessToken,
})
```

`RefreshAccessToken(ctx, input, options)` is also available and uses the same backend flow.

## Webhooks

Pvium webhooks are delivered as JSON `POST` requests to the `webhookUrl` configured on your client app. See the TypeScript SDK README for the canonical webhook body format, JWT semantics, [event catalog and payload shapes](../node-sdk/README.md#events-and-payloads), and [idempotency guidance](../node-sdk/README.md#idempotency-and-retries).

### `ResolvePviumWebhookPayload` (recommended)

The high-level helper verifies `body["token"]` when present, enforces that the token event matches the outer `event` or `type`, and returns the unwrapped payload.

```go
var body map[string]any
if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
	http.Error(w, "invalid JSON", http.StatusBadRequest)
	return
}

webhook, err := pvium.ResolvePviumWebhookPayload(
	body,
	os.Getenv("PVIUM_WEBHOOK_SECRET"),
)
if err != nil {
	http.Error(w, "invalid webhook", http.StatusUnauthorized)
	return
}

if webhook.Event == "oauth.invite.accepted" {
	data := webhook.Data
	fmt.Println(data["appId"])
}
```

If `body["token"]` is absent, the helper falls back to unsigned body data. Use that only in development.

### `VerifyPviumWebhookToken` (low-level)

Use this when you already have the raw token.

```go
payload, err := pvium.VerifyPviumWebhookToken(
	token,
	os.Getenv("PVIUM_WEBHOOK_SECRET"),
	&pvium.VerifyPviumWebhookTokenOptions{
		ExpectedEvent: "oauth.invite.accepted",
	},
)
if err != nil {
	log.Fatal(err)
}

fmt.Println(payload.Event, payload.Data, payload.Iat, payload.Exp)
```

`VerifyPviumWebhookToken` accepts `ExpectedEvent`, `Now`, and `AllowHashedSecretFallback`. See the TypeScript SDK README's [`verifyPviumWebhookToken` section](../node-sdk/README.md#verifypviumwebhooktoken-low-level) for shared verification behavior.

## OAuth Invite Links

The SDK can generate signed OAuth invite bundles for app invites and batch payment invites. See the TypeScript SDK README for the canonical explanation of bundle fields, generated OAuth query parameters, [scope values](../node-sdk/README.md#oauth-scopes), and shared state semantics.

### App Invite Example

```go
sdk := pvium.Init(pvium.Config{
	Environment: pvium.EnvironmentSandbox,
	APIKey:      os.Getenv("PVIUM_API_KEY"),
	ClientID:    os.Getenv("PVIUM_CLIENT_ID"),
})

bundle, err := sdk.Invites.CreateBundle(pvium.OAuthInviteBundleInput{
	Identities: []pvium.InviteIdentity{
		{Type: pvium.InviteIdentityEmail, Value: "payee@example.com"},
		{Type: pvium.InviteIdentityHandle, Value: "payee_handle"},
		{Type: pvium.InviteIdentityWallet, Value: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"},
	},
	Scopes: []string{"read:user", "read:ethereum_wallet"},
	Chain:  "ethereum",
	StateParams: map[string]any{
		"source": "admin-invite",
	},
})
if err != nil {
	log.Fatal(err)
}

signed, err := sdk.Invites.SignBundle(bundle, pvium.OAuthInviteSigner{
	Chain:      "ethereum",
	PrivateKey: os.Getenv("PVIUM_INVITE_SIGNER_PRIVATE_KEY"),
})
if err != nil {
	log.Fatal(err)
}

fmt.Println(signed.InviteLinks)
fmt.Println(signed.GroupInviteLink)

_, err = sdk.Invites.CommitBundle(ctx, signed, nil)
if err != nil {
	log.Fatal(err)
}
```

### Batch Invite Example

Use `BatchInvite.BatchID` for batch payment invite bundles. The SDK adds `batchId` as its own OAuth query parameter so `State` remains available for caller state.

```go
signed, err := sdk.Invites.CreateSignedBundle(
	pvium.OAuthInviteBundleInput{
		Identities: []pvium.InviteIdentity{
			{
				Type:                pvium.InviteIdentityEmail,
				Value:               "payee@example.com",
				DefaultPayoutAmount: 250,
			},
		},
		Scopes: []string{"read:user", "read:ethereum_wallet"},
		Chain:  "ethereum",
		BatchInvite: &pvium.OAuthInviteBatchOptions{
			BatchID: "batch_123",
			StateParams: map[string]any{
				"source": "bulk-payments",
			},
		},
	},
	pvium.OAuthInviteSigner{
		Chain:      "ethereum",
		PrivateKey: os.Getenv("PVIUM_INVITE_SIGNER_PRIVATE_KEY"),
	},
)
if err != nil {
	log.Fatal(err)
}

_, err = sdk.Invites.CommitBundle(ctx, signed, nil)
if err != nil {
	log.Fatal(err)
}
```

For batch bundles, `CommitBundle` posts to `/v1/batch-payments/:batchId/invites`. For non-batch bundles, it posts to `/v1/client-apps/:clientId/invites`.

### OAuth State

OAuth `state` is caller-owned state. Pass a plain state string when you already have one:

```go
bundle, err := sdk.Invites.CreateBundle(pvium.OAuthInviteBundleInput{
	Identities: []pvium.InviteIdentity{
		{Type: pvium.InviteIdentityEmail, Value: "payee@example.com"},
	},
	Chain: "ethereum",
	State: "return-to-admin",
})
```

Use `StateParams` when you want the SDK to encode multiple state values:

```go
bundle, err := sdk.Invites.CreateBundle(pvium.OAuthInviteBundleInput{
	Identities: []pvium.InviteIdentity{
		{Type: pvium.InviteIdentityEmail, Value: "payee@example.com"},
	},
	Chain: "ethereum",
	State: "return-to-admin",
	StateParams: map[string]any{
		"campaign":    "spring",
		"redirectTab": "payees",
	},
})
```

For compatibility, bundles without custom state still use `b_<batchId>` as legacy batch state. New integrations should read batch identity from the explicit `batchId` query parameter.

### Wallet Signing Callbacks

In client-side or wallet-mediated flows, do not pass a private key. Pass signing callbacks instead. You can use one `SignMessage` callback, or separate callbacks for the master secret and invite root messages.

```go
signed, err := sdk.Invites.SignBundle(bundle, pvium.OAuthInviteSigner{
	Chain:         "ethereum",
	SignerAddress: walletAddress,
	SignMasterSecret: func(message string) (string, error) {
		return wallet.SignMessage(message)
	},
	SignInviteRoot: func(message string) (string, error) {
		return wallet.SignMessage(message)
	},
	SignMessage: func(message string) (string, error) {
		return wallet.SignMessage(message)
	},
})
```

### One-Step Commit

Use `CreateSignedAndCommit` when you do not need to inspect or display generated links before submitting the bundle.

```go
_, err := sdk.Invites.CreateSignedAndCommit(
	ctx,
	pvium.OAuthInviteBundleInput{
		Identities: []pvium.InviteIdentity{
			{Type: pvium.InviteIdentityEmail, Value: "payee@example.com"},
		},
		Chain: "ethereum",
	},
	pvium.OAuthInviteSigner{
		Chain:      "ethereum",
		PrivateKey: os.Getenv("PVIUM_INVITE_SIGNER_PRIVATE_KEY"),
	},
	nil,
)
```

## Payout Workflows

The `sdk.Payouts` service supports Instant, Scheduled, Milestone, and Escrow payouts. Server-side integrations may pass a private key as the signer; wallet-based flows can pass signing callbacks on `PayoutSignerInput`.

Supported payout chains are `base`, `bsc`, `solana`, `base-testnet`, `solana-testnet`, and `localhost`. Supported payout currencies are `USDC` and `USDT`. For payment `Token`, pass `"usdc"`, `"usdt"`, or the supported token address or mint for the selected chain. The SDK maps it to the configured token address or mint and derives decimals.

Single-payout responses are payout intent objects. They keep `Meta` and `Data`
fields for compatibility, and expose payout fields plus proxy methods directly.

Batch detail responses include up to 250 embedded `Payments`. If the payout has
more, the response indicates truncation with fields such as `paymentsTruncated`,
`paymentsLimit`, and `paymentCount`. Use the dedicated paginated payment list
for larger payouts:

```go
payments, err := payoutIntent.ListPayments(ctx, &pvium.PayoutPaymentsListQuery{
	Page:    1,
	PerPage: 100,
}, nil)
if err != nil {
	log.Fatal(err)
}

for _, payment := range payments.Data {
	fmt.Println(payment["id"], payment["amount"])
}
```

Payout intents also expose management helpers for the batch and its payments:

```go
_, err = payoutIntent.EditPayment(ctx, paymentID, pvium.UpdatePayoutPaymentInput{
	Memo: "Updated memo",
}, nil)

_, err = payoutIntent.DeletePayment(ctx, paymentID, nil)
_, err = payoutIntent.RevokeInvite(ctx, inviteID, nil)
_, err = payoutIntent.Delete(ctx, nil)
```

### Instant Payouts

Instant payouts are created with payees, then finalized. Finalization signs the batch data so the payout cannot be modified silently after approval.

```go
payoutIntent, err := sdk.Payouts.Create(ctx, pvium.CreatePayoutInput{
	Type:  pvium.PayoutTypeInstant,
	Chain: string(pvium.PayoutChainBase),
	Name:  "Creator payroll",
	Payments: []pvium.PayoutPayment{
		{
			Receiver: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			Amount:   100,
			Token:    "usdc",
			Memo:     "February work",
		},
	},
}, nil)
if err != nil {
	log.Fatal(err)
}

_, err = payoutIntent.Finalize(
	ctx,
	pvium.PayoutSignerInput{PrivateKey: os.Getenv("PVIUM_SIGNER_PRIVATE_KEY")},
	pvium.PayoutFinalizeOptions{},
	nil,
)
if err != nil {
	log.Fatal(err)
}
```

### Scheduled Payouts

Scheduled payouts create Merkle proofs and a funding signature during finalization. Fund the returned `FundingURL` from the payer-facing payment screen.

Use `PayoutCurrency` with `pvium.PayoutCurrencyUSDC` / `pvium.PayoutCurrencyUSDT` or the matching lowercase symbol. When `PayoutCurrency` is set, omit per-payment `Token`; the SDK uses the payout currency for every payment and derives decimals.

```go
scheduleDate := int64(1777488000)

payoutIntent, err := sdk.Payouts.Create(ctx, pvium.CreatePayoutInput{
	Type:           pvium.PayoutTypeScheduled,
	Chain:          string(pvium.PayoutChainBase),
	Name:           "March creator payouts",
	PayoutCurrency: string(pvium.PayoutCurrencyUSDC),
	ScheduleDate:   scheduleDate,
	Payments: []pvium.PayoutPayment{
		{
			Receiver: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			Amount:   100,
			Memo:     "March work",
		},
	},
}, nil)
if err != nil {
	log.Fatal(err)
}

finalized, err := payoutIntent.Finalize(
	ctx,
	pvium.PayoutSignerInput{PrivateKey: os.Getenv("PVIUM_SIGNER_PRIVATE_KEY")},
	pvium.PayoutFinalizeOptions{},
	nil,
)
if err != nil {
	log.Fatal(err)
}

fmt.Println(finalized.FundingURL)
```

### Milestone Payouts

Milestone payouts use the scheduled payout machinery with `Type: PayoutTypeMilestone`. The SDK marks them as commitments for the API and injects `metadata.commitmentType = "milestone"`.

Create the commitment with the milestone structure in `Metadata`; actual payee payments can be added later after a recipient is selected. Milestone `dueDate` values, and the `scheduledDate` metadata field, should be ISO date strings. Once the milestone payout has payments, finalize it and fund the returned `FundingURL` before payees can claim milestone payments.

```go
commitment, err := sdk.Payouts.Create(ctx, pvium.CreatePayoutInput{
	Type:           pvium.PayoutTypeMilestone,
	Chain:          string(pvium.PayoutChainBase),
	Name:           "Website build",
	PayoutCurrency: string(pvium.PayoutCurrencyUSDC),
	Metadata: map[string]any{
		"gracePeriod":         7 * 24 * 60 * 60,
		"disapprovalDeadline": 24 * 60 * 60,
		"fundingOption":       "lock",
		"milestones": []map[string]any{
			{
				"name":    "Design approval",
				"amount":  500,
				"dueDate": "2026-07-01T00:00:00.000Z",
				"status":  "pending",
			},
			{
				"name":    "Production release",
				"amount":  1500,
				"dueDate": "2026-08-01T00:00:00.000Z",
				"status":  "pending",
			},
		},
	},
}, nil)
if err != nil {
	log.Fatal(err)
}

fmt.Println(commitment.ID)
```

### Escrow Payouts

Escrow payouts are funded before payees are attached:

1. Create and finalize the escrow payout. This produces the escrow batch hash and funding signature.
2. Fund the escrow on the payer-facing payment screen.
3. Add payees only after the escrow status is `funded`.

When you add payments to a funded escrow payout object, `AddPayments` creates a linked Scheduled child payout and finalizes/signs it automatically using the provided `Signer`. This is the same signer flow used for scheduled payouts. The child batch is hidden from the top-level batch list; payees appear under the escrow.

```go
lockDuration := int64(7 * 24 * 60 * 60)

escrow, err := sdk.Payouts.Create(ctx, pvium.CreatePayoutInput{
	Type:           pvium.PayoutTypeEscrow,
	Chain:          string(pvium.PayoutChainBase),
	Name:           "Open creator escrow",
	LockDuration:   &lockDuration,
	PayoutCurrency: string(pvium.PayoutCurrencyUSDC),
}, nil)
if err != nil {
	log.Fatal(err)
}

finalizedEscrow, err := escrow.Finalize(
	ctx,
	pvium.PayoutSignerInput{PrivateKey: os.Getenv("PVIUM_SIGNER_PRIVATE_KEY")},
	pvium.PayoutFinalizeOptions{},
	nil,
)
if err != nil {
	log.Fatal(err)
}

fmt.Println(finalizedEscrow.FundingURL)

// Fund finalizedEscrow.FundingURL in the Pvium payment UI, then refresh
// the escrow from the API so Status is "funded".
fundedEscrow, err := sdk.Payouts.Get(ctx, escrow.ID, nil)
if err != nil {
	log.Fatal(err)
}

claimDate := int64(1777488000)

_, err = fundedEscrow.AddPayments(ctx, pvium.AddPayoutPaymentsInput{
	Payments: []pvium.PayoutPayment{
		{
			Receiver: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			Amount:   100,
			Memo:     "Approved payout",
		},
	},
	Signer: &pvium.PayoutSignerInput{
		PrivateKey: os.Getenv("PVIUM_SIGNER_PRIVATE_KEY"),
	},
	FinalizeOptions: &pvium.PayoutFinalizeOptions{
		ClaimDate: claimDate,
	},
}, nil)
if err != nil {
	log.Fatal(err)
}
```

If `AddPayments` receives a normal payout id or non-escrow payout object, it uses the standard add-payments endpoint. If it receives an escrow payout object, it requires `Signer` and runs the linked scheduled payout creation/finalization flow for you.

## Package Layout

- `config`: environment/config constants and defaults
- `models`: request/response and domain models
- `transport`: HTTP client and transport helpers
- `services`: endpoints, OAuth, payouts, and invites services
- `crypto`: signing, hashing, nonce, and invite Merkle utilities
- `webhooks`: webhook token and payload verification helpers
- `tests/parity`: fixture-based parity tests
- `testdata/fixtures`: canonical TypeScript parity fixtures
- `examples`: runnable examples
- `docs`: release and maintenance notes

## Tests

```bash
go test ./...
```

## Notes

- Responses are parsed JSON payloads from the API.
- Path handling prevents duplicate `/v1` when your base URL already ends with `/v1`.
- Keep invite signing private keys on trusted servers only. Browser and wallet-mediated flows should use signing callbacks.
- Shared event catalogs, payload examples, scope tables, and other schema-oriented explanations live in [../node-sdk/README.md](../node-sdk/README.md).

## Versioning

Use semantic version tags. See [docs/releasing.md](docs/releasing.md).
