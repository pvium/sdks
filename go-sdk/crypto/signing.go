package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/pvium/sdks/go-sdk/models"
	"golang.org/x/crypto/sha3"
)

type SignerInput struct {
	PrivateKey string
}

type MessageSigner func(hash []byte) (string, error)

const PVIUMSignatureDomainText = "PVIUM_SIGNATURE_MESSAGE"

func SignatureDomainFromText(text string) string {
	return KeccakHex([]byte(text))
}

func KeccakHex(input []byte) string {
	h := sha3.NewLegacyKeccak256()
	h.Write(input)
	return "0x" + hex.EncodeToString(h.Sum(nil))
}

func HashJSONPayload(payload any) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return KeccakHex(b), nil
}

func HashABIEncodedPayload(types []string, values []any) (string, error) {
	if len(types) != len(values) {
		return "", errors.New("abi types and values length mismatch")
	}
	args := make(abi.Arguments, 0, len(types))
	converted := make([]any, 0, len(values))
	for i, typeName := range types {
		typ, err := abi.NewType(typeName, "", nil)
		if err != nil {
			return "", err
		}
		args = append(args, abi.Argument{Type: typ})
		converted = append(converted, convertABIValue(typeName, values[i]))
	}
	packed, err := args.Pack(converted...)
	if err != nil {
		return "", err
	}
	return KeccakHex(packed), nil
}

func SignMessageHash(hashHex string, privateKey string) (string, error) {
	pk, err := ethcrypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return "", err
	}
	msg := strings.TrimPrefix(hashHex, "0x")
	hashBytes, err := hex.DecodeString(msg)
	if err != nil {
		return "", err
	}
	prefixed := ethcrypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(hashBytes))), hashBytes)
	sig, err := ethcrypto.Sign(prefixed, pk)
	if err != nil {
		return "", err
	}
	if len(sig) == 65 {
		sig[64] += 27
	}
	return "0x" + hex.EncodeToString(sig), nil
}

func CreateSignerFromPrivateKey(privateKey string) MessageSigner {
	return func(hash []byte) (string, error) {
		return SignMessageHash("0x"+hex.EncodeToString(hash), privateKey)
	}
}

func hashPayload(payload any) (string, error) {
	return HashJSONPayload(payload)
}

type CreateProjectRequestPayload map[string]any

type CreateProjectSignatureOptions struct {
	PviumFeeBps     any
	ChainID         any
	SignatureDomain string
}

type CreateClaimRequestPayload map[string]any

type FinalizeClaimRequestPayload []map[string]any

type ResolveDisputeRequestPayload struct {
	ClaimID  string
	Approved bool
	ChainID  uint64
}

type RelayedCallRequestPayload map[string]any

