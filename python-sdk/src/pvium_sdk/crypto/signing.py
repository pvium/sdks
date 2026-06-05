from __future__ import annotations

from typing import Any, Dict, Iterable, List, Optional, Protocol, Sequence, Union

from eth_abi import encode
from eth_account import Account
from eth_account.messages import encode_defunct
from eth_utils import keccak, to_hex


HexString = str
Numeric = Union[int, str]


class MessageSigner(Protocol):
    def sign_message_hash(self, message_hash: str) -> str: ...


PVIUM_SIGNATURE_DOMAIN = to_hex(keccak(text="PVIUM_SIGNATURE_MESSAGE"))


def _int_like(value: Numeric) -> int:
    if isinstance(value, str):
        if value.startswith("0x"):
            return int(value, 16)
        return int(value)
    return int(value)


def _hex_to_bytes(value: str) -> bytes:
    raw = value[2:] if value.startswith("0x") else value
    return bytes.fromhex(raw)


def _ensure_0x(value: str) -> str:
    return value if value.startswith("0x") else f"0x{value}"


def _to_bytes32(value: Numeric) -> bytes:
    return _int_like(value).to_bytes(32, byteorder="big", signed=False)


def createSignerFromPrivateKey(private_key: str):
    return Account.from_key(private_key)


def hashAbiEncodedPayload(types: Sequence[str], values: Sequence[Any]) -> HexString:
    return to_hex(keccak(encode(list(types), list(values))))


def signMessageHash(message_hash: str, signer_or_private_key: Any) -> str:
    if isinstance(signer_or_private_key, str):
        account = Account.from_key(signer_or_private_key)
        signed = account.sign_message(encode_defunct(hexstr=message_hash))
        return _ensure_0x(signed.signature.hex())

    if hasattr(signer_or_private_key, "sign_message_hash"):
        return signer_or_private_key.sign_message_hash(message_hash)

    if hasattr(signer_or_private_key, "sign_message"):
        signed = signer_or_private_key.sign_message(encode_defunct(hexstr=message_hash))
        signature = signed.signature if hasattr(signed, "signature") else signed
        if isinstance(signature, bytes):
            return _ensure_0x(signature.hex())
        return _ensure_0x(str(signature))

    raise RuntimeError("Unsupported signer input")


def hashCreateProjectRequest(payload: Dict[str, Any], options: Dict[str, Any]) -> HexString:
    signature_domain = options.get("signatureDomain") or PVIUM_SIGNATURE_DOMAIN
    return hashAbiEncodedPayload(
        [
            "bytes32",
            "string",
            "string",
            "string",
            "address",
            "address",
            "address",
            "address",
            "uint256",
            "uint256",
            "uint256",
            "uint256",
            "uint256",
            "uint256",
        ],
        [
            _hex_to_bytes(signature_domain),
            payload["app"],
            payload["projectId"],
            payload["metadata"],
            payload["tokenAddress"],
            payload["refundAddress"],
            payload["appFeeAddress"],
            payload["appAdminAddress"],
            _int_like(payload["appFeeBps"]),
            _int_like(payload["disputeWindowSeconds"]),
            _int_like(payload["lockDuration"]),
            _int_like(payload["minimumBalancePerVendor"]),
            _int_like(options["pviumFeeBps"]),
            _int_like(options["chainId"]),
        ],
    )


def signCreateProjectRequest(payload: Dict[str, Any], signer_or_private_key: Any, options: Dict[str, Any]) -> str:
    return signMessageHash(hashCreateProjectRequest(payload, options), signer_or_private_key)


def hashCreateProjectAttestation(app_signature: str, chain_id: Numeric, signature_domain: HexString = PVIUM_SIGNATURE_DOMAIN) -> HexString:
    return hashAbiEncodedPayload(
        ["bytes32", "bytes", "uint256"],
        [_hex_to_bytes(signature_domain), _hex_to_bytes(app_signature), _int_like(chain_id)],
    )


def signCreateProjectAttestation(app_signature: str, signer_or_private_key: Any, chain_id: Numeric, signature_domain: HexString = PVIUM_SIGNATURE_DOMAIN) -> str:
    return signMessageHash(hashCreateProjectAttestation(app_signature, chain_id, signature_domain), signer_or_private_key)


def hashCreateClaimRequest(payload: Dict[str, Any]) -> HexString:
    return hashAbiEncodedPayload(
        ["string", "string", "bytes32", "address", "uint256", "uint256", "uint256", "uint256"],
        [
            payload["app"],
            payload["projectId"],
            _hex_to_bytes(payload["claimId"]),
            payload["receiver"],
            _int_like(payload["amount"]),
            _int_like(payload["claimableAfter"]),
            _int_like(payload["claimDeadline"]),
            _int_like(payload["nonce"]),
        ],
    )


def signCreateClaimRequest(payload: Dict[str, Any], signer_or_private_key: Any) -> str:
    return signMessageHash(hashCreateClaimRequest(payload), signer_or_private_key)


def hashFinalizeClaimRequest(claims: Sequence[Dict[str, Any]], chain_id: Numeric) -> HexString:
    data_packed = b""
    for claim in claims:
        data_packed += claim["app"].encode("utf-8")
        data_packed += claim["projectId"].encode("utf-8")
        data_packed += _hex_to_bytes(claim["claimId"])

    return to_hex(keccak(data_packed + _to_bytes32(chain_id)))


def signFinalizeClaimRequest(claims: Sequence[Dict[str, Any]], signer_or_private_key: Any, chain_id: Numeric) -> str:
    return signMessageHash(hashFinalizeClaimRequest(claims, chain_id), signer_or_private_key)


def hashRelayedCallRequest(payload: Dict[str, Any]) -> HexString:
    return hashAbiEncodedPayload(
        ["string", "string", "bytes", "uint256", "uint256"],
        [
            payload["appId"],
            payload["projectId"],
            _hex_to_bytes(payload["payload"]),
            _int_like(payload["nonce"]),
            _int_like(payload["chainId"]),
        ],
    )


def signRelayedCallRequest(payload: Dict[str, Any], signer_or_private_key: Any) -> str:
    return signMessageHash(hashRelayedCallRequest(payload), signer_or_private_key)


def hashDisputeRequest(claim_id: str, chain_id: Numeric) -> HexString:
    return hashAbiEncodedPayload(["bytes32", "uint256"], [_hex_to_bytes(claim_id), _int_like(chain_id)])


def signDisputeRequest(claim_id: str, signer_or_private_key: Any, chain_id: Numeric) -> str:
    return signMessageHash(hashDisputeRequest(claim_id, chain_id), signer_or_private_key)


def hashResolveDisputeRequest(payload: Dict[str, Any]) -> HexString:
    return hashAbiEncodedPayload(
        ["bytes32", "bool", "uint256"],
        [_hex_to_bytes(payload["claimId"]), bool(payload["approved"]), _int_like(payload["chainId"])],
    )


def signResolveDisputeRequest(payload: Dict[str, Any], signer_or_private_key: Any) -> str:
    return signMessageHash(hashResolveDisputeRequest(payload), signer_or_private_key)


def signatureDomainFromText(message: str) -> HexString:
    return to_hex(keccak(text=message))
