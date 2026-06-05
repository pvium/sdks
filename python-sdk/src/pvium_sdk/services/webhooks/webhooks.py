from __future__ import annotations

import base64
import hashlib
import hmac
import json
from typing import Any, Dict, Optional


def _b64url_decode(value: str) -> bytes:
    pad_len = (4 - len(value) % 4) % 4
    return base64.urlsafe_b64decode(value + ("=" * pad_len))


def _parse_b64url_json(value: str) -> Optional[Dict[str, Any]]:
    try:
        return json.loads(_b64url_decode(value).decode("utf-8"))
    except Exception:
        return None


def _b64url_hmac_sha256(signing_input: str, secret: str) -> str:
    digest = hmac.new(secret.encode("utf-8"), signing_input.encode("utf-8"), hashlib.sha256).digest()
    return base64.urlsafe_b64encode(digest).decode("utf-8").rstrip("=")


def _safe_equal(a: str, b: str) -> bool:
    return hmac.compare_digest(a, b)


def verifyPviumWebhookToken(token: str, secret: str, options: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    options = options or {}
    parts = token.split(".")
    if len(parts) != 3:
        raise RuntimeError("Invalid Pvium webhook token")

    encoded_header, encoded_payload, encoded_signature = parts
    header = _parse_b64url_json(encoded_header)
    if not header or header.get("alg") != "HS256":
        raise RuntimeError("Unsupported Pvium webhook token algorithm")

    signing_input = f"{encoded_header}.{encoded_payload}"
    secrets = [secret]
    if options.get("allowHashedSecretFallback", True):
        hashed_secret = hashlib.sha256(secret.encode("utf-8")).hexdigest()
        if hashed_secret != secret:
            secrets.append(hashed_secret)

    signature_valid = any(_safe_equal(encoded_signature, _b64url_hmac_sha256(signing_input, candidate)) for candidate in secrets)
    if not signature_valid:
        raise RuntimeError("Invalid Pvium webhook token signature")

    payload = _parse_b64url_json(encoded_payload)
    if not payload:
        raise RuntimeError("Invalid Pvium webhook token payload")

    now = options.get("now")
    if isinstance(now, (int, float)):
        now_seconds = int(now // 1000)
    else:
        import time

        now_seconds = int(time.time())

    exp = payload.get("exp")
    if isinstance(exp, (int, float)) and now_seconds >= int(exp):
        raise RuntimeError("Expired Pvium webhook token")

    expected_event = options.get("expectedEvent")
    if expected_event and payload.get("event") and payload.get("event") != expected_event:
        raise RuntimeError("Pvium webhook token event mismatch")

    return payload


def resolvePviumWebhookPayload(body: Dict[str, Any], secret: str, options: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    options = options or {}
    event = body.get("event") or body.get("type")
    token = body.get("token")

    if not token:
        return {"event": event, "data": body.get("data") or {}}

    merged_options = dict(options)
    if merged_options.get("expectedEvent") is None:
        merged_options["expectedEvent"] = event

    token_payload = verifyPviumWebhookToken(token, secret, merged_options)
    return {
        "event": token_payload.get("event") or event,
        "data": token_payload.get("data") or {},
        "tokenPayload": token_payload,
    }