func HashCreateProjectRequest(payload CreateProjectRequestPayload, options CreateProjectSignatureOptions) (string, error) {
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	stringType, _ := abi.NewType("string", "", nil)
	addressType, _ := abi.NewType("address", "", nil)
	uintType, _ := abi.NewType("uint256", "", nil)
	args := abi.Arguments{
		{Type: bytes32Type},
		{Type: stringType},
		{Type: stringType},
		{Type: stringType},
		{Type: addressType},
		{Type: addressType},
		{Type: addressType},
		{Type: addressType},
		{Type: uintType},
		{Type: uintType},
		{Type: uintType},
		{Type: uintType},
		{Type: uintType},
		{Type: uintType},
	}
	domain := options.SignatureDomain
	if domain == "" {
		domain = SignatureDomainFromText(PVIUMSignatureDomainText)
	}
	packed, err := args.Pack(
		common.HexToHash(domain),
		stringFromMap(payload, "app"),
		stringFromMap(payload, "projectId"),
		stringFromMap(payload, "metadata"),
		common.HexToAddress(stringFromMap(payload, "tokenAddress")),
		common.HexToAddress(stringFromMap(payload, "refundAddress")),
		common.HexToAddress(stringFromMap(payload, "appFeeAddress")),
		common.HexToAddress(stringFromMap(payload, "appAdminAddress")),
		bigFromAny(payload["appFeeBps"]),
		bigFromAny(payload["disputeWindowSeconds"]),
		bigFromAny(payload["lockDuration"]),
		bigFromAny(payload["minimumBalancePerVendor"]),
		bigFromAny(options.PviumFeeBps),
		bigFromAny(options.ChainID),
	)
	if err != nil {
		return "", err
	}
	return KeccakHex(packed), nil
}
func HashCreateClaimRequest(payload CreateClaimRequestPayload) (string, error) {
	stringType, _ := abi.NewType("string", "", nil)
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	addressType, _ := abi.NewType("address", "", nil)
	uintType, _ := abi.NewType("uint256", "", nil)
	args := abi.Arguments{{Type: stringType}, {Type: stringType}, {Type: bytes32Type}, {Type: addressType}, {Type: uintType}, {Type: uintType}, {Type: uintType}, {Type: uintType}}
	packed, err := args.Pack(
		stringFromMap(payload, "app"),
		stringFromMap(payload, "projectId"),
		common.HexToHash(stringFromMap(payload, "claimId")),
		common.HexToAddress(stringFromMap(payload, "receiver")),
		bigFromAny(payload["amount"]),
		bigFromAny(payload["claimableAfter"]),
		bigFromAny(payload["claimDeadline"]),
		bigFromAny(payload["nonce"]),
	)
	if err != nil {
		return "", err
	}
	return KeccakHex(packed), nil
}
func HashFinalizeClaimRequest(claims FinalizeClaimRequestPayload, chainID any) (string, error) {
	data := []byte{}
	for _, claim := range claims {
		data = append(data, []byte(stringFromMap(claim, "app"))...)
		data = append(data, []byte(stringFromMap(claim, "projectId"))...)
		claimBytes, err := hex.DecodeString(strings.TrimPrefix(stringFromMap(claim, "claimId"), "0x"))
		if err != nil {
			return "", err
		}
		data = append(data, claimBytes...)
	}
	data = append(data, uint256Bytes(bigFromAny(chainID))...)
	return KeccakHex(data), nil
}

func HashDisputeRequest(claimID string, chainID uint64) (string, error) {
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return "", err
	}
	uintType, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return "", err
	}
	args := abi.Arguments{{Type: bytes32Type}, {Type: uintType}}

	claimHash := common.HexToHash(claimID)
	packed, err := args.Pack(claimHash, new(big.Int).SetUint64(chainID))
	if err != nil {
		return "", err
	}
	return KeccakHex(packed), nil
}

func HashResolveDisputeRequest(payload ResolveDisputeRequestPayload) (string, error) {
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return "", err
	}
	boolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return "", err
	}
	uintType, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return "", err
	}
	args := abi.Arguments{{Type: bytes32Type}, {Type: boolType}, {Type: uintType}}
	claimHash := common.HexToHash(payload.ClaimID)
	packed, err := args.Pack(claimHash, payload.Approved, new(big.Int).SetUint64(payload.ChainID))
	if err != nil {
		return "", err
	}
	return KeccakHex(packed), nil
}
func HashRelayedCallRequest(payload RelayedCallRequestPayload) (string, error) {
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	uintType, _ := abi.NewType("uint256", "", nil)
	args := abi.Arguments{{Type: stringType}, {Type: stringType}, {Type: bytesType}, {Type: uintType}, {Type: uintType}}
	payloadBytes, err := hex.DecodeString(strings.TrimPrefix(stringFromMap(payload, "payload"), "0x"))
	if err != nil {
		return "", err
	}
	packed, err := args.Pack(
		stringFromMap(payload, "appId"),
		stringFromMap(payload, "projectId"),
		payloadBytes,
		bigFromAny(payload["nonce"]),
		bigFromAny(payload["chainId"]),
	)
	if err != nil {
		return "", err
	}
	return KeccakHex(packed), nil
}

