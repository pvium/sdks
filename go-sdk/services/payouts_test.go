package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/pvium/sdks/go-sdk/config"
	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

func loadPayoutParityFixture(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../parity-fixtures/scheduled-payout-finalize.json")
	if err != nil {
		t.Fatalf("read parity fixture: %v", err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode parity fixture: %v", err)
	}
	return fixture
}

func TestPayoutAddRecipientsPostsOpenPayees(t *testing.T) {
	t.Parallel()

	var gotPath, gotMethod string
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":201,"success":true},"data":{"added":[],"errors":[]}}`))
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	defaultAmount := 25.0
	_, err := service.AddRecipients(context.Background(), "batch 1", []models.PayoutRecipient{{
		IdentityType:        "github",
		IdentityValue:       "@feminefa",
		DefaultPayoutAmount: &defaultAmount,
		Memo:                "github payout",
	}}, nil)
	if err != nil {
		t.Fatalf("add recipients: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/batch-payments/batch%201/open-payees" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	expected := map[string]any{
		"recipients": []any{
			map[string]any{
				"identityType":        "github",
				"identityValue":       "@feminefa",
				"defaultPayoutAmount": float64(25),
				"memo":                "github payout",
			},
		},
	}
	if !reflect.DeepEqual(gotBody, expected) {
		t.Fatalf("unexpected body: got %+v want %+v", gotBody, expected)
	}
}

func TestPayoutResolveRecipientsPostsResolverEndpoint(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":{"resolved":[],"errors":[]}}`))
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	_, err := service.ResolveRecipients(context.Background(), "batch_1", []models.ResolvePayoutRecipient{{IdentityType: "email", IdentityValue: "payee@example.com"}}, nil)
	if err != nil {
		t.Fatalf("resolve recipients: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/batch-payments/batch_1/resolve-recipients" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	expected := map[string]any{
		"recipients": []any{
			map[string]any{"identityType": "email", "identityValue": "payee@example.com"},
		},
	}
	if !reflect.DeepEqual(gotBody, expected) {
		t.Fatalf("unexpected body: got %+v want %+v", gotBody, expected)
	}
}

func TestPayoutRemovePaymentsDeletesPaymentIDs(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":{}}`))
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	_, err := service.RemovePayments(context.Background(), "batch_1", []any{"1", 2}, nil)
	if err != nil {
		t.Fatalf("remove payments: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/batch-payments/batch_1/payments" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	expected := map[string]any{"paymentIds": []any{float64(1), float64(2)}}
	if !reflect.DeepEqual(gotBody, expected) {
		t.Fatalf("unexpected body: got %+v want %+v", gotBody, expected)
	}
}

func TestPayoutListPaymentsRequestsPaginatedEndpoint(t *testing.T) {
	t.Parallel()

	var gotPath, gotMethod string
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true,"pagination":{"totalCount":251,"perPage":50}},"data":[{"id":77}]}`))
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	res, err := service.ListPayments(context.Background(), "batch_1", &models.PayoutPaymentsListQuery{Page: 2, PerPage: 50}, nil)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/batch-payments/batch_1/payments" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if gotQuery != "page=2&perPage=50" {
		t.Fatalf("unexpected query: %s", gotQuery)
	}
	if res.Meta.Pagination == nil || res.Meta.Pagination.TotalCount != 251 {
		t.Fatalf("pagination missing: %+v", res.Meta.Pagination)
	}
}

func TestPayoutIntentProxiesPaymentManagementRoutes(t *testing.T) {
	t.Parallel()

	var requests []struct {
		Method string
		Path   string
		Query  string
		Body   map[string]any
	}
	responses := []string{
		`{"meta":{"statusCode":200,"success":true},"data":[{"id":77}]}`,
		`{"meta":{"statusCode":200,"success":true},"data":{"id":77,"memo":"Updated"}}`,
		`{"meta":{"statusCode":200,"success":true},"data":{}}`,
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entry := struct {
			Method string
			Path   string
			Query  string
			Body   map[string]any
		}{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery}
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&entry.Body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		}
		requests = append(requests, entry)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[0]))
		responses = responses[1:]
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	intent := &PayoutIntent{Data: models.PayoutRecord{ID: "batch_1"}, PayoutRecord: models.PayoutRecord{ID: "batch_1"}, service: service}

	if _, err := intent.ListPayments(context.Background(), &models.PayoutPaymentsListQuery{Page: 1, PerPage: 25}, nil); err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if _, err := intent.EditPayment(context.Background(), 77, models.UpdatePayoutPaymentInput{Memo: "Updated"}, nil); err != nil {
		t.Fatalf("edit payment: %v", err)
	}
	if _, err := intent.DeletePayment(context.Background(), 77, nil); err != nil {
		t.Fatalf("delete payment: %v", err)
	}

	if requests[0].Method != http.MethodGet || requests[0].Path != "/batch-payments/batch_1/payments" || requests[0].Query != "page=1&perPage=25" {
		t.Fatalf("unexpected list request: %+v", requests[0])
	}
	if requests[1].Method != http.MethodPatch || requests[1].Path != "/batch-payments/batch_1/payments/77" {
		t.Fatalf("unexpected edit request: %+v", requests[1])
	}
	if !reflect.DeepEqual(requests[1].Body, map[string]any{"memo": "Updated"}) {
		t.Fatalf("unexpected edit body: %+v", requests[1].Body)
	}
	if requests[2].Method != http.MethodDelete || requests[2].Path != "/batch-payments/batch_1/payments" {
		t.Fatalf("unexpected delete request: %+v", requests[2])
	}
	if !reflect.DeepEqual(requests[2].Body, map[string]any{"paymentIds": []any{float64(77)}}) {
		t.Fatalf("unexpected delete body: %+v", requests[2].Body)
	}
}

func TestPayoutCreateReturnsIntentWithProxyMethods(t *testing.T) {
	t.Parallel()

	responses := []string{
		`{"meta":{"statusCode":201,"success":true},"data":{"id":"batch_1","chain":"base","paymentType":"Instant","nonce":"0x11111111111111111111111111111111","complianceMode":"Open","payments":[{"receiver":"0x0000000000000000000000000000000000000001","amount":25,"token":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","decimals":6}]}}`,
		`{"meta":{"statusCode":200,"success":true},"data":{"id":"batch_1","chain":"base","paymentType":"Instant","nonce":"0x11111111111111111111111111111111","batchDataHash":"0x2222222222222222222222222222222222222222222222222222222222222222"}}`,
	}
	var methods []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[0]))
		responses = responses[1:]
	}))
	defer ts.Close()

	decimals := 6
	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL, ClientID: "app_test"}))
	payoutIntent, err := service.Create(context.Background(), models.CreatePayoutInput{
		Type:  models.PayoutTypeInstant,
		Chain: "base",
		Name:  "Creator payroll",
		Payments: []models.PayoutPayment{{
			Receiver: "0x0000000000000000000000000000000000000001",
			Amount:   25,
			Token:    "usdc",
			Decimals: &decimals,
		}},
	}, nil)
	if err != nil {
		t.Fatalf("create payout intent: %v", err)
	}
	if payoutIntent.ID != "batch_1" || payoutIntent.Data.ID != "batch_1" {
		t.Fatalf("intent fields mismatch: %+v", payoutIntent)
	}

	finalized, err := payoutIntent.Finalize(
		context.Background(),
		models.PayoutSignerInput{PrivateKey: "0x59c6995e998f97a5a004497e5daaaa853d873599e62e568a0a7d3a57c5fd8d0d"},
		models.PayoutFinalizeOptions{Timestamp: 1777487451},
		nil,
	)
	if err != nil {
		t.Fatalf("finalize payout intent: %v", err)
	}
	if finalized.Payout.ID != "batch_1" || finalized.Data.Payout.ID != "batch_1" {
		t.Fatalf("finalization fields mismatch: %+v", finalized)
	}
	if !reflect.DeepEqual(methods, []string{http.MethodPost, http.MethodPatch}) {
		t.Fatalf("methods mismatch: %+v", methods)
	}
}

