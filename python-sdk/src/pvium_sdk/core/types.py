from __future__ import annotations

from typing import Any, Dict, Optional, TypedDict


class RequestOptions(TypedDict, total=False):
    accessToken: str
    apiKey: str
    headers: Dict[str, str]
    skipApiKey: bool


class ApiMeta(TypedDict, total=False):
    statusCode: int
    success: bool
    message: str
    developerMessage: str


class OAuthTokenData(TypedDict, total=False):
    accessToken: str
    refreshToken: str
    expiresIn: int
    expiresAt: str
    tokenType: str


class OAuthTokenResponse(TypedDict):
    meta: ApiMeta
    data: OAuthTokenData


class OAuthUserInfoResponse(TypedDict):
    meta: ApiMeta
    data: Dict[str, Any]


class CreateInvoiceRequest(TypedDict, total=False):
    name: str
    description: str
    amount: float
    dueDate: str
    paymentChannels: Any
    redirectUri: str


JsonDict = Dict[str, Any]
JsonValue = Any


def ensure_dict(value: Any) -> Dict[str, Any]:
    if isinstance(value, dict):
        return value
    return {"value": value}


def require_ok_response(body: Any, status_code: int) -> Any:
    if isinstance(body, dict):
        meta = body.get("meta")
        if isinstance(meta, dict) and meta.get("success") is False:
            msg = meta.get("message") or f"Pvium API request failed with status {status_code}"
            raise RuntimeError(str(msg))
        return body

    if 200 <= status_code < 300:
        return body

    raise RuntimeError(f"Pvium API request failed with status {status_code}")