func signPayload(payload any, signer SignerInput) (string, error) {
	h, err := hashPayload(payload)
	if err != nil {
		return "", err
	}
	return SignMessageHash(h, signer.PrivateKey)
}

func SignCreateProjectRequest(payload CreateProjectRequestPayload, signer SignerInput, options CreateProjectSignatureOptions) (string, error) {
	hash, err := HashCreateProjectRequest(payload, options)
	if err != nil {
		return "", err
	}
	return SignMessageHash(hash, signer.PrivateKey)
}

func HashCreateProjectAttestation(appSignature string, chainID any, signatureDomain ...string) (string, error) {
	domain := SignatureDomainFromText(PVIUMSignatureDomainText)
	if len(signatureDomain) > 0 && signatureDomain[0] != "" {
		domain = signatureDomain[0]
	}
	return HashABIEncodedPayload(
		[]string{"bytes32", "bytes", "uint256"},
		[]any{domain, appSignature, chainID},
	)
}

func SignCreateProjectAttestation(appSignature string, signer SignerInput, chainID any, signatureDomain ...string) (string, error) {
	hash, err := HashCreateProjectAttestation(appSignature, chainID, signatureDomain...)
	if err != nil {
		return "", err
	}
	return SignMessageHash(hash, signer.PrivateKey)
}

func SignCreateClaimRequest(payload CreateClaimRequestPayload, signer SignerInput) (string, error) {
	hash, err := HashCreateClaimRequest(payload)
	if err != nil {
		return "", err
	}
	return SignMessageHash(hash, signer.PrivateKey)
}

func SignFinalizeClaimRequest(payload FinalizeClaimRequestPayload, signer SignerInput, chainID any) (string, error) {
	hash, err := HashFinalizeClaimRequest(payload, chainID)
	if err != nil {
		return "", err
	}
	return SignMessageHash(hash, signer.PrivateKey)
}

func SignDisputeRequest(claimID string, signer SignerInput, chainID uint64) (string, error) {
	hash, err := HashDisputeRequest(claimID, chainID)
	if err != nil {
		return "", err
	}
	return SignMessageHash(hash, signer.PrivateKey)
}

func SignResolveDisputeRequest(payload ResolveDisputeRequestPayload, signer SignerInput) (string, error) {
	hash, err := HashResolveDisputeRequest(payload)
	if err != nil {
		return "", err
	}
	return SignMessageHash(hash, signer.PrivateKey)
}

func SignRelayedCallRequest(payload RelayedCallRequestPayload, signer SignerInput) (string, error) {
	hash, err := HashRelayedCallRequest(payload)
	if err != nil {
		return "", err
	}
	return SignMessageHash(hash, signer.PrivateKey)
}

func CreatePayoutNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(buf), nil
}

func GenerateInstantPayoutHash(payments []models.PayoutPayment, nonce string) (string, error) {
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return "", err
	}
	tupleType, err := abi.NewType("tuple[]", "struct Payout", []abi.ArgumentMarshaling{
		{Name: "receiver", Type: "address"},
		{Name: "amount", Type: "uint256"},
		{Name: "token", Type: "address"},
		{Name: "memo", Type: "string"},
	})
	if err != nil {
		return "", err
	}
	args := abi.Arguments{{Type: bytesType}, {Type: tupleType}}

	nonceBytes, err := normalizeHexBytes(nonce)
	if err != nil {
		return "", err
	}

	payouts := make([]struct {
		Receiver common.Address
		Amount   *big.Int
		Token    common.Address
		Memo     string
	}, 0, len(payments))
	for _, payment := range payments {
		if payment.Decimals == nil {
			return "", errors.New("payment decimals are required to hash instant payouts")
		}
		if payment.Token == "" {
			return "", errors.New("payment token is required to hash instant payouts")
		}
		amount, err := ParseUnits(payment.Amount, *payment.Decimals)
		if err != nil {
			return "", err
		}
		payouts = append(payouts, struct {
			Receiver common.Address
			Amount   *big.Int
			Token    common.Address
			Memo     string
		}{
			Receiver: common.HexToAddress(payment.Receiver),
			Amount:   amount,
			Token:    common.HexToAddress(payment.Token),
			Memo:     payment.Memo,
		})
	}

	packed, err := args.Pack(nonceBytes, payouts)
	if err != nil {
		return "", err
	}
	return KeccakHex(packed), nil
}

