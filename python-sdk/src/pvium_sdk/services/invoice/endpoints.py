from __future__ import annotations

from typing import Any, Dict, Optional
from urllib.parse import quote

from ...core.client import PviumHttpClient
from ...core.types import RequestOptions


class PviumEndpoints:
    def __init__(self, http: PviumHttpClient):
        self.http = http

    def createInvoice(self, body: Dict[str, Any], options: Optional[RequestOptions] = None) -> Any:
        response = self.http.request("POST", "/v1/invoices", body=body, options=options)
        return self.http.parseResponseBody(response)

    def listInvoices(self, options: Optional[RequestOptions] = None) -> Any:
        response = self.http.request("GET", "/v1/invoices", options=options)
        return self.http.parseResponseBody(response)

    def getInvoiceStatus(self, code: str, options: Optional[RequestOptions] = None) -> Any:
        response = self.http.request(
            "GET",
            f"/v1/invoices/{quote(code, safe='')}/status",
            options=options,
        )
        return self.http.parseResponseBody(response)

    def cancelInvoice(self, invoice_id: str | int, options: Optional[RequestOptions] = None) -> Any:
        response = self.http.request(
            "PATCH",
            f"/v1/invoices/{quote(str(invoice_id), safe='')}",
            body={"active": False},
            options=options,
        )
        return self.http.parseResponseBody(response)

    def getInstallmentPayments(self, installment_id: int, options: Optional[RequestOptions] = None) -> Any:
        response = self.http.request(
            "GET",
            f"/v1/payment-installments/{quote(str(installment_id), safe='')}/payments",
            options=options,
        )
        return self.http.parseResponseBody(response)
