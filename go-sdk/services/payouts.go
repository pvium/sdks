package services

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	pvcrypto "github.com/pvium/sdks/go-sdk/crypto"
	"github.com/pvium/sdks/go-sdk/models"
	"github.com/pvium/sdks/go-sdk/transport"
)

type PayoutService struct {
	client *transport.HTTPClient
}

type PayoutIntent struct {
	Meta models.APIMeta
	Data models.PayoutRecord
	models.PayoutRecord

	service *PayoutService
}

type PayoutFinalization struct {
	Meta models.APIMeta
	Data models.FinalizePayoutData
	models.FinalizePayoutData

	Payout *PayoutIntent
}

func (p *PayoutIntent) Finalize(ctx context.Context, signer models.PayoutSignerInput, options models.PayoutFinalizeOptions, requestOptions *models.RequestOptions) (*PayoutFinalization, error) {
	if p == nil || p.service == nil {
		return nil, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.Finalize(ctx, p.Data, signer, options, requestOptions)
}

func (p *PayoutIntent) AddPayments(ctx context.Context, input any, options *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[map[string]any]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.AddPayments(ctx, p.Data, input, options)
}

func (p *PayoutIntent) AddRecipients(ctx context.Context, recipients []models.PayoutRecipient, options *models.RequestOptions) (models.APIResponse[models.AddPayoutRecipientsResult], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[models.AddPayoutRecipientsResult]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.AddRecipients(ctx, p.ID, recipients, options)
}

func (p *PayoutIntent) ResolveRecipients(ctx context.Context, recipients []models.ResolvePayoutRecipient, options *models.RequestOptions) (models.APIResponse[models.ResolvePayoutRecipientsResult], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[models.ResolvePayoutRecipientsResult]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.ResolveRecipients(ctx, p.ID, recipients, options)
}

func (p *PayoutIntent) RemovePayments(ctx context.Context, paymentIDs any, options *models.RequestOptions) (models.APIResponse[any], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[any]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.RemovePayments(ctx, p.ID, paymentIDs, options)
}

func (p *PayoutIntent) DeletePayment(ctx context.Context, paymentID any, options *models.RequestOptions) (models.APIResponse[any], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[any]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.DeletePayment(ctx, p.ID, paymentID, options)
}

func (p *PayoutIntent) UpdatePayment(ctx context.Context, paymentID any, input models.UpdatePayoutPaymentInput, options *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[map[string]any]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.UpdatePayment(ctx, p.ID, paymentID, input, options)
}

func (p *PayoutIntent) EditPayment(ctx context.Context, paymentID any, input models.UpdatePayoutPaymentInput, options *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	return p.UpdatePayment(ctx, paymentID, input, options)
}

func (p *PayoutIntent) ListPayments(ctx context.Context, query *models.PayoutPaymentsListQuery, options *models.RequestOptions) (models.APIResponse[[]map[string]any], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[[]map[string]any]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.ListPayments(ctx, p.ID, query, options)
}

func (p *PayoutIntent) ListInvites(ctx context.Context, options *models.RequestOptions) (models.APIResponse[[]map[string]any], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[[]map[string]any]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.ListInvites(ctx, p.ID, options)
}

func (p *PayoutIntent) RevokeInvite(ctx context.Context, inviteID any, options *models.RequestOptions) (models.APIResponse[any], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[any]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.RevokeInvite(ctx, p.ID, inviteID, options)
}

func (p *PayoutIntent) RevokeInviteRoot(ctx context.Context, inviteRootID any, options *models.RequestOptions) (models.APIResponse[any], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[any]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.RevokeInviteRoot(ctx, p.ID, inviteRootID, options)
}

func (p *PayoutIntent) Delete(ctx context.Context, options *models.RequestOptions) (models.APIResponse[any], error) {
	if p == nil || p.service == nil {
		return models.APIResponse[any]{}, errors.New("payout intent is not bound to a payout service")
	}
	return p.service.Delete(ctx, p.ID, options)
}

type supportedPayoutCurrency struct {
	ContractAddress string
	Decimals        int
}

var stablecoinTokenAddresses = map[string]map[string]supportedPayoutCurrency{
	"base": {
		"USDC": {ContractAddress: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", Decimals: 6},
		"USDT": {ContractAddress: "0xfde4C96c8593536E31F229EA8f37b2ADa2699bb2", Decimals: 6},
	},
	"bsc": {
		"USDT": {ContractAddress: "0x55d398326f99059fF775485246999027B3197955", Decimals: 18},
		"USDC": {ContractAddress: "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", Decimals: 18},
	},
	"solana": {
		"USDC": {ContractAddress: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", Decimals: 6},
		"USDT": {ContractAddress: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", Decimals: 6},
	},
	"base-testnet": {
		"USDC": {ContractAddress: "0x7dCEd3bFcC97948a665BB665a5D7eEfdfce39C3A", Decimals: 18},
		"USDT": {ContractAddress: "0x9d0C28036AC12d2150a23DE40Bc4A92f7Aa1A79E", Decimals: 18},
	},
	"solana-testnet": {
		"USDC": {ContractAddress: "CmBGSxKZtv22ZiVpKGMP1oMPZfc5rsgr3pEGBDRcjiAy", Decimals: 6},
		"USDT": {ContractAddress: "SPFPKg9zeE7ReqW3j9QU6p7XhPP8JDU5Dx4fgrTwVyF", Decimals: 6},
	},
	"localhost": {
		"USDT": {ContractAddress: "0x5FbDB2315678afecb367f032d93F642f64180aa3", Decimals: 18},
	},
}

var payoutChainAliases = map[string]string{
	"8453": "base", "base": "base", "basemainnet": "base", "base-mainnet": "base",
	"56": "bsc", "bsc": "bsc", "binance": "bsc", "binancesmartchain": "bsc", "binance-smart-chain": "bsc",
	"101": "solana", "solana": "solana",
	"84532": "base-testnet", "basetestnet": "base-testnet", "base-testnet": "base-testnet", "basesepolia": "base-testnet", "base-sepolia": "base-testnet",
	"1012": "solana-testnet", "solanatestnet": "solana-testnet", "solana-testnet": "solana-testnet",
	"31337": "localhost", "localhost": "localhost",
}

var payoutChainIDs = map[string]uint64{
	"base": 8453, "bsc": 56, "solana": 101, "base-testnet": 84532, "solana-testnet": 1012, "localhost": 31337,
}

func NewPayoutService(client *transport.HTTPClient) *PayoutService {
	return &PayoutService{client: client}
}

func (s *PayoutService) Create(ctx context.Context, input models.CreatePayoutInput, options *models.RequestOptions) (*PayoutIntent, error) {
	body, err := buildCreatePayoutBody(input)
	if err != nil {
		return nil, err
	}
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: "/batch-payments", Body: body, Options: options})
	if err != nil {
		return nil, err
	}
	res, err := transport.Decode[models.APIResponse[models.PayoutRecord]](raw)
	if err != nil {
		return nil, err
	}
	return s.wrapPayoutResponse(res), nil
}

func (s *PayoutService) List(ctx context.Context, query *models.PayoutListQuery, options *models.RequestOptions) (models.APIResponse[[]models.PayoutRecord], error) {
	q := map[string]string{}
	if query != nil {
		if query.Page > 0 {
			q["page"] = strconv.Itoa(query.Page)
		}
		if query.Limit > 0 {
			q["limit"] = strconv.Itoa(query.Limit)
		}
		if query.PaymentType != "" {
			q["paymentType"] = query.PaymentType
		}
		if query.IsCommitment != nil {
			q["isCommitment"] = strconv.FormatBool(*query.IsCommitment)
		}
		if query.Status != "" {
			q["status"] = query.Status
		}
	}
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "GET", Path: "/batch-payments", Query: q, Options: options})
	if err != nil {
		return models.APIResponse[[]models.PayoutRecord]{}, err
	}
	return transport.Decode[models.APIResponse[[]models.PayoutRecord]](raw)
}