type ScheduledPayoutHashParams struct {
	PayoutID            string
	FundingToken        string
	GracePeriod         uint64
	DisapprovalDeadline uint64
	Timestamp           uint64
	ChainID             uint64
}

type EscrowPayoutHashParams struct {
	PayoutID     string
	FundingToken string
	LockDuration uint64
	Timestamp    uint64
	ChainID      uint64
}

func ComputeScheduledPayoutHash(params ScheduledPayoutHashParams) (string, error) {
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return "", err
	}
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return "", err
	}
	uintType, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return "", err
	}
	args := abi.Arguments{
		{Type: bytes32Type},
		{Type: addressType},
		{Type: uintType},
		{Type: uintType},
		{Type: uintType},
		{Type: uintType},
	}

	payoutHex := strings.ReplaceAll(params.PayoutID, "-", "")
	if len(payoutHex) < 64 {
		payoutHex = payoutHex + strings.Repeat("0", 64-len(payoutHex))
	}
	payoutIDBytes := common.HexToHash("0x" + payoutHex)
	packed, err := args.Pack(
		payoutIDBytes,
		common.HexToAddress(params.FundingToken),
		new(big.Int).SetUint64(params.GracePeriod),
		new(big.Int).SetUint64(params.DisapprovalDeadline),
		new(big.Int).SetUint64(params.Timestamp),
		new(big.Int).SetUint64(params.ChainID),
	)
	if err != nil {
		return "", err
	}
	return KeccakHex(packed), nil
}

func ComputeEscrowPayoutHash(params EscrowPayoutHashParams) (string, error) {
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	addressType, _ := abi.NewType("address", "", nil)
	uintType, _ := abi.NewType("uint256", "", nil)
	args := abi.Arguments{{Type: bytes32Type}, {Type: addressType}, {Type: uintType}, {Type: uintType}, {Type: uintType}}
	payoutHex := strings.ReplaceAll(params.PayoutID, "-", "")
	if len(payoutHex) < 64 {
		payoutHex += strings.Repeat("0", 64-len(payoutHex))
	}
	packed, err := args.Pack(
		common.HexToHash("0x"+payoutHex),
		common.HexToAddress(params.FundingToken),
		new(big.Int).SetUint64(params.LockDuration),
		new(big.Int).SetUint64(params.Timestamp),
		new(big.Int).SetUint64(params.ChainID),
	)
	if err != nil {
		return "", err
	}
	return KeccakHex(packed), nil
}

func ComputeEscrowFundingDigest(escrowBatchHash, withdrawalWallet string) string {
	packed := []byte{}
	packed = append(packed, common.HexToHash(escrowBatchHash).Bytes()...)
	packed = append(packed, common.HexToAddress(withdrawalWallet).Bytes()...)
	return KeccakHex(packed)
}

func ComputeEscrowScheduledFundingDigest(escrowBatchHash, merkleRoot string) string {
	packed := []byte{}
	packed = append(packed, common.HexToHash(escrowBatchHash).Bytes()...)
	packed = append(packed, common.HexToHash(merkleRoot).Bytes()...)
	return KeccakHex(packed)
}

func MustBigInt(v string) *big.Int {
	b, ok := new(big.Int).SetString(v, 10)
	if !ok {
		panic(errors.New("invalid big int"))
	}
	return b
}

func normalizeHexBytes(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, errors.New("payout nonce is required")
	}
	body := strings.TrimPrefix(trimmed, "0x")
	if len(body)%2 != 0 {
		body = "0" + body
	}
	out, err := hex.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("payout nonce must be hex-compatible: %s", value)
	}
	return out, nil
}

