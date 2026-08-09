from __future__ import annotations

import base64
import hashlib
import json
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Callable, Dict, List, Optional
from urllib.parse import quote, urlencode

from eth_account import Account
from eth_account.messages import encode_defunct

from ...core.client import PviumHttpClient, PviumSdkConfig, resolvePviumConsentHost
from ...crypto.invite_merkle import (
    buildInviteMasterSecretMessage,
    createInviteNonce,
    createInviteSecret,
    createRootNonce,
    deriveInviteSecret,
    deriveMasterSecret,
    generateBatchInviteMerkleDataV2,
    normalizeOrgReferenceId,
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


def _is_evm_invite_chain(chain: Optional[str]) -> bool:
    return chain in {"base", "bsc", "ethereum"}


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


def _normalize_open_invite_identity_types(identity_types: Optional[List[str]]) -> List[str]:
    return sorted({t.strip() for t in (identity_types or []) if t and t.strip()})


def _normalize_open_invite_email_domains(domains: Optional[List[str]]) -> List[str]:
    return sorted({d.strip().lower() for d in (domains or []) if d and d.strip()})


def _compact_json(value: Any) -> str:
    return json.dumps(value, separators=(",", ":"), ensure_ascii=False)


def _sha256_hex(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def _to_iso_millis(value: Any) -> Optional[str]:
    if value is None or value == "":
        return None
    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        dt = datetime.fromtimestamp(value / 1000, tz=timezone.utc)
    else:
        text = str(value).strip()
        if text.endswith("Z"):
            text = text[:-1] + "+00:00"
        dt = datetime.fromisoformat(text)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        dt = dt.astimezone(timezone.utc)
    return dt.strftime("%Y-%m-%dT%H:%M:%S.") + f"{dt.microsecond // 1000:03d}Z"


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
            "signingKey": input.get("signingKey"),
            "orgReferenceId": normalizeOrgReferenceId(input.get("orgReferenceId")),
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
                "signingKey": bundle.get("signingKey"),
                "orgReferenceId": bundle.get("orgReferenceId"),
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
                    "leafVersion": "2",
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

        root = {
            "root": merkle["root"],
            "nonce": merkle["rootNonce"],
            "signature": root_signature["signature"],
            "signatureType": root_signature["signatureType"],
            "scopes": merkle["scopes"],
            "signingKey": merkle.get("signingKey"),
            "signingKeyType": merkle.get("signingKeyType"),
            "orgReferenceId": merkle.get("orgReferenceId"),
            "derivedOrgClientId": merkle.get("derivedOrgClientId"),
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
        }

        return {
            "clientId": bundle["clientId"],
            "consentHost": bundle["consentHost"],
            "batchId": batch_id,
            "batchInvite": bundle.get("batchInvite"),
            "scopes": merkle["scopes"],
            "chain": bundle.get("chain"),
            "masterSecret": master_secret,
            "root": {key: value for key, value in root.items() if value is not None},
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
            else f"/v1/client-apps/{(options or {}).get('commitClientAppId') or bundle['clientId']}/invites"
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

    def createOpenOrganizationInvite(self, input: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        input = input or {}
        client_id = self._require_client_id()
        consent_host = self._require_consent_host()
        created_at = int(input["createdAt"] if input.get("createdAt") is not None else int(time.time()))

        return {
            "clientId": client_id,
            "consentHost": consent_host,
            "label": input.get("label"),
            "scopes": _normalize_scopes(input.get("scopes") or _default_scopes_for_chain("")),
            "allowedIdentityTypes": _normalize_open_invite_identity_types(input.get("allowedIdentityTypes") or []),
            "allowedEmailDomains": _normalize_open_invite_email_domains(input.get("allowedEmailDomains") or []),
            "requireKyc": bool(input.get("requireKyc")),
            "requireTaxProfile": bool(input.get("requireTaxProfile")),
            "maxUses": input.get("maxUses"),
            "expiresAt": _to_iso_millis(input.get("expiresAt")),
            "redirectUri": input.get("redirectUri"),
            "state": input.get("state"),
            "stateParams": input.get("stateParams"),
            "createdAt": created_at,
            "inviteNonce": input.get("inviteNonce") or createInviteNonce(),
            "inviteSecret": input.get("inviteSecret") or createInviteSecret(),
        }

    def signOpenOrganizationInvite(self, draft: Dict[str, Any], signer: Dict[str, Any]) -> Dict[str, Any]:
        policy = self._build_open_organization_invite_policy(draft)
        policy_json = _compact_json(policy)
        message = "PVIUM_OPEN_ORGANIZATION_INVITE_V1" + "\n" + policy_json
        signature_info = self._sign_open_invite_message(message, signer)

        result: Dict[str, Any] = {
            "clientId": draft["clientId"],
            "consentHost": draft["consentHost"],
            "label": draft.get("label"),
            "inviteNonce": draft["inviteNonce"],
            "inviteSecret": draft["inviteSecret"],
            "secretHash": _sha256_hex(draft["inviteSecret"]),
            "policyHash": _sha256_hex(policy_json),
            "signature": signature_info["signature"],
            "signatureType": signature_info["signatureType"],
            "signatureMessage": message,
            "signatureTimestamp": draft["createdAt"],
            "signerAddress": signature_info.get("signerAddress"),
            "scopes": policy["scopes"],
            "allowedIdentityTypes": policy["allowedIdentityTypes"],
            "allowedEmailDomains": policy["allowedEmailDomains"],
            "requireKyc": policy["requireKyc"],
            "requireTaxProfile": policy["requireTaxProfile"],
            "redirectUri": draft.get("redirectUri"),
            "state": draft.get("state"),
            "stateParams": draft.get("stateParams"),
            "metadata": {"version": "1", "encoding": "PVIUM_OPEN_ORGANIZATION_INVITE_V1"},
        }
        if policy["maxUses"] > 0:
            result["maxUses"] = policy["maxUses"]
        if policy["expiresAt"]:
            result["expiresAt"] = policy["expiresAt"]
        return result

    def commitOpenOrganizationInvite(self, invite: Dict[str, Any], options: Optional[RequestOptions] = None) -> Dict[str, Any]:
        exclude = {"inviteSecret", "consentHost", "redirectUri", "state", "stateParams"}
        body = {k: v for k, v in invite.items() if k not in exclude}

        response = self.http.request(
            "POST",
            f"/v1/client-apps/{quote(str(invite['clientId']), safe='')}/open-invites",
            body=body,
            options=options,
        )
        raw = self.http.parseResponseBody(response)
        record = self._get_response_value(raw)
        invite_id = ""
        if isinstance(record, dict):
            invite_id = str(record.get("id") or record.get("_id") or "")

        invite_link = None
        if invite_id:
            invite_link = self._generate_open_organization_invite_link(
                {
                    "consentHost": invite["consentHost"],
                    "clientId": invite["clientId"],
                    "inviteId": invite_id,
                    "inviteSecret": invite["inviteSecret"],
                    "scopes": invite["scopes"],
                    "redirectUri": invite.get("redirectUri"),
                    "state": _build_invite_state(invite.get("state"), invite.get("stateParams"), None),
                }
            )

        return {
            "raw": raw,
            "invite": {**record, "inviteLink": invite_link} if isinstance(record, dict) else None,
            "inviteLink": invite_link,
        }

    def createSignedOpenOrganizationInvite(self, input: Dict[str, Any], signer: Dict[str, Any]) -> Dict[str, Any]:
        return self.signOpenOrganizationInvite(self.createOpenOrganizationInvite(input), signer)

    def createSignedAndCommitOpenOrganizationInvite(
        self, input: Dict[str, Any], signer: Dict[str, Any], options: Optional[RequestOptions] = None
    ) -> Dict[str, Any]:
        invite = self.createSignedOpenOrganizationInvite(input, signer)
        return self.commitOpenOrganizationInvite(invite, options)

    def _build_open_organization_invite_policy(self, draft: Dict[str, Any]) -> Dict[str, Any]:
        return {
            "appClientId": draft["clientId"],
            "allowedEmailDomains": _normalize_open_invite_email_domains(draft.get("allowedEmailDomains") or []),
            "allowedIdentityTypes": _normalize_open_invite_identity_types(draft.get("allowedIdentityTypes") or []),
            "createdAt": int(draft.get("createdAt") or 0),
            "expiresAt": _to_iso_millis(draft.get("expiresAt")) or "",
            "inviteNonce": str(draft.get("inviteNonce")),
            "maxUses": 0 if draft.get("maxUses") is None else int(draft["maxUses"]),
            "requireKyc": bool(draft.get("requireKyc")),
            "requireTaxProfile": bool(draft.get("requireTaxProfile")),
            "scopes": _normalize_scopes(draft.get("scopes") or []),
        }

    def _sign_open_invite_message(self, message: str, signer: Dict[str, Any]) -> Dict[str, Any]:
        is_evm = isinstance(signer, dict) and _is_evm_invite_chain(signer.get("chain"))
        if is_evm:
            if signer.get("privateKey"):
                account = Account.from_key(signer["privateKey"])
                signature = account.sign_message(encode_defunct(text=message)).signature.hex()
                return {
                    "signature": _ensure_0x(signature),
                    "signatureType": "evm-personal-sign",
                    "signerAddress": account.address,
                }

            fn = signer.get("signInviteRoot") or signer.get("signMessage")
            if not callable(fn):
                raise RuntimeError("EVM invite signer requires signMessage(message)")
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

        return self._sign_root_message(message, signer)

    def _get_response_value(self, response: Any) -> Any:
        if not isinstance(response, dict):
            return None
        if response.get("data") is not None:
            return response["data"]
        if response.get("value") is not None:
            return response["value"]
        return response

    def _generate_open_organization_invite_link(self, params: Dict[str, Any]) -> str:
        query: Dict[str, Any] = {
            "client_id": params["clientId"],
            "response_type": "code",
            "scope": " ".join(_normalize_scopes(params["scopes"])),
        }
        if params.get("redirectUri"):
            query["redirect_uri"] = params["redirectUri"]
        if params.get("state"):
            query["state"] = params["state"]

        base = params["consentHost"].rstrip("/")
        url = f"{base}/o/{quote(str(params['inviteId']), safe='')}?{urlencode(query)}"
        url += f"#s={quote(str(params['inviteSecret']), safe='')}"
        return url

    def _sign_message_for_master_secret(self, message: str, signer: Dict[str, Any]) -> Dict[str, Any]:
        chain = signer.get("chain")
        if _is_evm_invite_chain(chain) and signer.get("privateKey"):
            account = Account.from_key(signer["privateKey"])
            signature = account.sign_message(encode_defunct(text=message)).signature.hex()
            return {"signatureHex": signature.replace("0x", "").lower(), "signerAddress": account.address}

        if _is_evm_invite_chain(chain):
            fn = signer.get("signMasterSecret") or signer.get("signMessage")
            if not callable(fn):
                raise RuntimeError("EVM signer requires signMessage(message)")
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
        if _is_evm_invite_chain(chain) and signer.get("privateKey"):
            account = Account.from_key(signer["privateKey"])
            return {
                "signature": _ensure_0x(account.sign_message(encode_defunct(text=message)).signature.hex()),
                "signatureType": "evm-personal-sign",
                "signerAddress": account.address,
            }

        if _is_evm_invite_chain(chain):
            fn = signer.get("signInviteRoot") or signer.get("signMessage")
            if not callable(fn):
                raise RuntimeError("EVM signer requires signMessage(message)")
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
