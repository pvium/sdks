from __future__ import annotations

import hashlib
import os
import re
import time
from dataclasses import dataclass
from typing import Any, Dict, List, Optional, Sequence, Tuple

from eth_account.messages import encode_defunct
from eth_account import Account
from eth_utils import keccak


DEFAULT_INVITE_TTL_SECONDS = 7 * 24 * 60 * 60
INVITE_SECRET_DOMAIN_V2 = "PVIUM_INVITE_SECRET_V2"

SUPPORTED_INVITE_IDENTITY_TYPES = [
    "email",
    "handle",
    "wallet",
    "x",
    "github",
    "twitter",
    "discord",
    "telegram",
]

EVM_ADDRESS_RE = re.compile(r"^0x[0-9a-fA-F]{40}$")
EVM_ADDRESS_LOWER_RE = re.compile(r"^0x[0-9a-f]{40}$")
SOLANA_ADDRESS_RE = re.compile(r"^[1-9A-HJ-NP-Za-km-z]{32,44}$")
HANDLE_RE = re.compile(r"^[a-z0-9](?:[a-z0-9._-]{0,30}[a-z0-9])?$")
EMAIL_RE = re.compile(r"^[^\s@]+@[^\s@]+\.[^\s@]+$")


def _sha256(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def _random_hex(size: int) -> str:
    return os.urandom(size).hex()


def _to_unix_seconds(value: Any = None) -> int:
    if value is None:
        return 0
    if isinstance(value, (int, float)):
        return int(value)
    if hasattr(value, "timestamp"):
        return int(value.timestamp())
    if isinstance(value, str):
        from datetime import datetime

        try:
            return int(datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp())
        except Exception:
            return 0
    return 0


def _normalize_scopes(scopes: Sequence[str]) -> List[str]:
    return sorted({s.strip() for s in scopes if s and s.strip()})


def _normalize_amount(value: Any = None) -> str:
    if value is None:
        return ""
    try:
        return str(float(value)).rstrip("0").rstrip(".") if "." in str(value) else str(value)
    except Exception:
        return str(value)


def createRootNonce(batch_id: Optional[str], scopes: Sequence[str], salt: Optional[str] = None) -> str:
    root_salt = salt or _random_hex(16)
    return _sha256(":".join(["payy.invite.root.v1", batch_id or "", " ".join(scopes), root_salt]))


def createInviteNonce() -> str:
    return _random_hex(16)


def createInviteSecret() -> str:
    return _random_hex(32)


def buildSecretHash(invite_secret: str) -> str:
    return _sha256(invite_secret)


def buildInviteMasterSecretMessage(root_nonce: str) -> str:
    return f"{INVITE_SECRET_DOMAIN_V2}:{root_nonce}"


def deriveMasterSecret(raw_signature_hex: str) -> str:
    normalized = raw_signature_hex.lower().replace("0x", "")
    if not normalized:
        raise RuntimeError("Cannot derive master secret from empty signature")
    return _sha256(normalized)


def deriveInviteSecret(master_secret: str, invite_nonce: str) -> str:
    return _sha256(f"{master_secret}:{invite_nonce}")


def detectInviteIdentityType(raw: str) -> Optional[Dict[str, Any]]:
    trimmed = (raw or "").strip()
    if not trimmed:
        return None

    if EMAIL_RE.match(trimmed):
        return {"type": "email", "ambiguous": False}

    if trimmed.startswith("@"):
        rest = trimmed[1:].lower()
        if HANDLE_RE.match(rest):
            return {"type": "handle", "ambiguous": False}

    if EVM_ADDRESS_RE.match(trimmed):
        return {"type": "wallet", "ambiguous": False}

    if SOLANA_ADDRESS_RE.match(trimmed):
        lowered = trimmed.lower()
        also_valid_handle = len(trimmed) <= 32 and HANDLE_RE.match(lowered) is not None
        return {"type": "wallet", "ambiguous": bool(also_valid_handle)}

    if HANDLE_RE.match(trimmed.lower()):
        return {"type": "handle", "ambiguous": False}

    return None


def normalizeIdentityValue(identity_type: str, value: str) -> str:
    trimmed = (value or "").strip()
    if identity_type == "email":
        return trimmed.lower()
    if identity_type == "handle":
        return trimmed.lower().lstrip("@")
    if identity_type == "wallet":
        return trimmed.lower() if EVM_ADDRESS_RE.match(trimmed) else trimmed
    if identity_type in {"x", "twitter", "github", "discord", "telegram"}:
        return trimmed.lower().lstrip("@")
    raise RuntimeError(f"Unsupported identity type: {identity_type}")


def validateIdentityValue(identity_type: str, value: str) -> Optional[str]:
    if not value or not value.strip():
        return "Identity value is required"
    if identity_type not in SUPPORTED_INVITE_IDENTITY_TYPES:
        return f"Identity type not yet supported: {identity_type}"

    normalized = normalizeIdentityValue(identity_type, value)

    if identity_type == "email":
        return None if EMAIL_RE.match(normalized) else "Invalid email format"
    if identity_type == "handle":
        return None if HANDLE_RE.match(normalized) else "Invalid handle format"
    if identity_type == "wallet":
        if EVM_ADDRESS_LOWER_RE.match(normalized) or SOLANA_ADDRESS_RE.match(normalized):
            return None
        return "Invalid wallet address format"
    if identity_type in {"github", "x", "twitter", "discord", "telegram"}:
        return None if HANDLE_RE.match(normalized) else f"Invalid {identity_type} handle format"

    return f"Identity type not yet supported: {identity_type}"


def buildIdentityCommitment(identity_type: str, value: str, invite_nonce: str) -> str:
    return _sha256(":".join(["pvium.invite.identity.v2", identity_type, normalizeIdentityValue(identity_type, value), invite_nonce]))


def _build_leaf_message_v2(params: Dict[str, Any]) -> str:
    return "\n".join(
        [
            "PVIUM_INVITE_LEAF_V2",
            f"appClientId={params['appClientId']}",
            f"batchId={params['batchId']}",
            f"identityType={params['identityType']}",
            f"identityCommitment={params['identityCommitment']}",
            f"inviteNonce={params['inviteNonce']}",
            f"secretHash={params['secretHash']}",
            f"defaultPayoutAmount={_normalize_amount(params.get('defaultPayoutAmount'))}",
            f"expiresAt={params['expiresAt']}",
        ]
    )


def _build_root_message_v2(params: Dict[str, Any]) -> str:
    return "\n".join(
        [
            "PVIUM_INVITE_ROOT_V2",
            "version=2",
            f"appClientId={params['appClientId']}",
            f"batchId={params['batchId']}",
            f"root={params['root']}",
            f"rootNonce={params['rootNonce']}",
            f"scopes={' '.join(params['scopes'])}",
            f"createdAt={params['createdAt']}",
            f"expiresAt={params['expiresAt']}",
        ]
    )


def _build_leaf_message_v1(params: Dict[str, Any]) -> str:
    return "\n".join(
        [
            "PVIUM_INVITE_LEAF_V1",
            f"appClientId={params['appClientId']}",
            f"batchId={params['batchId']}",
            f"emailCommitment={params['emailCommitment']}",
            f"inviteNonce={params['inviteNonce']}",
            f"loginHintHash={params['loginHintHash']}",
            f"defaultPayoutAmount={_normalize_amount(params.get('defaultPayoutAmount'))}",
            f"expiresAt={params['expiresAt']}",
        ]
    )


def _build_root_message_v1(params: Dict[str, Any]) -> str:
    return "\n".join(
        [
            "PVIUM_INVITE_ROOT_V1",
            f"version={params['version']}",
            f"appClientId={params['appClientId']}",
            f"batchId={params['batchId']}",
            f"root={params['root']}",
            f"rootNonce={params['rootNonce']}",
            f"scopes={' '.join(params['scopes'])}",
            f"createdAt={params['createdAt']}",
            f"expiresAt={params['expiresAt']}",
        ]
    )


def _keccak_hex(data: bytes) -> str:
    return "0x" + keccak(data).hex()


def _hex_bytes(value: str) -> bytes:
    return bytes.fromhex(value[2:] if value.startswith("0x") else value)


def _pair_hash(left: bytes, right: bytes) -> bytes:
    a, b = (left, right) if left <= right else (right, left)
    return keccak(a + b)


def _build_merkle(leaves: List[bytes]) -> Tuple[str, List[List[str]]]:
    if not leaves:
        raise RuntimeError("Cannot build Merkle tree without leaves")

    levels: List[List[bytes]] = [leaves]
    while len(levels[-1]) > 1:
        prev = levels[-1]
        nxt: List[bytes] = []
        for i in range(0, len(prev), 2):
            if i + 1 >= len(prev):
                nxt.append(prev[i])
                continue
            left = prev[i]
            right = prev[i + 1]
            nxt.append(_pair_hash(left, right))
        levels.append(nxt)

    root_hex = "0x" + levels[-1][0].hex()

    proofs: List[List[str]] = []
    for leaf_index in range(len(leaves)):
        proof: List[str] = []
        idx = leaf_index
        for level in levels[:-1]:
            sibling_idx = idx ^ 1
            if sibling_idx < len(level):
                proof.append("0x" + level[sibling_idx].hex())
            idx //= 2
        proofs.append(proof)

    return root_hex, proofs


def _verify_merkle(leaf: bytes, proof: Sequence[str], root: str) -> bool:
    current = leaf
    for sibling_hex in proof:
        sibling = _hex_bytes(sibling_hex)
        current = _pair_hash(current, sibling)
    return current.hex() == (root[2:] if root.startswith("0x") else root).lower()


def generateBatchInviteMerkleDataV2(input: Dict[str, Any]) -> Dict[str, Any]:
    invites = input.get("invites") or []
    if not invites:
        raise RuntimeError("Cannot generate invite Merkle data without invites")

    for invite in invites:
        err = validateIdentityValue(invite["identityType"], invite["identityValue"])
        if err:
            raise RuntimeError(f"Invalid invite identity ({invite['identityType']}={invite['identityValue']}): {err}")

    scopes = _normalize_scopes(input.get("scopes") or [])
    batch_id = input.get("batchId") or ""
    created_at = int(input.get("createdAt") or int(time.time()))
    root_nonce = input.get("rootNonce") or createRootNonce(batch_id, scopes)

    invites_without_proofs: List[Dict[str, Any]] = []
    leaf_bytes: List[bytes] = []

    for invite in invites:
        invite_nonce = invite.get("inviteNonce") or createInviteNonce()
        invite_secret = invite.get("inviteSecret") or createInviteSecret()
        secret_hash = buildSecretHash(invite_secret)
        identity_value = normalizeIdentityValue(invite["identityType"], invite["identityValue"])
        identity_commitment = buildIdentityCommitment(invite["identityType"], identity_value, invite_nonce)
        expires_at = _to_unix_seconds(invite.get("expiresAt")) or (created_at + DEFAULT_INVITE_TTL_SECONDS)
        leaf_message = _build_leaf_message_v2(
            {
                "appClientId": input["appClientId"],
                "batchId": batch_id,
                "identityType": invite["identityType"],
                "identityCommitment": identity_commitment,
                "inviteNonce": invite_nonce,
                "secretHash": secret_hash,
                "defaultPayoutAmount": invite.get("defaultPayoutAmount"),
                "expiresAt": expires_at,
            }
        )
        leaf = keccak(leaf_message.encode("utf-8"))
        leaf_hex = "0x" + leaf.hex()

        leaf_bytes.append(leaf)
        invites_without_proofs.append(
            {
                "inviteId": invite.get("inviteId"),
                "identityType": invite["identityType"],
                "identityValue": identity_value,
                "identityValueRaw": invite["identityValue"],
                "identityCommitment": identity_commitment,
                "inviteNonce": invite_nonce,
                "inviteSecret": invite_secret,
                "secretHash": secret_hash,
                "defaultPayoutAmount": invite.get("defaultPayoutAmount"),
                "expiresAt": expires_at,
                "leaf": leaf_hex,
                "leafMessage": leaf_message,
            }
        )

    root, proofs = _build_merkle(leaf_bytes)
    expires_at = max(item["expiresAt"] for item in invites_without_proofs)

    invites_with_proofs = []
    for idx, item in enumerate(invites_without_proofs):
        invite = dict(item)
        invite["proof"] = proofs[idx]
        invites_with_proofs.append(invite)

    signature_message = _build_root_message_v2(
        {
            "appClientId": input["appClientId"],
            "batchId": batch_id,
            "root": root,
            "rootNonce": root_nonce,
            "scopes": scopes,
            "createdAt": created_at,
            "expiresAt": expires_at,
        }
    )

    return {
        "version": "2",
        "appClientId": input["appClientId"],
        "batchId": batch_id,
        "chain": input.get("chain"),
        "scopes": scopes,
        "root": root,
        "rootNonce": root_nonce,
        "inviteCount": len(invites_with_proofs),
        "createdAt": created_at,
        "expiresAt": expires_at,
        "signatureMessage": signature_message,
        "invites": invites_with_proofs,
    }


def verifyBatchInviteProofV2(input: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []
    batch_id = input.get("batchId") or ""

    identity_err = validateIdentityValue(input["identityType"], input["identityValue"])
    if identity_err:
        errors.append(identity_err)

    identity_commitment = buildIdentityCommitment(input["identityType"], input["identityValue"], input["inviteNonce"])
    secret_hash = buildSecretHash(input["inviteSecret"])
    expires_at = _to_unix_seconds(input.get("expiresAt"))
    leaf_message = _build_leaf_message_v2(
        {
            "appClientId": input["appClientId"],
            "batchId": batch_id,
            "identityType": input["identityType"],
            "identityCommitment": identity_commitment,
            "inviteNonce": input["inviteNonce"],
            "secretHash": secret_hash,
            "defaultPayoutAmount": input.get("defaultPayoutAmount"),
            "expiresAt": expires_at,
        }
    )
    leaf = keccak(leaf_message.encode("utf-8"))
    leaf_hex = "0x" + leaf.hex()

    if input.get("identityCommitment") and str(input["identityCommitment"]).lower() != identity_commitment.lower():
        errors.append("Identity commitment does not match signed-in user")
    if input.get("secretHash") and str(input["secretHash"]).lower() != secret_hash.lower():
        errors.append("Secret hash does not match provided invite secret")
    if str(input["leaf"]).lower() != leaf_hex.lower():
        errors.append("Invite leaf does not match invite data")

    proof_valid = _verify_merkle(leaf, input.get("proof") or [], input["root"])
    if not proof_valid:
        errors.append("Invite proof is not in the Merkle root")

    signature_valid = None
    recovered_signer = None
    if input.get("signatureType") == "evm-personal-sign" and input.get("signature") and input.get("signatureMessage"):
        try:
            recovered_signer = Account.recover_message(encode_defunct(text=input["signatureMessage"]), signature=input["signature"])
            signer_address = input.get("signerAddress")
            signature_valid = (not signer_address) or recovered_signer.lower() == str(signer_address).lower()
            if not signature_valid:
                errors.append("Invite root signature signer does not match")
        except Exception:
            signature_valid = False
            errors.append("Invite root signature is invalid")

    if input.get("signatureMessage") and input["root"] not in str(input["signatureMessage"]):
        errors.append("Invite root signature message does not contain root")

    return {
        "valid": len(errors) == 0,
        "leaf": leaf_hex,
        "leafMessage": leaf_message,
        "identityCommitment": identity_commitment,
        "secretHash": secret_hash,
        "proofValid": proof_valid,
        "signatureValid": signature_valid,
        "recoveredSigner": recovered_signer,
        "errors": errors,
    }


# Backward-compatible V1 helpers

def _build_email_commitment(batch_id: Optional[str], email: str, invite_nonce: str) -> str:
    return _sha256(":".join(["payy.invite.email.v1", batch_id or "", email.strip().lower(), invite_nonce]))


def _build_login_hint_hash(email: str, invite_nonce: str) -> str:
    return _sha256(f"{email.strip().lower()}:{invite_nonce}")[:12]


def generateBatchInviteMerkleData(input: Dict[str, Any]) -> Dict[str, Any]:
    invites = input.get("invites") or []
    if not invites:
        raise RuntimeError("Cannot generate invite Merkle data without invites")

    scopes = _normalize_scopes(input.get("scopes") or [])
    batch_id = input.get("batchId") or ""
    created_at = int(input.get("createdAt") or int(time.time()))
    root_nonce = input.get("rootNonce") or createRootNonce(batch_id, scopes)

    without_proofs: List[Dict[str, Any]] = []
    leaves: List[bytes] = []

    for invite in invites:
        invite_nonce = invite.get("inviteNonce") or createInviteNonce()
        login_hint_hash = _build_login_hint_hash(invite["email"], invite_nonce)
        email_commitment = _build_email_commitment(input.get("batchId"), invite["email"], invite_nonce)
        expires_at = _to_unix_seconds(invite.get("expiresAt")) or (created_at + DEFAULT_INVITE_TTL_SECONDS)

        leaf_message = _build_leaf_message_v1(
            {
                "appClientId": input["appClientId"],
                "batchId": batch_id,
                "emailCommitment": email_commitment,
                "inviteNonce": invite_nonce,
                "loginHintHash": login_hint_hash,
                "defaultPayoutAmount": invite.get("defaultPayoutAmount"),
                "expiresAt": expires_at,
            }
        )
        leaf = keccak(leaf_message.encode("utf-8"))
        leaf_hex = "0x" + leaf.hex()

        leaves.append(leaf)
        without_proofs.append(
            {
                "inviteId": invite.get("inviteId"),
                "email": invite["email"],
                "inviteNonce": invite_nonce,
                "loginHintHash": login_hint_hash,
                "emailCommitment": email_commitment,
                "defaultPayoutAmount": invite.get("defaultPayoutAmount"),
                "expiresAt": expires_at,
                "leaf": leaf_hex,
                "leafMessage": leaf_message,
            }
        )

    root, proofs = _build_merkle(leaves)
    expires_at = max(item["expiresAt"] for item in without_proofs)

    invites_out = []
    for idx, item in enumerate(without_proofs):
        x = dict(item)
        x["proof"] = proofs[idx]
        invites_out.append(x)

    signature_message = _build_root_message_v1(
        {
            "version": "1",
            "appClientId": input["appClientId"],
            "batchId": batch_id,
            "root": root,
            "rootNonce": root_nonce,
            "scopes": scopes,
            "createdAt": created_at,
            "expiresAt": expires_at,
        }
    )

    return {
        "version": "1",
        "appClientId": input["appClientId"],
        "batchId": batch_id,
        "chain": input.get("chain"),
        "scopes": scopes,
        "root": root,
        "rootNonce": root_nonce,
        "inviteCount": len(invites_out),
        "createdAt": created_at,
        "expiresAt": expires_at,
        "signatureMessage": signature_message,
        "invites": invites_out,
    }


def verifyBatchInviteProof(input: Dict[str, Any]) -> Dict[str, Any]:
    errors: List[str] = []
    batch_id = input.get("batchId") or ""
    login_hint_hash = _build_login_hint_hash(input["email"], input["inviteNonce"])
    email_commitment = _build_email_commitment(batch_id, input["email"], input["inviteNonce"])
    expires_at = _to_unix_seconds(input.get("expiresAt"))

    leaf_message = _build_leaf_message_v1(
        {
            "appClientId": input["appClientId"],
            "batchId": batch_id,
            "emailCommitment": email_commitment,
            "inviteNonce": input["inviteNonce"],
            "loginHintHash": login_hint_hash,
            "defaultPayoutAmount": input.get("defaultPayoutAmount"),
            "expiresAt": expires_at,
        }
    )
    leaf = keccak(leaf_message.encode("utf-8"))
    leaf_hex = "0x" + leaf.hex()

    if input.get("loginHintHash") != login_hint_hash:
        errors.append("Login hint hash does not match signed-in user")
    if input.get("emailCommitment") and str(input["emailCommitment"]).lower() != email_commitment.lower():
        errors.append("Email commitment does not match signed-in user")
    if str(input["leaf"]).lower() != leaf_hex.lower():
        errors.append("Invite leaf does not match invite data")

    proof_valid = _verify_merkle(leaf, input.get("proof") or [], input["root"])
    if not proof_valid:
        errors.append("Invite proof is not in the Merkle root")

    signature_valid = None
    recovered_signer = None
    if input.get("signatureType") == "evm-personal-sign" and input.get("signature") and input.get("signatureMessage"):
        try:
            recovered_signer = Account.recover_message(encode_defunct(text=input["signatureMessage"]), signature=input["signature"])
            signer_address = input.get("signerAddress")
            signature_valid = (not signer_address) or recovered_signer.lower() == str(signer_address).lower()
            if not signature_valid:
                errors.append("Invite root signature signer does not match")
        except Exception:
            signature_valid = False
            errors.append("Invite root signature is invalid")

    if input.get("signatureMessage") and input["root"] not in str(input["signatureMessage"]):
        errors.append("Invite root signature message does not contain root")

    return {
        "valid": len(errors) == 0,
        "leaf": leaf_hex,
        "leafMessage": leaf_message,
        "loginHintHash": login_hint_hash,
        "emailCommitment": email_commitment,
        "proofValid": proof_valid,
        "signatureValid": signature_valid,
        "recoveredSigner": recovered_signer,
        "errors": errors,
    }