func (s *PayoutService) Get(ctx context.Context, payoutID string, options *models.RequestOptions) (*PayoutIntent, error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "GET", Path: fmt.Sprintf("/batch-payments/%s", payoutID), Options: options})
	if err != nil {
		return nil, err
	}
	res, err := transport.Decode[models.APIResponse[models.PayoutRecord]](raw)
	if err != nil {
		return nil, err
	}
	return s.wrapPayoutResponse(res), nil
}

func (s *PayoutService) Finalize(ctx context.Context, payout any, args ...any) (*PayoutFinalization, error) {
	if len(args) == 2 {
		if payoutID, ok := payout.(string); ok {
			if payload, ok := args[0].(map[string]any); ok {
				options, _ := args[1].(*models.RequestOptions)
				return s.finalizeRaw(ctx, payoutID, payload, options)
			}
		}
	}

	if len(args) < 2 || len(args) > 3 {
		return nil, errors.New("finalize requires signer, options, and optional request options")
	}
	signer, ok := args[0].(models.PayoutSignerInput)
	if !ok {
		if signerPtr, ptrOK := args[0].(*models.PayoutSignerInput); ptrOK && signerPtr != nil {
			signer = *signerPtr
		} else {
			return nil, errors.New("finalize signer must be models.PayoutSignerInput")
		}
	}
	options, ok := args[1].(models.PayoutFinalizeOptions)
	if !ok {
		if optionsPtr, ptrOK := args[1].(*models.PayoutFinalizeOptions); ptrOK && optionsPtr != nil {
			options = *optionsPtr
		} else {
			return nil, errors.New("finalize options must be models.PayoutFinalizeOptions")
		}
	}
	var requestOptions *models.RequestOptions
	if len(args) == 3 {
		requestOptions, _ = args[2].(*models.RequestOptions)
	}

	record, err := s.resolvePayoutRecord(ctx, payout, requestOptions)
	if err != nil {
		return nil, err
	}
	payload, batchDataHash, batchHash, merkleRoot, err := s.buildFinalizePayload(ctx, record, signer, options, requestOptions)
	if err != nil {
		return nil, err
	}
	res, err := s.finalizeRaw(ctx, record.ID, payload, requestOptions)
	if err != nil {
		return res, err
	}
	res.Data.BatchDataHash = batchDataHash
	res.Data.BatchHash = batchHash
	res.Data.MerkleRoot = merkleRoot
	identifier := batchDataHash
	if res.Data.Payout.PaymentType == models.PayoutTypeScheduled {
		identifier = firstNonEmpty(res.Data.Payout.MerkleRoot, merkleRoot)
	} else if res.Data.Payout.BatchDataHash != "" {
		identifier = res.Data.Payout.BatchDataHash
	}
	if identifier != "" {
		res.Data.FundingURL = strings.TrimRight(s.client.Config().ConsentHost, "/") + "/batch/" + identifier
	}
	return res, nil
}

func (s *PayoutService) finalizeRaw(ctx context.Context, payoutID string, payload map[string]any, options *models.RequestOptions) (*PayoutFinalization, error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "PATCH", Path: fmt.Sprintf("/batch-payments/%s", payoutID), Body: payload, Options: options})
	if err != nil {
		return nil, err
	}
	finalized, err := transport.Decode[models.APIResponse[models.PayoutRecord]](raw)
	if err != nil {
		return nil, err
	}
	return s.wrapFinalizationResponse(models.APIResponse[models.FinalizePayoutData]{
		Meta: finalized.Meta,
		Data: models.FinalizePayoutData{Payout: finalized.Data},
	}), nil
}

func (s *PayoutService) AddPayments(ctx context.Context, payout any, input any, options *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	payments, addInput, err := normalizeAddPaymentsInput(input)
	if err != nil {
		return models.APIResponse[map[string]any]{}, err
	}
	requestOptions := options
	if addInput != nil && addInput.RequestOptions != nil {
		requestOptions = addInput.RequestOptions
	}

	if payoutRecord, ok := payout.(models.PayoutRecord); ok && payoutRecord.PaymentType == models.PayoutTypeEscrow {
		if addInput == nil || addInput.Signer == nil {
			return models.APIResponse[map[string]any]{}, errors.New("a signer or private key is required to add payments to escrow payouts")
		}
		finalizeOptions := models.PayoutFinalizeOptions{}
		if addInput.FinalizeOptions != nil {
			finalizeOptions = *addInput.FinalizeOptions
		}
		return s.addEscrowPayees(ctx, payoutRecord, payments, *addInput.Signer, finalizeOptions, requestOptions)
	}
	if payoutRecord, ok := payout.(*models.PayoutRecord); ok && payoutRecord != nil && payoutRecord.PaymentType == models.PayoutTypeEscrow {
		if addInput == nil || addInput.Signer == nil {
			return models.APIResponse[map[string]any]{}, errors.New("a signer or private key is required to add payments to escrow payouts")
		}
		finalizeOptions := models.PayoutFinalizeOptions{}
		if addInput.FinalizeOptions != nil {
			finalizeOptions = *addInput.FinalizeOptions
		}
		return s.addEscrowPayees(ctx, *payoutRecord, payments, *addInput.Signer, finalizeOptions, requestOptions)
	}
	if payoutIntent, ok := payout.(*PayoutIntent); ok && payoutIntent != nil && payoutIntent.PaymentType == models.PayoutTypeEscrow {
		if addInput == nil || addInput.Signer == nil {
			return models.APIResponse[map[string]any]{}, errors.New("a signer or private key is required to add payments to escrow payouts")
		}
		finalizeOptions := models.PayoutFinalizeOptions{}
		if addInput.FinalizeOptions != nil {
			finalizeOptions = *addInput.FinalizeOptions
		}
		return s.addEscrowPayees(ctx, payoutIntent.Data, payments, *addInput.Signer, finalizeOptions, requestOptions)
	}
	if payoutIntent, ok := payout.(PayoutIntent); ok && payoutIntent.PaymentType == models.PayoutTypeEscrow {
		if addInput == nil || addInput.Signer == nil {
			return models.APIResponse[map[string]any]{}, errors.New("a signer or private key is required to add payments to escrow payouts")
		}
		finalizeOptions := models.PayoutFinalizeOptions{}
		if addInput.FinalizeOptions != nil {
			finalizeOptions = *addInput.FinalizeOptions
		}
		return s.addEscrowPayees(ctx, payoutIntent.Data, payments, *addInput.Signer, finalizeOptions, requestOptions)
	}

	payoutID := resolvePayoutID(payout)
	if payoutID == "" {
		return models.APIResponse[map[string]any]{}, fmt.Errorf("unsupported payout reference %T", payout)
	}
	chain := ""
	if payoutRecord, ok := payout.(models.PayoutRecord); ok {
		chain = payoutRecord.Chain
	}
	if payoutRecord, ok := payout.(*models.PayoutRecord); ok && payoutRecord != nil {
		chain = payoutRecord.Chain
	}
	if payoutIntent, ok := payout.(*PayoutIntent); ok && payoutIntent != nil {
		chain = payoutIntent.Chain
	}
	if payoutIntent, ok := payout.(PayoutIntent); ok {
		chain = payoutIntent.Chain
	}
	normalizedPayments, err := normalizePaymentsForCreate(chain, payments, "", nil, "")
	if err != nil {
		return models.APIResponse[map[string]any]{}, err
	}
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: fmt.Sprintf("/batch-payments/%s/payments", payoutID), Body: map[string]any{"payments": normalizedPayments}, Options: requestOptions})
	if err != nil {
		return models.APIResponse[map[string]any]{}, err
	}
	return transport.Decode[models.APIResponse[map[string]any]](raw)
}

