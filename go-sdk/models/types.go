package models

import "time"

type RequestOptions struct {
	AccessToken string
	APIKey      string
	Headers     map[string]string
	SkipAPIKey  bool
}

type APIMeta struct {
	StatusCode       int             `json:"statusCode"`
	Success          bool            `json:"success"`
	Message          string          `json:"message,omitempty"`
	DeveloperMessage string          `json:"developerMessage,omitempty"`
	Pagination       *PaginationMeta `json:"pagination,omitempty"`
}

type PaginationMeta struct {
	TotalCount  int    `json:"totalCount,omitempty"`
	PerPage     int    `json:"perPage,omitempty"`
	Current     int    `json:"current,omitempty"`
	CurrentPage string `json:"currentPage,omitempty"`
	Next        int    `json:"next,omitempty"`
	NextPage    string `json:"nextPage,omitempty"`
}

type APIResponse[T any] struct {
	Meta APIMeta `json:"meta"`
	Data T       `json:"data"`
}

type InvoicePaymentChannel struct {
	Chain    string `json:"chain"`
	Currency string `json:"currency"`
}

type CreateInvoiceRequest map[string]any

type InstallmentPlanItem struct {
	ID      int     `json:"id,omitempty"`
	Amount  float64 `json:"amount,omitempty"`
	DueDate string  `json:"dueDate,omitempty"`
}

type InvoiceItem struct {
	Name     string  `json:"name,omitempty"`
	Quantity int     `json:"quantity,omitempty"`
	Amount   float64 `json:"amount,omitempty"`
}

type InvoiceListItem struct {
	ID             int                   `json:"id"`
	Code           string                `json:"code"`
	Name           string                `json:"name"`
	DocumentType   string                `json:"documentType,omitempty"`
	ContractType   string                `json:"contractType,omitempty"`
	CurrencySymbol string                `json:"currencySymbol,omitempty"`
	ActualAmount   float64               `json:"actualAmount,omitempty"`
	TotalPaid      float64               `json:"totalPaid,omitempty"`
	TotalUnpaid    float64               `json:"totalUnpaid,omitempty"`
	Plan           []InstallmentPlanItem `json:"plan,omitempty"`
	Items          []InvoiceItem         `json:"items,omitempty"`
}

type CreateInvoiceData map[string]any

type InvoiceStatusInstallment struct {
	ID          int     `json:"id"`
	Amount      float64 `json:"amount"`
	DueDate     string  `json:"dueDate"`
	TotalPaid   float64 `json:"totalPaid"`
	TotalUnpaid float64 `json:"totalUnpaid"`
	Payments    []any   `json:"payments"`
}

type InvoiceStatusData struct {
	ContractID     int                        `json:"contractId"`
	ContractCode   string                     `json:"contractCode"`
	ContractName   string                     `json:"contractName"`
	CurrencySymbol string                     `json:"currencySymbol"`
	TotalAmount    float64                    `json:"totalAmount"`
	TotalPaid      float64                    `json:"totalPaid"`
	TotalUnpaid    float64                    `json:"totalUnpaid"`
	Installments   []InvoiceStatusInstallment `json:"installments"`
}

type InstallmentPayment struct {
	ID              int     `json:"id"`
	Installment     int     `json:"installment"`
	Amount          float64 `json:"amount"`
	Status          string  `json:"status"`
	PaymentMethod   string  `json:"paymentMethod,omitempty"`
	TransactionHash string  `json:"transactionHash,omitempty"`
	PaymentDate     string  `json:"paymentDate,omitempty"`
}

