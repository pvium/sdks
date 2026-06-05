from __future__ import annotations

import json
import logging
import socket
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass, field
from typing import Any, Callable, Dict, Mapping, Optional

from .types import RequestOptions


PVIUM_BASE_URLS: Dict[str, str] = {
    "test": "http://localhost:4005/v1",
    "sandbox": "https://api-sandbox.pvium.com/v1",
    "production": "https://api.pvium.com/v1",
}

PVIUM_CONSENT_HOSTS: Dict[str, str] = {
    "test": "http://localhost:3000",
    "sandbox": "https://sandbox.pvium.com",
    "production": "https://pvium.com",
}


@dataclass
class PviumSdkConfig:
    baseUrl: Optional[str] = None
    apiKey: Optional[str] = None
    clientId: Optional[str] = None
    environment: Optional[str] = "production"
    consentHost: Optional[str] = None
    timeoutMs: int = 30000
    fetchFn: Optional[Callable[[str, str, Dict[str, str], Optional[str], float], tuple[int, Dict[str, str], str]]] = None
    defaultHeaders: Dict[str, str] = field(default_factory=dict)
    logging: Optional[Dict[str, Any]] = None


def resolvePviumBaseUrl(config: PviumSdkConfig) -> str:
    environment = config.environment or "production"
    base = config.baseUrl or PVIUM_BASE_URLS.get(environment) or PVIUM_BASE_URLS["production"]
    return base.rstrip("/")


def resolvePviumConsentHost(config: PviumSdkConfig) -> str:
    environment = config.environment or "production"
    host = config.consentHost or PVIUM_CONSENT_HOSTS.get(environment) or PVIUM_CONSENT_HOSTS["production"]
    return host.rstrip("/")


@dataclass
class HttpResponse:
    status: int
    headers: Dict[str, str]
    text: str

    @property
    def ok(self) -> bool:
        return 200 <= self.status < 300


class PviumHttpClient:
    def __init__(self, config: PviumSdkConfig):
        self.base_url = resolvePviumBaseUrl(config)
        self.timeout_ms = config.timeoutMs or 30000
        self.fetch_fn = config.fetchFn
        self.api_key = config.apiKey
        self.default_headers = config.defaultHeaders or {}
        logging_config = config.logging or {}
        self.log_requests = bool(logging_config.get("requests"))
        self.logger = logging_config.get("logger") or logging.getLogger("pvium-sdk")

    def setApiKey(self, key: Optional[str] = None) -> None:
        self.api_key = key

    def request(
        self,
        method: str,
        path: str,
        query: Optional[Mapping[str, Any]] = None,
        body: Optional[Any] = None,
        options: Optional[RequestOptions] = None,
    ) -> HttpResponse:
        options = options or {}
        url = self._build_url(path, query)

        headers: Dict[str, str] = {"Accept": "application/json"}
        headers.update(self.default_headers)
        headers.update(options.get("headers", {}))

        access_token = options.get("accessToken")
        if access_token:
            headers["Authorization"] = f"Bearer {access_token}"
        else:
            api_key = options.get("apiKey") or self.api_key
            if api_key and not options.get("skipApiKey"):
                headers["x-api-key"] = api_key

        payload: Optional[str] = None
        if body is not None:
            headers["Content-Type"] = "application/json"
            payload = json.dumps(body)

        if self.log_requests:
            self.logger.debug(
                "[pvium-sdk] request",
                extra={
                    "method": method,
                    "url": url,
                    "timeoutMs": self.timeout_ms,
                },
            )

        try:
            if self.fetch_fn:
                status, response_headers, text = self.fetch_fn(
                    method,
                    url,
                    headers,
                    payload,
                    self.timeout_ms / 1000,
                )
                return HttpResponse(status=status, headers=response_headers, text=text)

            req = urllib.request.Request(
                url=url,
                data=payload.encode("utf-8") if payload is not None else None,
                method=method,
                headers=headers,
            )
            with urllib.request.urlopen(req, timeout=self.timeout_ms / 1000) as response:
                response_headers = {k.lower(): v for k, v in response.headers.items()}
                text = response.read().decode("utf-8")
                return HttpResponse(status=response.status, headers=response_headers, text=text)
        except urllib.error.HTTPError as err:
            text = err.read().decode("utf-8") if err.fp else ""
            return HttpResponse(
                status=err.code,
                headers={k.lower(): v for k, v in err.headers.items()} if err.headers else {},
                text=text,
            )
        except (urllib.error.URLError, socket.timeout) as err:
            raise RuntimeError(str(err)) from err

    def parseResponseBody(self, response: HttpResponse) -> Any:
        content_type = response.headers.get("content-type", "")
        if "application/json" in content_type:
            if not response.text:
                return None
            return json.loads(response.text)
        return response.text if response.text else None

    def _build_url(self, path: str, query: Optional[Mapping[str, Any]] = None) -> str:
        normalized_path = self._normalize_path(path)
        url = f"{self.base_url}{normalized_path}"

        if query:
            pairs = []
            for key, value in query.items():
                if value is None:
                    continue
                pairs.append((key, str(value)))
            if pairs:
                url = f"{url}?{urllib.parse.urlencode(pairs)}"

        return url

    def _normalize_path(self, path: str) -> str:
        if not path.startswith("/"):
            path = f"/{path}"
        if self.base_url.endswith("/v1") and path.startswith("/v1/"):
            return path[3:]
        return path
