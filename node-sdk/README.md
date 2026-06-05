# Pvium TypeScript SDK

Pvium provides programmable zero-custody stablecoin payments for the global gig economy.

This SDK provides typed access to Pvium APIs and helpers for generating signed OAuth invite links, including Merkle-root-backed invite bundles for invite and batch payment workflows.

- Developer documentation: https://pvium.gitbook.io/documentation
- Pvium website: https://pvium.com
- Pvium sandbox: https://sandbox.pvium.com

## Install

```bash
npm install @pvium/sdk
```

## Quick Start

```ts
import { PviumSdk } from "@pvium/sdk";

const pvium = PviumSdk.init({
  environment: "sandbox",
  apiKey: process.env.PVIUM_API_KEY as string,
});

async function run() {
  const invoices = await pvium.endpoints.listInvoices();
  console.log(invoices.meta.success, invoices.data.length);
}

run().catch(console.error);
```

## Base URLs

- Sandbox API: `https://api-sandbox.pvium.com/v1`
- Production API: `https://api.pvium.com/v1`
- Sandbox app: `https://sandbox.pvium.com`
- Production app: `https://pvium.com`

## Configuration

`PviumSdk.init(config)` supports:

- `clientId` for OAuth invite link generation.
- `apiKey` for authenticated API requests && for OAuth code exchange.
- `environment`, either `sandbox` or `production`.
- `baseUrl` to override the API base URL.
- `consentHost` to override the OAuth consent host.
- `timeoutMs`
- `fetchFn`
- `defaultHeaders`

## API Endpoints

The `pvium.endpoints` service exposes Pvium API operations:

- `createInvoice(body)`
- `listInvoices()`
- `getInvoiceStatus(code)`
- `getInstallmentPayments(id)`

All API methods accept a typed request options object. Pass `accessToken` to
make a Bearer-token request on behalf of an authorized user. When `accessToken`
is present, the SDK suppresses the configured `apiKey`.

```ts
await pvium.endpoints.createInvoice(invoiceBody, { accessToken });
await pvium.endpoints.listInvoices({ accessToken });
```

## OAuth

The `pvium.oauth` service exposes OAuth helper operations:

- `exchangeCodeForToken({ code, redirectUri })`
- `refreshAccessToken({ refreshToken })`
- `getAccessTokenFromRefreshToken({ refreshToken })`
- `getUserInfo({ accessToken })`

When creating invite bundles for an OAuth flow, pass `redirectUri` so generated
invite links include the registered callback URL:

```ts
const signed = await pvium.invites.createSignedBundle(
  {
    identities: [{ type: "github", value: "octocat" }],
    scopes: ["read:user", "read:github", "write:invoice"],
    redirectUri: "https://example.com/api/pvium/oauth/callback",
    chain: "ethereum",
  },
  {
    chain: "ethereum",
    privateKey: process.env.PVIUM_INVITE_SIGNER_PRIVATE_KEY as string,
  },
);
```

Exchange the returned OAuth code on your server:

```ts
const tokens = await pvium.oauth.exchangeCodeForToken({
  code,
  redirectUri: "https://example.com/api/pvium/oauth/callback",
});

const invoice = await pvium.endpoints.createInvoice(invoiceBody, {
  accessToken: tokens.data.accessToken,
});
```

Cancel an invoice by marking the underlying invoice contract inactive:

```ts
await pvium.endpoints.cancelInvoice(invoice.data.id, {
  accessToken: tokens.data.accessToken,
});
```

Refresh an expired access token on your server with the refresh token returned
by the code exchange. This calls `POST /v1/client-apps/oauth2/token` with
`grantType: "refresh_token"` and returns a new access token:

```ts
const refreshed = await pvium.oauth.getAccessTokenFromRefreshToken({
  refreshToken: tokens.data.refreshToken as string,
});

await pvium.endpoints.listInvoices({
  accessToken: refreshed.data.accessToken,
});
```

`refreshAccessToken({ refreshToken })` is also available and uses the same
backend flow.

## Webhooks

Pvium webhooks are delivered as a JSON `POST` to the `webhookUrl` configured on
your client app. The request body has the shape:

```json
{ "event": "<event-name>", "token": "<signed-jwt>", "data": { /* optional */ } }
```