type OAuthTokenData struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresIn    int    `json:"expiresIn,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
	TokenType    string `json:"tokenType,omitempty"`
}

type OAuthSocialHandle struct {
	Provider   string `json:"provider"`
	Handle     string `json:"handle"`
	Subject    string `json:"subject,omitempty"`
	Name       string `json:"name,omitempty"`
	Email      string `json:"email,omitempty"`
	VerifiedAt string `json:"verifiedAt,omitempty"`
}

type OAuthLinkedAccount struct {
	Type     string         `json:"type,omitempty"`
	Username string         `json:"username,omitempty"`
	Login    string         `json:"login,omitempty"`
	Handle   string         `json:"handle,omitempty"`
	Profile  map[string]any `json:"profile,omitempty"`
}

type OAuthUserInfo struct {
	ID                 string               `json:"id,omitempty"`
	AltID              string               `json:"_id,omitempty"`
	Email              string               `json:"email,omitempty"`
	Handle             string               `json:"handle,omitempty"`
	SocialHandles      []OAuthSocialHandle  `json:"socialHandles,omitempty"`
	PrivyLinkedAccount []OAuthLinkedAccount `json:"privyLinkedAccounts,omitempty"`
	AuthorizedWallets  []any                `json:"authorizedWallets,omitempty"`
}

type ExchangeAuthorizationCodeInput struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirectUri"`
	ClientID    string `json:"clientId,omitempty"`
	APIKey      string `json:"apiKey,omitempty"`
}

type RefreshAccessTokenInput struct {
	RefreshToken string `json:"refreshToken"`
	ClientID     string `json:"clientId,omitempty"`
	APIKey       string `json:"apiKey,omitempty"`
}

type InviteIdentityType string

const (
	InviteIdentityEmail    InviteIdentityType = "email"
	InviteIdentityHandle   InviteIdentityType = "handle"
	InviteIdentityWallet   InviteIdentityType = "wallet"
	InviteIdentityX        InviteIdentityType = "x"
	InviteIdentityGitHub   InviteIdentityType = "github"
	InviteIdentityTwitter  InviteIdentityType = "twitter"
	InviteIdentityDiscord  InviteIdentityType = "discord"
	InviteIdentityTelegram InviteIdentityType = "telegram"
)

type InviteIdentity struct {
	Type                InviteIdentityType `json:"type"`
	Value               string             `json:"value"`
	DefaultPayoutAmount float64            `json:"defaultPayoutAmount,omitempty"`
	ExpiresAt           string             `json:"expiresAt,omitempty"`
}

type OAuthInviteBatchOptions struct {
	BatchID     string         `json:"batchId"`
	StateParams map[string]any `json:"stateParams,omitempty"`
}

type OAuthInviteBundleInput struct {
	Identities  []InviteIdentity         `json:"identities"`
	Scopes      []string                 `json:"scopes,omitempty"`
	Chain       string                   `json:"chain,omitempty"`
	BatchID     string                   `json:"batchId,omitempty"`
	CreatedAt   int64                    `json:"createdAt,omitempty"`
	RootNonce   string                   `json:"rootNonce,omitempty"`
	StateParams map[string]any           `json:"stateParams,omitempty"`
	State       string                   `json:"state,omitempty"`
	RedirectURI string                   `json:"redirectUri,omitempty"`
	BatchInvite *OAuthInviteBatchOptions `json:"batchInvite,omitempty"`
}

type OAuthInviteSigner struct {
	Chain            string
	PrivateKey       string
	SignerAddress    string
	SignMessage      func(message string) (string, error)
	SignMasterSecret func(message string) (string, error)
	SignInviteRoot   func(message string) (string, error)
}

type OAuthInviteBundleDraft struct {
	ClientID    string                   `json:"clientId"`
	ConsentHost string                   `json:"consentHost"`
	Identities  []InviteIdentity         `json:"identities"`
	Scopes      []string                 `json:"scopes"`
	BatchID     string                   `json:"batchId,omitempty"`
	BatchInvite *OAuthInviteBatchOptions `json:"batchInvite,omitempty"`
	Chain       string                   `json:"chain,omitempty"`
	State       string                   `json:"state,omitempty"`
	StateParams map[string]any           `json:"stateParams,omitempty"`
	RedirectURI string                   `json:"redirectUri,omitempty"`
	CreatedAt   int64                    `json:"createdAt,omitempty"`
	RootNonce   string                   `json:"rootNonce,omitempty"`
}