func TestPayoutCreatePostsNodeCompatibleBody(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"meta":{"statusCode":201,"success":true},"data":{"id":"batch_1","chain":"base","paymentType":"Scheduled"}}`))
	}))
	defer ts.Close()

	lockDuration := int64(3600)
	decimals := 6
	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	_, err := service.Create(context.Background(), models.CreatePayoutInput{
		ID:           "batch_1",
		Type:         models.PayoutTypeScheduled,
		Chain:        "base",
		Nonce:        "0x1234",
		LockDuration: &lockDuration,
		Name:         "Creator payout",
		Description:  "SDK parity test",
		Metadata: map[string]any{
			"payoutCurrency": "0x0000000000000000000000000000000000000002",
		},
		Payments: []models.PayoutPayment{{
			Receiver: "0x0000000000000000000000000000000000000001",
			Amount:   "25",
			Decimals: &decimals,
			Memo:     "escrow work",
		}},
	}, nil)
	if err != nil {
		t.Fatalf("create payout: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/batch-payments" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if gotBody["paymentType"] != "Scheduled" || gotBody["complianceMode"] != "Open" || gotBody["label"] != "Creator payout" {
		t.Fatalf("unexpected create body: %+v", gotBody)
	}
	payments, ok := gotBody["payments"].([]any)
	if !ok || len(payments) != 1 {
		t.Fatalf("unexpected payments: %+v", gotBody["payments"])
	}
	payment := payments[0].(map[string]any)
	if payment["amount"] != float64(25) || payment["token"] != "0x0000000000000000000000000000000000000002" {
		t.Fatalf("payment was not normalized like node: %+v", payment)
	}
}

func TestPayoutCreateMapsDirectCurrencyAndRejectsMismatchedPaymentToken(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":201,"success":true},"data":{"id":"batch_1","chain":"base","paymentType":"Scheduled"}}`))
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	_, err := service.Create(context.Background(), models.CreatePayoutInput{
		Type:           models.PayoutTypeScheduled,
		Chain:          "base",
		PayoutCurrency: string(models.PayoutCurrencyUSDC),
		ScheduleDate:   int64(1777488000),
		Payments: []models.PayoutPayment{{
			Receiver: "0x0000000000000000000000000000000000000001",
			Amount:   "25",
		}},
	}, nil)
	if err != nil {
		t.Fatalf("create payout: %v", err)
	}
	metadata := gotBody["metadata"].(map[string]any)
	if metadata["payoutCurrency"] != "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913" || metadata["scheduledDate"] != float64(1777488000) {
		t.Fatalf("metadata mismatch: %+v", metadata)
	}
	payment := gotBody["payments"].([]any)[0].(map[string]any)
	if payment["token"] != "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913" || payment["decimals"] != float64(6) {
		t.Fatalf("payment mismatch: %+v", payment)
	}

	_, err = service.Create(context.Background(), models.CreatePayoutInput{
		Type:           models.PayoutTypeScheduled,
		Chain:          "base",
		PayoutCurrency: string(models.PayoutCurrencyUSDC),
		Payments: []models.PayoutPayment{{
			Receiver: "0x0000000000000000000000000000000000000001",
			Amount:   "25",
			Token:    "usdt",
		}},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "payment token must match payoutCurrency") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestPayoutCreateInjectsMilestoneCommitmentMetadata(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"statusCode":201,"success":true},"data":{"id":"batch_1","chain":"base","paymentType":"Scheduled","isCommitment":true}}`))
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	_, err := service.Create(context.Background(), models.CreatePayoutInput{
		Type:           models.PayoutTypeMilestone,
		Chain:          "base",
		Name:           "Website build",
		PayoutCurrency: "usdc",
		Metadata: map[string]any{
			"milestones": []map[string]any{{
				"name":    "Design approval",
				"amount":  500,
				"dueDate": "2026-07-01T00:00:00.000Z",
				"status":  "pending",
			}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("create milestone: %v", err)
	}
	if gotBody["paymentType"] != string(models.PayoutTypeScheduled) || gotBody["isCommitment"] != true {
		t.Fatalf("milestone mapping mismatch: %+v", gotBody)
	}
	metadata := gotBody["metadata"].(map[string]any)
	if metadata["commitmentType"] != "milestone" {
		t.Fatalf("commitment metadata missing: %+v", metadata)
	}
}

func TestPayoutFinalizeInstantWithMessageSigner(t *testing.T) {
	t.Parallel()

	var requests []struct {
		Method string
		Path   string
		Body   map[string]any
	}
	decimals := 6
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured := struct {
			Method string
			Path   string
			Body   map[string]any
		}{Method: r.Method, Path: r.URL.Path}
		if r.Body != nil && r.Method == http.MethodPatch {
			if err := json.NewDecoder(r.Body).Decode(&captured.Body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		}
		requests = append(requests, captured)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"meta": map[string]any{"statusCode": 200, "success": true},
				"data": map[string]any{
					"id":             "120bdabb-5790-415c-ae75-c2fca1cc5232",
					"chain":          "base",
					"paymentType":    "Instant",
					"complianceMode": "Open",
					"nonce":          "0x1234",
					"app":            map[string]any{"clientId": "app_test"},
					"payments": []any{map[string]any{
						"receiver": "0x0000000000000000000000000000000000000001",
						"amount":   "1",
						"token":    "0x0000000000000000000000000000000000000002",
						"decimals": decimals,
						"memo":     "",
					}},
				},
			})
			return
		}
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":{"id":"120bdabb-5790-415c-ae75-c2fca1cc5232","chain":"base","paymentType":"Instant","batchDataHash":"0xabc"}}`))
	}))
	defer ts.Close()

	signedMessages := []string{}
	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	_, err := service.Finalize(
		context.Background(),
		"120bdabb-5790-415c-ae75-c2fca1cc5232",
		models.PayoutSignerInput{
			SignerAddress: "0x0000000000000000000000000000000000000003",
			SignMessage: func(message string) (string, error) {
				signedMessages = append(signedMessages, message)
				return "0xsigned", nil
			},
		},
		models.PayoutFinalizeOptions{SignerAddress: "0x0000000000000000000000000000000000000003", Timestamp: 123},
		nil,
	)
	if err != nil {
		t.Fatalf("finalize instant: %v", err)
	}
	if len(signedMessages) != 1 || !strings.HasPrefix(signedMessages[0], "PVIUM_SIGNED_BATCH:app_test:") {
		t.Fatalf("unexpected signed messages: %+v", signedMessages)
	}
	if requests[1].Method != http.MethodPatch {
		t.Fatalf("expected patch, got %+v", requests)
	}
	payload := requests[1].Body
	if payload["signer"] != "0x0000000000000000000000000000000000000003" {
		t.Fatalf("signer mismatch: %+v", payload)
	}
	if payload["batchSignature"] != "123:0x0000000000000000000000000000000000000003:0xsigned" {
		t.Fatalf("batch signature mismatch: %+v", payload)
	}
}