The `token` is an HS256-signed JWT whose payload is
`{ event, data, iat, exp }`. The signing secret is the `webhookSecret` shown
when configuring or regenerating the client app's webhook secret. Always verify
the token before trusting the body — `event`/`data` on the outer body are
convenience fields and are not authenticated on their own.

### `resolvePviumWebhookPayload` (recommended)

The high-level helper. Verifies `body.token` (if present), enforces that the
token's `event` matches the outer body's `event`/`type`, and returns the
unwrapped payload.

```ts
import { resolvePviumWebhookPayload } from "@pvium/sdk";

const body = await request.json();
const webhook = resolvePviumWebhookPayload(
  body,
  process.env.PVIUM_WEBHOOK_SECRET as string,
);

if (webhook.event === "oauth.invite.accepted") {
  const data = webhook.data; // verified, typed via generic
}
```

Returns `{ event, data, tokenPayload? }`. If `body.token` is absent it falls
back to the unsigned `body.data` — only do this in dev. Throws on signature,
expiry, or event-mismatch failures.

### `verifyPviumWebhookToken` (low-level)

Use this when you already have the raw token (e.g. you store the entire JWT for
audit, or you receive the token from somewhere other than a webhook POST).

```ts
import { verifyPviumWebhookToken } from "@pvium/sdk";

const payload = verifyPviumWebhookToken<{ appId: string }>(
  token,
  process.env.PVIUM_WEBHOOK_SECRET as string,
  {
    expectedEvent: "oauth.invite.accepted", // optional; throws on mismatch
    now: Date.now(),                         // optional; for testing/clock skew
    allowHashedSecretFallback: true,         // optional; default true
  },
);

console.log(payload.event, payload.data, payload.iat, payload.exp);
```

Signature:

```ts
function verifyPviumWebhookToken<TData = Record<string, unknown>>(
  token: string,
  secret: string,
  options?: {
    expectedEvent?: string;
    now?: Date | number;
    allowHashedSecretFallback?: boolean;
  },
): { event?: string; data?: TData; iat?: number; exp?: number; [k: string]: unknown };
```

Throws when the JWT shape is wrong, the algorithm isn't `HS256`, the signature
doesn't match either the raw `secret` or its SHA-256 (for legacy hashed
secrets — disable via `allowHashedSecretFallback: false`), the token has
expired, or `expectedEvent` is set and doesn't match.

### Events and payloads

