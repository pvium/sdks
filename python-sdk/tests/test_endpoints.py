import json
from urllib.parse import unquote

from pvium_sdk import PviumSdk, PviumSdkConfig


def make_fetch(response_body, requests):
    def _fetch(method, url, headers, payload, timeout):
        requests.append(
            {
                "method": method,
                "url": url,
                "headers": headers,
                "payload": payload,
                "timeout": timeout,
            }
        )
        return response_body.get("meta", {}).get("statusCode", 200), {"content-type": "application/json"}, json.dumps(response_body)

    return _fetch


def test_list_invoices_calls_endpoint_with_api_key():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            apiKey="pk_test_dummy",
            fetchFn=make_fetch({"meta": {"statusCode": 200, "success": True}, "data": []}, requests),
        )
    )

    result = sdk.endpoints.listInvoices()

    assert result["data"] == []
    assert len(requests) == 1
    assert requests[0]["url"] == "http://localhost:4005/v1/invoices"
    assert requests[0]["method"] == "GET"
    assert requests[0]["headers"]["x-api-key"] == "pk_test_dummy"


def test_get_invoice_status_encodes_code():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch({"meta": {"statusCode": 200, "success": True}, "data": {"contractCode": "INV 123"}}, requests),
        )
    )

    result = sdk.endpoints.getInvoiceStatus("INV 123")

    assert result["data"]["contractCode"] == "INV 123"
    assert requests[0]["url"].endswith("/invoices/INV%20123/status")


def test_cancel_invoice_patches_active_false():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch({"meta": {"statusCode": 200, "success": True}, "data": {"id": 1, "active": False}}, requests),
        )
    )

    result = sdk.endpoints.cancelInvoice(42232)

    assert result["data"]["active"] is False
    assert requests[0]["url"].endswith("/invoices/42232")
    assert requests[0]["method"] == "PATCH"
    assert json.loads(requests[0]["payload"]) == {"active": False}


def test_get_installment_payments_endpoint():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch({"meta": {"statusCode": 200, "success": True}, "data": []}, requests),
        )
    )

    result = sdk.endpoints.getInstallmentPayments(42232)

    assert result["data"] == []
    assert requests[0]["url"].endswith("/payment-installments/42232/payments")


def test_create_invoice_posts_json_body():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="http://localhost:4005/v1",
            fetchFn=make_fetch({"meta": {"statusCode": 201, "success": True}, "data": {"id": "invoice_123"}}, requests),
        )
    )

    payload = {
        "name": "SDK Test",
        "description": "Integration test invoice",
        "amount": 50,
        "dueDate": "2026-04-28T00:00:00.000Z",
    }

    result = sdk.endpoints.createInvoice(payload)

    assert result["data"]["id"] == "invoice_123"
    assert requests[0]["url"].endswith("/invoices")
    assert requests[0]["method"] == "POST"
    assert json.loads(requests[0]["payload"]) == payload


def test_access_token_uses_bearer_and_suppresses_api_key():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="https://api.example.test/v1",
            apiKey="app_key",
            fetchFn=make_fetch({"meta": {"statusCode": 200, "success": True}, "data": []}, requests),
        )
    )

    sdk.endpoints.listInvoices({"accessToken": "access_user"})

    assert requests[0]["headers"]["Authorization"] == "Bearer access_user"
    assert "x-api-key" not in requests[0]["headers"]


def test_configured_api_key_is_used_when_access_token_is_not_provided():
    requests = []
    sdk = PviumSdk.init(
        PviumSdkConfig(
            baseUrl="https://api.example.test/v1",
            apiKey="app_key",
            fetchFn=make_fetch({"meta": {"statusCode": 200, "success": True}, "data": []}, requests),
        )
    )

    sdk.endpoints.listInvoices()

    assert requests[0]["headers"]["x-api-key"] == "app_key"
    assert "Authorization" not in requests[0]["headers"]
