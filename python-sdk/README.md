# Pvium Python SDK

Python SDK equivalent of the Pvium TypeScript SDK.

## Project Structure

```text
python-sdk/
  src/pvium_sdk/
    core/
      client.py
      types.py
      sdk.py
      async_sdk.py
    services/
      invoice/
        endpoints.py
      oauth/
        oauth.py
      invites/
        invites.py
      payout/
        payout.py
      webhooks/
        webhooks.py
      ...compatibility wrapper modules...
    crypto/
      signing.py
      invite_merkle.py
    __init__.py
    ...compatibility wrapper modules...
  tests/
    ...pytest suite...
  pyproject.toml
  README.md
```

## Install

```bash
pip install pvium
```

For local development and tests:

```bash
pip install -e '.[dev]'
```

## Quick Start

```python
from pvium_sdk import PviumSdk, PviumSdkConfig

pvium = PviumSdk.init(
    PviumSdkConfig(
        environment="sandbox",
        apiKey="your_api_key",
        clientId="your_client_id",
    )
)

invoices = pvium.endpoints.listInvoices()
print(invoices)
```

## Scheduled Payout Quick Start

Use scheduled payouts for larger payout batches. If a payout has more than 200
payees, create a scheduled payout instead of an instant payout.

```python
from uuid import uuid4

from pvium_sdk import PayoutCurrency, PviumSdk, PviumSdkConfig

pvium = PviumSdk.init(
    PviumSdkConfig(
        environment="sandbox",
        apiKey="your_api_key",
        clientId="your_client_id",
    )
)

scheduled = pvium.payout.createFinalized(
    {
        "id": str(uuid4()),
        "type": "Scheduled",
        "chain": "base",
        "name": "March creator payouts",
        "payoutCurrency": PayoutCurrency.USDC,
        "scheduleDate": 1777488000,
        "payments": [
            {
                "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
                "amount": 100,
                "memo": "March payout",
            },
            # Add the rest of the payees here.
        ],
    },
    "your_signer_private_key",
    {
        "timestamp": 1777487451,
        "claimDate": 1777488000,
    },
)

print(scheduled.fundingUrl)
```

## Async Usage

`AsyncPviumSdk` preserves the same service surface as `PviumSdk` and runs service calls in a worker thread.

```python
from pvium_sdk import AsyncPviumSdk, PviumSdkConfig

sdk = AsyncPviumSdk.init(
  PviumSdkConfig(baseUrl="https://api-sandbox.pvium.com/v1", apiKey="your_api_key")
)

# await any service method
invoices = await sdk.endpoints.listInvoices()
```

## Run Tests

```bash
pytest
```

## Webhook events

`oauth.invite.accepted` is emitted when an invited user's authorization becomes
active immediately. If the invite scopes require payee screening, the
authorization remains pending until those checks complete; then handle
`oauth.authorization.activated`. That event confirms an active authorization,
not batch-specific payability. For Strict payouts, call
`pvium.payout.isPayable(batch_id, identities)` before adding recipients or
finalizing.

