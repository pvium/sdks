# Pvium Python SDK

Python SDK equivalent of the Pvium TypeScript SDK.

## Project Structure

```text
pvium-sdk/
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
pip install -e .
```

For development and tests:

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

## Utilities

The package exports signing helpers, webhook verification helpers, invite merkle helpers, and payout hash helpers similar to the TypeScript SDK.