func (s *PayoutService) RemovePayments(ctx context.Context, payoutID string, paymentIDs any, options *models.RequestOptions) (models.APIResponse[any], error) {
	normalized, err := normalizePaymentIDs(paymentIDs)
	if err != nil {
		return models.APIResponse[any]{}, err
	}
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "DELETE", Path: fmt.Sprintf("/batch-payments/%s/payments", payoutID), Body: map[string]any{"paymentIds": normalized}, Options: options})
	if err != nil {
		return models.APIResponse[any]{}, err
	}
	return transport.Decode[models.APIResponse[any]](raw)
}

func (s *PayoutService) DeletePayment(ctx context.Context, payoutID string, paymentID any, options *models.RequestOptions) (models.APIResponse[any], error) {
	return s.RemovePayments(ctx, payoutID, []any{paymentID}, options)
}

func (s *PayoutService) UpdatePayment(ctx context.Context, payoutID string, paymentID any, input models.UpdatePayoutPaymentInput, options *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "PATCH", Path: fmt.Sprintf("/batch-payments/%s/payments/%v", payoutID, paymentID), Body: input, Options: options})
	if err != nil {
		return models.APIResponse[map[string]any]{}, err
	}
	return transport.Decode[models.APIResponse[map[string]any]](raw)
}

func (s *PayoutService) EditPayment(ctx context.Context, payoutID string, paymentID any, input models.UpdatePayoutPaymentInput, options *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	return s.UpdatePayment(ctx, payoutID, paymentID, input, options)
}

func (s *PayoutService) ListPayments(ctx context.Context, payoutID string, query *models.PayoutPaymentsListQuery, options *models.RequestOptions) (models.APIResponse[[]map[string]any], error) {
	q := map[string]string{}
	if query != nil {
		if query.Page > 0 {
			q["page"] = strconv.Itoa(query.Page)
		}
		if query.PerPage > 0 {
			q["perPage"] = strconv.Itoa(query.PerPage)
		}
		if query.Limit > 0 {
			q["limit"] = strconv.Itoa(query.Limit)
		}
	}
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "GET", Path: fmt.Sprintf("/batch-payments/%s/payments", payoutID), Query: q, Options: options})
	if err != nil {
		return models.APIResponse[[]map[string]any]{}, err
	}
	return transport.Decode[models.APIResponse[[]map[string]any]](raw)
}

func (s *PayoutService) AddRecipients(ctx context.Context, payoutID string, recipients []models.PayoutRecipient, options *models.RequestOptions) (models.APIResponse[models.AddPayoutRecipientsResult], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: fmt.Sprintf("/batch-payments/%s/open-payees", payoutID), Body: map[string]any{"recipients": recipients}, Options: options})
	if err != nil {
		return models.APIResponse[models.AddPayoutRecipientsResult]{}, err
	}
	return transport.Decode[models.APIResponse[models.AddPayoutRecipientsResult]](raw)
}

func (s *PayoutService) ResolveRecipients(ctx context.Context, payoutID string, recipients []models.ResolvePayoutRecipient, options *models.RequestOptions) (models.APIResponse[models.ResolvePayoutRecipientsResult], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: fmt.Sprintf("/batch-payments/%s/resolve-recipients", payoutID), Body: map[string]any{"recipients": recipients}, Options: options})
	if err != nil {
		return models.APIResponse[models.ResolvePayoutRecipientsResult]{}, err
	}
	return transport.Decode[models.APIResponse[models.ResolvePayoutRecipientsResult]](raw)
}

func (s *PayoutService) ListInvites(ctx context.Context, payoutID string, options *models.RequestOptions) (models.APIResponse[[]map[string]any], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "GET", Path: fmt.Sprintf("/batch-payments/%s/invites", payoutID), Options: options})
	if err != nil {
		return models.APIResponse[[]map[string]any]{}, err
	}
	return transport.Decode[models.APIResponse[[]map[string]any]](raw)
}

func (s *PayoutService) RevokeInvite(ctx context.Context, payoutID string, inviteID any, options *models.RequestOptions) (models.APIResponse[any], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "DELETE", Path: fmt.Sprintf("/batch-payments/%s/invites/%v", payoutID, inviteID), Options: options})
	if err != nil {
		return models.APIResponse[any]{}, err
	}
	return transport.Decode[models.APIResponse[any]](raw)
}

func (s *PayoutService) RevokeInviteRoot(ctx context.Context, payoutID string, inviteRootID any, options *models.RequestOptions) (models.APIResponse[any], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "DELETE", Path: fmt.Sprintf("/batch-payments/%s/invite-roots/%v", payoutID, inviteRootID), Options: options})
	if err != nil {
		return models.APIResponse[any]{}, err
	}
	return transport.Decode[models.APIResponse[any]](raw)
}

func (s *PayoutService) Delete(ctx context.Context, payoutID string, options *models.RequestOptions) (models.APIResponse[any], error) {
	raw, _, err := s.client.Do(ctx, transport.Request{Method: "DELETE", Path: fmt.Sprintf("/batch-payments/%s", payoutID), Options: options})
	if err != nil {
		return models.APIResponse[any]{}, err
	}
	return transport.Decode[models.APIResponse[any]](raw)
}

