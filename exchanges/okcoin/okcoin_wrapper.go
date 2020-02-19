package okcoin

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
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

	if o.Features.Supports.RESTCapabilities.AutoPairUpdates {
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
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},

		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot:   new(currency.PairStore),
			asset.Margin: new(currency.PairStore),
		},
	}

	o.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				CancelOrders:        true,
				SubmitOrder:         true,
				SubmitOrders:        true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	o.Requester = request.New(o.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		// TODO: Specify each individual endpoint rate limits as per docs
		request.NewBasicRateLimit(okCoinRateInterval, okCoinStandardRequestRate),
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
	format, err := o.GetPairFormat(asset.Spot, false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update currencies. Err: %s\n",
			o.Name,
			err)
		return
	}
	enabled, err := o.CurrencyPairs.GetPairs(asset.Spot, true)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update currencies. Err: %s\n",
			o.Name,
			err)
		return
	}

	avail, err := o.CurrencyPairs.GetPairs(asset.Spot, false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update currencies. Err: %s\n",
			o.Name,
			err)
		return
	}

	if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
		!common.StringDataContains(avail.Strings(), format.Delimiter) {
		p, err := currency.NewPairsFromStrings([]string{currency.BTC.String() +
			format.Delimiter +
			currency.USD.String()})
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies.\n",
				o.Name)
		} else {
			log.Warnf(log.ExchangeSys,
				"Enabled pairs for %v reset due to config upgrade, please enable the ones you would like again.\n",
				o.Name)
			forceUpdate = true

			err := o.UpdatePairs(p, asset.Spot, true, true)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies. Err: %s\n",
					o.Name,
					err)
				return
			}
		}
	}

	if !o.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err = o.UpdateTradablePairs(forceUpdate)
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

	format, err := o.GetPairFormat(asset, false)
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range prods {
		pairs = append(pairs, prods[x].BaseCurrency+
			format.Delimiter+
			prods[x].QuoteCurrency)
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
	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return o.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKCoin) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if assetType == asset.Spot {
		resp, err := o.GetSpotAllTokenPairsInformation()
		if err != nil {
			return nil, err
		}
		pairs, err := o.GetEnabledPairs(assetType)
		if err != nil {
			return nil, err
		}
		for i := range pairs {
			for j := range resp {
				if !pairs[i].Equal(resp[j].InstrumentID) {
					continue
				}

				err = ticker.ProcessTicker(&ticker.Price{
					Last:         resp[j].Last,
					High:         resp[j].High24h,
					Low:          resp[j].Low24h,
					Bid:          resp[j].BestBid,
					Ask:          resp[j].BestAsk,
					Volume:       resp[j].BaseVolume24h,
					QuoteVolume:  resp[j].QuoteVolume24h,
					Open:         resp[j].Open24h,
					Pair:         pairs[i],
					LastUpdated:  resp[j].Timestamp,
					ExchangeName: o.Name,
					AssetType:    assetType})
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return ticker.GetTicker(o.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (o *OKCoin) FetchTicker(p currency.Pair, assetType asset.Item) (tickerData *ticker.Price, err error) {
	tickerData, err = ticker.GetTicker(o.Name, p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return
}