func TestPayoutFinalizeScheduledSolanaSkipsFundingSignature(t *testing.T) {
	t.Parallel()

	var patchBody map[string]any
	decimals := 6
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"meta": map[string]any{"statusCode": 200, "success": true},
				"data": scheduledPayoutFixture(decimals),
			})
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&patchBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":{"id":"batch_1","chain":"solana","paymentType":"Scheduled","merkleRoot":"0xabc"}}`))
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	_, err := service.Finalize(
		context.Background(),
		"120bdabb-5790-415c-ae75-c2fca1cc5232",
		models.PayoutSignerInput{
			SignerAddress: "0x0000000000000000000000000000000000000003",
			SignMessage:   func(string) (string, error) { return "signed", nil },
		},
		models.PayoutFinalizeOptions{
			Chain:         "solana",
			ChainID:       1,
			SignerAddress: "0x0000000000000000000000000000000000000003",
			Timestamp:     123,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("finalize scheduled solana: %v", err)
	}
	if _, ok := patchBody["fundingSignature"]; ok {
		t.Fatalf("funding signature should be omitted for solana: %+v", patchBody)
	}
	if !strings.HasSuffix(patchBody["batchSignature"].(string), ":signed") {
		t.Fatalf("batch signature mismatch: %+v", patchBody)
	}
}