func (s *PayoutService) addEscrowPayees(ctx context.Context, escrowPayout models.PayoutRecord, payments []models.PayoutPayment, signer models.PayoutSignerInput, options models.PayoutFinalizeOptions, requestOptions *models.RequestOptions) (models.APIResponse[map[string]any], error) {
	if escrowPayout.PaymentType != models.PayoutTypeEscrow {
		return models.APIResponse[map[string]any]{}, errors.New("addEscrowPayees requires an escrow payout")
	}
	if escrowPayout.BatchHash == "" {
		return models.APIResponse[map[string]any]{}, errors.New("escrow payout must be finalized before adding payees")
	}
	if escrowPayout.Status != "" && escrowPayout.Status != "funded" {
		return models.APIResponse[map[string]any]{}, errors.New("escrow payout must be funded before adding payees")
	}
	if len(payments) == 0 {
		return models.APIResponse[map[string]any]{}, errors.New("at least one payee is required")
	}

	claimDate := options.ClaimDate
	if claimDate == 0 {
		claimDate = timeNowUnix()
	}
	fundingToken := firstNonEmpty(normalizeTokenAddress(options.FundingToken), resolvePayoutFundingTokenCandidate(escrowPayout))
	scheduledPayments := make([]models.PayoutPayment, 0, len(payments))
	for _, payment := range payments {
		if payment.Token == "" {
			payment.Token = fundingToken
		}
		if payment.ClaimDate == nil {
			payment.ClaimDate = &claimDate
		}
		scheduledPayments = append(scheduledPayments, payment)
	}

	childID := options.ID
	if childID == "" {
		childID = createPayoutID()
	}
	metadata := map[string]any{}
	for k, v := range options.Metadata {
		metadata[k] = v
	}
	if _, ok := metadata["payoutCurrency"]; !ok {
		metadata["payoutCurrency"] = fundingToken
	}
	metadata["escrowBatch"] = escrowPayout.ID
	metadata["escrowBatchHash"] = escrowPayout.BatchHash
	metadata["scheduledDate"] = claimDate

	complianceMode := options.ComplianceMode
	if complianceMode == "" {
		complianceMode = escrowPayout.ComplianceMode
	}
	if complianceMode == "" {
		complianceMode = models.PayoutComplianceOpen
	}
	name := options.Name
	if name == "" {
		name = firstNonEmpty(escrowPayout.Name, "Escrow payout") + " Payees"
	}

	payout := models.PayoutRecord{
		ID:             childID,
		Chain:          firstNonEmpty(options.Chain, escrowPayout.Chain),
		PaymentType:    models.PayoutTypeScheduled,
		ComplianceMode: complianceMode,
		EscrowBatch:    escrowPayout.ID,
		Metadata:       metadata,
		Payments:       scheduledPayments,
		App:            escrowPayout.App,
	}
	finalizeOptions := options
	finalizeOptions.ClientID = firstNonEmpty(options.ClientID, appClientID(escrowPayout.App))
	finalizeOptions.Chain = firstNonEmpty(options.Chain, escrowPayout.Chain)
	finalizeOptions.EscrowBatch = escrowPayout
	finalizeOptions.FundingToken = fundingToken
	finalizeOptions.Payments = scheduledPayments
	finalizeOptions.ClaimDate = claimDate
	finalized, batchDataHash, batchHash, merkleRoot, err := s.buildFinalizePayload(ctx, payout, signer, finalizeOptions, requestOptions)
	if err != nil {
		return models.APIResponse[map[string]any]{}, err
	}

	body := map[string]any{
		"id":             childID,
		"chain":          payout.Chain,
		"nonce":          firstNonEmpty(payout.Nonce, mustCreatePayoutNonce()),
		"paymentType":    models.PayoutTypeScheduled,
		"isCommitment":   false,
		"escrowBatch":    escrowPayout.ID,
		"payments":       mustNormalizePaymentsForCreate(payout.Chain, scheduledPayments, fundingToken, nil, ""),
		"label":          name,
		"name":           name,
		"complianceMode": complianceMode,
		"metadata":       metadata,
	}
	if options.Description != "" {
		body["description"] = options.Description
	}
	for k, v := range finalized {
		body[k] = v
	}

	raw, _, err := s.client.Do(ctx, transport.Request{Method: "POST", Path: "/batch-payments", Body: body, Options: requestOptions})
	if err != nil {
		return models.APIResponse[map[string]any]{}, err
	}
	res, err := transport.Decode[models.APIResponse[map[string]any]](raw)
	if err != nil {
		return res, err
	}
	if res.Data == nil {
		res.Data = map[string]any{}
	}
	res.Data["batchDataHash"] = batchDataHash
	res.Data["batchHash"] = batchHash
	res.Data["merkleRoot"] = merkleRoot
	return res, nil
}

func (s *PayoutService) resolvePayoutRecord(ctx context.Context, payout any, options *models.RequestOptions) (models.PayoutRecord, error) {
	switch v := payout.(type) {
	case string:
		res, err := s.Get(ctx, v, options)
		if err != nil {
			return models.PayoutRecord{}, err
		}
		return res.Data, nil
	case PayoutIntent:
		return v.Data, nil
	case *PayoutIntent:
		if v == nil {
			return models.PayoutRecord{}, errors.New("payout intent is nil")
		}
		return v.Data, nil
	case models.PayoutRecord:
		return v, nil
	case *models.PayoutRecord:
		if v == nil {
			return models.PayoutRecord{}, errors.New("payout record is nil")
		}
		return *v, nil
	default:
		return models.PayoutRecord{}, fmt.Errorf("unsupported payout reference %T", payout)
	}
}

func (s *PayoutService) wrapPayoutResponse(res models.APIResponse[models.PayoutRecord]) *PayoutIntent {
	return &PayoutIntent{
		Meta:         res.Meta,
		Data:         res.Data,
		PayoutRecord: res.Data,
		service:      s,
	}
}

func (s *PayoutService) wrapFinalizationResponse(res models.APIResponse[models.FinalizePayoutData]) *PayoutFinalization {
	return &PayoutFinalization{
		Meta:               res.Meta,
		Data:               res.Data,
		FinalizePayoutData: res.Data,
		Payout:             s.wrapPayoutResponse(models.APIResponse[models.PayoutRecord]{Meta: res.Meta, Data: res.Data.Payout}),
	}
}

