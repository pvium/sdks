package services

import (
	"context"
	"fmt"

	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

type EndpointsService struct {
	client *transport.HTTPClient
}

func NewEndpointsService(client *transport.HTTPClient) *EndpointsService {
	return &EndpointsService{client: client}
}

func (s *EndpointsService) CreateInvoice(ctx context.Context, body models.CreateInvoiceRequest, options *models.RequestOptions) (models.APIResponse[models.CreateInvoiceData], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: "/invoices", Body: body, Options: options})
	if err != nil {
		return models.APIResponse[models.CreateInvoiceData]{}, err
	}
	return transport.Decode[models.APIResponse[models.CreateInvoiceData]](raw)
}

func (s *EndpointsService) ListInvoices(ctx context.Context, options *models.RequestOptions) (models.APIResponse[[]models.InvoiceListItem], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "GET", Path: "/invoices", Options: options})
	if err != nil {
		return models.APIResponse[[]models.InvoiceListItem]{}, err
	}
	return transport.Decode[models.APIResponse[[]models.InvoiceListItem]](raw)
}

func (s *EndpointsService) GetInvoiceStatus(ctx context.Context, code string, options *models.RequestOptions) (models.APIResponse[models.InvoiceStatusData], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "GET", Path: fmt.Sprintf("/invoices/%s/status", code), Options: options})
	if err != nil {
		return models.APIResponse[models.InvoiceStatusData]{}, err
	}
	return transport.Decode[models.APIResponse[models.InvoiceStatusData]](raw)
}

func (s *EndpointsService) CancelInvoice(ctx context.Context, id string, options *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "PATCH", Path: fmt.Sprintf("/invoices/%s", id), Body: map[string]any{"active": false}, Options: options})
	if err != nil {
		return models.APIResponse[map[string]any]{}, err
	}
	return transport.Decode[models.APIResponse[map[string]any]](raw)
}

func (s *EndpointsService) GetInstallmentPayments(ctx context.Context, installmentID int, options *models.RequestOptions) (models.APIResponse[[]models.InstallmentPayment], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "GET", Path: fmt.Sprintf("/payment-installments/%d/payments", installmentID), Options: options})
	if err != nil {
		return models.APIResponse[[]models.InstallmentPayment]{}, err
	}
	return transport.Decode[models.APIResponse[[]models.InstallmentPayment]](raw)
}
