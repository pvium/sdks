from __future__ import annotations

import asyncio
from typing import Any

from .client import PviumSdkConfig
from .sdk import PviumSdk


class _AsyncServiceProxy:
    def __init__(self, service: Any):
        self._service = service

    def __getattr__(self, name: str) -> Any:
        attr = getattr(self._service, name)
        if callable(attr):
            async def _wrapped(*args: Any, **kwargs: Any) -> Any:
                return await asyncio.to_thread(attr, *args, **kwargs)

            return _wrapped

        return attr


class AsyncPviumSdk:
    def __init__(self, config: PviumSdkConfig):
        self._sync_sdk = PviumSdk(config)
        self.http = _AsyncServiceProxy(self._sync_sdk.http)
        self.endpoints = _AsyncServiceProxy(self._sync_sdk.endpoints)
        self.invites = _AsyncServiceProxy(self._sync_sdk.invites)
        self.oauth = _AsyncServiceProxy(self._sync_sdk.oauth)
        self.payout = _AsyncServiceProxy(self._sync_sdk.payout)

    @staticmethod
    def init(config: PviumSdkConfig) -> "AsyncPviumSdk":
        return AsyncPviumSdk(config)