type InviteRootSignature struct {
	Root               string         `json:"root"`
	Nonce              string         `json:"nonce"`
	Signature          string         `json:"signature"`
	SignatureType      string         `json:"signatureType"`
	Scopes             []string       `json:"scopes"`
	SignatureMessage   string         `json:"signatureMessage"`
	SignatureTimestamp int64          `json:"signatureTimestamp"`
	SignerAddress      string         `json:"signerAddress,omitempty"`
	InviteCount        int            `json:"inviteCount"`
	ExpiresAt          string         `json:"expiresAt,omitempty"`
	Metadata           map[string]any `json:"metadata"`
}

type SignedInvite struct {
	IdentityType        InviteIdentityType `json:"identityType"`
	IdentityValue       string             `json:"identityValue"`
	IdentityCommitment  string             `json:"identityCommitment"`
	SecretHash          string             `json:"secretHash"`
	LeafVersion         string             `json:"leafVersion"`
	InviteNonce         string             `json:"inviteNonce"`
	InviteSecret        string             `json:"inviteSecret,omitempty"`
	InviteLink          string             `json:"inviteLink,omitempty"`
	DefaultPayoutAmount float64            `json:"defaultPayoutAmount,omitempty"`
	AppClientID         string             `json:"appClientId"`
	Leaf                string             `json:"leaf"`
	Proof               []string           `json:"proof"`
	ExpiresAt           string             `json:"expiresAt,omitempty"`
}

type BatchInviteMerkleData struct {
	Version          string         `json:"version"`
	AppClientID      string         `json:"appClientId"`
	BatchID          string         `json:"batchId"`
	Chain            string         `json:"chain,omitempty"`
	Scopes           []string       `json:"scopes"`
	Root             string         `json:"root"`
	RootNonce        string         `json:"rootNonce"`
	InviteCount      int            `json:"inviteCount"`
	CreatedAt        int64          `json:"createdAt"`
	ExpiresAt        int64          `json:"expiresAt"`
	SignatureMessage string         `json:"signatureMessage"`
	Invites          []SignedInvite `json:"invites"`
}

type SignedOAuthInviteBundle struct {
	ClientID        string                   `json:"clientId"`
	ConsentHost     string                   `json:"consentHost"`
	BatchID         string                   `json:"batchId"`
	BatchInvite     *OAuthInviteBatchOptions `json:"batchInvite,omitempty"`
	Scopes          []string                 `json:"scopes"`
	Chain           string                   `json:"chain,omitempty"`
	MasterSecret    string                   `json:"masterSecret"`
	Root            InviteRootSignature      `json:"root"`
	Invites         []SignedInvite           `json:"invites"`
	InviteLinks     []string                 `json:"inviteLinks"`
	GroupInviteLink string                   `json:"groupInviteLink,omitempty"`
	Merkle          BatchInviteMerkleData    `json:"merkle"`
}

type PayoutType string

const (
	PayoutTypeInstant   PayoutType = "Instant"
	PayoutTypeScheduled PayoutType = "Scheduled"
	PayoutTypeEscrow    PayoutType = "Escrow"
	PayoutTypeMilestone PayoutType = "Milestone"
)

type PayoutChain string

const (
	PayoutChainBase          PayoutChain = "base"
	PayoutChainBSC           PayoutChain = "bsc"
	PayoutChainSolana        PayoutChain = "solana"
	PayoutChainBaseTestnet   PayoutChain = "base-testnet"
	PayoutChainSolanaTestnet PayoutChain = "solana-testnet"
	PayoutChainLocalhost     PayoutChain = "localhost"
)

type PayoutCurrency string

const (
	PayoutCurrencyUSDC PayoutCurrency = "USDC"
	PayoutCurrencyUSDT PayoutCurrency = "USDT"
)

type PayoutComplianceMode string