The current event catalog and the shape of `data` for each. All payloads carry
`appId` (your client app's `_id`). Sample objects below are illustrative.

#### `contract.created`

Fired after a contract is created against your app. `paymentData` is omitted.

```jsonc
{
  "appId": "65f...",
  "contract": {
    "id": "8a3...",
    "name": "Invoice #1042",
    "code": "INV-1042",
    "appId": "65f...",
    "user": "67d...",
    "contractType": "Invoice"
  },
  "paymentData": null
}
```

#### `payment.attached`

Fired when a transfer is attached to a contract installment.

```jsonc
{
  "appId": "65f...",
  "contract": { "id": "8a3...", "name": "Invoice #1042" },
  "paymentData": {
    "id": "9b2...",
    "amount": 250,
    "paymentDate": "2026-05-12T18:31:04.000Z",
    "transactionHash": "0xabc..."
  }
}
```

#### `oauth.invite.accepted`

Fired when an invited identity completes the OAuth flow against your app.

```jsonc
{
  "appId": "65f...",
  "clientId": "app_abcd1234",
  "pviumUserId": "67d...",
  "pviumHandle": "alice",
  "githubLogin": "alice-gh",
  "user": {
    "id": "67d...",
    "handle": "alice",
    "email": "alice@example.com",
    "githubLogin": "alice-gh"
  },
  "authorization": {
    "id": "aa1...",
    "scopes": ["profile", "payments.read"],
    "expiresAt": "2027-05-12T18:31:04.000Z",
    "expiresIn": 31536000,
    "tokenType": "Bearer"
  },
  "accessToken": "access_...",
  "refreshToken": "refresh_...",
  "expiresAt": "2027-05-12T18:31:04.000Z",
  "expiresIn": 31536000,
  "tokenType": "Bearer",
  "invite": {
    "identityType": "email",
    "identityValue": "alice@example.com",
    "batchId": "uuid-or-null",
    "acceptedAt": "2026-05-12T18:31:04.000Z"
  }
}
```

#### `batch.payee.added`

Fired when a payee gets attached to a batch. Emitted in two flows: when an
invitee accepts OAuth and is auto-attached to the linked batch, and when a
batch admin directly adds a recipient. The exact field set therefore varies
slightly between flows — code against the union below.

```jsonc
{
  "appId": "65f...",
  "clientId": "app_abcd1234",       // OAuth-flow only
  "batch": {
    "id": "uuid-batch",
    "chain": "base",                // batchWebhook flow
    "status": "pending"             // batchWebhook flow
  },
  "payee": {
    "identityType": "email",
    "identityValue": "alice@example.com",
    "receiver": "0xRecipient...",
    "amount": 100,
    "memo": "Bonus payment",        // direct-add flow
    "attachedAt": "2026-05-12T..."  // OAuth-flow only
  },
  "user": {
    "id": "67d...",
    "handle": "alice",
    "email": "alice@example.com"
  },
  "invite": {                        // shape varies by flow
    "id": "inv-id",
    "batchId": "uuid-batch",
    "identityType": "email",
    "identityValue": "alice@example.com",
    "acceptedAt": "2026-05-12T..."
  }
}
```

#### `batch.funded`

Fired when on-chain funding lands for a batch. Two emit paths:

- **Instant batches** — fires once when the single instant-pay transaction is
  observed (the early-return on `batchTransactionHash` guarantees no replays).
- **Merkle batches** — fires on every funding transaction (each unique tx
  hash). Use `batch.fullyFunded` to detect the transition into fully-funded.

```jsonc
{
  "appId": "65f...",
  "batch": {
    "id": "uuid-batch",
    "chain": "base",
    "status": "funded",              // or "partially_funded"
    "batchDataHash": "0x...",        // instant flow
    "batchHash": "0x...",            // merkle flow
    "batchTransactionHash": "0x...",
    "batchContract": "0x...",        // instant flow
    "contractAddress": "0x...",      // merkle flow
    "merkleBatchContract": "0x...",  // merkle flow
    "totalFunded": 1500,             // merkle flow
    "fullyFunded": true              // merkle flow
  },
  "funding": {
    "amount": 1500,
    "token": "0xTokenContract...",   // or 0x0 for native
    "payer": "0xPayer...",           // instant flow
    "fundedAt": 1747068664,          // unix seconds
    "transactionHash": "0x..."
  }
}
```

#### `batch.payee.claimed`

Fired when a recipient claims their portion of a Merkle batch on-chain. The
backend debounces duplicate claim events — the webhook only fires the first
time a given payment row transitions from unclaimed to claimed.

```jsonc
{
  "appId": "65f...",
  "batch": {
    "id": "uuid-batch",
    "chain": "base",
    "status": "funded",
    "batchHash": "0x...",
    "merkleBatchContract": "0x..."
  },
  "payee": {
    "paymentId": "uuid-payment",
    "receiver": "0xRecipient...",
    "amount": 250,
    "token": "0xTokenContract...",
    "decimals": 6,
    "memo": "INV-1042:install-3",
    "orderIndex": 0,
    "claimDate": 1747068664
  },
  "claim": {
    "claimedAt": "2026-05-12T18:31:04.000Z",
    "transactionHash": "0xclaim...",
    "onchainAmount": "250000000",
    "onchainToken": "0xTokenContract..."
  }
}
```

### Idempotency and retries

Pvium retries failed deliveries with exponential backoff. Treat your handler
as idempotent — use `transactionHash` (for `batch.funded` / `batch.payee.claimed`),
`paymentData.id` (for `payment.attached`), or `invite.id` / `authorization.id`
(for OAuth) as the dedup key.

## OAuth Invite Links

The SDK can generate signed OAuth invite bundles for app invites and batch payment invites. A bundle contains:

- `inviteLinks`: one OAuth link per invited identity.
- `groupInviteLink`: a shared OAuth link backed by the same invite root.
- `root`: the signed Merkle root payload to submit to the Pvium API.
- `invites`: invite commitments and proofs submitted with the root.

Generated links use `/oauth2/authorize` and include the standard OAuth values plus invite-specific query parameters:

- `client_id`
- `response_type=code`
- `scope`
- `state`, when provided.
- `batchId`, for batch payment invite bundles.
- `invite_nonce`, `invite_secret`, `identity_type`, and `identity_hint` for identity-specific links.
- `batch_link_secret` for group invite links.

### OAuth Scopes

Use these scope values in the `scopes` array when creating OAuth invite bundles.

| Category       | Scope                        | Description                                                   |
| -------------- | ---------------------------- | ------------------------------------------------------------- |
| Invoices       | `read:invoice`               | Read invoices created by this app.                            |
| Invoices       | `write:invoice`              | Create and update invoices for this app.                      |
| Invoices       | `read:accepted_invoice`      | Read invoices accepted by authorized users.                   |
| Invoices       | `write:accepted_invoice`     | Create and update accepted invoice records.                   |
| User           | `read:user`                  | Read the authorized user basic profile (handle, email, name). |
| User           | `read:business_profile`      | Read business profiles linked to the authorized user.         |
| User           | `write:business_profile`     | Create and update business profiles.                          |
| Wallets        | `read:ethereum_wallet`       | Read authorized Ethereum wallet details.                      |
| Wallets        | `read:solana_wallet`         | Read authorized Solana wallet details.                        |
| KYC and AML    | `read:kyc_status`            | Read KYC verification status.                                 |
| KYC and AML    | `read:aml_status`            | Read AML screening status.                                    |
| KYC and AML    | `read:kyc_legal_name`        | Read verified legal name details.                             |
| KYC and AML    | `read:kyc_country`           | Read verified country details.                                |
| KYC and AML    | `read:kyc_tax_id`            | Read verified tax ID details.                                 |
| KYC and AML    | `read:kyc_dob`               | Read verified date of birth details.                          |
| KYC and AML    | `read:kyc_address`           | Read verified address details.                                |
| KYC and AML    | `read:kyc_document_metadata` | Read verification document metadata.                          |
| Batch Payments | `read:batch_payment`         | Read batch payment records.                                   |
| Batch Payments | `write:batch_payment`        | Create and update batch payments.                             |

### App Invite Example

```ts
import { PviumSdk } from "@pvium/sdk";

const pvium = PviumSdk.init({
  environment: "sandbox",
  apiKey: process.env.PVIUM_API_KEY as string,
  clientId: process.env.PVIUM_CLIENT_ID as string,
});

const bundle = pvium.invites.createBundle({
  identities: [
    { type: "email", value: "payee@example.com" },
    { type: "handle", value: "payee_handle" },
    { type: "address", value: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e" },
  ],
  scopes: ["read:user", "read:ethereum_wallet"],
  chain: "ethereum",
  stateParams: {
    source: "admin-invite",
  },
});

const signed = await pvium.invites.signBundle(bundle, {
  chain: "ethereum",
  privateKey: process.env.PVIUM_INVITE_SIGNER_PRIVATE_KEY as string,
});

console.log(signed.inviteLinks);
console.log(signed.groupInviteLink);

await pvium.invites.commitBundle(signed);
```



### Batch Invite Example

Use `batchInvite.batchId` for batch payment invite bundles. The SDK adds `batchId` as its own OAuth query parameter so `state` remains available for caller state.

```ts
const signed = await pvium.invites.createSignedBundle(
  {
    identities: [
      {
        type: "email",
        value: "payee@example.com",
        defaultPayoutAmount: 250,
      },
    ],
    scopes: ["read:user", "read:ethereum_wallet"],
    chain: "ethereum",
    batchInvite: {
      batchId: "batch_123",
      stateParams: {
        source: "bulk-payments",
      },
    },
  },
  {
    chain: "ethereum",
    privateKey: process.env.PVIUM_INVITE_SIGNER_PRIVATE_KEY as string,
  },
);

await pvium.invites.commitBundle(signed);
```

For batch bundles, `commitBundle` posts to `/v1/batch-payments/:batchId/invites`. For non-batch bundles, it posts to `/v1/client-apps/:clientId/invites`.

### OAuth State

OAuth `state` is caller-owned state. Pass a plain state string when you already have one:

```ts
const bundle = pvium.invites.createBundle({
  identities: [{ type: "email", value: "payee@example.com" }],
  chain: "ethereum",
  state: "return-to-admin",
});
```

Use `stateParams` when you want the SDK to encode multiple state values:

```ts
const bundle = pvium.invites.createBundle({
  identities: [{ type: "email", value: "payee@example.com" }],
  chain: "ethereum",
  state: "return-to-admin",
  stateParams: {
    campaign: "spring",
    redirectTab: "payees",
  },
});
```

For compatibility, bundles without custom state still use `b_<batchId>` as legacy batch state. New integrations should read batch identity from the explicit `batchId` query parameter.


## Payout Workflows

The `pvium.payout` service supports Instant, Scheduled, Milestone, and Escrow
payouts. Server-side integrations may pass a private key as the signer; browser
apps should pass wallet signing callbacks instead.

Single-payout responses are returned as payout intent objects. Payout fields are
available at the top level and helper methods can be called directly.

### Instant Payouts

Instant payouts are created with payees, then finalized. Finalization signs the
batch data so the payout cannot be modified silently after approval.

```ts
const payoutIntent = await pvium.payout.create({
  type: "Instant",
  chain: "base",
  name: "Creator payroll",
  payments: [
    {
      receiver: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
      amount: 100,
      token: "usdc",
      memo: "February work",
    },
  ],
});

await payoutIntent.finalize(process.env.PVIUM_SIGNER_PRIVATE_KEY!);
```

### Scheduled Payouts

Scheduled payouts create Merkle proofs and a funding signature during
finalization. Fund the returned `fundingUrl` from the payer-facing payment
screen. Use `payoutCurrency` with `PayoutCurrency.USDC` / `PayoutCurrency.USDT`
or the matching lowercase symbol. When `payoutCurrency` is set, omit per-payment
`token`; the SDK uses `payoutCurrency` for every payment and derives `decimals`.

```ts
const payoutIntent = await pvium.payout.create({
  type: "Scheduled",
  chain: "base",
  name: "March creator payouts",
  payoutCurrency: PayoutCurrency.USDC,
  scheduleDate: 1777488000, //unix timestamp in seconds
  payments: [
    {
      receiver: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
      amount: 100,
      memo: "March work",
    },
  ],
});

const finalized = await payoutIntent.finalize(
  process.env.PVIUM_SIGNER_PRIVATE_KEY!,
);

console.log(finalized.fundingUrl);
```

### Milestone Payouts

Milestone payouts use the scheduled payout machinery with `type: "Milestone"`.
The SDK marks them as commitments for the API. Create the commitment with the
milestone structure in `metadata`; actual payee payments can be added later
after a recipient is selected. Milestone `dueDate` values, and the
`scheduledDate` metadata field, should be ISO date strings. Finalize the
commitment after creating it, then fund the returned `fundingUrl` before payees
can claim milestone payments.

```ts
const commitment = await pvium.payout.create({
  type: "Milestone",
  chain: "base",
  name: "Website build",
  payoutCurrency: PayoutCurrency.USDC,
  metadata: {
    gracePeriod: 7 * 24 * 60 * 60,
    disapprovalDeadline: 24 * 60 * 60,
    milestones: [
      {
        name: "Design approval",
        amount: 500,
        dueDate: "2026-07-01T00:00:00.000Z",
        status: "pending",
      },
      {
        name: "Production release",
        amount: 1500,
        dueDate: "2026-08-01T00:00:00.000Z",
        status: "pending",
      },
    ],
  },
});

const finalizedCommitment = await commitment.finalize(
  process.env.PVIUM_SIGNER_PRIVATE_KEY!,
);

console.log(finalizedCommitment.fundingUrl);
```



### Escrow Payouts

Escrow payouts are funded before payees are attached:

1. Create and finalize the escrow payout. This produces the escrow batch hash
   and funding signature.
2. Fund the escrow on the payer-facing payment screen.
3. Add payees only after the escrow status is `funded`.

When you add payments to a funded escrow payout object, `addPayments` creates a
linked Scheduled child payout and finalizes/signs it automatically using the
provided `signer`. This is the same signer flow used for scheduled payouts. The
child batch is hidden from the top-level batch list; payees appear under the
escrow.

```ts
const escrow = await pvium.payout.create({
  type: "Escrow",
  chain: "base",
  name: "Open creator escrow",
  lockDuration: 7 * 24 * 60 * 60,
  payoutCurrency: "usdc",
});

const finalizedEscrow = await escrow.finalize(
  process.env.PVIUM_SIGNER_PRIVATE_KEY!,
);

// Fund finalizedEscrow.fundingUrl in the Pvium payment UI, then refresh
// the escrow from the API so status is "funded".
const fundedEscrow = await pvium.payout.get(escrow.id);

// This creates a linked
// Scheduled payout for these payees, signs/finalizes it with `signer`, and
// links it back to the funded escrow.
await fundedEscrow.addPayments({
  payments: [
    {
      receiver: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
      amount: 100,
      memo: "Approved payout",
    },
  ],
  signer: process.env.PVIUM_SIGNER_PRIVATE_KEY!,
  finalizeOptions: {
    claimDate: 1777488000,
  },
});
```

### Managing Payout Intents

Pending payout intents can be updated or deleted before finalization. Payment
rows can also be updated or removed while the backend batch is still pending.
After `addPayments`, the API returns the refreshed payout intent with its
`payments` array capped to the backend batch-detail limit. Use
`listPayments({ page, perPage })` for the dedicated paginated payment list, and
use each payment row's `id` for later edits or deletes. When the embedded
payment list is capped, `payoutIntent.paymentsTruncated` is `true`;
`payoutIntent.paymentsLimit` is the embedded row limit, and
`payoutIntent.paymentCount` is the full payment count.

```ts
const payoutIntent = await pvium.payout.create({
  type: "Instant",
  chain: "base",
  name: "Draft payout",
  payments: [
    {
      receiver: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
      amount: 100,
      token: "usdc",
      memo: "Initial amount",
    },
  ],
});

await payoutIntent.editPayment(123, {
  amount: 125,
  memo: "Adjusted amount",
});

await payoutIntent.deletePayment(123);

const payments = await payoutIntent.listPayments({ page: 1, perPage: 50 });
console.log(payments.meta.pagination?.totalCount, payments.data);

if (payoutIntent.paymentsTruncated) {
  const firstPage = await payoutIntent.listPayments({ page: 1, perPage: 250 });
  console.log(firstPage.meta.pagination?.totalCount, firstPage.data);
}

// Soft-deletes the payout intent. The API only allows this for pending/failed
// batches.
await payoutIntent.delete();
```

Batch invite roots and individual batch invites can be revoked by id:

```ts
await payoutIntent.revokeInviteRoot("invite_root_123");
await payoutIntent.revokeInvite("invite_123");
```



### Browser Wallet Signing

In browser apps, do not pass a private key. Pass signing callbacks instead. You can use one `signMessage` callback, or separate callbacks for the master secret and invite root messages.

```ts
const signed = await pvium.invites.signBundle(bundle, {
  chain: "ethereum",
  signerAddress: walletAddress,
  signMasterSecret: async (message) => wallet.signMessage(message),
  signInviteRoot: async (message) => wallet.signMessage(message),
  signMessage: async (message) => wallet.signMessage(message),
});
```

For Solana, the SDK passes `Uint8Array` messages to the signing callbacks and stores Solana signatures as base64 strings.

### One-Step Commit

Use `createSignedAndCommit` when you do not need to inspect or display generated links before submitting the bundle.

```ts
await pvium.invites.createSignedAndCommit(
  {
    identities: [{ type: "email", value: "payee@example.com" }],
    chain: "ethereum",
  },
  {
    chain: "ethereum",
    privateKey: process.env.PVIUM_INVITE_SIGNER_PRIVATE_KEY as string,
  },
);
```

## Tests

```bash
npm run build
npm test
```

## Notes

- Responses are parsed JSON payloads from the API.
- Path handling prevents duplicate `/v1` when your base URL already ends with `/v1`.
- Keep invite signing private keys on trusted servers only. Browser apps should use wallet signing callbacks.
