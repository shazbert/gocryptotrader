package withdraw

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/cache"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

// RequestType used for easy matching of int type to Word
type RequestType uint8

const (
	// Crypto request type
	Crypto RequestType = iota
	// Fiat request type
	Fiat
	// Unknown request type
	Unknown
)

const (
	// ErrStrAmountMustBeGreaterThanZero message to return when requested amount
	// is less than 0
	ErrStrAmountMustBeGreaterThanZero = "amount must be greater than 0"
	// ErrStrAddressisInvalid message to return when address is invalid for
	// crypto request
	ErrStrAddressisInvalid = "address is not valid"
	// ErrStrAddressNotSet message to return when address is empty
	ErrStrAddressNotSet = "address cannot be empty"
	// ErrStrNoCurrencySet message to return when no currency is set
	ErrStrNoCurrencySet = "currency not set"
	// ErrStrCurrencyNotCrypto message to return when requested currency is not
	// crypto
	ErrStrCurrencyNotCrypto = "requested currency is not a cryptocurrency"
	// ErrStrCurrencyNotFiat message to return when requested currency is not
	// fiat
	ErrStrCurrencyNotFiat = "requested currency is not fiat"
	// ErrStrFeeCannotBeNegative message to return when fee amount is negative
	ErrStrFeeCannotBeNegative = "fee amount cannot be negative"
	// ErrStrAddressNotWhiteListed message to return when attempting to withdraw
	// to non-whitelisted address
	ErrStrAddressNotWhiteListed = "address is not whitelisted for withdrawals in config.json in portfolio addresses"
	// ErrStrExchangeNotSupportedByAddress message to return when attempting to
	// withdraw to an unsupported exchange
	ErrStrExchangeNotSupportedByAddress = "address is not supported by exchange in config.json in portfolio addresses"
)

var (
	// ErrRequestCannotBeNil message to return when a request is nil
	ErrRequestCannotBeNil = errors.New("request cannot be nil")
	// ErrExchangeNameUnset message to return when an exchange name is unset
	ErrExchangeNameUnset = errors.New("exchange name unset")
	// ErrInvalidRequest message to return when a request type is invalid
	ErrInvalidRequest = errors.New("invalid request type")
	// CacheSize cache size to use for withdrawal request history
	CacheSize uint64 = 25
	// Cache LRU cache for recent requests
	Cache = cache.New(CacheSize)
	// DryRunID uuid to use for dryruns
	DryRunID, _ = uuid.FromString("3e7e2c25-5a0b-429b-95a1-0960079dce56")
)

// CryptoRequest stores the info required for a crypto withdrawal request
type CryptoRequest struct {
	Address    string
	AddressTag string
	FeeAmount  float64
}

// FiatRequest used for fiat withdrawal requests
type FiatRequest struct {
	Bank banking.Account

	IsExpressWire bool
	// Intermediary bank information
	RequiresIntermediaryBank      bool
	IntermediaryBankAccountNumber float64
	IntermediaryBankName          string
	IntermediaryBankAddress       string
	IntermediaryBankCity          string
	IntermediaryBankCountry       string
	IntermediaryBankPostalCode    string
	IntermediarySwiftCode         string
	IntermediaryBankCode          float64
	IntermediaryIBAN              string
	WireCurrency                  string
}

// Request holds complete details for request
type Request struct {
	Exchange    string        `json:"exchange"`
	Currency    currency.Code `json:"currency"`
	Asset       asset.Item
	Account     string
	Description string      `json:"description"`
	Amount      float64     `json:"amount"`
	Type        RequestType `json:"type"`

	TradePassword   string
	OneTimePassword int64
	PIN             int64

	Crypto CryptoRequest `json:",omitempty"`
	Fiat   FiatRequest   `json:",omitempty"`
}

// Details holds complete details for response transactions and database
// deployment
type Details struct {
	// InternalWithdrawalID is a UUID database identifier for a withdrawal
	// transaction
	InternalWithdrawalID uuid.UUID
	// InternalExchangeID is a UUID database identifier for cross referencing
	// against an exchange
	InternalExchangeID uuid.UUID
	// Response is the individual reply from the exchange
	Response Response
	// The outgoing accepted request for reference
	Request *Request
	// Time at which accepted and deployed in the database
	CreatedAt time.Time
	// Time at which an update occured in the database
	UpdatedAt time.Time
}

// ExchangeResponse holds information returned from an exchange
type Response struct {
	// WithdrawalID is an exchange defined withdrawal ID
	WithdrawalID string
	// TransactionID is a protocol level transaction ID for a blockchain. This
	// can cross reference transaction details by its block explorer.
	TransactionID string
	// Generalisation bucket
	Status string
	// In the event that the request is done via REST and there is no ability
	// for the websocket endpoint to reduce position in holdings this will be
	// set to true
	ReduceAccountHoldings bool
}
