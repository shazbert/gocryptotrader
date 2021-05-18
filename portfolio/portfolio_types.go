package portfolio

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Base holds the portfolio base addresses
type Base struct {
	s *State
	m sync.Mutex
}

// Designation is a type to specify what this address is defined as.
type Designation string

var (
	// DepositDestination defines a portfolio address being a wallet you can
	// withdraw to, the balance of this wallet will not be tracked as this will
	// usually be an exchange owned wallet.
	DepositDestination Designation = "DepositDestination"
	// Exchange defines a portfolio address of being a balance that an api key
	// set would be able to interact and manipulate.
	Exchange Designation = "Exchange"
	// ColdWallet defines a potential client owned address that will be used
	// mainly for long term cryptocurrency storage.
	ColdWallet Designation = "ColdWallet"
	// ColdWallet defines a potential client owned address that will be used
	// mainly for short term cryptocurrency storage.
	HotWallet Designation = "HotWallet"
)

var errInvalidDesignation = errors.New("invalid designation")

// Validate checks designation
func (d *Designation) Validate() error {
	if *d == DepositDestination ||
		*d == Exchange ||
		*d == ColdWallet ||
		*d == HotWallet {
		return nil
	}
	return fmt.Errorf("cannot use %s %w", *d, errInvalidDesignation)
}

// Address sub type holding address information for portfolio
type Address struct {
	Type               Designation
	Address            string
	Account            string
	Asset              asset.Item
	CoinType           currency.Code
	Balance            float64
	Description        string
	WhiteListed        bool
	ColdStorage        bool
	TagRequired        bool
	SupportedExchanges string
}

// EtherchainBalanceResponse holds JSON incoming and outgoing data for
// Etherchain
type EtherchainBalanceResponse struct {
	Status int `json:"status"`
	Data   []struct {
		Address   string      `json:"address"`
		Balance   float64     `json:"balance"`
		Nonce     interface{} `json:"nonce"`
		Code      string      `json:"code"`
		Name      interface{} `json:"name"`
		Storage   interface{} `json:"storage"`
		FirstSeen interface{} `json:"firstSeen"`
	} `json:"data"`
}