See the [canonical webhook event reference](../node-sdk/README.md#events-and-payloads)
for payload shapes and delivery semantics.

## Services

- `pvium.endpoints`
  - `createInvoice(body, options=None)`
  - `listInvoices(options=None)`
  - `getInvoiceStatus(code, options=None)`
  - `cancelInvoice(invoice_id, options=None)`
  - `getInstallmentPayments(installment_id, options=None)`

- `pvium.oauth`
  - `exchangeCodeForToken(input, options=None)`
  - `refreshAccessToken(input, options=None)`
  - `getAccessTokenFromRefreshToken(input, options=None)`
  - `getUserInfo(options=None)`

- `pvium.invites`
  - `createBundle(input)`
  - `signBundle(bundle, signer)`
  - `createSignedBundle(input, signer)`
  - `commitBundle(bundle, options=None)`
  - `createSignedAndCommit(input, signer, options=None)`

- `pvium.payout`
  - `create(input, options=None)`
  - `createFinalized(input, signer, options=None, request_options=None)`
  - `list(query=None, options=None)`
  - `get(payout_id, options=None)`
  - `addPayments(payout, input, options=None)`
  - `addRecipients(payout_id, input, options=None)`
  - `resolveRecipients(payout_id, input, options=None)`
  - `isPayable(payout_id, identities, options=None)`
  - `removePayments(payout_id, input, options=None)`
  - `deletePayment(payout_id, payment_id, options=None)`
  - `updatePayment(payout_id, payment_id, input, options=None)`
  - `editPayment(payout_id, payment_id, input, options=None)`
  - `listPayments(payout_id, query=None, options=None)`
  - `listInvites(payout_id, options=None)`
  - `revokeInvite(payout_id, invite_id, options=None)`
  - `revokeInviteRoot(payout_id, invite_root_id, options=None)`
  - `delete(payout_id, options=None)`
  - `finalize(payout_input, signer, options=None, request_options=None)`

Single-payout responses are payout intent objects. They keep dictionary
compatibility with `["meta"]` and `["data"]`, and expose payout fields and proxy
methods directly:

```python
payout_intent = pvium.payout.create({
    "type": "Instant",
    "chain": "base",
    "name": "Creator payroll",
    "payments": [
        {
            "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
            "amount": 100,
            "token": "usdc",
        }
    ],
})

finalized = payout_intent.finalize("your_signer_private_key")
print(finalized.fundingUrl)
```

Instant payouts are best for smaller batches. Prefer scheduled payouts when the
batch has more than 200 payees.

Batch detail responses include up to 250 embedded `payments`. If the payout has
more, the response meta/data includes truncation fields such as
`paymentsTruncated`, `paymentsLimit`, and `paymentCount`. Use the paginated
payment list for larger payouts:

```python
payments = payout_intent.listPayments({"page": 1, "perPage": 100})

for payment in payments["data"]:
    print(payment["id"], payment["amount"])
```

You can manage a payout intent and its payments through the intent object:

```python
payout_intent.editPayment(payment_id, {"memo": "Updated memo"})
payout_intent.deletePayment(payment_id)
payout_intent.revokeInvite(invite_id)
payout_intent.delete()
```

For funded escrow payouts, use the intent proxy instead of passing the batch
back into the service:

```python
funded_escrow = pvium.payout.get("escrow_batch_id")

funded_escrow.addPayments({
    "payments": [
        {
            "receiver": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
            "amount": 100,
            "memo": "Approved payout",
        }
    ],
    "signer": "your_signer_private_key",
    "finalizeOptions": {"claimDate": 1777488000},
})
```

### Check recipient payability (read-only)

```python
result = pvium.payout.isPayable(payout_id, [
    {"type": "email", "value": "alice@example.com"},
    {"type": "email", "value": "bob@example.com"},
])
for recipient in result["data"]["recipients"]:
    print(recipient["value"], recipient["isPayable"], recipient["blockers"])
# Payout intent: payout_intent.isPayable(identities, options=None)
# Async client: await async_pvium.payout.isPayable(payout_id, identities)
```

Uses `POST /v1/batch-payments/:batchId/is-payable` with a JSON body:

```json
{"identities":[{"type":"email","value":"alice@example.com"}]}
```

POST and GET use the same read-only handler, validation, and response. The endpoint
requires `read:batch_payment`, allows 1–50 identities and at most 4096 bytes of
JSON-serialized identities, and returns `Cache-Control: no-store`. Split larger
lookups into smaller requests. GET remains available with an `identities`
URL-encoded JSON query parameter; the SDK uses POST so recipient identifiers
are carried in the body instead of the URL.

The response includes the effective `complianceMode` (including a parent pool),
`checksRequired`, `requiredScopes`, and ordered `recipients`. Each recipient has
`type`, normalized `value`, `isPayable`, `isRegistered`, `isInvited`,
`invitationStatus`, `authorizationStatus`, `missingScopes`, and `blockers`.
Unregistered and not-yet-invited identities return results rather than 404s.
Invitation status describes the latest applicable organization or batch invite;
an accepted invitation alone does not establish payability. An existing valid
authorization can make a recipient payable without a new invite.

Strict requires `read:user`, `read:legal_id`, `read:tax_forms` and the batch's
wallet scope (`read:ethereum_wallet` or `read:solana_wallet`). Legacy `read:kyc`
also satisfies `read:legal_id`. Checks include current authorization, payee
identity verification, a current qualifying tax form for the payer business,
and the authorized wallet. Pending payee screening blocks payment; the lookup
does not initiate screening.

For Open batches, `checksRequired` is false, all syntactically valid identities
have `isPayable: true`, and registration/invitation fields are null because no
user lookup is required. This means compliance eligibility, not successful
identity resolution, funding, or execution. The endpoint does not create invites
or payment rows. Strict recipient addition rechecks the same eligibility rules.

## Utilities

The package exports signing helpers, webhook verification helpers, invite merkle helpers, and payout hash helpers similar to the TypeScript SDK.