func (s *PayoutService) buildFinalizePayload(ctx context.Context, payout models.PayoutRecord, signer models.PayoutSignerInput, options models.PayoutFinalizeOptions, requestOptions *models.RequestOptions) (map[string]any, string, string, string, error) {
	timestamp := options.Timestamp
	if timestamp == 0 {
		timestamp = timeNowUnix()
	}
	signerAddress, err := resolveSignerAddress(signer, options.SignerAddress)
	if err != nil {
		return nil, "", "", "", err
	}
	complianceMode := options.ComplianceMode
	if complianceMode == "" {
		complianceMode = payout.ComplianceMode
	}
	if complianceMode == "" {
		complianceMode = models.PayoutComplianceOpen
	}
	clientID := options.ClientID
	if clientID == "" {
		clientID, _ = payout.App["clientId"].(string)
	}
	if clientID == "" {
		clientID = s.client.Config().ClientID
	}
	if clientID == "" {
		return nil, "", "", "", errors.New("clientId is required to finalize this payout")
	}
	chain := firstNonEmpty(options.Chain, payout.Chain)
	payments := options.Payments
	if payments == nil {
		payments = payout.Payments
	}

	payload := map[string]any{}
	var batchDataHash, batchHash string
	if payout.PaymentType == models.PayoutTypeScheduled || payout.IsCommitment {
		chainID, err := resolvePayoutChainID(chain, options.ChainID, "scheduled payout finalization")
		if err != nil {
			return nil, "", "", "", err
		}
		gracePeriod := options.GracePeriod
		if gracePeriod == 0 {
			gracePeriod = uint64FromMetadata(payout.Metadata, "gracePeriod")
		}
		disapprovalDeadline := options.DisapprovalDeadline
		if disapprovalDeadline == 0 {
			disapprovalDeadline = uint64FromMetadata(payout.Metadata, "disapprovalDeadline")
		}
		fundingToken := firstNonEmpty(normalizeTokenAddress(options.FundingToken), resolvePayoutFundingTokenCandidate(payout))
		if fundingToken == "" {
			return nil, "", "", "", errors.New("fundingToken must be provided as an address to finalize scheduled payouts")
		}
		payments, err = normalizePaymentsForSigning(payments, chain, fundingToken)
		if err != nil {
			return nil, "", "", "", err
		}
		batchHash, err = pvcrypto.ComputeScheduledPayoutHash(pvcrypto.ScheduledPayoutHashParams{
			PayoutID:            payout.ID,
			FundingToken:        fundingToken,
			GracePeriod:         gracePeriod,
			DisapprovalDeadline: disapprovalDeadline,
			Timestamp:           uint64(timestamp),
			ChainID:             chainID,
		})
		if err != nil {
			return nil, "", "", "", err
		}
		message := fmt.Sprintf("PVIUM_SIGNED_SCHEDULE:%s:%s:%s:%d", clientID, batchHash, complianceMode, timestamp)
		signature, returnedAddress, err := signFinalizeMessage(signer, message)
		if err != nil {
			return nil, "", "", "", err
		}
		if signerAddress == "" {
			signerAddress = returnedAddress
		}
		if signerAddress == "" {
			return nil, "", "", "", errors.New("scheduled payout finalization requires signerAddress")
		}
		claimDate := options.ClaimDate
		if claimDate == 0 {
			claimDate = int64FromMetadata(payout.Metadata, "scheduledDate")
		}
		if claimDate == 0 {
			claimDate = int64FromMetadata(payout.Metadata, "claimableDate")
		}
		merkleRoot, proofs, err := generateMerkleTreeForPayout(batchHash, payments, claimDate)
		if err != nil {
			return nil, "", "", "", err
		}
		batchDataHash, err = computeScheduledBatchDataHash(batchHash, merkleRoot, signerAddress)
		if err != nil {
			return nil, "", "", "", err
		}
		payload["signer"] = strings.ToLower(signerAddress)
		payload["batchSignature"] = fmt.Sprintf("%d:%s:%s", timestamp, strings.ToLower(signerAddress), signature)
		payload["batchHash"] = batchHash
		payload["merkleRoot"] = merkleRoot
		payload["batchDataHash"] = batchDataHash
		payload["proofs"] = proofs
		payload["gracePeriod"] = gracePeriod
		payload["disapprovalDeadline"] = disapprovalDeadline
		if !strings.Contains(strings.ToLower(chain), "solana") {
			fundingDigest := batchDataHash
			if escrowBatch, ok := options.EscrowBatch.(models.PayoutRecord); ok && escrowBatch.BatchHash != "" {
				fundingDigest = computeEscrowScheduledFundingDigest(escrowBatch.BatchHash, merkleRoot)
			}
			if escrowBatch, ok := options.EscrowBatch.(*models.PayoutRecord); ok && escrowBatch != nil && escrowBatch.BatchHash != "" {
				fundingDigest = computeEscrowScheduledFundingDigest(escrowBatch.BatchHash, merkleRoot)
			}
			payload["fundingSignature"], err = signFundingDigest(signer, fundingDigest)
			if err != nil {
				return nil, "", "", "", err
			}
		}
		return payload, batchDataHash, batchHash, merkleRoot, nil
	}

	if payout.PaymentType == models.PayoutTypeEscrow {
		chainID, err := resolvePayoutChainID(chain, options.ChainID, "escrow payout finalization")
		if err != nil {
			return nil, "", "", "", err
		}
		nonce := payout.Nonce
		if nonce == "" && timestamp != 0 {
			nonce = strconv.FormatInt(timestamp, 10)
		}
		if nonce == "" {
			return nil, "", "", "", errors.New("payout nonce is required to finalize escrow payouts")
		}
		fundingToken := firstNonEmpty(normalizeTokenAddress(options.FundingToken), resolvePayoutFundingTokenCandidate(payout))
		if fundingToken == "" {
			return nil, "", "", "", errors.New("fundingToken must be provided as an address to finalize escrow payouts")
		}
		payments, err = normalizePaymentsForSigning(payments, chain, fundingToken)
		if err != nil {
			return nil, "", "", "", err
		}
		lockDuration := options.LockDuration
		if lockDuration == 0 {
			lockDuration = int64FromMetadata(payout.Metadata, "lockDuration")
		}
		if lockDuration == 0 {
			lockDuration = int64(payout.LockDuration)
		}
		if lockDuration <= 0 {
			return nil, "", "", "", errors.New("lockDuration is required to finalize escrow payouts")
		}
		batchDataHash, err = pvcrypto.GenerateInstantPayoutHash(payments, nonce)
		if err != nil {
			return nil, "", "", "", err
		}
		message := fmt.Sprintf("PVIUM_SIGNED_BATCH:%s:%s:%s:%d", clientID, batchDataHash, complianceMode, timestamp)
		signature, returnedAddress, err := signFinalizeMessage(signer, message)
		if err != nil {
			return nil, "", "", "", err
		}
		if signerAddress == "" {
			signerAddress = returnedAddress
		}
		if signerAddress == "" {
			return nil, "", "", "", errors.New("escrow payout finalization requires signerAddress")
		}
		batchHash, err = pvcrypto.ComputeEscrowPayoutHash(pvcrypto.EscrowPayoutHashParams{
			PayoutID:     payout.ID,
			FundingToken: fundingToken,
			LockDuration: uint64(lockDuration),
			Timestamp:    uint64(timestamp),
			ChainID:      chainID,
		})
		if err != nil {
			return nil, "", "", "", err
		}
		fundingDigest := pvcrypto.ComputeEscrowFundingDigest(batchHash, signerAddress)
		fundingSignature, err := signFundingDigest(signer, fundingDigest)
		if err != nil {
			return nil, "", "", "", err
		}
		metadata := map[string]any{}
		for k, v := range payout.Metadata {
			metadata[k] = v
		}
		metadata["lockDuration"] = lockDuration
		payload["signer"] = strings.ToLower(signerAddress)
		payload["batchSignature"] = fmt.Sprintf("%d:%s:%s", timestamp, strings.ToLower(signerAddress), signature)
		payload["fundingSignature"] = fmt.Sprintf("%d:%s:%s", timestamp, strings.ToLower(signerAddress), fundingSignature)
		payload["batchHash"] = batchHash
		payload["batchDataHash"] = batchDataHash
		payload["metadata"] = metadata
		return payload, batchDataHash, batchHash, "", nil
	}

	nonce := payout.Nonce
	if nonce == "" && timestamp != 0 {
		nonce = strconv.FormatInt(timestamp, 10)
	}
	batchDataHash, err = pvcrypto.GenerateInstantPayoutHash(payments, nonce)
	if err != nil {
		return nil, "", "", "", err
	}
	message := fmt.Sprintf("PVIUM_SIGNED_BATCH:%s:%s:%s:%d", clientID, batchDataHash, complianceMode, timestamp)
	signature, returnedAddress, err := signFinalizeMessage(signer, message)
	if err != nil {
		return nil, "", "", "", err
	}
	if signerAddress == "" {
		signerAddress = returnedAddress
	}
	if signerAddress == "" {
		return nil, "", "", "", errors.New("instant payout finalization requires signerAddress")
	}
	payload["signer"] = strings.ToLower(signerAddress)
	payload["batchSignature"] = fmt.Sprintf("%d:%s:%s", timestamp, strings.ToLower(signerAddress), signature)
	payload["batchDataHash"] = batchDataHash
	return payload, batchDataHash, "", "", nil
}

