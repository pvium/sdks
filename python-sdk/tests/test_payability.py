import asyncio
import json
from pathlib import Path
from urllib.parse import urlparse

import pytest
from pvium_sdk import AsyncPviumSdk, PviumSdk, PviumSdkConfig

FIXTURE = json.loads((Path(__file__).resolve().parents[2] / "parity-fixtures/payability.json").read_text())


@pytest.mark.parametrize("response", FIXTURE["responses"])
@pytest.mark.parametrize("mode", ["service", "intent", "async"])
def test_payability_matches_shared_fixture(response, mode):
    requests = []

    def fetch(method, url, headers, payload, timeout):
        requests.append((method, url, headers, payload))
        body = {"meta": {"success": True}, "data": {"id": FIXTURE["payoutId"], "chain": "base", "paymentType": "Scheduled"}} if method == "GET" else response
        return 200, {"content-type": "application/json"}, json.dumps(body)

    config = PviumSdkConfig(baseUrl="https://api.example.test/v1", apiKey="app-key", fetchFn=fetch)
    identities = FIXTURE["body"]["identities"]
    options = {"accessToken": "parity-token"}
    if mode == "async":
        result = asyncio.run(AsyncPviumSdk.init(config).payout.isPayable(FIXTURE["payoutId"], identities, options))
    else:
        sdk = PviumSdk.init(config)
        result = sdk.payout.get(FIXTURE["payoutId"]).isPayable(identities, options) if mode == "intent" else sdk.payout.isPayable(FIXTURE["payoutId"], identities, options)
    method, url, headers, payload = requests[-1]
    assert method == FIXTURE["method"]
    assert urlparse(url).path == FIXTURE["path"]
    assert urlparse(url).query == ""
    assert json.loads(payload) == FIXTURE["body"]
    assert headers.get("Authorization") == "Bearer parity-token"
    assert "x-api-key" not in {key.lower(): value for key, value in headers.items()}
    assert result == response


def test_payability_propagates_api_errors():
    def fetch(*args):
        return 404, {"content-type": "application/json"}, json.dumps({"meta": {"success": False, "message": "Batch not found"}})
    sdk = PviumSdk.init(PviumSdkConfig(baseUrl="https://api.example.test/v1", fetchFn=fetch))
    with pytest.raises(RuntimeError, match="Batch not found"):
        sdk.payout.isPayable("missing", FIXTURE["body"]["identities"])
