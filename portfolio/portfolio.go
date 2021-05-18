package portfolio

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	cryptoIDAPIURL = "https://chainz.cryptoid.info"
	xrpScanAPIURL  = "https://api.xrpscan.com/api/v1/account/"

	ethplorerAPIURL      = "https://api.ethplorer.io"
	ethplorerAddressInfo = "getAddressInfo"

	// PortfolioAddressExchange is a label for an exchange address
	PortfolioAddressExchange = "Exchange"
	// PortfolioAddressPersonal is a label for a personal/offline address
	PortfolioAddressPersonal = "Personal"
)

var (
	errExchangeIsEmpty           = errors.New("exchange name not set")
	errHoldingsIsNil             = errors.New("holdings is nil")
	errNotEthAddress             = errors.New("not an Ethereum address")
	errInvalidAddress            = errors.New("invalid address")
	errWalletIsNil               = errors.New("wallet is nil")
	errAssetIsEmpty              = errors.New("asset is empty")
	errAccountIsEmpty            = errors.New("account is empty")
	errInvalidBalance            = errors.New("balance amount is invalid")
	errAddressIsEmpty            = errors.New("address is empty")
	errCurrencyIsEmpty           = errors.New("currency not set")
	errExchangePortfolioNotFound = errors.New("exchange portfolio not found")
	errPortfolioAlreadySeeded    = errors.New("portfolio addresses already seeded")
	errAddressCannotMatch        = errors.New("address cannot match")
	errNoDepositAddressesFound   = errors.New("no deposit addresses found for exchange")
	errDepositAddressNotFound    = errors.New("deposit address not found")
	errNoBalanceReturned         = errors.New("no balance info returned")
)

// Portfolio is variable store holding an array of portfolioAddress
var Portfolio Base

// Verbose allows for debug output when sending an http request
var Verbose bool

// GetEthereumBalance single or multiple address information as
// EtherchainBalanceResponse
func GetEthereumBalance(address string) (EthplorerResponse, error) {
	valid, _ := common.IsValidCryptoAddress(address, "eth")
	if !valid {
		return EthplorerResponse{}, errNotEthAddress
	}

	urlPath := fmt.Sprintf("%s/%s/%s?apiKey=freekey",
		ethplorerAPIURL,
		ethplorerAddressInfo,
		address)

	result := EthplorerResponse{}
	return result, common.SendHTTPGetRequest(urlPath, true, Verbose, &result)
}