func buildCreatePayoutBody(input models.CreatePayoutInput) (map[string]any, error) {
	paymentType := input.PaymentType
	if input.Type != "" {
		paymentType = input.Type
	}
	if paymentType == "" {
		paymentType = models.PayoutTypeInstant
	}
	isCommitment := false
	if paymentType == models.PayoutTypeMilestone {
		paymentType = models.PayoutTypeScheduled
		isCommitment = true
	}
	nonce := input.Nonce
	if nonce == "" {
		generated, err := pvcrypto.CreatePayoutNonce()
		if err != nil {
			return nil, err
		}
		nonce = generated
	}
	metadata := map[string]any{}
	for k, v := range input.Metadata {
		metadata[k] = v
	}
	if input.PayoutCurrency != "" {
		currency, err := resolvePayoutCurrencyConfig(input.Chain, input.PayoutCurrency)
		if err != nil {
			return nil, err
		}
		metadata["payoutCurrency"] = formatConfiguredToken(currency)
		metadata["payoutCurrencyDecimals"] = currency.Decimals
	}
	if input.ScheduleDate != nil {
		metadata["scheduledDate"] = input.ScheduleDate
	}
	if input.LockDuration != nil {
		if _, ok := metadata["lockDuration"]; !ok {
			metadata["lockDuration"] = *input.LockDuration
		}
	}
	if input.Type == models.PayoutTypeMilestone || input.PaymentType == models.PayoutTypeMilestone {
		metadata["commitmentType"] = "milestone"
	}
	complianceMode := input.ComplianceMode
	if complianceMode == "" {
		complianceMode = models.PayoutComplianceOpen
	}
	body := map[string]any{
		"chain":          input.Chain,
		"nonce":          nonce,
		"paymentType":    paymentType,
		"isCommitment":   isCommitment,
		"complianceMode": complianceMode,
		"metadata":       metadata,
	}
	if input.ID != "" {
		body["id"] = input.ID
	}
	if escrowBatch := resolvePayoutID(input.EscrowBatch); escrowBatch != "" {
		body["escrowBatch"] = escrowBatch
	}
	if input.Payments != nil {
		tokenFallback := tokenFallbackFromMetadata(metadata)
		decimalsFallback := decimalsFromMetadata(metadata)
		expectedToken := ""
		if input.PayoutCurrency != "" {
			expectedToken = tokenFallback
		}
		payments, err := normalizePaymentsForCreate(input.Chain, input.Payments, tokenFallback, decimalsFallback, expectedToken)
		if err != nil {
			return nil, err
		}
		body["payments"] = payments
	}
	if input.LockDuration != nil {
		body["lockDuration"] = *input.LockDuration
	}
	if label := firstNonEmpty(input.Label, input.Name); label != "" {
		body["label"] = label
	}
	if input.Name != "" {
		body["name"] = input.Name
	}
	if input.Description != "" {
		body["description"] = input.Description
	}
	return body, nil
}

func normalizePaymentsForCreate(chain string, payments []models.PayoutPayment, tokenFallback string, decimalsFallback *int, expectedToken string) ([]models.PayoutPayment, error) {
	if payments == nil {
		return nil, nil
	}
	normalizedExpectedToken := normalizeTokenValue(expectedToken)
	out := make([]models.PayoutPayment, 0, len(payments))
	for _, payment := range payments {
		token, currency, err := resolvePaymentToken(chain, payment, tokenFallback)
		if err != nil {
			return nil, err
		}
		normalizedPaymentToken := normalizeTokenValue(token)
		if normalizedExpectedToken != "" && normalizedPaymentToken != "" && normalizedPaymentToken != normalizedExpectedToken {
			return nil, errors.New("payment token must match payoutCurrency when payoutCurrency is provided")
		}
		payment.Token = token
		payment.TokenSymbol = ""
		if payment.Decimals == nil {
			if currency != nil {
				decimals := currency.Decimals
				payment.Decimals = &decimals
			} else if decimalsFallback != nil {
				decimals := *decimalsFallback
				payment.Decimals = &decimals
			}
		}
		if s, ok := payment.Amount.(string); ok {
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				payment.Amount = n
			}
		}
		out = append(out, payment)
	}
	return out, nil
}

func mustNormalizePaymentsForCreate(chain string, payments []models.PayoutPayment, tokenFallback string, decimalsFallback *int, expectedToken string) []models.PayoutPayment {
	normalized, err := normalizePaymentsForCreate(chain, payments, tokenFallback, decimalsFallback, expectedToken)
	if err != nil {
		return payments
	}
	return normalized
}

func normalizePaymentsForSigning(payments []models.PayoutPayment, chain string, tokenFallback string) ([]models.PayoutPayment, error) {
	tokenDecimals := (*int)(nil)
	if currency := resolvePayoutCurrencyByToken(chain, tokenFallback); currency != nil {
		decimals := currency.Decimals
		tokenDecimals = &decimals
	}
	return normalizePaymentsForCreate(chain, payments, tokenFallback, tokenDecimals, "")
}

func normalizeAddPaymentsInput(input any) ([]models.PayoutPayment, *models.AddPayoutPaymentsInput, error) {
	switch v := input.(type) {
	case []models.PayoutPayment:
		return v, nil, nil
	case models.AddPayoutPaymentsInput:
		return v.Payments, &v, nil
	case *models.AddPayoutPaymentsInput:
		if v == nil {
			return nil, nil, errors.New("add payments input is nil")
		}
		return v.Payments, v, nil
	default:
		return nil, nil, fmt.Errorf("unsupported add payments input %T", input)
	}
}

func normalizePaymentIDs(input any) ([]int64, error) {
	switch ids := input.(type) {
	case models.RemovePayoutPaymentsInput:
		return normalizePaymentIDs(ids.PaymentIDs)
	case []any:
		out := make([]int64, 0, len(ids))
		for _, id := range ids {
			n, err := normalizePaymentID(id)
			if err != nil {
				return nil, err
			}
			out = append(out, n)
		}
		return out, nil
	case []string:
		out := make([]int64, 0, len(ids))
		for _, id := range ids {
			n, err := normalizePaymentID(id)
			if err != nil {
				return nil, err
			}
			out = append(out, n)
		}
		return out, nil
	case []int:
		out := make([]int64, 0, len(ids))
		for _, id := range ids {
			out = append(out, int64(id))
		}
		return out, nil
	case []int64:
		return ids, nil
	default:
		return nil, fmt.Errorf("unsupported payment ids type %T", input)
	}
}

