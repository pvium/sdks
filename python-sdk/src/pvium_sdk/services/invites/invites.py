from __future__ import annotations

import base64
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Callable, Dict, List, Optional
from urllib.parse import urlencode

from eth_account import Account
from eth_account.messages import encode_defunct

from ...core.client import PviumHttpClient, PviumSdkConfig, resolvePviumConsentHost
from ...crypto.invite_merkle import (
    buildInviteMasterSecretMessage,
    createInviteNonce,
    createRootNonce,
    deriveInviteSecret,
    deriveMasterSecret,
    generateBatchInviteMerkleDataV2,
    normalizeIdentityValue,
    validateIdentityValue,
)
from ...core.types import RequestOptions


def _normalize_scopes(scopes: List[str]) -> List[str]:
    return sorted({s.strip() for s in scopes if s and s.strip()})


def _default_scopes_for_chain(chain: Optional[str]) -> List[str]:
    chain_lower = (chain or "").lower()
    scopes = ["read:user"]
    if "solana" in chain_lower:
        scopes.append("read:solana_wallet")
    elif chain_lower:
        scopes.append("read:ethereum_wallet")
    return _normalize_scopes(scopes)


def _to_iso(seconds: int) -> str:
    return datetime.fromtimestamp(seconds, tz=timezone.utc).isoformat().replace("+00:00", "Z")


def _normalize_state_value(value: Any) -> Optional[str]:
    if value is None:
        return None
    return str(value)


def _ensure_0x(value: str) -> str:
    return value if value.startswith("0x") else f"0x{value}"


def _build_invite_state(state: Optional[str], state_params: Optional[Dict[str, Any]], batch_id: Optional[str]) -> Optional[str]:
    state_params = state_params or {}
    entries = [(k, v) for k, v in state_params.items() if v is not None]
    if not entries:
        return state or (f"b_{batch_id}" if batch_id else None)

    payload: Dict[str, str] = {}
    if state:
        payload["state"] = state
    if batch_id:
        payload["batchId"] = batch_id
    for key, value in entries:
        normalized = _normalize_state_value(value)
        if normalized is not None:
            payload[key] = normalized

    return urlencode(payload)


