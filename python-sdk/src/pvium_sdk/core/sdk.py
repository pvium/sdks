from __future__ import annotations

from .client import PviumHttpClient, PviumSdkConfig
from ..services.endpoints import PviumEndpoints
from ..services.invites import PviumInviteService
from ..services.oauth import PviumOAuth
from ..services.payout import PviumPayoutService


class PviumSdk:
    def __init__(self, config: PviumSdkConfig):
        self.http = PviumHttpClient(config)
        self.endpoints = PviumEndpoints(self.http)
        self.invites = PviumInviteService(self.http, config)
        self.oauth = PviumOAuth(self.http, config)
        self.payout = PviumPayoutService(self.http, config)

    @staticmethod
    def init(config: PviumSdkConfig) -> "PviumSdk":
        return PviumSdk(config)