func normalizePaymentID(input any) (int64, error) {
	switch id := input.(type) {
	case string:
		return strconv.ParseInt(id, 10, 64)
	case int:
		return int64(id), nil
	case int64:
		return id, nil
	case float64:
		if math.Trunc(id) != id {
			return 0, errors.New("payment id must be an integer")
		}
		return int64(id), nil
	default:
		return 0, fmt.Errorf("unsupported payment id type %T", input)
	}
}

func resolvePayoutID(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case PayoutIntent:
		return v.ID
	case *PayoutIntent:
		if v == nil {
			return ""
		}
		return v.ID
	case models.PayoutRecord:
		return v.ID
	case *models.PayoutRecord:
		if v == nil {
			return ""
		}
		return v.ID
	default:
		return ""
	}
}

func tokenFallbackFromMetadata(metadata map[string]any) string {
	for _, key := range []string{"payoutToken", "payoutCurrency", "fundingToken"} {
		if token := normalizeTokenAddress(metadata[key]); token != "" {
			return token
		}
		if token, ok := metadata[key].(string); ok && len(token) > 10 {
			return token
		}
	}
	return ""
}

func decimalsFromMetadata(metadata map[string]any) *int {
	switch v := metadata["payoutCurrencyDecimals"].(type) {
	case int:
		return &v
	case int64:
		n := int(v)
		return &n
	case float64:
		n := int(v)
		return &n
	default:
		return nil
	}
}

func normalizeTokenAddress(value any) string {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "0x") {
			return common.HexToAddress(v).Hex()
		}
	case map[string]any:
		for _, key := range []string{"contractAddress", "address", "token", "payoutToken", "fundingToken", "current"} {
			if token := normalizeTokenAddress(v[key]); token != "" {
				return token
			}
		}
	}
	return ""
}

func normalizeTokenValue(token string) string {
	if normalized := normalizeTokenAddress(token); normalized != "" {
		return normalized
	}
	if len(token) > 10 {
		return token
	}
	return ""
}

func normalizePayoutCurrency(value string) string {
	switch strings.ToLower(value) {
	case "usdc":
		return "USDC"
	case "usdt":
		return "USDT"
	default:
		return ""
	}
}

func chainKeyFor(chain string) string {
	return payoutChainAliases[strings.ToLower(strings.ReplaceAll(chain, " ", ""))]
}

func resolvePayoutCurrencyConfig(chain string, currency string) (*supportedPayoutCurrency, error) {
	normalizedCurrency := normalizePayoutCurrency(currency)
	if normalizedCurrency == "" {
		return nil, nil
	}
	chainKey := chainKeyFor(chain)
	if chainKey == "" {
		return nil, fmt.Errorf("payoutCurrency %s is not supported on chain %s", normalizedCurrency, chain)
	}
	config, ok := stablecoinTokenAddresses[chainKey][normalizedCurrency]
	if !ok {
		return nil, fmt.Errorf("payoutCurrency %s is not supported on chain %s", normalizedCurrency, chain)
	}
	return &config, nil
}

func formatConfiguredToken(currency *supportedPayoutCurrency) string {
	if currency == nil {
		return ""
	}
	if strings.HasPrefix(currency.ContractAddress, "0x") {
		return common.HexToAddress(currency.ContractAddress).Hex()
	}
	return currency.ContractAddress
}

func resolvePayoutCurrencyByToken(chain string, token string) *supportedPayoutCurrency {
	if token == "" {
		return nil
	}
	chainKey := chainKeyFor(chain)
	if chainKey == "" {
		return nil
	}
	normalizedToken := normalizeTokenValue(token)
	for _, currency := range stablecoinTokenAddresses[chainKey] {
		if normalizeTokenValue(currency.ContractAddress) == normalizedToken {
			c := currency
			return &c
		}
	}
	return nil
}

func resolvePaymentToken(chain string, payment models.PayoutPayment, tokenFallback string) (string, *supportedPayoutCurrency, error) {
	symbol := payment.TokenSymbol
	if symbol == "" {
		symbol = normalizePayoutCurrency(payment.Token)
	}
	if symbol != "" {
		currency, err := resolvePayoutCurrencyConfig(chain, symbol)
		if err != nil {
			return "", nil, err
		}
		return formatConfiguredToken(currency), currency, nil
	}
	if payment.Token != "" {
		token := normalizeTokenValue(payment.Token)
		currency := resolvePayoutCurrencyByToken(chain, token)
		if token == "" || currency == nil {
			fallbackToken := normalizeTokenValue(tokenFallback)
			if token != "" && fallbackToken == token {
				return token, nil, nil
			}
			return "", nil, fmt.Errorf("payment token %s is not supported on chain %s", payment.Token, chain)
		}
		return formatConfiguredToken(currency), currency, nil
	}
	token := normalizeTokenValue(tokenFallback)
	currency := resolvePayoutCurrencyByToken(chain, token)
	if currency != nil {
		return formatConfiguredToken(currency), currency, nil
	}
	return token, nil, nil
}

func resolvePayoutChainID(chain string, chainID uint64, context string) (uint64, error) {
	if chainID != 0 {
		return chainID, nil
	}
	chainKey := chainKeyFor(chain)
	if chainKey == "" || payoutChainIDs[chainKey] == 0 {
		return 0, fmt.Errorf("chainId is required for %s", context)
	}
	return payoutChainIDs[chainKey], nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}

func resolveSignerAddress(signer models.PayoutSignerInput, fallback string) (string, error) {
	if fallback != "" {
		return strings.ToLower(fallback), nil
	}
	if signer.SignerAddress != "" {
		return strings.ToLower(signer.SignerAddress), nil
	}
	if signer.PrivateKey == "" {
		return "", nil
	}
	privateKey, err := ethcrypto.HexToECDSA(strings.TrimPrefix(signer.PrivateKey, "0x"))
	if err != nil {
		return "", err
	}
	return strings.ToLower(ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()), nil
}

func signFinalizeMessage(signer models.PayoutSignerInput, message string) (string, string, error) {
	if signer.PrivateKey != "" {
		signature, err := signPersonalMessage(message, signer.PrivateKey)
		address, addrErr := resolveSignerAddress(signer, "")
		if err != nil {
			return "", "", err
		}
		if addrErr != nil {
			return "", "", addrErr
		}
		return signature, address, nil
	}
	sign := signer.SignFinalize
	if sign == nil {
		sign = signer.SignMessage
	}
	if sign == nil {
		return "", "", errors.New("signer must provide privateKey, signMessage, or signFinalize")
	}
	signature, err := sign(message)
	return signature, signer.SignerAddress, err
}

