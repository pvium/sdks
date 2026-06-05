from __future__ import annotations

import hashlib
import os
import time
import uuid
from dataclasses import dataclass
from decimal import Decimal
from typing import Any, Callable, Dict, List, Optional, Sequence, Tuple, Union
from urllib.parse import quote

from eth_abi import encode
from eth_abi.packed import encode_packed
from eth_account import Account
from eth_account.messages import encode_defunct
from eth_keys import keys
from eth_utils import keccak, to_checksum_address

from ...core.client import PviumHttpClient, PviumSdkConfig, resolvePviumConsentHost
from ...core.types import RequestOptions


HexString = str
PayoutCurrency = type("PayoutCurrency", (), {"USDC": "USDC", "USDT": "USDT"})
PayoutSignerInput = Union[
    str,
    Dict[str, Any],
    Callable[[str], Union[str, Dict[str, Any]]],
]


STABLECOIN_TOKEN_ADDRESSES = {
    "base": {
        "USDC": {"contractAddress": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", "decimals": 6},
        "USDT": {"contractAddress": "0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2", "decimals": 6},
    },
    "bsc": {
        "USDT": {"contractAddress": "0x55d398326f99059fF775485246999027B3197955", "decimals": 18},
        "USDC": {"contractAddress": "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", "decimals": 18},
    },
    "solana": {
        "USDC": {"contractAddress": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "decimals": 6},
        "USDT": {"contractAddress": "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "decimals": 6},
    },
    "base-testnet": {
        "USDC": {"contractAddress": "0x7dCEd3bFcC97948a665BB665a5D7eEfdfce39C3A", "decimals": 18},
        "USDT": {"contractAddress": "0x9d0C28036AC12d2150a23DE40Bc4A92f7Aa1A79E", "decimals": 18},
    },
    "solana-testnet": {
        "USDC": {"contractAddress": "CmBGSxKZtv22ZiVpKGMP1oMPZfc5rsgr3pEGBDRcjiAy", "decimals": 6},
        "USDT": {"contractAddress": "SPFPKg9zeE7ReqW3j9QU6p7XhPP8JDU5Dx4fgrTwVyF", "decimals": 6},
    },
    "localhost": {
        "USDT": {"contractAddress": "0x5FbDB2315678afecb367f032d93F642f64180aa3", "decimals": 18},
    },
}

CHAIN_ALIASES = {
    "8453": "base",
    "base": "base",
    "basemainnet": "base",
    "base-mainnet": "base",
    "56": "bsc",
    "bsc": "bsc",
    "binance": "bsc",
    "binancesmartchain": "bsc",
    "binance-smart-chain": "bsc",
    "101": "solana",
    "solana": "solana",
    "84532": "base-testnet",
    "basetestnet": "base-testnet",
    "base-testnet": "base-testnet",
    "basesepolia": "base-testnet",
    "base-sepolia": "base-testnet",
    "1012": "solana-testnet",
    "solanatestnet": "solana-testnet",
    "solana-testnet": "solana-testnet",
    "31337": "localhost",
    "localhost": "localhost",
}

PAYOUT_CHAIN_IDS = {
    "base": 8453,
    "bsc": 56,
    "solana": 101,
    "base-testnet": 84532,
    "solana-testnet": 1012,
    "localhost": 31337,
}


class PayoutIntent(dict):
    def __init__(self, service: "PviumPayoutService", meta: Dict[str, Any], data: Dict[str, Any]):
        super().__init__({"meta": meta, "data": data})
        object.__setattr__(self, "_service", service)

    def __getattr__(self, name: str) -> Any:
        data = self.get("data") or {}
        if name in data:
            return data[name]
        try:
            return self[name]
        except KeyError as err:
            raise AttributeError(name) from err

    def finalize(
        self,
        signer: PayoutSignerInput,
        options: Optional[Dict[str, Any]] = None,
        request_options: Optional[RequestOptions] = None,
    ) -> "PayoutFinalization":
        return self._service.finalize(self["data"], signer, options, request_options)

    def addPayments(
        self,
        input: Union[Dict[str, Any], List[Dict[str, Any]]],
        options: Optional[RequestOptions] = None,
    ):
        return self._service.addPayments(self["data"], input, options)

    def addRecipients(self, input: Union[Dict[str, Any], List[Dict[str, Any]]], options: Optional[RequestOptions] = None):
        return self._service.addRecipients(self["data"]["id"], input, options)

    def resolveRecipients(self, input: Union[Dict[str, Any], List[Dict[str, Any]]], options: Optional[RequestOptions] = None):
        return self._service.resolveRecipients(self["data"]["id"], input, options)

    def removePayments(self, input: Union[Dict[str, Any], List[Union[int, str]]], options: Optional[RequestOptions] = None):
        return self._service.removePayments(self["data"]["id"], input, options)

    def deletePayment(self, payment_id: Union[int, str], options: Optional[RequestOptions] = None):
        return self._service.deletePayment(self["data"]["id"], payment_id, options)

    def updatePayment(self, payment_id: Union[int, str], input: Dict[str, Any], options: Optional[RequestOptions] = None):
        return self._service.updatePayment(self["data"]["id"], payment_id, input, options)

    def editPayment(self, payment_id: Union[int, str], input: Dict[str, Any], options: Optional[RequestOptions] = None):
        return self.updatePayment(payment_id, input, options)

    def listPayments(self, query: Optional[Dict[str, Any]] = None, options: Optional[RequestOptions] = None):
        return self._service.listPayments(self["data"]["id"], query, options)

    def listInvites(self, options: Optional[RequestOptions] = None):
        return self._service.listInvites(self["data"]["id"], options)

    def revokeInvite(self, invite_id: Union[int, str], options: Optional[RequestOptions] = None):
        return self._service.revokeInvite(self["data"]["id"], invite_id, options)

    def revokeInviteRoot(self, invite_root_id: Union[int, str], options: Optional[RequestOptions] = None):
        return self._service.revokeInviteRoot(self["data"]["id"], invite_root_id, options)

    def delete(self, options: Optional[RequestOptions] = None):
        return self._service.delete(self["data"]["id"], options)


class PayoutFinalization(dict):
    def __init__(self, service: "PviumPayoutService", meta: Dict[str, Any], data: Dict[str, Any]):
        super().__init__({"meta": meta, "data": data})
        payout = data.get("payout") or {}
        object.__setattr__(self, "payout", PayoutIntent(service, meta, payout))

    def __getattr__(self, name: str) -> Any:
        data = self.get("data") or {}
        if name in data:
            return data[name]
        try:
            return self[name]
        except KeyError as err:
            raise AttributeError(name) from err


def _parse_units(amount: Union[str, int, float, Decimal], decimals: int) -> int:
    value = Decimal(str(amount))
    scale = Decimal(10) ** int(decimals)
    return int(value * scale)


def _hex_bytes(value: str) -> bytes:
    raw = value[2:] if value.startswith("0x") else value
    return bytes.fromhex(raw)


def _to_hex(value: bytes) -> str:
    return "0x" + value.hex()


def _ensure_0x(value: str) -> str:
    return value if value.startswith("0x") else f"0x{value}"


def _normalize_hex_address(value: str) -> str:
    return value if value.lower().startswith("0x") else f"0x{value}"


def _normalize_instant_nonce(nonce: str) -> str:
    trimmed = nonce.strip()
    if not trimmed:
        raise RuntimeError("Payout nonce is required")

    body = trimmed[2:] if trimmed.startswith("0x") else trimmed
    try:
        int(body, 16)
    except ValueError as err:
        raise RuntimeError(f"Payout nonce must be hex-compatible: {nonce}") from err

    if len(body) % 2 == 1:
        body = "0" + body
    return "0x" + body


def _normalize_token_address(value: Any) -> Optional[str]:
    if isinstance(value, str):
        if value.startswith("0x") and len(value) == 42:
            return to_checksum_address(value)
        return None

    if isinstance(value, dict):
        for key in ["contractAddress", "address", "token", "payoutToken", "fundingToken", "current"]:
            nested = _normalize_token_address(value.get(key))
            if nested:
                return nested

    return None


def _chain_key(chain: str) -> Optional[str]:
    if not chain:
        return None
    lowered = chain.lower()
    return CHAIN_ALIASES.get(lowered.replace(" ", "")) or CHAIN_ALIASES.get(lowered)


def _normalize_payout_currency(value: Optional[str]) -> Optional[str]:
    if not value:
        return None
    normalized = str(value).lower()
    if normalized == "usdc":
        return "USDC"
    if normalized == "usdt":
        return "USDT"
    return None


def _format_configured_token(currency: Dict[str, Any]) -> str:
    address = currency["contractAddress"]
    return to_checksum_address(address) if address.startswith("0x") else address


def _resolve_payout_currency_config(chain: str, currency: Optional[str]) -> Optional[Dict[str, Any]]:
    normalized_currency = _normalize_payout_currency(currency)
    if not normalized_currency:
        return None

    chain_key = _chain_key(chain)
    config = (STABLECOIN_TOKEN_ADDRESSES.get(chain_key or "") or {}).get(normalized_currency)
    if not config:
        raise RuntimeError(f"payoutCurrency {normalized_currency} is not supported on chain {chain}")
    return config


def _resolve_payout_currency_by_token(chain: str, token: Optional[str]) -> Optional[Dict[str, Any]]:
    if not token:
        return None
    chain_key = _chain_key(chain)
    if not chain_key:
        return None

    normalized_token = _normalize_token_value(token)
    for currency in (STABLECOIN_TOKEN_ADDRESSES.get(chain_key) or {}).values():
        normalized_address = _normalize_token_value(currency["contractAddress"])
        if normalized_address == normalized_token:
            return currency
    return None


def _normalize_token_value(token: Optional[str]) -> Optional[str]:
    normalized = _normalize_token_address(token)
    if normalized:
        return normalized
    if isinstance(token, str) and len(token) > 10:
        return token
    return None


def _resolve_payment_token(
    chain: str,
    payment: Dict[str, Any],
    token_fallback: Optional[str] = None,
) -> Dict[str, Any]:
    symbol = payment.get("tokenSymbol") or _normalize_payout_currency(payment.get("token"))
    if symbol:
        currency = _resolve_payout_currency_config(chain, symbol)
        if not currency:
            return {}
        return {"token": _format_configured_token(currency), "currency": currency}

    if payment.get("token"):
        token = _normalize_token_value(payment.get("token"))
        currency = _resolve_payout_currency_by_token(chain, token)
        if not token or not currency:
            fallback_token = _normalize_token_value(token_fallback)
            if token and fallback_token == token:
                return {"token": token}
            raise RuntimeError(f"Payment token {payment.get('token')} is not supported on chain {chain}")
        return {"token": _format_configured_token(currency), "currency": currency}

    token = _normalize_token_value(token_fallback)
    currency = _resolve_payout_currency_by_token(chain, token)
    return {"token": _format_configured_token(currency) if currency else token, "currency": currency}


def _build_payout_metadata(input: Dict[str, Any]) -> Dict[str, Any]:
    metadata = dict(input.get("metadata") or {})
    payout_currency_config = _resolve_payout_currency_config(input.get("chain") or "", input.get("payoutCurrency"))
    if payout_currency_config:
        metadata["payoutCurrency"] = _format_configured_token(payout_currency_config)
        metadata["payoutCurrencyDecimals"] = payout_currency_config["decimals"]
    if input.get("scheduleDate") is not None:
        metadata["scheduledDate"] = input.get("scheduleDate")
    if input.get("lockDuration") and metadata.get("lockDuration") is None:
        metadata["lockDuration"] = input["lockDuration"]
    if input.get("type") == "Milestone":
        metadata["commitmentType"] = "milestone"
    return metadata


def _resolve_payout_chain_id(chain: str, chain_id: Optional[int] = None, context: str = "payout finalization") -> int:
    if chain_id:
        return int(chain_id)
    chain_key = _chain_key(chain)
    resolved = PAYOUT_CHAIN_IDS.get(chain_key or "")
    if not resolved:
        raise RuntimeError(f"chainId is required for {context}")
    return resolved


def _map_payout_type(payout_type: Optional[str]) -> Dict[str, Any]:
    if payout_type == "Milestone":
        return {"paymentType": "Scheduled", "isCommitment": True}
    if payout_type == "Escrow":
        return {"paymentType": "Escrow", "isCommitment": False}
    if payout_type == "Scheduled":
        return {"paymentType": "Scheduled", "isCommitment": False}
    return {"paymentType": "Instant", "isCommitment": False}


def createPayoutNonce() -> HexString:
    return "0x" + os.urandom(16).hex()


def _create_payout_id() -> str:
    return str(uuid.uuid4())


def _get_payout_id_bytes32(payout_id: str) -> bytes:
    payout_id_hex = payout_id.replace("-", "")
    return bytes.fromhex(payout_id_hex.ljust(64, "0"))


def generateInstantPayoutHash(payments: Sequence[Dict[str, Any]], nonce: str) -> HexString:
    payouts = []
    for payment in payments:
        if payment.get("decimals") is None:
            raise RuntimeError("Payment decimals are required to hash instant payouts")
        if not payment.get("token"):
            raise RuntimeError("Payment token is required to hash instant payouts")
        payouts.append(
            (
                _normalize_hex_address(payment["receiver"]),
                _parse_units(payment["amount"], int(payment["decimals"])),
                _normalize_hex_address(payment["token"]),
                payment.get("memo") or "",
            )
        )

    encoded = encode(
        ["bytes", "(address,uint256,address,string)[]"],
        [_hex_bytes(_normalize_instant_nonce(nonce)), payouts],
    )
    return _to_hex(keccak(encoded))


def computeScheduledPayoutHash(params: Dict[str, Any]) -> HexString:
    encoded = encode(
        ["bytes32", "address", "uint256", "uint256", "uint256", "uint256"],
        [
            _get_payout_id_bytes32(params["payoutId"]),
            to_checksum_address(params["fundingToken"]),
            int(params["gracePeriod"]),
            int(params["disapprovalDeadline"]),
            int(params["timestamp"]),
            int(params["chainId"]),
        ],
    )
    return _to_hex(keccak(encoded))


def computeEscrowPayoutHash(params: Dict[str, Any]) -> HexString:
    encoded = encode(
        ["bytes32", "address", "uint256", "uint256", "uint256"],
        [
            _get_payout_id_bytes32(params["payoutId"]),
            to_checksum_address(params["fundingToken"]),
            int(params["lockDuration"]),
            int(params["timestamp"]),
            int(params["chainId"]),
        ],
    )
    return _to_hex(keccak(encoded))


def computeEscrowFundingDigest(params: Dict[str, Any]) -> HexString:
    packed = encode_packed(
        ["bytes32", "address"],
        [_hex_bytes(params["escrowBatchHash"]), to_checksum_address(params["withdrawalWallet"])],
    )
    return _to_hex(keccak(packed))


def computeEscrowScheduledFundingDigest(params: Dict[str, Any]) -> HexString:
    packed = encode_packed(
        ["bytes32", "bytes32"],
        [_hex_bytes(params["escrowBatchHash"]), _hex_bytes(params["merkleRoot"])],
    )
    return _to_hex(keccak(packed))


def _hash_merkle_node(value: bytes) -> bytes:
    return keccak(value)


def _pair_hash(left: bytes, right: bytes) -> bytes:
    a, b = (left, right) if left <= right else (right, left)
    return keccak(a + b)


def _generate_leaf_hash(batch_hash: str, entry: Dict[str, Any]) -> bytes:
    encoded = encode_packed(
        ["bytes32", "address", "uint256", "uint256", "string"],
        [
            _hex_bytes(batch_hash),
            to_checksum_address(entry["receiverAddress"]),
            int(entry["amount"]),
            int(entry["claimableDate"]),
            entry["memo"],
        ],
    )
    return keccak(encoded)


def _generate_merkle_tree_for_payout(batch_hash: str, payments: Sequence[Dict[str, Any]], default_claim_date: Optional[int] = None) -> Dict[str, Any]:
    if not payments:
        raise RuntimeError("Cannot finalize scheduled payouts without payments")

    entries = []
    for payment in payments:
        if payment.get("decimals") is None:
            raise RuntimeError("Payment decimals are required to hash scheduled payouts")
        entries.append(
            {
                "receiverAddress": payment["receiver"].lower(),
                "amount": str(_parse_units(payment["amount"], int(payment["decimals"]))),
                "claimableDate": int(payment.get("claimDate") or default_claim_date or 0),
                "memo": payment.get("memo") or "",
            }
        )

    leaves = [_generate_leaf_hash(batch_hash, entry) for entry in entries]
    levels: List[List[bytes]] = [leaves]
    while len(levels[-1]) > 1:
        prev = levels[-1]
        nxt = []
        for i in range(0, len(prev), 2):
            if i + 1 >= len(prev):
                nxt.append(prev[i])
                continue
            left = prev[i]
            right = prev[i + 1]
            nxt.append(_pair_hash(left, right))
        levels.append(nxt)

    merkle_root = _to_hex(levels[-1][0])

    proofs = []
    for idx, entry in enumerate(entries):
        proof = []
        index = idx
        for level in levels[:-1]:
            sibling = index ^ 1
            if sibling < len(level):
                proof.append(_to_hex(level[sibling]))
            index //= 2
        proofs.append({"receiver": entry["receiverAddress"], "proof": proof, "leaf": _to_hex(leaves[idx])})

    return {"merkleRoot": merkle_root, "proofs": proofs}


def _normalize_payments_for_create(
    payments: Optional[Sequence[Dict[str, Any]]],
    chain: str = "",
    token_fallback: Optional[str] = None,
    decimals_fallback: Optional[int] = None,
    expected_token: Optional[str] = None,
) -> Optional[List[Dict[str, Any]]]:
    if payments is None:
        return None

    normalized = []
    normalized_expected_token = _normalize_token_value(expected_token)
    for payment in payments:
        resolved = _resolve_payment_token(chain, payment, token_fallback)
        normalized_payment_token = _normalize_token_value(resolved.get("token"))
        if normalized_expected_token and normalized_payment_token and normalized_payment_token != normalized_expected_token:
            raise RuntimeError("Payment token must match payoutCurrency when payoutCurrency is provided")
        amount = float(payment["amount"]) if isinstance(payment.get("amount"), str) else payment.get("amount")
        item = dict(payment)
        item.pop("tokenSymbol", None)
        item["token"] = resolved.get("token")
        item["decimals"] = payment.get("decimals") or (resolved.get("currency") or {}).get("decimals") or decimals_fallback
        item["amount"] = amount
        normalized.append(item)
    return normalized


def _normalize_payments_for_signing(
    payments: Sequence[Dict[str, Any]],
    chain: str,
    token_fallback: Optional[str] = None,
) -> List[Dict[str, Any]]:
    token_decimals = (_resolve_payout_currency_by_token(chain, token_fallback) or {}).get("decimals")
    normalized = []
    for payment in payments:
        resolved = _resolve_payment_token(chain, payment, token_fallback)
        item = dict(payment)
        item.pop("tokenSymbol", None)
        item["token"] = resolved.get("token")
        item["decimals"] = payment.get("decimals") or (resolved.get("currency") or {}).get("decimals") or token_decimals
        normalized.append(item)
    return normalized


def _resolve_payout_funding_token_candidate(payout: Optional[Dict[str, Any]]) -> Optional[str]:
    if not payout:
        return None

    return (
        _normalize_token_address(payout.get("payoutToken"))
        or _normalize_token_address(payout.get("token"))
        or _normalize_token_address(payout.get("fundingToken"))
        or _normalize_token_address((payout.get("metadata") or {}).get("payoutToken"))
        or _normalize_token_address((payout.get("metadata") or {}).get("payoutCurrency"))
        or _normalize_token_address((payout.get("metadata") or {}).get("fundingToken"))
        or _normalize_token_address((payout.get("payments") or [{}])[0].get("token") if payout.get("payments") else None)
    )


def _resolve_funding_token(payout: Dict[str, Any], options: Dict[str, Any]) -> str:
    linked_escrow = options.get("escrowBatch") if isinstance(options.get("escrowBatch"), dict) else None
    token = (
        _normalize_token_address(options.get("fundingToken"))
        or _resolve_payout_funding_token_candidate(payout)
        or _resolve_payout_funding_token_candidate(linked_escrow)
    )
    if not token:
        raise RuntimeError("fundingToken must be provided as an address to finalize scheduled payouts")
    return token


def _resolve_signer_address(signer: PayoutSignerInput, fallback: Optional[str] = None) -> Optional[str]:
    if isinstance(signer, str):
        return Account.from_key(signer).address.lower()
    if callable(signer):
        return fallback.lower() if fallback else None
    if signer.get("privateKey"):
        return Account.from_key(signer["privateKey"]).address.lower()
    address = fallback or signer.get("signerAddress") or signer.get("address")
    return address.lower() if isinstance(address, str) else None


def _normalize_signature_result(result: Any) -> Dict[str, Any]:
    if isinstance(result, str):
        return {"signature": result}
    if isinstance(result, bytes):
        import base64

        return {"signature": base64.b64encode(result).decode("utf-8")}
    if isinstance(result, dict):
        signature = result.get("signature")
        if isinstance(signature, bytes):
            import base64

            signature = base64.b64encode(signature).decode("utf-8")
        return {"signature": signature, "signerAddress": result.get("signerAddress")}
    return {"signature": str(result)}


def _to_evm_signature_hex(signature: Any) -> str:
    raw = bytearray(signature.to_bytes())
    if len(raw) == 65 and raw[64] in (0, 1):
        raw[64] += 27
    return _to_hex(bytes(raw))


def _sign_payout_finalize_message(signer: PayoutSignerInput, message: str, chain: Optional[str] = None) -> Dict[str, Any]:
    if isinstance(signer, str):
        account = Account.from_key(signer)
        sig = account.sign_message(encode_defunct(text=message)).signature.hex()
        return {"signature": _ensure_0x(sig), "signerAddress": account.address.lower()}

    if callable(signer):
        return _normalize_signature_result(signer(message))

    if signer.get("privateKey"):
        account = Account.from_key(signer["privateKey"])
        sig = account.sign_message(encode_defunct(text=message)).signature.hex()
        return {"signature": _ensure_0x(sig), "signerAddress": account.address.lower()}

    sign_fn = signer.get("signFinalize") or signer.get("signMessage")
    if not callable(sign_fn):
        raise RuntimeError("Signer must provide signMessage(message) or signFinalize(message)")

    is_solana = "solana" in ((chain or signer.get("chain") or "").lower())
    payload = message.encode("utf-8") if is_solana else message
    return _normalize_signature_result(sign_fn(payload))


def _require_signer_address(signer_address: Optional[str], context: str) -> str:
    if not signer_address:
        raise RuntimeError(
            f"{context} requires signerAddress, or signMessage/signFinalize must return {{ signature, signerAddress }}"
        )
    return signer_address.lower()


def _sign_funding_digest(signer: PayoutSignerInput, digest: HexString) -> str:
    if isinstance(signer, str):
        private_key = keys.PrivateKey(_hex_bytes(signer))
        signature = private_key.sign_msg_hash(_hex_bytes(digest))
        return _to_evm_signature_hex(signature)

    if callable(signer):
        raise RuntimeError(
            "EVM payout finalization requires signFunding(digest), signDigest(digest), provider.request({ method: 'secp256k1_sign' }), or a private key for the funding signature"
        )

    if signer.get("privateKey"):
        private_key = keys.PrivateKey(_hex_bytes(signer["privateKey"]))
        signature = private_key.sign_msg_hash(_hex_bytes(digest))
        return _to_evm_signature_hex(signature)

    sign_fn = signer.get("signFunding") or signer.get("signDigest")
    if callable(sign_fn):
        result = sign_fn(digest)
        if isinstance(result, dict):
            result = result.get("signature")
        return str(result)

    request_fn = signer.get("request")
    if callable(request_fn):
        return str(request_fn({"method": "secp256k1_sign", "params": [digest]}))

    raise RuntimeError(
        "EVM payout finalization requires signFunding(digest), signDigest(digest), provider.request({ method: 'secp256k1_sign' }), or a private key for the funding signature"
    )


class PviumPayoutService:
    def __init__(self, http: PviumHttpClient, config: PviumSdkConfig):
        self.http = http
        self.consentHost = resolvePviumConsentHost(config)
        self.clientId = config.clientId

    def create(self, input: Dict[str, Any], options: Optional[RequestOptions] = None) -> Dict[str, Any]:
        mapped = _map_payout_type(input.get("type"))
        escrow_batch = input.get("escrowBatch")
        escrow_batch_id = escrow_batch.get("id") if isinstance(escrow_batch, dict) else escrow_batch
        metadata = _build_payout_metadata(input)
        token_fallback = (
            _normalize_token_address(metadata.get("payoutToken"))
            or _normalize_token_address(metadata.get("payoutCurrency"))
            or _normalize_token_address(metadata.get("fundingToken"))
            or (metadata.get("payoutCurrency") if isinstance(metadata.get("payoutCurrency"), str) else None)
            or (metadata.get("fundingToken") if isinstance(metadata.get("fundingToken"), str) else None)
        )
        decimals_fallback = (
            metadata.get("payoutCurrencyDecimals")
            if isinstance(metadata.get("payoutCurrencyDecimals"), int)
            else (_resolve_payout_currency_by_token(input.get("chain") or "", token_fallback) or {}).get("decimals")
        )

        response = self.http.request(
            "POST",
            "/v1/batch-payments",
            body={
                "id": input.get("id"),
                "chain": input.get("chain"),
                "nonce": input.get("nonce") or createPayoutNonce(),
                "paymentType": mapped["paymentType"],
                "isCommitment": mapped["isCommitment"],
                "escrowBatch": escrow_batch_id,
                "payments": _normalize_payments_for_create(
                    input.get("payments"),
                    input.get("chain") or "",
                    token_fallback,
                    decimals_fallback,
                    token_fallback if input.get("payoutCurrency") else None,
                ),
                "lockDuration": input.get("lockDuration"),
                "label": input.get("label") or input.get("name"),
                "name": input.get("name"),
                "description": input.get("description"),
                "complianceMode": input.get("complianceMode") or "Open",
                "metadata": metadata,
            },
            options=options,
        )
        return self._wrap_payout_response(self._parse(response))

    def createFinalized(
        self,
        input: Dict[str, Any],
        signer: PayoutSignerInput,
        options: Optional[Dict[str, Any]] = None,
        request_options: Optional[RequestOptions] = None,
    ) -> Dict[str, Any]:
        options = options or {}
        if not input.get("id"):
            raise RuntimeError("id is required to create finalized payouts")

        mapped = _map_payout_type(input.get("type"))
        if mapped["paymentType"] != "Scheduled" and not mapped["isCommitment"]:
            raise RuntimeError("createFinalized currently supports scheduled payouts")

        escrow_batch = input.get("escrowBatch")
        escrow_batch_id = escrow_batch.get("id") if isinstance(escrow_batch, dict) else escrow_batch
        metadata = _build_payout_metadata(input)
        token_fallback = (
            _normalize_token_address(metadata.get("payoutToken"))
            or _normalize_token_address(metadata.get("payoutCurrency"))
            or _normalize_token_address(metadata.get("fundingToken"))
            or (metadata.get("payoutCurrency") if isinstance(metadata.get("payoutCurrency"), str) else None)
            or (metadata.get("fundingToken") if isinstance(metadata.get("fundingToken"), str) else None)
        )
        decimals_fallback = (
            metadata.get("payoutCurrencyDecimals")
            if isinstance(metadata.get("payoutCurrencyDecimals"), int)
            else (_resolve_payout_currency_by_token(input.get("chain") or "", token_fallback) or {}).get("decimals")
        )
        payments = _normalize_payments_for_create(
            input.get("payments"),
            input.get("chain") or "",
            token_fallback,
            decimals_fallback,
            token_fallback if input.get("payoutCurrency") else None,
        )

        payout = {
            "id": input["id"],
            "chain": input.get("chain"),
            "nonce": input.get("nonce") or createPayoutNonce(),
            "paymentType": mapped["paymentType"],
            "isCommitment": mapped["isCommitment"],
            "complianceMode": input.get("complianceMode") or "Open",
            "escrowBatch": escrow_batch_id,
            "metadata": metadata,
            "payments": payments,
        }

        finalized = self._build_scheduled_finalize_payload(payout, signer, options, request_options)

        response = self.http.request(
            "POST",
            "/v1/batch-payments",
            body={
                "id": input.get("id"),
                "chain": input.get("chain"),
                "nonce": payout["nonce"],
                "paymentType": mapped["paymentType"],
                "isCommitment": mapped["isCommitment"],
                "escrowBatch": escrow_batch_id,
                "payments": _normalize_payments_for_create(
                    payments,
                    input.get("chain") or "",
                    finalized["fundingToken"],
                    (_resolve_payout_currency_by_token(input.get("chain") or "", finalized["fundingToken"]) or {}).get("decimals"),
                    finalized["fundingToken"] if input.get("payoutCurrency") else None,
                ),
                "label": input.get("label") or input.get("name"),
                "name": input.get("name"),
                "description": input.get("description"),
                "complianceMode": input.get("complianceMode") or "Open",
                "metadata": metadata,
                **finalized["updatePayload"],
            },
            options=request_options,
        )

        created = self._parse(response)
        created_data = created.get("data") or {}
        identifier = created_data.get("merkleRoot") or finalized["merkleRoot"]
        return self._wrap_finalization_response({
            "meta": created.get("meta", {}),
            "data": {
                "payout": created_data,
                "fundingUrl": f"{self.consentHost}/batch/{identifier}" if identifier else None,
                "batchDataHash": finalized["batchDataHash"],
                "batchHash": finalized["batchHash"],
                "merkleRoot": finalized["merkleRoot"],
            },
        })

    def list(self, query: Optional[Dict[str, Any]] = None, options: Optional[RequestOptions] = None) -> Dict[str, Any]:
        response = self.http.request("GET", "/v1/batch-payments", query=query, options=options)
        return self._parse(response)

    def get(self, payout_id: str, options: Optional[RequestOptions] = None) -> Dict[str, Any]:
        response = self.http.request("GET", f"/v1/batch-payments/{quote(payout_id, safe='')}", options=options)
        return self._wrap_payout_response(self._parse(response))

    def addPayments(self, payout: Union[str, Dict[str, Any]], input: Union[Dict[str, Any], List[Dict[str, Any]]], options: Optional[RequestOptions] = None):
        payments = input if isinstance(input, list) else input.get("payments")
        request_options = input.get("requestOptions") if isinstance(input, dict) and input.get("requestOptions") else options

        payout_data = payout.get("data") if isinstance(payout, PayoutIntent) else payout

        if isinstance(payout_data, dict) and payout_data.get("paymentType") == "Escrow":
            signer = None if isinstance(input, list) else input.get("signer")
            if not signer:
                raise RuntimeError("A signer or private key is required to add payments to escrow payouts")
            finalize_options = {} if isinstance(input, list) else (input.get("finalizeOptions") or {})
            return self._add_escrow_payees(payout_data, payments, signer, finalize_options, request_options)

        payout_id = payout_data if isinstance(payout_data, str) else payout_data["id"]
        response = self.http.request(
            "POST",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}/payments",
            body={"payments": _normalize_payments_for_create(payments, payout_data.get("chain") if isinstance(payout_data, dict) else "")},
            options=request_options,
        )
        return self._wrap_payout_response(self._parse(response))

    def _add_escrow_payees(
        self,
        escrow_batch: Union[str, Dict[str, Any]],
        payments: List[Dict[str, Any]],
        signer: PayoutSignerInput,
        options: Optional[Dict[str, Any]] = None,
        request_options: Optional[RequestOptions] = None,
    ):
        options = options or {}
        escrow_payout = self.get(escrow_batch, request_options).get("data") if isinstance(escrow_batch, str) else escrow_batch
        if isinstance(escrow_payout, PayoutIntent):
            escrow_payout = escrow_payout.get("data") or {}

        if escrow_payout.get("paymentType") != "Escrow":
            raise RuntimeError("addEscrowPayees requires an escrow payout")
        if not escrow_payout.get("batchHash"):
            raise RuntimeError("Escrow payout must be finalized before adding payees")
        if escrow_payout.get("status") and escrow_payout.get("status") != "funded":
            raise RuntimeError("Escrow payout must be funded before adding payees")
        if not payments:
            raise RuntimeError("At least one payee is required")

        claim_date = int(options.get("claimDate") or int(time.time()))
        funding_token = _normalize_token_address(options.get("fundingToken")) or _resolve_payout_funding_token_candidate(escrow_payout)
        scheduled_payments = []
        for payment in payments:
            item = dict(payment)
            item["token"] = _normalize_token_address(payment.get("token")) or funding_token
            item["claimDate"] = payment.get("claimDate") or claim_date
            scheduled_payments.append(item)

        return self.createFinalized(
            {
                "id": options.get("id") or _create_payout_id(),
                "type": "Scheduled",
                "chain": options.get("chain") or escrow_payout.get("chain"),
                "name": options.get("name") or f"{str(escrow_payout.get('name') or 'Escrow payout')} Payees",
                "description": options.get("description"),
                "complianceMode": options.get("complianceMode") or escrow_payout.get("complianceMode") or "Open",
                "escrowBatch": escrow_payout,
                "payments": scheduled_payments,
                "metadata": {
                    **(options.get("metadata") or {}),
                    "payoutCurrency": (options.get("metadata") or {}).get("payoutCurrency") or funding_token,
                    "escrowBatch": escrow_payout.get("id"),
                    "escrowBatchHash": escrow_payout.get("batchHash"),
                    "scheduledDate": claim_date,
                },
            },
            signer,
            {
                **options,
                "clientId": options.get("clientId") or (escrow_payout.get("app") or {}).get("clientId"),
                "chain": options.get("chain") or escrow_payout.get("chain"),
                "escrowBatch": escrow_payout,
                "fundingToken": funding_token,
                "payments": scheduled_payments,
                "claimDate": claim_date,
            },
            request_options,
        )

    def addRecipients(self, payout_id: str, input: Union[Dict[str, Any], List[Dict[str, Any]]], options: Optional[RequestOptions] = None):
        recipients = input if isinstance(input, list) else input.get("recipients")
        response = self.http.request(
            "POST",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}/open-payees",
            body={"recipients": recipients},
            options=options,
        )
        return self._parse(response)

    def resolveRecipients(self, payout_id: str, input: Union[Dict[str, Any], List[Dict[str, Any]]], options: Optional[RequestOptions] = None):
        recipients = input if isinstance(input, list) else input.get("recipients")
        response = self.http.request(
            "POST",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}/resolve-recipients",
            body={"recipients": recipients},
            options=options,
        )
        return self._parse(response)

    def removePayments(self, payout_id: str, input: Union[Dict[str, Any], List[Union[int, str]]], options: Optional[RequestOptions] = None):
        payment_ids = input if isinstance(input, list) else input.get("paymentIds")
        response = self.http.request(
            "DELETE",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}/payments",
            body={"paymentIds": [int(x) for x in payment_ids]},
            options=options,
        )
        return self._parse(response)

    def deletePayment(self, payout_id: str, payment_id: Union[int, str], options: Optional[RequestOptions] = None):
        return self.removePayments(payout_id, [payment_id], options)

    def updatePayment(self, payout_id: str, payment_id: Union[int, str], input: Dict[str, Any], options: Optional[RequestOptions] = None):
        response = self.http.request(
            "PATCH",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}/payments/{quote(str(payment_id), safe='')}",
            body=input,
            options=options,
        )
        return self._parse(response)

    def editPayment(self, payout_id: str, payment_id: Union[int, str], input: Dict[str, Any], options: Optional[RequestOptions] = None):
        return self.updatePayment(payout_id, payment_id, input, options)

    def listPayments(self, payout_id: str, query: Optional[Dict[str, Any]] = None, options: Optional[RequestOptions] = None):
        response = self.http.request(
            "GET",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}/payments",
            query=query,
            options=options,
        )
        return self._parse(response)

    def listInvites(self, payout_id: str, options: Optional[RequestOptions] = None):
        response = self.http.request(
            "GET",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}/invites",
            options=options,
        )
        return self._parse(response)

    def revokeInvite(self, payout_id: str, invite_id: Union[int, str], options: Optional[RequestOptions] = None):
        response = self.http.request(
            "DELETE",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}/invites/{quote(str(invite_id), safe='')}",
            options=options,
        )
        return self._parse(response)

    def revokeInviteRoot(self, payout_id: str, invite_root_id: Union[int, str], options: Optional[RequestOptions] = None):
        response = self.http.request(
            "DELETE",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}/invite-roots/{quote(str(invite_root_id), safe='')}",
            options=options,
        )
        return self._parse(response)

    def delete(self, payout_id: str, options: Optional[RequestOptions] = None):
        response = self.http.request(
            "DELETE",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}",
            options=options,
        )
        return self._parse(response)

    def _build_scheduled_finalize_payload(self, payout: Dict[str, Any], signer: PayoutSignerInput, options: Dict[str, Any], request_options: Optional[RequestOptions] = None):
        payout_id = payout["id"]
        timestamp = int(options.get("timestamp") or int(time.time()))
        signer_address = _resolve_signer_address(signer, options.get("signerAddress"))
        compliance_mode = options.get("complianceMode") or payout.get("complianceMode") or "Open"
        client_id = options.get("clientId") or (payout.get("app") or {}).get("clientId") or self.clientId
        if not client_id:
            raise RuntimeError("clientId is required to finalize this payout")
        chain = options.get("chain") or payout.get("chain") or ""
        payments = options.get("payments") or payout.get("payments")
        if not isinstance(payments, list):
            raise RuntimeError("Payout response does not include payments")

        grace_period = int(options.get("gracePeriod", (payout.get("metadata") or {}).get("gracePeriod", 0)))
        disapproval_deadline = int(options.get("disapprovalDeadline", (payout.get("metadata") or {}).get("disapprovalDeadline", 0)))
        funding_token = _resolve_funding_token(payout, options)

        chain_id = _resolve_payout_chain_id(chain, options.get("chainId"), "scheduled payout finalization")
        payments = _normalize_payments_for_signing(payments, chain, funding_token)

        batch_hash = computeScheduledPayoutHash(
            {
                "payoutId": payout_id,
                "fundingToken": funding_token,
                "gracePeriod": grace_period,
                "disapprovalDeadline": disapproval_deadline,
                "timestamp": timestamp,
                "chainId": chain_id,
            }
        )

        backend_message = f"PVIUM_SIGNED_SCHEDULE:{client_id}:{batch_hash}:{compliance_mode}:{timestamp}"
        backend_signature = _sign_payout_finalize_message(signer, backend_message, chain)
        signer_address = _require_signer_address(signer_address or backend_signature.get("signerAddress"), "Scheduled payout finalization")

        merkle = _generate_merkle_tree_for_payout(
            batch_hash,
            payments,
            int(options.get("claimDate") or (payout.get("metadata") or {}).get("scheduledDate") or (payout.get("metadata") or {}).get("claimableDate") or 0),
        )
        merkle_root = merkle["merkleRoot"]
        batch_data_hash = _to_hex(keccak(encode_packed(["bytes32", "bytes32", "address"], [_hex_bytes(batch_hash), _hex_bytes(merkle_root), to_checksum_address(signer_address)])))

        update_payload: Dict[str, Any] = {
            "signer": signer_address,
            "batchSignature": f"{timestamp}:{signer_address}:{backend_signature['signature']}",
            "batchHash": batch_hash,
            "merkleRoot": merkle_root,
            "batchDataHash": batch_data_hash,
            "proofs": [{"receiver": p["receiver"], "proof": p["proof"]} for p in merkle["proofs"]],
            "gracePeriod": grace_period,
            "disapprovalDeadline": disapproval_deadline,
        }

        if "solana" not in chain.lower():
            funding_digest = batch_data_hash
            linked_escrow = options.get("escrowBatch") or payout.get("escrowBatch") or (payout.get("metadata") or {}).get("escrowBatch")
            if linked_escrow:
                escrow_payout = self.get(linked_escrow, request_options).get("data") if isinstance(linked_escrow, str) else linked_escrow
                if isinstance(escrow_payout, PayoutIntent):
                    escrow_payout = escrow_payout.get("data") or {}
                if not escrow_payout.get("batchHash"):
                    raise RuntimeError("Linked escrow payout must be finalized before finalizing scheduled payouts")
                funding_digest = computeEscrowScheduledFundingDigest(
                    {"escrowBatchHash": escrow_payout["batchHash"], "merkleRoot": merkle_root}
                )

            update_payload["fundingSignature"] = _sign_funding_digest(signer, funding_digest)

        return {
            "updatePayload": update_payload,
            "batchDataHash": batch_data_hash,
            "batchHash": batch_hash,
            "merkleRoot": merkle_root,
            "fundingToken": funding_token,
        }

    def finalize(
        self,
        payout_input: Union[str, Dict[str, Any]],
        signer: PayoutSignerInput,
        options: Optional[Dict[str, Any]] = None,
        request_options: Optional[RequestOptions] = None,
    ):
        options = options or {}
        payout = self.get(payout_input, request_options).get("data") if isinstance(payout_input, str) else payout_input
        if isinstance(payout, PayoutIntent):
            payout = payout.get("data") or {}
        payout_id = payout["id"]
        timestamp = int(options.get("timestamp") or int(time.time()))
        signer_address = _resolve_signer_address(signer, options.get("signerAddress"))
        compliance_mode = options.get("complianceMode") or payout.get("complianceMode") or "Open"
        client_id = options.get("clientId") or (payout.get("app") or {}).get("clientId") or self.clientId
        if not client_id:
            raise RuntimeError("clientId is required to finalize this payout")
        chain = options.get("chain") or payout.get("chain") or ""
        payments = options.get("payments") or payout.get("payments")
        if not isinstance(payments, list):
            raise RuntimeError("Payout response does not include payments")

        update_payload: Dict[str, Any] = {}
        batch_data_hash: HexString
        batch_hash: Optional[HexString] = None
        merkle_root: Optional[HexString] = None

        if payout.get("paymentType") == "Scheduled" or payout.get("isCommitment"):
            grace_period = int(options.get("gracePeriod", (payout.get("metadata") or {}).get("gracePeriod", 0)))
            disapproval_deadline = int(options.get("disapprovalDeadline", (payout.get("metadata") or {}).get("disapprovalDeadline", 0)))
            funding_token = _resolve_funding_token(payout, options)
            chain_id = _resolve_payout_chain_id(chain, options.get("chainId"), "scheduled payout finalization")
            payments = _normalize_payments_for_signing(payments, chain, funding_token)

            batch_hash = computeScheduledPayoutHash(
                {
                    "payoutId": payout_id,
                    "fundingToken": funding_token,
                    "gracePeriod": grace_period,
                    "disapprovalDeadline": disapproval_deadline,
                    "timestamp": timestamp,
                    "chainId": chain_id,
                }
            )

            backend_message = f"PVIUM_SIGNED_SCHEDULE:{client_id}:{batch_hash}:{compliance_mode}:{timestamp}"
            backend_signature = _sign_payout_finalize_message(signer, backend_message, chain)
            signer_address = _require_signer_address(
                signer_address or backend_signature.get("signerAddress"),
                "Scheduled payout finalization",
            )

            merkle = _generate_merkle_tree_for_payout(
                batch_hash,
                payments,
                int(options.get("claimDate") or (payout.get("metadata") or {}).get("scheduledDate") or (payout.get("metadata") or {}).get("claimableDate") or 0),
            )
            merkle_root = merkle["merkleRoot"]
            batch_data_hash = _to_hex(keccak(encode_packed(["bytes32", "bytes32", "address"], [_hex_bytes(batch_hash), _hex_bytes(merkle_root), to_checksum_address(signer_address)])))

            update_payload["signer"] = signer_address
            update_payload["batchSignature"] = f"{timestamp}:{signer_address}:{backend_signature['signature']}"
            update_payload["batchHash"] = batch_hash
            update_payload["merkleRoot"] = merkle_root
            update_payload["batchDataHash"] = batch_data_hash
            update_payload["proofs"] = [{"receiver": p["receiver"], "proof": p["proof"]} for p in merkle["proofs"]]
            update_payload["gracePeriod"] = grace_period
            update_payload["disapprovalDeadline"] = disapproval_deadline

            if "solana" not in chain.lower():
                funding_digest = batch_data_hash
                linked_escrow = options.get("escrowBatch") or payout.get("escrowBatch") or (payout.get("metadata") or {}).get("escrowBatch")
                if linked_escrow:
                    escrow_payout = self.get(linked_escrow, request_options).get("data") if isinstance(linked_escrow, str) else linked_escrow
                    if isinstance(escrow_payout, PayoutIntent):
                        escrow_payout = escrow_payout.get("data") or {}
                    if not escrow_payout.get("batchHash"):
                        raise RuntimeError("Linked escrow payout must be finalized before finalizing scheduled payouts")
                    funding_digest = computeEscrowScheduledFundingDigest(
                        {"escrowBatchHash": escrow_payout["batchHash"], "merkleRoot": merkle_root}
                    )
                update_payload["fundingSignature"] = _sign_funding_digest(signer, funding_digest)

        elif payout.get("paymentType") == "Escrow":
            nonce = payout.get("nonce") or (str(options.get("timestamp")) if options.get("timestamp") else None)
            if not nonce:
                raise RuntimeError("Payout nonce is required to finalize escrow payouts")

            funding_token = _resolve_funding_token(payout, options)
            chain_id = _resolve_payout_chain_id(chain, options.get("chainId"), "escrow payout finalization")
            payments = _normalize_payments_for_signing(payments, chain, funding_token)
            lock_duration = int(options.get("lockDuration") or (payout.get("metadata") or {}).get("lockDuration") or payout.get("lockDuration") or 0)
            if lock_duration <= 0:
                raise RuntimeError("lockDuration is required to finalize escrow payouts")

            batch_data_hash = generateInstantPayoutHash(payments, nonce)
            message = f"PVIUM_SIGNED_BATCH:{client_id}:{batch_data_hash}:{compliance_mode}:{timestamp}"
            signature = _sign_payout_finalize_message(signer, message, chain)
            signer_address = _require_signer_address(
                signer_address or signature.get("signerAddress"),
                "Escrow payout finalization",
            )

            escrow_batch_hash = computeEscrowPayoutHash(
                {
                    "payoutId": payout_id,
                    "fundingToken": funding_token,
                    "lockDuration": lock_duration,
                    "timestamp": timestamp,
                    "chainId": chain_id,
                }
            )
            escrow_funding_digest = computeEscrowFundingDigest(
                {"escrowBatchHash": escrow_batch_hash, "withdrawalWallet": signer_address}
            )

            update_payload["signer"] = signer_address
            update_payload["batchSignature"] = f"{timestamp}:{signer_address}:{signature['signature']}"
            update_payload["fundingSignature"] = f"{timestamp}:{signer_address}:{_sign_funding_digest(signer, escrow_funding_digest)}"
            update_payload["batchHash"] = escrow_batch_hash
            update_payload["batchDataHash"] = batch_data_hash
            update_payload["metadata"] = {**(payout.get("metadata") or {}), "lockDuration": lock_duration}
            batch_hash = escrow_batch_hash

        else:
            nonce = payout.get("nonce") or (str(options.get("timestamp")) if options.get("timestamp") else None)
            if not nonce:
                raise RuntimeError("Payout nonce is required to finalize instant payouts")

            batch_data_hash = generateInstantPayoutHash(payments, nonce)
            message = f"PVIUM_SIGNED_BATCH:{client_id}:{batch_data_hash}:{compliance_mode}:{timestamp}"
            signature = _sign_payout_finalize_message(signer, message, chain)
            signer_address = _require_signer_address(
                signer_address or signature.get("signerAddress"),
                "Instant payout finalization",
            )

            update_payload["signer"] = signer_address
            update_payload["batchSignature"] = f"{timestamp}:{signer_address}:{signature['signature']}"
            update_payload["batchDataHash"] = batch_data_hash

        response = self.http.request(
            "PATCH",
            f"/v1/batch-payments/{quote(str(payout_id), safe='')}",
            body=update_payload,
            options=request_options,
        )
        finalized = self._parse(response)
        finalized_payout = finalized.get("data") or {}

        identifier = (
            finalized_payout.get("merkleRoot") or merkle_root
            if finalized_payout.get("paymentType") == "Scheduled"
            else finalized_payout.get("batchDataHash") or batch_data_hash
        )

        return self._wrap_finalization_response({
            "meta": finalized.get("meta", {}),
            "data": {
                "payout": finalized_payout,
                "fundingUrl": f"{self.consentHost}/batch/{identifier}" if identifier else None,
                "batchDataHash": batch_data_hash,
                "batchHash": batch_hash,
                "merkleRoot": merkle_root,
            },
        })

    def _wrap_payout_response(self, response: Dict[str, Any]) -> PayoutIntent:
        return PayoutIntent(self, response.get("meta", {}), response.get("data") or {})

    def _wrap_finalization_response(self, response: Dict[str, Any]) -> PayoutFinalization:
        return PayoutFinalization(self, response.get("meta", {}), response.get("data") or {})

    def _parse(self, response):
        body = self.http.parseResponseBody(response)
        if not response.ok:
            message = None
            if isinstance(body, dict):
                message = (
                    (body.get("meta") or {}).get("message")
                    or body.get("message")
                    or body.get("error")
                )
            if not message:
                message = f"Pvium API request failed with status {response.status}"
            raise RuntimeError(str(message))
        return body