const (
	PayoutComplianceOpen   PayoutComplianceMode = "Open"
	PayoutComplianceStrict PayoutComplianceMode = "Strict"
)

type PayoutPayment struct {
	Receiver    string `json:"receiver"`
	Amount      any    `json:"amount"`
	Token       string `json:"token,omitempty"`
	TokenSymbol string `json:"tokenSymbol,omitempty"`
	Decimals    *int   `json:"decimals,omitempty"`
	Memo        string `json:"memo,omitempty"`
	PublicID    string `json:"publicId,omitempty"`
	ClaimDate   *int64 `json:"claimDate,omitempty"`
	ClaimEnd    *int64 `json:"claimEnd,omitempty"`
}

type PayoutRecipient struct {
	IdentityType        string   `json:"identityType"`
	IdentityValue       string   `json:"identityValue"`
	DefaultPayoutAmount *float64 `json:"defaultPayoutAmount,omitempty"`
	Memo                string   `json:"memo,omitempty"`
}

type PayoutRecord struct {
	ID                string               `json:"id"`
	Chain             string               `json:"chain"`
	PaymentType       PayoutType           `json:"paymentType"`
	IsCommitment      bool                 `json:"isCommitment,omitempty"`
	Status            string               `json:"status,omitempty"`
	Nonce             string               `json:"nonce,omitempty"`
	BatchDataHash     string               `json:"batchDataHash,omitempty"`
	BatchHash         string               `json:"batchHash,omitempty"`
	MerkleRoot        string               `json:"merkleRoot,omitempty"`
	BatchSignature    string               `json:"batchSignature,omitempty"`
	FundingSignature  string               `json:"fundingSignature,omitempty"`
	LockDuration      int                  `json:"lockDuration,omitempty"`
	Metadata          map[string]any       `json:"metadata,omitempty"`
	Payments          []PayoutPayment      `json:"payments,omitempty"`
	PaymentCount      int                  `json:"paymentCount,omitempty"`
	PaymentsLimit     int                  `json:"paymentsLimit,omitempty"`
	PaymentsTruncated bool                 `json:"paymentsTruncated,omitempty"`
	EscrowBatch       string               `json:"escrowBatch,omitempty"`
	ComplianceMode    PayoutComplianceMode `json:"complianceMode,omitempty"`
	App               map[string]any       `json:"app,omitempty"`
	Name              string               `json:"name,omitempty"`
	Description       string               `json:"description,omitempty"`
}

type CreatePayoutInput struct {
	ID             string               `json:"id,omitempty"`
	Type           PayoutType           `json:"type,omitempty"`
	Chain          string               `json:"chain"`
	Nonce          string               `json:"nonce,omitempty"`
	PaymentType    PayoutType           `json:"paymentType,omitempty"`
	Payments       []PayoutPayment      `json:"payments,omitempty"`
	EscrowBatch    any                  `json:"escrowBatch,omitempty"`
	LockDuration   *int64               `json:"lockDuration,omitempty"`
	Label          string               `json:"label,omitempty"`
	Name           string               `json:"name,omitempty"`
	Description    string               `json:"description,omitempty"`
	Metadata       map[string]any       `json:"metadata,omitempty"`
	ComplianceMode PayoutComplianceMode `json:"complianceMode,omitempty"`
	PayoutCurrency string               `json:"payoutCurrency,omitempty"`
	ScheduleDate   any                  `json:"scheduleDate,omitempty"`
}

type PayoutListQuery struct {
	Page         int    `json:"page,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	PaymentType  string `json:"paymentType,omitempty"`
	IsCommitment *bool  `json:"isCommitment,omitempty"`
	Status       string `json:"status,omitempty"`
}

type PayoutPaymentsListQuery struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"perPage,omitempty"`
	Limit   int `json:"limit,omitempty"`
}