func ParseUnits(value any, decimals int) (*big.Int, error) {
	if decimals < 0 {
		return nil, errors.New("decimals must be non-negative")
	}

	switch v := value.(type) {
	case string:
		return parseDecimalUnits(v, decimals)
	case int:
		return new(big.Int).Mul(big.NewInt(int64(v)), unitMultiplier(decimals)), nil
	case int64:
		return new(big.Int).Mul(big.NewInt(v), unitMultiplier(decimals)), nil
	case float64:
		return parseDecimalUnits(fmt.Sprintf("%v", v), decimals)
	case float32:
		return parseDecimalUnits(fmt.Sprintf("%v", v), decimals)
	default:
		rv := reflect.ValueOf(value)
		if rv.IsValid() {
			switch rv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return new(big.Int).Mul(big.NewInt(rv.Int()), unitMultiplier(decimals)), nil
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				n := new(big.Int).SetUint64(rv.Uint())
				return n.Mul(n, unitMultiplier(decimals)), nil
			}
		}
		return nil, fmt.Errorf("unsupported amount type %T", value)
	}
}

func parseDecimalUnits(raw string, decimals int) (*big.Int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("amount is required")
	}
	if strings.HasPrefix(value, "-") {
		return nil, errors.New("amount must be non-negative")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid decimal amount: %s", raw)
	}
	whole := parts[0]
	if whole == "" {
		whole = "0"
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > decimals {
		return nil, fmt.Errorf("amount has too many decimal places: %s", raw)
	}
	fraction += strings.Repeat("0", decimals-len(fraction))
	combined := strings.TrimLeft(whole+fraction, "0")
	if combined == "" {
		combined = "0"
	}
	out, ok := new(big.Int).SetString(combined, 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal amount: %s", raw)
	}
	return out, nil
}

func unitMultiplier(decimals int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
}

func stringFromMap(payload map[string]any, key string) string {
	if value, ok := payload[key].(string); ok {
		return value
	}
	return fmt.Sprint(payload[key])
}

func bigFromAny(value any) *big.Int {
	switch v := value.(type) {
	case nil:
		return big.NewInt(0)
	case *big.Int:
		return new(big.Int).Set(v)
	case big.Int:
		return new(big.Int).Set(&v)
	case string:
		if strings.HasPrefix(v, "0x") {
			n, _ := new(big.Int).SetString(strings.TrimPrefix(v, "0x"), 16)
			if n == nil {
				return big.NewInt(0)
			}
			return n
		}
		n, _ := new(big.Int).SetString(v, 10)
		if n == nil {
			return big.NewInt(0)
		}
		return n
	case int:
		return big.NewInt(int64(v))
	case int64:
		return big.NewInt(v)
	case uint64:
		return new(big.Int).SetUint64(v)
	case float64:
		return big.NewInt(int64(v))
	default:
		rv := reflect.ValueOf(value)
		if rv.IsValid() {
			switch rv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return big.NewInt(rv.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return new(big.Int).SetUint64(rv.Uint())
			}
		}
		return big.NewInt(0)
	}
}

func uint256Bytes(value *big.Int) []byte {
	out := make([]byte, 32)
	if value == nil {
		return out
	}
	return value.FillBytes(out)
}

func convertABIValue(typeName string, value any) any {
	switch {
	case typeName == "address":
		return common.HexToAddress(fmt.Sprint(value))
	case typeName == "bytes32":
		return common.HexToHash(fmt.Sprint(value))
	case strings.HasPrefix(typeName, "uint"):
		return bigFromAny(value)
	case typeName == "bytes":
		if s, ok := value.(string); ok && strings.HasPrefix(s, "0x") {
			out, _ := hex.DecodeString(strings.TrimPrefix(s, "0x"))
			return out
		}
		if b, ok := value.([]byte); ok {
			return b
		}
		return []byte(fmt.Sprint(value))
	default:
		return value
	}
}