// EthplorerResponse holds JSON address data for Ethplorer
type EthplorerResponse struct {
	Address string `json:"address"`
	ETH     struct {
		Balance  float64 `json:"balance"`
		TotalIn  float64 `json:"totalIn"`
		TotalOut float64 `json:"totalOut"`
	} `json:"ETH"`
	CountTxs     int `json:"countTxs"`
	ContractInfo struct {
		CreatorAddress  string `json:"creatorAddress"`
		TransactionHash string `json:"transactionHash"`
		Timestamp       int    `json:"timestamp"`
	} `json:"contractInfo"`
	TokenInfo struct {
		Address        string `json:"address"`
		Name           string `json:"name"`
		Decimals       int    `json:"decimals"`
		Symbol         string `json:"symbol"`
		TotalSupply    string `json:"totalSupply"`
		Owner          string `json:"owner"`
		LastUpdated    int    `json:"lastUpdated"`
		TotalIn        int64  `json:"totalIn"`
		TotalOut       int64  `json:"totalOut"`
		IssuancesCount int    `json:"issuancesCount"`
		HoldersCount   int    `json:"holdersCount"`
		Image          string `json:"image"`
		Description    string `json:"description"`
		Price          struct {
			Rate      int    `json:"rate"`
			Diff      int    `json:"diff"`
			Timestamp int64  `json:"ts"`
			Currency  string `json:"currency"`
		} `json:"price"`
	} `json:"tokenInfo"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ExchangeAccountInfo : Generic type to hold each exchange's holdings in all
// enabled currencies
type ExchangeAccountInfo struct {
	ExchangeName string
	Currencies   []ExchangeAccountCurrencyInfo
}

// ExchangeAccountCurrencyInfo : Sub type to store currency name and value
type ExchangeAccountCurrencyInfo struct {
	CurrencyName string
	TotalValue   float64
	Hold         float64
}

// Coin stores a coin type, balance, address and percentage relative to the total
// amount.
type Coin struct {
	Coin       currency.Code `json:"coin"`
	Balance    float64       `json:"balance"`
	Address    string        `json:"address,omitempty"`
	Percentage float64       `json:"percentage,omitempty"`
}

// OfflineCoinSummary stores a coin types address, balance and percentage
// relative to the total amount.
type OfflineCoinSummary struct {
	Address    string  `json:"address"`
	Balance    float64 `json:"balance"`
	Percentage float64 `json:"percentage,omitempty"`
}

// OnlineCoinSummary stores a coin types balance and percentage relative to the
// total amount.
type OnlineCoinSummary struct {
	Balance    float64 `json:"balance"`
	Percentage float64 `json:"percentage,omitempty"`
}

// Summary Stores the entire portfolio summary
type Summary struct {
	Totals         []Coin                                         `json:"coin_totals"`
	Offline        []Coin                                         `json:"coins_offline"`
	OfflineSummary map[currency.Code][]OfflineCoinSummary         `json:"offline_summary"`
	Online         []Coin                                         `json:"coins_online"`
	OnlineSummary  map[string]map[currency.Code]OnlineCoinSummary `json:"online_summary"`
}

// XRPScanAccount defines the return type for account data
type XRPScanAccount struct {
	Sequence                                  int     `json:"sequence"`
	XRPBalance                                float64 `json:"xrpBalance,string"`
	OwnerCount                                int     `json:"ownerCount"`
	PreviousAffectingTransactionID            string  `json:"previousAffectingTransactionID"`
	PreviousAffectingTransactionLedgerVersion int     `json:"previousAffectingTransactionLedgerVersion"`
	Settings                                  struct {
		RequireDestinationTag bool   `json:"requireDestinationTag"`
		EmailHash             string `json:"emailHash"`
		Domain                string `json:"domain"`
	} `json:"settings"`
	Account        string      `json:"account"`
	Parent         string      `json:"parent"`
	InitialBalance float64     `json:"initial_balance"`
	Inception      time.Time   `json:"inception"`
	LedgerIndex    int         `json:"ledger_index"`
	TxHash         string      `json:"tx_hash"`
	AccountName    AccountInfo `json:"accountName"`
	ParentName     AccountInfo `json:"parentName"`
	Advisory       interface{} `json:"advisory"`
}

// AccountInfo is a XRPScan subtype for account associations
type AccountInfo struct {
	Name        string `json:"name"`
	Description string `json:"desc"`
	Account     string `json:"account"`
	Domain      string `json:"domain"`
	Twitter     string `json:"twitter"`
	Verified    bool   `json:"verified"`
}

// State defines snapshot of total portfolio tracked items and balances
type State struct {
	// Exchanges tracks balances across exchanges
	Exchanges map[string]*[]Holdings `json:"exchangeBalances,omitempty"`
	// ColdWallets tracks deposit wallets that are used for long term storage
	ColdWallets []Wallet `json:"coldWallets"`
	// HotWallets tracks deposit wallets that are used for short term storage
	HotWallets []Wallet `json:"hotWallets"`
	// Deposits track deposit addresses for exchanges
	// TODO: Deprecate because this is only for POC and to visual the deposit
	// addresses, not really need to be tied in with the portfolio but easier
	// to manage as a singular type got the short term.
	Deposits map[string]*[]Wallet `json:"depositAddresses"`
}

// Deposit is a deposit address for an exchange
type Deposit struct {
	Wallet
	// SupportedExchanges defines a comma seperated list of exchanges that can
	// withdraw to this deposit address.
	SupportedExchanges string `json:"supportedExchanges"`
}

// Wallet defines a wallet with address
type Wallet struct {
	Address     string `json:"address"`
	WhiteListed bool   `json:"whiteListed"`
	// TagMemoRequired requirement for a deposit this allows for a check before
	// a withdrawal occurs. This needs to be vetted by end user.
	TagMemoRequired    bool   `json:"tagMemoRequired"`
	TagMemo            string `json:"tagMemo"`
	SupportedExchanges string `json:"supportedExchanges"`
	Account            string `json:"account,omitempty"`
	Holding
}

// Holdings defines a a holdings associated with an account
type Holdings struct {
	Account string `json:"account"`
	Holding
}

// Holding defines the currency type and balance
type Holding struct {
	Currency string `json:"currency"`
	Asset    string `json:"asset"`
	// Balance omitempty used for deposit addresses which will not get tracked
	// as these are usually exchange aggregated accounts.
	Balance float64 `json:"balance,omitempty"`
}