// GetCryptoIDAddress queries CryptoID for an address balance for a
// specified cryptocurrency
func GetCryptoIDAddress(address string, coinType currency.Code) (float64, error) {
	valid, _ := common.IsValidCryptoAddress(address, coinType.String())
	if !valid {
		return 0, errInvalidAddress
	}

	var result float64
	url := fmt.Sprintf("%s/%s/api.dws?q=getbalance&a=%s",
		cryptoIDAPIURL,
		coinType.Lower(),
		address)

	err := common.SendHTTPGetRequest(url, true, Verbose, &result)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// GetRippleBalance returns the value for a ripple address
func GetRippleBalance(address string) (float64, error) {
	var result XRPScanAccount
	err := common.SendHTTPGetRequest(xrpScanAPIURL+address, true, Verbose, &result)
	if err != nil {
		return 0, err
	}

	if (result == XRPScanAccount{}) {
		return 0, errNoBalanceReturned
	}

	return result.XRPBalance, nil
}

// GetAddressBalance acceses the portfolio base and returns the balance by passed
// in address, coin type and description
func (p *Base) GetAddressBalance(address, description string, coinType currency.Code) (float64, bool) {
	p.m.Lock()
	defer p.m.Unlock()
	for x := range p.s.HotWallets {
		if p.s.HotWallets[x].Address == address &&
			p.s.HotWallets[x].Currency == coinType.String() {
			return p.s.HotWallets[x].Balance, true
		}
	}
	return 0, false
}

// // ExchangeExists checks to see if an exchange exists in the portfolio base
// func (p *Base) ExchangeExists(exchangeName string) bool {
// 	p.m.Lock()
// 	defer p.m.Unlock()
// 	for exch := range p.s.Deposits {
// 		if exch == exchangeName {
// 			return true
// 		}
// 	}
// 	return false
// }

// // AddressExists checks to see if there is an address associated with the
// // portfolio base
// func (p *Base) AddressExists(address string) bool {
// 	p.m.Lock()
// 	defer p.m.Unlock()
// 	for x := range p.s.HotWallets {
// 		if p.s.HotWallets[x].Address == address {
// 			return true
// 		}
// 	}
// 	return false
// }

// Validate checks holdings values
func (h *Holdings) Validate() error {
	if h == nil {
		return errHoldingsIsNil
	}

	if h.Account == "" {
		return errAccountIsEmpty
	}

	if h.Asset == "" {
		return errAssetIsEmpty
	}

	if h.Currency == "" {
		return errCurrencyIsEmpty
	}

	if h.Balance < 0 {
		return errInvalidBalance
	}

	return nil
}

// UpdateInsertExchangeBalance if found will update holdings, if not found will
// insert balance.
func (p *Base) UpdateInsertExchangeBalance(exch string, h *Holdings) error {
	if exch == "" {
		return errExchangeIsEmpty
	}

	err := h.Validate()
	if err != nil {
		return err
	}

	p.m.Lock()
	defer p.m.Unlock()
	val, ok := p.s.Exchanges[exch]
	if !ok {
		if h.Balance == 0 {
			return nil
		}
		p.s.Exchanges[exch] = &[]Holdings{*h}
		return nil
	}

	for x := range *val {
		if (*val)[x].Account == h.Account &&
			(*val)[x].Asset == h.Asset &&
			(*val)[x].Currency == h.Currency { // Check for match
			if (*val)[x].Balance == 0 {
				// If found remove entry as its not needed
				(*val)[x] = (*val)[len((*val))-1]
				(*val) = (*val)[:len((*val))-1]
			} else {
				// Adjust balance
				(*val)[x].Balance = h.Balance
			}
			return nil
		}
	}

	if h.Balance != 0 { // If balance is zero we can skip here
		*val = append(*val, *h)
	}
	return nil
}

//  Validate checks wallet data and returns error on incorrect values
func (w *Wallet) Validate() error {
	if w == nil {
		return errWalletIsNil
	}

	if w.Address == "" {
		return errAddressIsEmpty
	}

	if w.Asset == "" {
		return errAssetIsEmpty
	}

	if w.Balance < 0 {
		return errInvalidBalance
	}
	return nil
}

// UpdateInsertDepositAddress adds or updates a new deposit address for an
// exchange for a potential destination to an exchange.
func (p *Base) UpdateInsertDepositAddress(exch string, w *Wallet) error {
	if exch == "" {
		return errExchangeIsEmpty
	}

	err := w.Validate()
	if err != nil {
		return err
	}

	p.m.Lock()
	defer p.m.Unlock()
	// NOTE: In this context balance is not needed as these addresses are
	// exchange hot wallets.
	val, ok := p.s.Deposits[exch]
	if !ok {
		p.s.Deposits[exch] = &[]Wallet{*w}
		return nil
	}

	for x := range *val { // search for loaded address
		if (*val)[x].Address == w.Address && (*val)[x].Currency == w.Currency {
			// Address and currency should be the only two fields that are
			// needed to be checked as it will act as a unique ID associated
			// with the crypto.
			(*val)[x] = *w
			return nil
		}
	}
	// insert wallet address
	*val = append(*val, *w)
	return nil
}

// UpdateInsertColdWallet adds or updates a cold wallet for long term storage
func (p *Base) UpdateInsertColdWallet(w *Wallet) error {
	err := w.Validate()
	if err != nil {
		return err
	}

	if !w.WhiteListed {
		log.Warnf(log.PortfolioMgr,
			"Cannot withdraw to wallet %s for %s as it is not whitelisted.",
			w.Address,
			w.Currency)
	}

	for x := range p.s.ColdWallets {
		if p.s.ColdWallets[x].Address == w.Address &&
			p.s.ColdWallets[x].Currency == w.Currency {
			// Address and currency should be the only two fields that are
			// needed to be checked as it will act as a unique ID associated
			// with the crypto.
			p.s.ColdWallets[x] = *w
		}
	}
	// insert wallet address
	p.s.ColdWallets = append(p.s.ColdWallets, *w)
	return nil
}

// UpdateInsertHotWallet adds or updates a hot wallet for short term storage
func (p *Base) UpdateInsertHotWallet(w *Wallet) error {
	err := w.Validate()
	if err != nil {
		return err
	}

	if !w.WhiteListed {
		log.Warnf(log.PortfolioMgr,
			"Cannot withdraw to wallet %s for %s as it is not whitelisted",
			w.Address,
			w.Currency)
	}

	for x := range p.s.HotWallets {
		if p.s.HotWallets[x].Address == w.Address &&
			p.s.HotWallets[x].Currency == w.Currency {
			// Address and currency should be the only two fields that are
			// needed to be checked as it will act as a unique ID associated
			// with the crypto.
			log.Debugf(log.PortfolioMgr,
				"Updating hot wallet: '%s' '%s' '%s' '%s' entry with balance '%f'.\n",
				w.Address,
				w.Account,
				w.Asset,
				w.Currency,
				w.Balance)
			p.s.HotWallets[x] = *w
			return nil
		}
	}
	log.Debugf(log.PortfolioMgr,
		"Inserting hot wallet: '%s' '%s' '%s' '%s' entry with balance '%f'.\n",
		w.Address,
		w.Account,
		w.Asset,
		w.Currency,
		w.Balance)
	// insert wallet address
	p.s.HotWallets = append(p.s.HotWallets, *w)
	return nil
}

// UpdatePortfolio adds to the portfolio addresses by coin type
func (p *Base) UpdatePortfolio(addresses []string, coinType currency.Code) error {
	p.m.Lock()
	defer p.m.Unlock()
	if strings.Contains(strings.Join(addresses, ","), PortfolioAddressExchange) ||
		strings.Contains(strings.Join(addresses, ","), PortfolioAddressPersonal) {
		return nil
	}

	switch coinType {
	case currency.ETH:
		for x := range addresses {
			result, err := GetEthereumBalance(addresses[x])
			if err != nil {
				return err
			}

			if result.Error.Message != "" {
				return errors.New(result.Error.Message)
			}

			// err = p.AddAddress(addresses[x],
			// 	PortfolioAddressPersonal,
			// 	coinType,
			// 	result.ETH.Balance)
			// if err != nil {
			// 	return err
			// }
		}
	case currency.XRP:
		// for x := range addresses {
		// 	// result, err := GetRippleBalance(addresses[x])
		// 	// if err != nil {
		// 	// 	return err
		// 	// }
		// 	// err = p.AddAddress(addresses[x],
		// 	// 	PortfolioAddressPersonal,
		// 	// 	coinType,
		// 	// 	result)
		// 	// if err != nil {
		// 	// 	return err
		// 	// }
		// }
	default:
		// for x := range addresses {
		// 	result, err := GetCryptoIDAddress(addresses[x], coinType)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	err = p.AddAddress(addresses[x],
		// 		PortfolioAddressPersonal,
		// 		coinType,
		// 		result)
		// 	if err != nil {
		// 		return err
		// 	}
		// }
	}
	return nil
}

// GetPortfolioByExchange returns currency portfolio amount by exchange
func (p *Base) GetPortfolioByExchange(exch string) (map[currency.Code]float64, error) {
	if exch == "" {
		return nil, errExchangeIsEmpty
	}

	p.m.Lock()
	defer p.m.Unlock()
	holdings, ok := p.s.Exchanges[exch]
	if !ok {
		return nil, errExchangePortfolioNotFound
	}

	result := make(map[currency.Code]float64)
	for x := range *holdings {
		code := currency.NewCode((*holdings)[x].Currency)
		// Amounts could be spread across acounts, for now amalgamate.
		result[code] = (*holdings)[x].Balance + result[code]
	}
	return result, nil
}

// GetCombinedExchangesPortfolio returns portfolio map for aggregated balances
// for all matching currency items for a full snapshot.
func (p *Base) getCombinedExchangesPortfolio() map[currency.Code]float64 {
	result := make(map[currency.Code]float64)
	for _, holdings := range p.s.Exchanges {
		for x := range *holdings {
			code := currency.NewCode((*holdings)[x].Currency)
			result[code] = (*holdings)[x].Balance + result[code]
		}
	}
	return result
}

// GetPersonalPortfolio returns current portfolio base information
func (p *Base) getPersonalPortfolio() map[currency.Code]float64 {
	result := make(map[currency.Code]float64)
	for x := range p.s.ColdWallets {
		code := currency.NewCode(p.s.ColdWallets[x].Currency)
		result[code] = p.s.ColdWallets[x].Balance + result[code]
	}
	for x := range p.s.HotWallets {
		code := currency.NewCode(p.s.HotWallets[x].Currency)
		result[code] = p.s.HotWallets[x].Balance + result[code]
	}
	return result
}

// getPercentage returns the percentage of the target coin amount against the
// total coin amount.
func getPercentage(input, totals map[currency.Code]float64, target currency.Code) float64 {
	subtotal := input[target]
	total := totals[target]
	percentage := (subtotal / total) * 100 / 1
	return percentage
}

// getPercentageSpecific returns the percentage a specific value of a target
// coin amount against the total coin amount.
func getPercentageSpecific(input float64, target currency.Code, totals map[currency.Code]float64) float64 {
	total := totals[target]
	percentage := (input / total) * 100 / 1
	return percentage
}

// GetPortfolioSummary returns the complete portfolio summary, showing coin
// totals, offline and online summaries with their relative percentages.
func (p *Base) GetPortfolioSummary() Summary {
	p.m.Lock()
	defer p.m.Unlock()

	totalCoins := make(map[currency.Code]float64)

	personalHoldings := p.getPersonalPortfolio()
	for c, bal := range personalHoldings {
		totalCoins[c] = bal
	}

	exchangeHoldings := p.getCombinedExchangesPortfolio()
	for code, balance := range exchangeHoldings {
		totalCoins[code] = balance + totalCoins[code]
	}

	var output Summary
	for code, balance := range totalCoins {
		output.Totals = append(output.Totals, Coin{Coin: code, Balance: balance})
	}

	for code, balance := range personalHoldings {
		output.Offline = append(output.Offline, Coin{
			Coin:       code,
			Balance:    balance,
			Percentage: getPercentage(personalHoldings, totalCoins, code),
		})
	}

	for code, balance := range exchangeHoldings {
		output.Online = append(output.Online, Coin{
			Coin:       code,
			Balance:    balance,
			Percentage: getPercentage(exchangeHoldings, totalCoins, code),
		})
	}

	exchangeSummary := make(map[string]map[currency.Code]OnlineCoinSummary)
	for exch := range p.s.Exchanges {
		result, err := p.GetPortfolioByExchange(exch)
		if err != nil {
			continue
		}

		coinSummary := make(map[currency.Code]OnlineCoinSummary)
		for code, balance := range result {
			coinSummary[code] = OnlineCoinSummary{
				Balance:    balance,
				Percentage: getPercentageSpecific(balance, code, totalCoins),
			}
		}
		exchangeSummary[exch] = coinSummary
	}
	output.OnlineSummary = exchangeSummary

	offlineSummary := make(map[currency.Code][]OfflineCoinSummary)
	setSummary(p.s.ColdWallets, &offlineSummary, totalCoins)
	setSummary(p.s.HotWallets, &offlineSummary, totalCoins)
	output.OfflineSummary = offlineSummary
	return output
}

func setSummary(wallet []Wallet, sum *map[currency.Code][]OfflineCoinSummary, totalCoins map[currency.Code]float64) {
	for x := range wallet {
		code := currency.NewCode(wallet[x].Currency)
		(*sum)[code] = append((*sum)[code], OfflineCoinSummary{
			Address: wallet[x].Address,
			Balance: wallet[x].Balance,
			Percentage: getPercentageSpecific(wallet[x].Balance,
				code,
				totalCoins),
		})
	}
}

// GetPortfolioGroupedCoin returns portfolio base information grouped by coin
func (p *Base) GetPortfolioGroupedCoin() map[currency.Code][]string {
	p.m.Lock()
	defer p.m.Unlock()
	result := make(map[currency.Code][]string)
	for x := range p.s.ColdWallets {
		c := currency.NewCode(p.s.ColdWallets[x].Currency)
		result[c] = append(result[c], p.s.ColdWallets[x].Address)
	}
	for x := range p.s.HotWallets {
		c := currency.NewCode(p.s.HotWallets[x].Currency)
		result[c] = append(result[c], p.s.HotWallets[x].Address)
	}
	return result
}

// Seed loads a new portfolio state, will error if already seeded
func (p *Base) Seed(port State) error {
	p.m.Lock()
	defer p.m.Unlock()
	if p.s != nil {
		return errPortfolioAlreadySeeded
	}
	p.s = &port
	return nil
}

// GetExchangeCount returns the exchange count
func (p *Base) GetExchangeCount() int {
	p.m.Lock()
	defer p.m.Unlock()
	return len(p.s.Exchanges)
}

// GetHotWalletsCount returns the hot wallets count
func (p *Base) GetHotWalletsCount() int {
	p.m.Lock()
	defer p.m.Unlock()
	return len(p.s.HotWallets)
}

// GetColdWalletsCount returns the cold wallets count
func (p *Base) GetColdWalletsCount() int {
	p.m.Lock()
	defer p.m.Unlock()
	return len(p.s.ColdWallets)
}

// StartPortfolioWatcher observes the portfolio object
func StartPortfolioWatcher() {
	log.Debugf(log.PortfolioMgr,
		"PortfolioWatcher started: Monitoring over %d exchange(s), %d hot wallet(s) & %d cold wallet(s) entries.\n",
		Portfolio.GetExchangeCount(),
		Portfolio.GetHotWalletsCount(),
		Portfolio.GetColdWalletsCount())

	for {
		data := Portfolio.GetPortfolioGroupedCoin()
		for code, offlineAddresses := range data {
			err := Portfolio.UpdatePortfolio(offlineAddresses, code)
			if err != nil {
				log.Errorf(log.PortfolioMgr,
					"PortfolioWatcher error %s for currency %s, val %v\n",
					err,
					code,
					offlineAddresses)
				continue
			}

			log.Debugf(log.PortfolioMgr,
				"PortfolioWatcher: Successfully updated address balance for %s address(es) %s\n",
				code,
				offlineAddresses)
		}
		time.Sleep(time.Minute * 10)
	}
}

// GetPortfolio returns a pointer to the portfolio base
func GetPortfolio() *Base {
	return &Portfolio
}

// Supported checks if an address that is matched is allowed to be deposited to
// by calling exchange.
func (b *Base) Supported(exch string, address string, c currency.Code) bool {
	b.m.Lock()
	defer b.m.Unlock()
	for x := range b.s.ColdWallets {
		if b.s.ColdWallets[x].Address == address &&
			b.s.ColdWallets[x].Currency == c.String() {
			return strings.Contains(b.s.ColdWallets[x].SupportedExchanges, exch)
		}
	}

	for x := range b.s.HotWallets {
		if b.s.HotWallets[x].Address == address &&
			b.s.HotWallets[x].Currency == c.String() {
			return strings.Contains(b.s.HotWallets[x].SupportedExchanges, exch)
		}
	}
	return false
}

// IsExchangeSupported checks if exchange is supported by portfolio address
func IsExchangeSupported(exch, address string, c currency.Code) bool {
	return Portfolio.Supported(exch, address, c)
}

// ColdStorage checks to see if an address is a cold storage wallet
func (b *Base) ColdStorage(address string, c currency.Code) bool {
	b.m.Lock()
	defer b.m.Unlock()
	for x := range b.s.ColdWallets {
		if b.s.ColdWallets[x].Address == address &&
			b.s.ColdWallets[x].Currency == c.String() {
			return true
		}
	}
	return false
}

// IsColdStorage checks if address is a cold storage wallet
func IsColdStorage(address string, c currency.Code) bool {
	return Portfolio.ColdStorage(address, c)
}

// WhiteListed checks to see if an address is white listed for interaction
func (b *Base) WhiteListed(address string, c currency.Code) bool {
	b.m.Lock()
	defer b.m.Unlock()
	for x := range b.s.ColdWallets {
		if b.s.ColdWallets[x].Address == address &&
			b.s.ColdWallets[x].Currency == c.String() {
			return b.s.ColdWallets[x].WhiteListed
		}
	}
	for x := range b.s.HotWallets {
		if b.s.HotWallets[x].Address == address &&
			b.s.HotWallets[x].Currency == c.String() {
			return b.s.HotWallets[x].WhiteListed
		}
	}
	return false
}

// IsWhiteListed checks if address is whitelisted for withdraw transfers
func IsWhiteListed(address string, c currency.Code) bool {
	return Portfolio.WhiteListed(address, c)
}

// IsTagOrMemoRequired checks if address is needs a tag or memo
func (b *Base) TagOrMemoRequired(address string, c currency.Code) (bool, error) {
	b.m.Lock()
	defer b.m.Unlock()
	for x := range b.s.ColdWallets {
		if b.s.ColdWallets[x].Address == address &&
			b.s.ColdWallets[x].Currency == c.String() {
			return b.s.ColdWallets[x].TagMemoRequired, nil
		}
	}
	for x := range b.s.HotWallets {
		if b.s.HotWallets[x].Address == address &&
			b.s.HotWallets[x].Currency == c.String() {
			return b.s.HotWallets[x].TagMemoRequired, nil
		}
	}
	return false, errAddressCannotMatch
}

// IsTagOrMemoRequired checks if address is needs a tag or memo
func IsTagOrMemoRequired(address string, c currency.Code) (bool, error) {
	return Portfolio.TagOrMemoRequired(address, c)
}

// GetState returns portfolio state for configuration
func (b *Base) GetState() State {
	return *b.s
}

// GetDepositAddressByExchange returns a deposit address for the specified
// exchange and cryptocurrency if it exists
func (b *Base) GetDepositAddressByExchange(exch string, c currency.Code) (string, error) {
	if exch == "" {
		return "", errExchangeIsEmpty
	}

	if c.String() == "" {
		return "", errCurrencyIsEmpty
	}

	b.m.Lock()
	defer b.m.Unlock()
	val, ok := b.s.Deposits[exch]
	if !ok {
		return "", errNoDepositAddressesFound
	}

	for x := range *val {
		if (*val)[x].Currency == c.String() {
			// TODO: Add in tag requirements, need to return full info. When we
			// upgrade to a multikey situation we will have different tag
			// requirements per key so maybe a keychain map might be needed.
			return (*val)[x].Address, nil
		}
	}
	return "", errDepositAddressNotFound
}

// GetDepositAddressesByExchange returns a list of cryptocurrency addresses for
// the specified exchange if they exist
func (b *Base) GetDepositAddressesByExchange(exch string) (map[string]string, error) {
	if exch == "" {
		return nil, errExchangeIsEmpty
	}

	b.m.Lock()
	defer b.m.Unlock()

	val, ok := b.s.Deposits[exch]
	if !ok {
		return nil, errNoDepositAddressesFound
	}

	r := make(map[string]string)
	for x := range *val {
		r[(*val)[x].Currency] = (*val)[x].Address
	}
	return r, nil
}

// LoadDepositAddress inserts the manageable deposit address for an exchange
// account.
func (b *Base) LoadDepositAddress(exch, acc, addr, tagMemo, curr string) error {
	b.m.Lock()
	defer b.m.Unlock()

	w := Wallet{
		Address:     addr,
		WhiteListed: true, // These will be automatically white listed because
		// these are specifically tied to the exchange.
		TagMemoRequired:    tagMemo != "",
		SupportedExchanges: "ALL", // Can allow all enables exchanges to
		// withdraw to these addresses as needed.
		TagMemo: tagMemo,
		Account: acc,
		Holding: Holding{
			Currency: curr,
			Asset:    asset.Spot.String(),
		},
	}

	if b.s.Deposits == nil {
		b.s.Deposits = make(map[string]*[]Wallet)
	}

	addresses, ok := b.s.Deposits[exch]
	if !ok {
		// Deploy first instance
		b.s.Deposits[exch] = &[]Wallet{w}
		return nil
	}

	for x := range *addresses {
		if (*addresses)[x].Address == addr &&
			(*addresses)[x].Account == acc &&
			(*addresses)[x].Currency == curr {
			(*addresses)[x] = w // Update
			return nil
		}
	}
	*addresses = append(*addresses, w)
	return nil
}
