import { PviumHttpClient } from "./client";
import {
  CreateInvoiceRequest,
  RequestOptions,
  CancelInvoiceResponse,
  CreateInvoiceResponse,
  ListInvoicesResponse,
  InvoiceStatusResponse,
  InstallmentPaymentsResponse,
} from "./types";

export class PviumEndpoints {
  constructor(private readonly http: PviumHttpClient) {}

  async createInvoice(
    body: CreateInvoiceRequest,
    options?: RequestOptions,
  ): Promise<CreateInvoiceResponse> {
    const response = await this.http.request({
      method: "POST",
      path: "/v1/invoices",
      body,
      options,
    });
    return this.http.parseResponseBody<CreateInvoiceResponse>(response);
  }

  async listInvoices(options?: RequestOptions): Promise<ListInvoicesResponse> {
    const response = await this.http.request({
      method: "GET",
      path: "/v1/invoices",
      options,
    });
    return this.http.parseResponseBody<ListInvoicesResponse>(response);
  }

  async getInvoiceStatus(
    code: string,
    options?: RequestOptions,
  ): Promise<InvoiceStatusResponse> {
    const response = await this.http.request({
      method: "GET",
      path: `/v1/invoices/${encodeURIComponent(code)}/status`,
      options,
    });

    return this.http.parseResponseBody<InvoiceStatusResponse>(response);
  }

  async cancelInvoice(
    id: string | number,
    options?: RequestOptions,
  ): Promise<CancelInvoiceResponse> {
    const response = await this.http.request({
      method: "PATCH",
      path: `/v1/invoices/${encodeURIComponent(String(id))}`,
      body: { active: false },
      options,
    });

    return this.http.parseResponseBody<CancelInvoiceResponse>(response);
  }

  async getInstallmentPayments(
    id: number,
    options?: RequestOptions,
  ): Promise<InstallmentPaymentsResponse> {
    const response = await this.http.request({
      method: "GET",
      path: `/v1/payment-installments/${encodeURIComponent(String(id))}/payments`,
      options,
    });

    return this.http.parseResponseBody<InstallmentPaymentsResponse>(response);
  }
}
