# TypeScript vs Go SDK Parity Checklist

This checklist tracks parity between the TypeScript SDK and this Go SDK.

## Completed in this pass

- OAuth token exchange input overrides:
  - `ExchangeAuthorizationCodeInput.clientId`
  - `ExchangeAuthorizationCodeInput.apiKey`
  - `RefreshAccessTokenInput.clientId`
  - `RefreshAccessTokenInput.apiKey`
- OAuth token exchange now skips `x-api-key` header and uses body `apiKey`, matching TypeScript behavior.
- OAuth now returns explicit errors when required `clientId` or `apiKey` is missing.
- Webhook verification supports SHA-256 hashed secret fallback by default.
- Webhook verification supports `expectedEvent` validation.
- Webhook payload resolution now supports tokenless payloads (`event` + `data`) like TypeScript.
- Invite state encoding parity improvements:
  - no quoted state payloads
  - `batchInvite.stateParams` merged with `stateParams` (input values override batch values)
  - TypeScript-style nested state payload behavior with optional `state` and `batchId`
  - fallback state of `b_<batchId>` when batch invites omit state inputs
- Invite scope normalization parity improvements:
  - trimmed/deduplicated/sorted scopes
  - default scopes now derived from chain and sorted (TypeScript-compatible behavior)
- Invite input validation parity improvements:
  - require at least one identity
  - require non-empty `batchInvite.batchId` when batch invites are used
  - require configured `clientId` for invite methods

## Remaining high-impact gaps

- Invite bundle parity:
  - richer V2 root payload structure and signature metadata
  - invite proof payload/validation parity fields
- Payout parity:
  - high-level `createFinalized` flow
  - escrow payee helper flow parity
  - richer signer/provider adapters and finalize helper ergonomics
- Signing parity:
  - typed ABI encoding/hash behavior matching TypeScript utility signatures across all request types

## Suggested implementation order

1. Invite V2 payload and state encoding parity
2. Payout high-level finalized flows
3. Remaining signing utility shape parity