func TestPayoutFinalizeScheduledSupportsSeparateSigners(t *testing.T) {
	t.Parallel()

	var patchBody map[string]any
	calls := []string{}
	decimals := 6
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"meta": map[string]any{"statusCode": 200, "success": true},
				"data": scheduledPayoutFixture(decimals),
			})
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&patchBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":{"id":"120bdabb-5790-415c-ae75-c2fca1cc5232","chain":"base","paymentType":"Scheduled","merkleRoot":"0xabc"}}`))
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	_, err := service.Finalize(
		context.Background(),
		"120bdabb-5790-415c-ae75-c2fca1cc5232",
		models.PayoutSignerInput{
			Chain:         "ethereum",
			SignerAddress: "0x0000000000000000000000000000000000000003",
			SignMessage: func(string) (string, error) {
				t.Fatal("fallback signMessage should not be called")
				return "", nil
			},
			SignFinalize: func(message string) (string, error) {
				calls = append(calls, "finalize:"+message)
				return "finalize-signature", nil
			},
			SignFunding: func(digest string) (string, error) {
				calls = append(calls, "funding:"+digest)
				return "funding-signature", nil
			},
		},
		models.PayoutFinalizeOptions{Chain: "base", ChainID: 8453, Timestamp: 123},
		nil,
	)
	if err != nil {
		t.Fatalf("finalize scheduled separate signers: %v", err)
	}
	if len(calls) != 2 || !strings.HasPrefix(calls[0], "finalize:PVIUM_SIGNED_SCHEDULE:") || !strings.HasPrefix(calls[1], "funding:0x") {
		t.Fatalf("unexpected calls: %+v", calls)
	}
	if !strings.HasSuffix(patchBody["batchSignature"].(string), ":finalize-signature") {
		t.Fatalf("batch signature mismatch: %+v", patchBody)
	}
	if patchBody["fundingSignature"] != "funding-signature" {
		t.Fatalf("funding signature mismatch: %+v", patchBody)
	}
}