type FinalizePayoutData struct {
	Payout        PayoutRecord   `json:"payout"`
	FundingURL    string         `json:"fundingUrl,omitempty"`
	FundingCall   map[string]any `json:"fundingCall,omitempty"`
	BatchDataHash string         `json:"batchDataHash"`
	BatchHash     string         `json:"batchHash,omitempty"`
	MerkleRoot    string         `json:"merkleRoot,omitempty"`
}

type AddPayoutPaymentsInput struct {
	Payments        []PayoutPayment        `json:"payments"`
	Signer          *PayoutSignerInput     `json:"-"`
	FinalizeOptions *PayoutFinalizeOptions `json:"-"`
	RequestOptions  *RequestOptions        `json:"-"`
}

type RemovePayoutPaymentsInput struct {
	PaymentIDs []any `json:"paymentIds"`
}

type UpdatePayoutPaymentInput struct {
	Amount    any    `json:"amount,omitempty"`
	Memo      string `json:"memo,omitempty"`
	ClaimDate any    `json:"claimDate,omitempty"`
}

type AddPayoutRecipientsInput struct {
	Recipients []PayoutRecipient `json:"recipients"`
}

type ResolvePayoutRecipient struct {
	IdentityType  string `json:"identityType,omitempty"`
	IdentityValue string `json:"identityValue,omitempty"`
}

type ResolvePayoutRecipientsInput struct {
	Recipients []ResolvePayoutRecipient `json:"recipients"`
}

type PayoutRecipientResult struct {
	Identity       string `json:"identity,omitempty"`
	IdentityType   string `json:"identityType,omitempty"`
	IdentityValue  string `json:"identityValue,omitempty"`
	UserID         string `json:"userId,omitempty"`
	Email          string `json:"email,omitempty"`
	Handle         string `json:"handle,omitempty"`
	EthereumWallet string `json:"ethereumWallet,omitempty"`
	SolanaWallet   string `json:"solanaWallet,omitempty"`
	Receiver       string `json:"receiver,omitempty"`
}

type PayoutRecipientError struct {
	Identity      string `json:"identity,omitempty"`
	IdentityType  string `json:"identityType,omitempty"`
	IdentityValue string `json:"identityValue,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

type AddPayoutRecipientsResult struct {
	Added  []PayoutRecipientResult `json:"added"`
	Errors []PayoutRecipientError  `json:"errors"`
}

type ResolvePayoutRecipientsResult struct {
	Resolved []PayoutRecipientResult `json:"resolved"`
	Errors   []PayoutRecipientError  `json:"errors"`
}

type PayoutMessageSignature struct {
	Signature     string `json:"signature"`
	SignatureType string `json:"signatureType,omitempty"`
	SignerAddress string `json:"signerAddress,omitempty"`
}

type PayoutSignerInput struct {
	Chain         string
	PrivateKey    string
	SignerAddress string
	SignMessage   func(message string) (string, error)
	SignFinalize  func(message string) (string, error)
	SignFunding   func(digest string) (string, error)
	SignDigest    func(digest string) (string, error)
}

type PayoutFinalizeOptions struct {
	ClientID            string
	Chain               string
	ChainID             uint64
	EscrowBatch         any
	FundingToken        string
	Payments            []PayoutPayment
	ComplianceMode      PayoutComplianceMode
	GracePeriod         uint64
	DisapprovalDeadline uint64
	ClaimDate           int64
	LockDuration        int64
	Timestamp           int64
	SignerAddress       string
	ID                  string
	Name                string
	Description         string
	Metadata            map[string]any
}

type PviumWebhookTokenPayload struct {
	Event string         `json:"event"`
	Data  map[string]any `json:"data"`
	Iat   int64          `json:"iat,omitempty"`
	Exp   int64          `json:"exp,omitempty"`
}

type VerifyPviumWebhookTokenOptions struct {
	ExpectedEvent             string    `json:"expectedEvent,omitempty"`
	Now                       time.Time `json:"now,omitempty"`
	AllowHashedSecretFallback *bool     `json:"allowHashedSecretFallback,omitempty"`
}