func signFundingDigest(signer models.PayoutSignerInput, digest string) (string, error) {
	if signer.PrivateKey != "" {
		privateKey, err := ethcrypto.HexToECDSA(strings.TrimPrefix(signer.PrivateKey, "0x"))
		if err != nil {
			return "", err
		}
		digestBytes, err := hex.DecodeString(strings.TrimPrefix(digest, "0x"))
		if err != nil {
			return "", err
		}
		sig, err := ethcrypto.Sign(digestBytes, privateKey)
		if err != nil {
			return "", err
		}
		if len(sig) == 65 {
			sig[64] += 27
		}
		return "0x" + hex.EncodeToString(sig), nil
	}
	sign := signer.SignFunding
	if sign == nil {
		sign = signer.SignDigest
	}
	if sign == nil {
		return "", errors.New("EVM payout finalization requires signFunding, signDigest, or a private key")
	}
	return sign(digest)
}

func signPersonalMessage(message string, privateKeyHex string) (string, error) {
	privateKey, err := ethcrypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return "", err
	}
	messageBytes := []byte(message)
	prefixed := ethcrypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(messageBytes))), messageBytes)
	sig, err := ethcrypto.Sign(prefixed, privateKey)
	if err != nil {
		return "", err
	}
	if len(sig) == 65 {
		sig[64] += 27
	}
	return "0x" + hex.EncodeToString(sig), nil
}

func generateMerkleTreeForPayout(batchHash string, payments []models.PayoutPayment, defaultClaimDate int64) (string, []map[string]any, error) {
	if len(payments) == 0 {
		return "", nil, errors.New("cannot finalize scheduled payouts without payments")
	}
	leaves := make([][]byte, 0, len(payments))
	entries := make([]map[string]any, 0, len(payments))
	for _, payment := range payments {
		if payment.Decimals == nil {
			return "", nil, errors.New("payment decimals are required to hash scheduled payouts")
		}
		amount, err := pvcrypto.ParseUnits(payment.Amount, *payment.Decimals)
		if err != nil {
			return "", nil, err
		}
		claimDate := defaultClaimDate
		if payment.ClaimDate != nil {
			claimDate = *payment.ClaimDate
		}
		entry := map[string]any{
			"receiver":  strings.ToLower(payment.Receiver),
			"amount":    amount,
			"claimDate": claimDate,
			"memo":      payment.Memo,
		}
		leaf, err := generateLeafHash(batchHash, entry)
		if err != nil {
			return "", nil, err
		}
		leaves = append(leaves, leaf)
		entries = append(entries, entry)
	}
	levels := buildMerkleLevels(leaves)
	root := levels[len(levels)-1][0]
	proofs := make([]map[string]any, 0, len(entries))
	for i, entry := range entries {
		proof := merkleProof(levels, i)
		proofs = append(proofs, map[string]any{
			"receiver": entry["receiver"],
			"proof":    proof,
		})
	}
	return "0x" + hex.EncodeToString(root), proofs, nil
}

func generateLeafHash(batchHash string, entry map[string]any) ([]byte, error) {
	packed := []byte{}
	packed = append(packed, common.HexToHash(batchHash).Bytes()...)
	packed = append(packed, common.HexToAddress(entry["receiver"].(string)).Bytes()...)
	packed = append(packed, uint256Bytes(entry["amount"].(*big.Int))...)
	packed = append(packed, uint256Bytes(big.NewInt(entry["claimDate"].(int64)))...)
	packed = append(packed, []byte(entry["memo"].(string))...)
	return ethcrypto.Keccak256(packed), nil
}

func buildMerkleLevels(leaves [][]byte) [][][]byte {
	current := make([][]byte, len(leaves))
	for i := range leaves {
		current[i] = append([]byte(nil), leaves[i]...)
	}
	levels := [][][]byte{current}
	for len(current) > 1 {
		next := make([][]byte, 0, (len(current)+1)/2)
		for i := 0; i < len(current); i += 2 {
			if i+1 == len(current) {
				next = append(next, current[i])
				continue
			}
			left, right := current[i], current[i+1]
			if bytes.Compare(left, right) > 0 {
				left, right = right, left
			}
			next = append(next, ethcrypto.Keccak256(append(append([]byte(nil), left...), right...)))
		}
		current = next
		levels = append(levels, current)
	}
	return levels
}

func merkleProof(levels [][][]byte, index int) []string {
	proof := []string{}
	for level := 0; level < len(levels)-1; level++ {
		nodes := levels[level]
		sibling := index ^ 1
		if sibling < len(nodes) {
			proof = append(proof, "0x"+hex.EncodeToString(nodes[sibling]))
		}
		index /= 2
	}
	return proof
}

func computeScheduledBatchDataHash(batchHash, merkleRoot, signerAddress string) (string, error) {
	packed := []byte{}
	packed = append(packed, common.HexToHash(batchHash).Bytes()...)
	packed = append(packed, common.HexToHash(merkleRoot).Bytes()...)
	packed = append(packed, common.HexToAddress(signerAddress).Bytes()...)
	return pvcrypto.KeccakHex(packed), nil
}

func computeEscrowScheduledFundingDigest(escrowBatchHash, merkleRoot string) string {
	return pvcrypto.ComputeEscrowScheduledFundingDigest(escrowBatchHash, merkleRoot)
}

func uint256Bytes(value *big.Int) []byte {
	out := make([]byte, 32)
	if value == nil {
		return out
	}
	return value.FillBytes(out)
}

func resolvePayoutFundingTokenCandidate(payout models.PayoutRecord) string {
	if token := normalizeTokenAddress(payout.Metadata["payoutToken"]); token != "" {
		return token
	}
	if token, ok := payout.Metadata["payoutToken"].(string); ok && len(token) > 10 {
		return token
	}
	if token := normalizeTokenAddress(payout.Metadata["payoutCurrency"]); token != "" {
		return token
	}
	if token, ok := payout.Metadata["payoutCurrency"].(string); ok && len(token) > 10 {
		return token
	}
	if token := normalizeTokenAddress(payout.Metadata["fundingToken"]); token != "" {
		return token
	}
	if token, ok := payout.Metadata["fundingToken"].(string); ok && len(token) > 10 {
		return token
	}
	if len(payout.Payments) > 0 {
		return normalizeTokenValue(payout.Payments[0].Token)
	}
	return ""
}

func uint64FromMetadata(metadata map[string]any, key string) uint64 {
	switch v := metadata[key].(type) {
	case int:
		return uint64(v)
	case int64:
		return uint64(v)
	case uint64:
		return v
	case float64:
		return uint64(v)
	case string:
		n, _ := strconv.ParseUint(v, 10, 64)
		return n
	default:
		return 0
	}
}

func int64FromMetadata(metadata map[string]any, key string) int64 {
	switch v := metadata[key].(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}

func appClientID(app map[string]any) string {
	if app == nil {
		return ""
	}
	clientID, _ := app["clientId"].(string)
	return clientID
}

func mustCreatePayoutNonce() string {
	nonce, err := pvcrypto.CreatePayoutNonce()
	if err != nil {
		return ""
	}
	return nonce
}

func createPayoutID() string {
	buf := make([]byte, 16)
	if _, err := cryptorand.Read(buf); err != nil {
		return ""
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
