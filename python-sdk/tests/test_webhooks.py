import base64
import hashlib
import hmac
import json

import pytest

from pvium_sdk import resolvePviumWebhookPayload, verifyPviumWebhookToken


def base64url_json(value):
    return base64.urlsafe_b64encode(json.dumps(value).encode("utf-8")).decode("utf-8").rstrip("=")


def sign_jwt(payload, secret):
    encoded_header = base64url_json({"alg": "HS256", "typ": "JWT"})
    encoded_payload = base64url_json(payload)
    signature = hmac.new(
        secret.encode("utf-8"), f"{encoded_header}.{encoded_payload}".encode("utf-8"), hashlib.sha256
    ).digest()
    encoded_signature = base64.urlsafe_b64encode(signature).decode("utf-8").rstrip("=")
    return f"{encoded_header}.{encoded_payload}.{encoded_signature}"


def test_verify_webhook_hs256_token():
    token = sign_jwt(
        {
            "event": "oauth.invite.accepted",
            "data": {"githubLogin": "octocat"},
            "iat": 1_700_000_000,
            "exp": 4_000_000_000,
        },
        "webhook_secret",
    )

    payload = verifyPviumWebhookToken(token, "webhook_secret", {"expectedEvent": "oauth.invite.accepted"})
    assert payload["event"] == "oauth.invite.accepted"
    assert payload["data"] == {"githubLogin": "octocat"}


def test_verify_webhook_supports_hashed_secret_fallback():
    secret = "secret_abc123"
    hashed_secret = hashlib.sha256(secret.encode("utf-8")).hexdigest()
    token = sign_jwt({"event": "batch.payee.added", "data": {"batch": {"id": "batch_123"}}, "exp": 4_000_000_000}, hashed_secret)

    payload = verifyPviumWebhookToken(token, secret)
    assert payload["event"] == "batch.payee.added"


def test_resolve_webhook_payload_reads_token_data():
    token = sign_jwt({"event": "invoice.paid", "data": {"invoiceId": "inv_123"}, "exp": 4_000_000_000}, "webhook_secret")

    resolved = resolvePviumWebhookPayload({"event": "invoice.paid", "token": token}, "webhook_secret")
    assert resolved["event"] == "invoice.paid"
    assert resolved["data"] == {"invoiceId": "inv_123"}


def test_expired_webhook_token_rejected():
    token = sign_jwt({"event": "invoice.paid", "data": {}, "exp": 1_700_000_000}, "webhook_secret")

    with pytest.raises(RuntimeError, match="Expired Pvium webhook token"):
        verifyPviumWebhookToken(token, "webhook_secret", {"now": 1_800_000_000_000})
