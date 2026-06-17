# TypeScript vs Go/Python SDK Parity Checklist

This checklist tracks parity between the TypeScript SDK and the other SDKs.

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

- HTTP/client error parity:
  - expose a typed API error equivalent to TypeScript `PviumApiError`
  - preserve response `status`, `statusText`, parsed response body, and a useful message for non-2xx responses
  - canonical TypeScript test: `node-sdk/test/oauth.test.js` -> `refreshAccessToken rejects non-2xx responses with PviumApiError`
- App invite lookup parity:
  - expose an invite-service helper equivalent to TypeScript `invites.findAppInviteByIdentity`
  - call `GET /v1/batch-payments/app-invites`
  - support query fields `identityType`, `identityValue`, and optional `status`
  - normalize `identityValue` before matching returned invite records
  - return the matched invite, all returned invites, raw response, and an `accepted` boolean
  - canonical TypeScript test: `node-sdk/test/invites.test.js` -> `finds app invites by identity through the app-invites endpoint`
- Invite commit result parity:
  - `commitBundle` should return a structured result rather than only the raw backend response
  - compare returned invite nonces against the submitted bundle's nonces
  - expose `inviteCommitted`, `alreadyAccepted`, `committedInvites`, `existingInvites`, all returned invites, and raw response
  - canonical TypeScript test: `node-sdk/test/invites.test.js` -> `commitBundle detects returned accepted invites with different nonces`
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
