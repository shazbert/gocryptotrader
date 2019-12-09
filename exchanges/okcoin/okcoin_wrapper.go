package okcoin

import (
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (o *OKCoin) GetDefaultConfig() (*config.ExchangeConfig, error) {
	o.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = o.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = o.BaseCurrencies

	err := o.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if o.Features.REST.AutoPairUpdatesEnabled() {
		err = o.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults method assignes the default values for OKEX
func (o *OKCoin) SetDefaults() {
	o.SetErrorDefaults()
	o.SetCheckVarDefaults()
	o.Name = okCoinExchangeName
	o.Enabled = true
	o.Verbose = true

	o.API.CredentialsValidator.RequiresKey = true
	o.API.CredentialsValidator.RequiresSecret = true
	o.API.CredentialsValidator.RequiresClientID = true

	o.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
			asset.Margin,
		},

		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},

		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
	}

	withdrawPermissions := exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals

	globalRate := protocol.GetNewGlobalRate(time.Second,
		time.Second,
		okCoinAuthRate,
		okCoinUnauthRate)

	o.Features = &protocol.Features{
		REST: &protocol.Components{
			Enabled:             true,
			TickerBatching:      protocol.SetNewComponent(globalRate, true, false),
			TickerFetching:      protocol.SetNewComponent(globalRate, true, false),
			KlineFetching:       protocol.SetNewComponent(globalRate, true, false),
			TradeFetching:       protocol.SetNewComponent(globalRate, true, false),
			OrderbookFetching:   protocol.SetNewComponent(globalRate, true, false),
			AutoPairUpdates:     protocol.SetNewComponent(globalRate, true, false),
			AccountInfo:         protocol.SetNewComponent(globalRate, true, true),
			GetOrder:            protocol.SetNewComponent(globalRate, true, true),
			GetOrders:           protocol.SetNewComponent(globalRate, true, true),
			CancelOrder:         protocol.SetNewComponent(globalRate, true, true),
			CancelOrders:        protocol.SetNewComponent(globalRate, true, true),
			SubmitOrder:         protocol.SetNewComponent(globalRate, true, true),
			SubmitOrders:        protocol.SetNewComponent(globalRate, true, true),
			DepositHistory:      protocol.SetNewComponent(globalRate, true, true),
			WithdrawalHistory:   protocol.SetNewComponent(globalRate, true, true),
			UserTradeHistory:    protocol.SetNewComponent(globalRate, true, true),
			CryptoDeposit:       protocol.SetNewComponent(globalRate, true, true),
			CryptoWithdrawal:    protocol.SetNewComponent(globalRate, true, true),
			TradeFee:            protocol.SetNewComponent(globalRate, true, true),
			CryptoWithdrawalFee: protocol.SetNewComponent(globalRate, true, true),
			Withdraw:            &withdrawPermissions,
		},
		Websocket: &protocol.Components{
			Enabled:                true,
			TickerFetching:         protocol.SetNewComponentNoRate(true, false),
			TradeFetching:          protocol.SetNewComponentNoRate(true, false),
			KlineFetching:          protocol.SetNewComponentNoRate(true, false),
			OrderbookFetching:      protocol.SetNewComponentNoRate(true, false),
			Subscribe:              protocol.SetNewComponentNoRate(true, false),
			Unsubscribe:            protocol.SetNewComponentNoRate(true, false),
			AuthenticatedEndpoints: protocol.SetNewComponentNoRate(true, true),
			MessageCorrelation:     protocol.SetNewComponentNoRate(true, false),
		},
	}

	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second, okCoinAuthRate),
		request.NewRateLimit(time.Second, okCoinUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
	)

	o.API.Endpoints.URLDefault = okCoinAPIURL
	o.API.Endpoints.URL = okCoinAPIURL
	o.API.Endpoints.WebsocketURL = okCoinWebsocketURL
	o.APIVersion = okCoinAPIVersion
	o.Websocket = wshandler.New()
	o.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	o.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	o.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Start starts the OKGroup go routine
func (o *OKCoin) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (o *OKCoin) Run() {
	if o.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			o.Name,
			common.IsEnabled(o.Websocket.IsEnabled()),
			o.WebsocketURL)
	}

	forceUpdate := false
	delim := o.GetPairFormat(asset.Spot, false).Delimiter
	if !common.StringDataContains(o.CurrencyPairs.GetPairs(asset.Spot,
		true).Strings(), delim) ||
		!common.StringDataContains(o.CurrencyPairs.GetPairs(asset.Spot,
			false).Strings(), delim) {
		enabledPairs := currency.NewPairsFromStrings([]string{
			fmt.Sprintf("BTC%sUSD", delim),
		})
		log.Warnf(log.ExchangeSys,
			"Enabled pairs for %v reset due to config upgrade, please enable the ones you would like again.\n",
			o.Name)
		forceUpdate = true

		err := o.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies.\n",
				o.Name)
			return
		}
	}

	if !o.Features.REST.AutoPairUpdatesEnabled() && !forceUpdate {
		return
	}

	err := o.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			o.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (o *OKCoin) FetchTradablePairs(asset asset.Item) ([]string, error) {
	prods, err := o.GetSpotTokenPairDetails()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range prods {
		pairs = append(pairs, fmt.Sprintf("%v%v%v", prods[x].BaseCurrency,
			o.GetPairFormat(asset, false).Delimiter, prods[x].QuoteCurrency))
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (o *OKCoin) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := o.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return o.UpdatePairs(currency.NewPairsFromStrings(pairs),
		asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKCoin) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var tickerData ticker.Price
	if assetType == asset.Spot {
		resp, err := o.GetSpotAllTokenPairsInformation()
		if err != nil {
			return tickerData, err
		}
		pairs := o.GetEnabledPairs(assetType)
		for i := range pairs {
			for j := range resp {
				if !pairs[i].Equal(resp[j].InstrumentID) {
					continue
				}
				tickerData = ticker.Price{
					Last:        resp[j].Last,
					High:        resp[j].High24h,
					Low:         resp[j].Low24h,
					Bid:         resp[j].BestBid,
					Ask:         resp[j].BestAsk,
					Volume:      resp[j].BaseVolume24h,
					QuoteVolume: resp[j].QuoteVolume24h,
					Open:        resp[j].Open24h,
					Pair:        pairs[i],
					LastUpdated: resp[j].Timestamp,
				}
				err = ticker.ProcessTicker(o.Name, &tickerData, assetType)
				if err != nil {
					log.Error(log.Ticker, err)
				}
			}
		}
	}
	return ticker.GetTicker(o.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (o *OKCoin) FetchTicker(p currency.Pair, assetType asset.Item) (tickerData ticker.Price, err error) {
	tickerData, err = ticker.GetTicker(o.Name, p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return
}
