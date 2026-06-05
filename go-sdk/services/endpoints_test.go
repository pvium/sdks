package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/pvium/sdks/go-sdk/config"
	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

func TestEndpointsListInvoicesCallsInvoicesEndpoint(t *testing.T) {
	t.Parallel()

	var gotPath, gotMethod, gotAPIKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAPIKey = r.Header.Get("x-api-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":[]}`))
	}))
	defer ts.Close()

	service := NewEndpointsService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL, APIKey: "pk_test_dummy"}))
	res, err := service.ListInvoices(context.Background(), nil)
	if err != nil {
		t.Fatalf("list invoices: %v", err)
	}
	if len(res.Data) != 0 {
		t.Fatalf("expected empty list, got %d", len(res.Data))
	}
	if gotPath != "/invoices" || gotMethod != http.MethodGet {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if gotAPIKey != "pk_test_dummy" {
		t.Fatalf("expected api key header")
	}
}

func TestEndpointsGetInvoiceStatusEncodesPath(t *testing.T) {
	t.Parallel()

	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":{"contractCode":"INV 123"}}`))
	}))
	defer ts.Close()

	service := NewEndpointsService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	res, err := service.GetInvoiceStatus(context.Background(), "INV 123", nil)
	if err != nil {
		t.Fatalf("get invoice status: %v", err)
	}
	if res.Data.ContractCode != "INV 123" {
		t.Fatalf("contract code mismatch: %s", res.Data.ContractCode)
	}
	if gotPath != "/invoices/INV%20123/status" {
		t.Fatalf("unexpected escaped path: %s", gotPath)
	}
}

func TestEndpointsCancelInvoiceSendsInactivePatchBody(t *testing.T) {
	t.Parallel()

	var gotPath, gotMethod string
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":{"id":42232,"active":false}}`))
	}))
	defer ts.Close()

	service := NewEndpointsService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	res, err := service.CancelInvoice(context.Background(), "42232", nil)
	if err != nil {
		t.Fatalf("cancel invoice: %v", err)
	}
	if active, ok := res.Data["active"].(bool); !ok || active {
		t.Fatalf("expected active=false in response")
	}
	if gotPath != "/invoices/42232" || gotMethod != http.MethodPatch {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if gotBody["active"] != false {
		t.Fatalf("unexpected patch body: %+v", gotBody)
	}
}

func TestEndpointsGetInstallmentPaymentsCallsEndpoint(t *testing.T) {
	t.Parallel()

	var gotPath, gotMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":[]}`))
	}))
	defer ts.Close()

	service := NewEndpointsService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	res, err := service.GetInstallmentPayments(context.Background(), 42232, nil)
	if err != nil {
		t.Fatalf("get installment payments: %v", err)
	}
	if len(res.Data) != 0 {
		t.Fatalf("expected empty list")
	}
	if gotPath != "/payment-installments/42232/payments" || gotMethod != http.MethodGet {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
}

func TestEndpointsCreateInvoicePostsJSON(t *testing.T) {
	t.Parallel()

	var gotPath, gotMethod string
	var gotBody models.CreateInvoiceRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":201,"success":true},"data":{"id":"invoice_123"}}`))
	}))
	defer ts.Close()

	service := NewEndpointsService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	payload := models.CreateInvoiceRequest{
		"name":           "SDK Test",
		"description":    "Integration test invoice",
		"amount":         50,
		"dueDate":        "2026-04-28T00:00:00.000Z",
		"amountType":     "Flat",
		"discount":       0,
		"discountType":   "Flat",
		"tax":            0,
		"documentNumber": 123,
	}
	res, err := service.CreateInvoice(context.Background(), payload, nil)
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	if res.Data["id"] != "invoice_123" {
		t.Fatalf("response id mismatch: %+v", res.Data)
	}
	if gotPath != "/invoices" || gotMethod != http.MethodPost {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	expected := models.CreateInvoiceRequest{}
	rawPayload, _ := json.Marshal(payload)
	_ = json.Unmarshal(rawPayload, &expected)
	if !reflect.DeepEqual(gotBody, expected) {
		t.Fatalf("payload mismatch: %+v", gotBody)
	}
}
