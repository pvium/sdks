const test = require("node:test");
const assert = require("node:assert/strict");

const { PviumSdk } = require("../dist/index.js");

function createMockSdk(responseBody = { meta: { statusCode: 200, success: true } }) {
  const requests = [];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    apiKey: "pk_test_dummy",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      return new Response(JSON.stringify(responseBody), {
        status: responseBody.meta?.statusCode || 200,
        headers: { "content-type": "application/json" },
      });
    },
  });

  return { sdk, requests };
}

test("listInvoices calls the invoices list endpoint", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 200, success: true },
    data: [],
  });

  const result = await sdk.endpoints.listInvoices();

  assert.deepEqual(result.data, []);
  assert.equal(requests.length, 1);
  assert.equal(requests[0].url, "http://localhost:4005/v1/invoices");
  assert.equal(requests[0].init.method, "GET");
  assert.equal(requests[0].init.headers["x-api-key"], "pk_test_dummy");
});

test("getInvoiceStatus calls the encoded invoice status endpoint", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 200, success: true },
    data: { contractCode: "INV 123" },
  });

  const result = await sdk.endpoints.getInvoiceStatus("INV 123");

  assert.equal(result.data.contractCode, "INV 123");
  assert.equal(
    requests[0].url,
    "http://localhost:4005/v1/invoices/INV%20123/status",
  );
  assert.equal(requests[0].init.method, "GET");
});

test("cancelInvoice patches the invoice contract inactive", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 200, success: true },
    data: { id: 42232, active: false },
  });

  const result = await sdk.endpoints.cancelInvoice(42232);

  assert.equal(result.data.active, false);
  assert.equal(requests[0].url, "http://localhost:4005/v1/invoices/42232");
  assert.equal(requests[0].init.method, "PATCH");
  assert.deepEqual(JSON.parse(requests[0].init.body), { active: false });
});

test("getInstallmentPayments calls the installment payments endpoint", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 200, success: true },
    data: [],
  });

  const result = await sdk.endpoints.getInstallmentPayments(42232);

  assert.deepEqual(result.data, []);
  assert.equal(
    requests[0].url,
    "http://localhost:4005/v1/payment-installments/42232/payments",
  );
  assert.equal(requests[0].init.method, "GET");
});

test("createInvoice posts JSON to the invoices endpoint", async () => {
  const { sdk, requests } = createMockSdk({
    meta: { statusCode: 201, success: true },
    data: { id: "invoice_123" },
  });
  const payload = {
    name: "SDK Test",
    description: "Integration test invoice",
    amount: 50,
    dueDate: "2026-04-28T00:00:00.000Z",
    amountType: "Flat",
    discount: 0,
    discountType: "Flat",
    tax: 0,
    documentNumber: 123,
  };

  const result = await sdk.endpoints.createInvoice(payload);

  assert.equal(result.data.id, "invoice_123");
  assert.equal(requests[0].url, "http://localhost:4005/v1/invoices");
  assert.equal(requests[0].init.method, "POST");
  assert.deepEqual(JSON.parse(requests[0].init.body), payload);
});