func TestPayoutAddPaymentsCreatesSignedEscrowChildScheduledPayout(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"meta":{"statusCode":201,"success":true},"data":{"id":"22222222-2222-4222-8222-222222222222","chain":"base","paymentType":"Scheduled","escrowBatch":"7a6ca76d-77f7-4c0e-9da9-c64f1cb18a1f","merkleRoot":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`))
	}))
	defer ts.Close()

	privateKey := "0x59c6995e998f97a5a004497e5daaaa853d873599e62e568a0a7d3a57c5fd8d0d"
	escrowBatch := models.PayoutRecord{
		ID:             "7a6ca76d-77f7-4c0e-9da9-c64f1cb18a1f",
		Chain:          "base",
		PaymentType:    models.PayoutTypeEscrow,
		Status:         "funded",
		ComplianceMode: models.PayoutComplianceOpen,
		Name:           "Creator escrow",
		BatchHash:      "0x1111111111111111111111111111111111111111111111111111111111111111",
		Metadata: map[string]any{
			"payoutCurrency": "0x0000000000000000000000000000000000000002",
		},
		App: map[string]any{"clientId": "app_test"},
	}
	decimals := 6
	claimDate := int64(1777488000)
	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	escrowIntent := &PayoutIntent{
		Meta:         models.APIMeta{StatusCode: 200, Success: true},
		Data:         escrowBatch,
		PayoutRecord: escrowBatch,
		service:      service,
	}
	_, err := escrowIntent.AddPayments(context.Background(), models.AddPayoutPaymentsInput{
		Payments: []models.PayoutPayment{{
			Receiver: "0x0000000000000000000000000000000000000001",
			Amount:   "25",
			Decimals: &decimals,
			Memo:     "escrow work",
		}},
		Signer: &models.PayoutSignerInput{PrivateKey: privateKey},
		FinalizeOptions: &models.PayoutFinalizeOptions{
			ID:        "22222222-2222-4222-8222-222222222222",
			ChainID:   8453,
			Timestamp: 1777487451,
			ClaimDate: claimDate,
		},
	}, nil)
	if err != nil {
		t.Fatalf("add escrow payments: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/batch-payments" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if gotBody["id"] != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("id mismatch: %+v", gotBody)
	}
	if gotBody["paymentType"] != "Scheduled" || gotBody["escrowBatch"] != escrowBatch.ID {
		t.Fatalf("payout type/escrow mismatch: %+v", gotBody)
	}
	if !regexp.MustCompile(`^1777487451:0x[0-9a-f]{40}:0x`).MatchString(gotBody["batchSignature"].(string)) {
		t.Fatalf("batch signature mismatch: %+v", gotBody)
	}
	for _, key := range []string{"fundingSignature", "batchHash", "batchDataHash", "merkleRoot"} {
		value, ok := gotBody[key].(string)
		if !ok || !strings.HasPrefix(value, "0x") {
			t.Fatalf("%s mismatch: %+v", key, gotBody)
		}
	}
	proofs := gotBody["proofs"].([]any)
	if proofs[0].(map[string]any)["receiver"] != "0x0000000000000000000000000000000000000001" {
		t.Fatalf("proof receiver mismatch: %+v", proofs)
	}
	metadata := gotBody["metadata"].(map[string]any)
	if metadata["escrowBatch"] != escrowBatch.ID || metadata["escrowBatchHash"] != escrowBatch.BatchHash || metadata["scheduledDate"] != float64(claimDate) {
		t.Fatalf("metadata mismatch: %+v", metadata)
	}
	payments := gotBody["payments"].([]any)
	payment := payments[0].(map[string]any)
	if payment["claimDate"] != float64(claimDate) || payment["token"] != "0x0000000000000000000000000000000000000002" {
		t.Fatalf("payment mismatch: %+v", payment)
	}
}

func TestPayoutAddPaymentsRejectsEscrowWithoutSigner(t *testing.T) {
	t.Parallel()

	decimals := 6
	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: "https://api.example.test"}))
	_, err := service.AddPayments(context.Background(), models.PayoutRecord{
		ID:          "7a6ca76d-77f7-4c0e-9da9-c64f1cb18a1f",
		Chain:       "base",
		PaymentType: models.PayoutTypeEscrow,
		Status:      "funded",
		BatchHash:   "0x1111111111111111111111111111111111111111111111111111111111111111",
		Metadata: map[string]any{
			"payoutCurrency": "0x0000000000000000000000000000000000000002",
		},
		App: map[string]any{"clientId": "app_test"},
	}, []models.PayoutPayment{{
		Receiver: "0x0000000000000000000000000000000000000001",
		Amount:   "25",
		Decimals: &decimals,
	}}, nil)
	if err == nil || !strings.Contains(err.Error(), "signer or private key is required") {
		t.Fatalf("expected signer error, got %v", err)
	}
}

func TestScheduledPayoutMerkleRootsAndProofsMatchCanonicalParityValues(t *testing.T) {
	t.Parallel()

	decimals := 6
	basePayments := []models.PayoutPayment{
		{
			Receiver: "0x0000000000000000000000000000000000000001",
			Amount:   "1",
			Token:    "0x0000000000000000000000000000000000000002",
			Decimals: &decimals,
			Memo:     "a",
		},
		{
			Receiver: "0x0000000000000000000000000000000000000003",
			Amount:   "2",
			Token:    "0x0000000000000000000000000000000000000002",
			Decimals: &decimals,
			Memo:     "b",
		},
		{
			Receiver: "0x0000000000000000000000000000000000000004",
			Amount:   "3",
			Token:    "0x0000000000000000000000000000000000000002",
			Decimals: &decimals,
			Memo:     "c",
		},
	}

	cases := []struct {
		name       string
		count      int
		merkleRoot string
		proofs     [][]string
	}{
		{
			name:       "single leaf",
			count:      1,
			merkleRoot: "0xc40e60ab1b114ed2eb4bbe156b73023238565bf37b45766c3c922b8089d1d2e9",
			proofs:     [][]string{{}},
		},
		{
			name:       "even leaves",
			count:      2,
			merkleRoot: "0x7ade83cae70b4278f73144e9a99f77f7deeed719e4778245b9f3db71f6ae02b7",
			proofs: [][]string{
				{"0xdeaa9357ef2c59449293ed3ba3060e6aa9cf4bf4812ecc96f2b3a6500744f05b"},
				{"0xc40e60ab1b114ed2eb4bbe156b73023238565bf37b45766c3c922b8089d1d2e9"},
			},
		},
		{
			name:       "odd leaves",
			count:      3,
			merkleRoot: "0xf9eb35f3a7cf94793c4f0d440f9c510828d516e6ac308141819ab7265754a8d6",
			proofs: [][]string{
				{
					"0xdeaa9357ef2c59449293ed3ba3060e6aa9cf4bf4812ecc96f2b3a6500744f05b",
					"0x54acdb6206c4f539cbcda16b5132e71254f0ae9126b4efaa442f5c61659d544c",
				},
				{
					"0xc40e60ab1b114ed2eb4bbe156b73023238565bf37b45766c3c922b8089d1d2e9",
					"0x54acdb6206c4f539cbcda16b5132e71254f0ae9126b4efaa442f5c61659d544c",
				},
				{"0x7ade83cae70b4278f73144e9a99f77f7deeed719e4778245b9f3db71f6ae02b7"},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var patchBody map[string]any
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewDecoder(r.Body).Decode(&patchBody); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				_, _ = w.Write([]byte(`{"meta":{"statusCode":200,"success":true},"data":{"id":"x","paymentType":"Scheduled"}}`))
			}))
			defer ts.Close()

			service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
			_, err := service.Finalize(
				context.Background(),
				models.PayoutRecord{
					ID:             "120bdabb-5790-415c-ae75-c2fca1cc5232",
					Chain:          "base",
					PaymentType:    models.PayoutTypeScheduled,
					ComplianceMode: models.PayoutComplianceOpen,
					Metadata: map[string]any{
						"payoutCurrency":      "0x0000000000000000000000000000000000000002",
						"gracePeriod":         0,
						"disapprovalDeadline": 0,
						"scheduledDate":       0,
					},
					App:      map[string]any{"clientId": "app_test"},
					Payments: basePayments[:tc.count],
				},
				models.PayoutSignerInput{
					SignerAddress: "0x0000000000000000000000000000000000000005",
					SignFinalize:  func(string) (string, error) { return "finalize", nil },
					SignFunding:   func(string) (string, error) { return "funding", nil },
				},
				models.PayoutFinalizeOptions{Chain: "base", Timestamp: 123, ClaimDate: 1777488000},
				nil,
			)
			if err != nil {
				t.Fatalf("finalize scheduled: %v", err)
			}

			if patchBody["merkleRoot"] != tc.merkleRoot {
				t.Fatalf("merkle root mismatch: got %s want %s", patchBody["merkleRoot"], tc.merkleRoot)
			}
			proofs := patchBody["proofs"].([]any)
			if len(proofs) != len(tc.proofs) {
				t.Fatalf("proof count mismatch: got %d want %d", len(proofs), len(tc.proofs))
			}
			for i, proofAny := range proofs {
				proof := proofAny.(map[string]any)["proof"].([]any)
				gotProof := make([]string, 0, len(proof))
				for _, item := range proof {
					gotProof = append(gotProof, item.(string))
				}
				if !reflect.DeepEqual(gotProof, tc.proofs[i]) {
					t.Fatalf("proof %d mismatch: got %+v want %+v", i, gotProof, tc.proofs[i])
				}
			}
		})
	}
}

func TestScheduledPayoutFinalizationSignaturesMatchCanonicalParityValues(t *testing.T) {
	t.Parallel()

	fixture := loadPayoutParityFixture(t)
	options := fixture["options"].(map[string]any)
	var patchBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(fixture["getResponse"])
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&patchBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(fixture["patchResponse"])
	}))
	defer ts.Close()

	service := NewPayoutService(transport.NewHTTPClient(config.Config{BaseURL: ts.URL}))
	result, err := service.Finalize(
		context.Background(),
		fixture["payoutId"].(string),
		models.PayoutSignerInput{PrivateKey: fixture["privateKey"].(string)},
		models.PayoutFinalizeOptions{
			Chain:     options["chain"].(string),
			Timestamp: int64(options["timestamp"].(float64)),
			ClaimDate: int64(options["claimDate"].(float64)),
		},
		nil,
	)
	if err != nil {
		t.Fatalf("finalize scheduled: %v", err)
	}

	if !reflect.DeepEqual(patchBody, fixture["expectedPatchPayload"]) {
		t.Fatalf("patch payload mismatch:\ngot  %+v\nwant %+v", patchBody, fixture["expectedPatchPayload"])
	}
	expectedResult := fixture["expectedResult"].(map[string]any)
	expectedPayout := expectedResult["payout"].(map[string]any)
	if result.Data.Payout.ID != expectedPayout["id"] ||
		string(result.Data.Payout.PaymentType) != expectedPayout["paymentType"] ||
		result.Data.Payout.MerkleRoot != expectedPayout["merkleRoot"] {
		t.Fatalf("response payout mismatch: %+v", result.Data.Payout)
	}
	if result.Data.FundingURL != expectedResult["fundingUrl"] ||
		result.Data.BatchDataHash != expectedResult["batchDataHash"] ||
		result.Data.BatchHash != expectedResult["batchHash"] ||
		result.Data.MerkleRoot != expectedResult["merkleRoot"] {
		t.Fatalf("response metadata mismatch: %+v", result.Data)
	}
}

func scheduledPayoutFixture(decimals int) map[string]any {
	return map[string]any{
		"id":             "120bdabb-5790-415c-ae75-c2fca1cc5232",
		"chain":          "base",
		"paymentType":    "Scheduled",
		"complianceMode": "Open",
		"metadata": map[string]any{
			"payoutCurrency":      "0x0000000000000000000000000000000000000002",
			"gracePeriod":         0,
			"disapprovalDeadline": 0,
			"scheduledDate":       0,
		},
		"app": map[string]any{"clientId": "app_test"},
		"payments": []any{map[string]any{
			"receiver": "0x0000000000000000000000000000000000000001",
			"amount":   "1",
			"token":    "0x0000000000000000000000000000000000000002",
			"decimals": decimals,
			"memo":     "",
		}},
	}
}