class PviumInviteService:
    def __init__(self, http: PviumHttpClient, config: PviumSdkConfig):
        self.http = http
        self.config = config

    def createBundle(self, input: Dict[str, Any]) -> Dict[str, Any]:
        client_id = self._require_client_id()
        consent_host = self._require_consent_host()
        identities = input.get("identities") or []
        batch_id = (input.get("batchInvite") or {}).get("batchId") or input.get("batchId")

        if not identities:
            raise RuntimeError("At least one invite identity is required")

        if input.get("batchInvite") and not str(input["batchInvite"].get("batchId", "")).strip():
            raise RuntimeError("batchInvite.batchId is required for batch invite bundles")

        for identity in identities:
            err = validateIdentityValue(identity["type"], identity["value"])
            if err:
                raise RuntimeError(f"Invalid invite identity ({identity['type']}={identity['value']}): {err}")

        return {
            "clientId": client_id,
            "consentHost": consent_host,
            "identities": identities,
            "scopes": _normalize_scopes(input.get("scopes") or _default_scopes_for_chain(input.get("chain"))),
            "batchId": batch_id,
            "batchInvite": input.get("batchInvite") or ({"batchId": batch_id} if batch_id else None),
            "chain": input.get("chain"),
            "state": input.get("state"),
            "stateParams": {
                **((input.get("batchInvite") or {}).get("stateParams") or {}),
                **(input.get("stateParams") or {}),
            },
            "redirectUri": input.get("redirectUri"),
            "createdAt": input.get("createdAt"),
            "rootNonce": input.get("rootNonce"),
        }

    def signBundle(self, bundle: Dict[str, Any], signer: Dict[str, Any]) -> Dict[str, Any]:
        scopes = _normalize_scopes(bundle["scopes"])
        created_at = int(bundle.get("createdAt") or int(time.time()))
        batch_id = bundle.get("batchId") or ""
        root_nonce = bundle.get("rootNonce") or createRootNonce(batch_id, scopes)
        derivation_salt = batch_id or root_nonce

        master_message = buildInviteMasterSecretMessage(derivation_salt)
        master_signature = self._sign_message_for_master_secret(master_message, signer)
        master_secret = deriveMasterSecret(master_signature["signatureHex"])

        invite_entries = []
        for identity in bundle["identities"]:
            invite_nonce = createInviteNonce()
            invite_entries.append(
                {
                    "identityType": identity["type"],
                    "identityValue": identity["value"],
                    "inviteNonce": invite_nonce,
                    "inviteSecret": deriveInviteSecret(master_secret, invite_nonce),
                    "defaultPayoutAmount": identity.get("defaultPayoutAmount"),
                    "expiresAt": identity.get("expiresAt"),
                }
            )

        merkle = generateBatchInviteMerkleDataV2(
            {
                "appClientId": bundle["clientId"],
                "batchId": batch_id or None,
                "chain": bundle.get("chain"),
                "scopes": scopes,
                "createdAt": created_at,
                "rootNonce": root_nonce,
                "invites": invite_entries,
            }
        )

        root_signature = self._sign_root_message(merkle["signatureMessage"], signer)
        signing_chain = signer.get("chain") or bundle.get("chain")
        state = _build_invite_state(bundle.get("state"), bundle.get("stateParams"), batch_id)

        invites = []
        for invite in merkle["invites"]:
            expires_at_iso = _to_iso(invite["expiresAt"]) if invite.get("expiresAt") else None
            invite_link = self._generate_invite_link(
                {
                    "consentHost": bundle["consentHost"],
                    "clientId": bundle["clientId"],
                    "scopes": merkle["scopes"],
                    "state": state,
                    "redirectUri": bundle.get("redirectUri"),
                    "batchId": batch_id or None,
                    "inviteNonce": invite["inviteNonce"],
                    "inviteSecret": invite["inviteSecret"],
                    "identityType": invite["identityType"],
                    "identityHint": invite["identityValue"],
                }
            )
            invites.append(
                {
                    "identityType": invite["identityType"],
                    "identityValue": invite["identityValue"],
                    "identityCommitment": invite["identityCommitment"],
                    "secretHash": invite["secretHash"],
                    "leafVersion": merkle["version"],
                    "inviteNonce": invite["inviteNonce"],
                    "inviteSecret": invite["inviteSecret"],
                    "inviteLink": invite_link,
                    "defaultPayoutAmount": invite.get("defaultPayoutAmount"),
                    "appClientId": bundle["clientId"],
                    "leaf": invite["leaf"],
                    "proof": invite["proof"],
                    "expiresAt": expires_at_iso,
                }
            )

        group_invite_link = self._generate_group_invite_link(
            {
                "consentHost": bundle["consentHost"],
                "clientId": bundle["clientId"],
                "scopes": merkle["scopes"],
                "state": state,
                "redirectUri": bundle.get("redirectUri"),
                "batchId": batch_id or None,
                "masterSecret": master_secret,
            }
        )

        return {
            "clientId": bundle["clientId"],
            "consentHost": bundle["consentHost"],
            "batchId": batch_id,
            "batchInvite": bundle.get("batchInvite"),
            "scopes": merkle["scopes"],
            "chain": bundle.get("chain"),
            "masterSecret": master_secret,
            "root": {
                "root": merkle["root"],
                "nonce": merkle["rootNonce"],
                "signature": root_signature["signature"],
                "signatureType": root_signature["signatureType"],
                "scopes": merkle["scopes"],
                "signatureMessage": merkle["signatureMessage"],
                "signatureTimestamp": merkle["createdAt"],
                "signerAddress": root_signature.get("signerAddress"),
                "inviteCount": merkle["inviteCount"],
                "expiresAt": _to_iso(merkle["expiresAt"]) if merkle.get("expiresAt") else None,
                "metadata": {
                    "version": merkle["version"],
                    "leafEncoding": "PVIUM_INVITE_LEAF_V2",
                    "signingChain": signing_chain,
                },
            },
            "invites": invites,
            "inviteLinks": [invite["inviteLink"] for invite in invites],
            "groupInviteLink": group_invite_link,
            "merkle": merkle,
        }

    def commitBundle(self, bundle: Dict[str, Any], options: Optional[RequestOptions] = None) -> Any:
        batch_id = (bundle.get("batchInvite") or {}).get("batchId") or bundle.get("batchId")
        path = (
            f"/v1/batch-payments/{batch_id}/invites"
            if batch_id
            else f"/v1/client-apps/{bundle['clientId']}/invites"
        )

        response = self.http.request(
            "POST",
            path,
            body={
                "root": bundle["root"],
                "invites": [
                    {k: v for k, v in invite.items() if k not in {"inviteSecret", "inviteLink"}}
                    for invite in bundle["invites"]
                ],
            },
            options=options,
        )
        return self.http.parseResponseBody(response)

    def createSignedBundle(self, input: Dict[str, Any], signer: Dict[str, Any]) -> Dict[str, Any]:
        return self.signBundle(self.createBundle(input), signer)

    def createSignedAndCommit(self, input: Dict[str, Any], signer: Dict[str, Any], options: Optional[RequestOptions] = None) -> Any:
        bundle = self.createSignedBundle(input, signer)
        return self.commitBundle(bundle, options)

    def _sign_message_for_master_secret(self, message: str, signer: Dict[str, Any]) -> Dict[str, Any]:
        chain = signer.get("chain")
        if chain == "ethereum" and signer.get("privateKey"):
            account = Account.from_key(signer["privateKey"])
            signature = account.sign_message(encode_defunct(text=message)).signature.hex()
            return {"signatureHex": signature.replace("0x", "").lower(), "signerAddress": account.address}

        if chain == "ethereum":
            fn = signer.get("signMasterSecret") or signer.get("signMessage")
            if not callable(fn):
                raise RuntimeError("Ethereum signer requires signMessage(message)")
            result = fn(message)
            if isinstance(result, dict):
                signature = result["signature"]
                signer_address = result.get("signerAddress") or signer.get("signerAddress")
            else:
                signature = str(result)
                signer_address = signer.get("signerAddress")
            return {"signatureHex": str(signature).replace("0x", "").lower(), "signerAddress": signer_address}

        fn = signer.get("signMasterSecret") or signer.get("signMessage")
        if not callable(fn):
            raise RuntimeError("Solana signer requires signMessage(message_bytes)")
        result = fn(message.encode("utf-8"))

        signature: Any
        signer_address = signer.get("signerAddress")
        if isinstance(result, dict):
            signature = result["signature"]
            signer_address = result.get("signerAddress") or signer_address
        else:
            signature = result

        if isinstance(signature, bytes):
            signature_hex = signature.hex()
        else:
            try:
                signature_hex = base64.b64decode(str(signature)).hex()
            except Exception:
                signature_hex = str(signature).replace("0x", "").lower()

        return {"signatureHex": signature_hex, "signerAddress": signer_address}

    def _sign_root_message(self, message: str, signer: Dict[str, Any]) -> Dict[str, Any]:
        chain = signer.get("chain")
        if chain == "ethereum" and signer.get("privateKey"):
            account = Account.from_key(signer["privateKey"])
            return {
                "signature": _ensure_0x(account.sign_message(encode_defunct(text=message)).signature.hex()),
                "signatureType": "evm-personal-sign",
                "signerAddress": account.address,
            }

        if chain == "ethereum":
            fn = signer.get("signInviteRoot") or signer.get("signMessage")
            if not callable(fn):
                raise RuntimeError("Ethereum signer requires signMessage(message)")
            result = fn(message)
            if isinstance(result, dict):
                return {
                    "signature": result["signature"],
                    "signatureType": result.get("signatureType") or "evm-personal-sign",
                    "signerAddress": result.get("signerAddress") or signer.get("signerAddress"),
                }
            return {
                "signature": str(result),
                "signatureType": "evm-personal-sign",
                "signerAddress": signer.get("signerAddress"),
            }

        fn = signer.get("signInviteRoot") or signer.get("signMessage")
        if not callable(fn):
            raise RuntimeError("Solana signer requires signMessage(message_bytes)")
        result = fn(message.encode("utf-8"))

        if isinstance(result, bytes):
            signature = base64.b64encode(result).decode("utf-8")
            return {
                "signature": signature,
                "signatureType": "solana-message",
                "signerAddress": signer.get("signerAddress"),
            }

        if isinstance(result, dict):
            return {
                "signature": result["signature"],
                "signatureType": result.get("signatureType") or "solana-message",
                "signerAddress": result.get("signerAddress") or signer.get("signerAddress"),
            }

        return {
            "signature": str(result),
            "signatureType": "solana-message",
            "signerAddress": signer.get("signerAddress"),
        }

    def _generate_invite_link(self, params: Dict[str, Any]) -> str:
        query: Dict[str, Any] = {
            "client_id": params["clientId"],
            "response_type": "code",
            "scope": " ".join(_normalize_scopes(params["scopes"])),
            "invite_nonce": params["inviteNonce"],
            "invite_secret": params["inviteSecret"],
            "identity_type": params["identityType"],
        }
        if params.get("redirectUri"):
            query["redirect_uri"] = params["redirectUri"]
        if params.get("state"):
            query["state"] = params["state"]
        if params.get("batchId"):
            query["batchId"] = params["batchId"]
        if params.get("identityHint"):
            query["identity_hint"] = normalizeIdentityValue(params["identityType"], params["identityHint"])

        return f"{params['consentHost'].rstrip('/')}/oauth2/authorize?{urlencode(query)}"

    def _generate_group_invite_link(self, params: Dict[str, Any]) -> str:
        query: Dict[str, Any] = {
            "client_id": params["clientId"],
            "response_type": "code",
            "scope": " ".join(_normalize_scopes(params["scopes"])),
            "batch_link_secret": params["masterSecret"],
        }
        if params.get("redirectUri"):
            query["redirect_uri"] = params["redirectUri"]
        if params.get("state"):
            query["state"] = params["state"]
        if params.get("batchId"):
            query["batchId"] = params["batchId"]

        return f"{params['consentHost'].rstrip('/')}/oauth2/authorize?{urlencode(query)}"

    def _require_client_id(self) -> str:
        if not self.config.clientId:
            raise RuntimeError("PviumSdkConfig.clientId is required for invite methods")
        return self.config.clientId

    def _require_consent_host(self) -> str:
        return resolvePviumConsentHost(self.config)
