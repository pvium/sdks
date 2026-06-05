import pytest

from pvium_sdk import AsyncPviumSdk, PviumSdkConfig


@pytest.mark.asyncio
async def test_async_sdk_wraps_sync_services():
    calls = []

    def fetch(method, url, headers, payload, timeout):
        calls.append((method, url))
        return 200, {"content-type": "application/json"}, '{"meta":{"statusCode":200,"success":true},"data":[]}'

    sdk = AsyncPviumSdk.init(PviumSdkConfig(baseUrl="http://localhost:4005/v1", fetchFn=fetch))

    result = await sdk.endpoints.listInvoices()

    assert result["data"] == []
    assert calls[0][0] == "GET"
    assert calls[0][1] == "http://localhost:4005/v1/invoices"
