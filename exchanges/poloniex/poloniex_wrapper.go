package poloniex

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (p *Poloniex) GetDefaultConfig() (*config.ExchangeConfig, error) {
	p.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = p.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = p.BaseCurrencies

	err := p.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if p.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = p.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default settings for poloniex
func (p *Poloniex) SetDefaults() {
	p.Name = "Poloniex"
	p.Enabled = true
	p.Verbose = true
	p.API.CredentialsValidator.RequiresKey = true
	p.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}

	configFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}

	err := p.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	p.Features = exchange.Features{
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
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.TwoHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.OneDay.Word():     true,
				},
			},
		},
	}

	p.Requester = request.New(p.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	p.API.Endpoints = p.NewEndpoints()
	err = p.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      poloniexAPIURL,
		exchange.WebsocketSpot: poloniexWebsocketAddress,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	p.Websocket = stream.New()
	p.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	p.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	p.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (p *Poloniex) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		p.SetEnabled(false)
		return nil
	}

	err := p.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := p.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = p.Websocket.Setup(&stream.WebsocketSetup{
		Enabled:                          exch.Features.Enabled.Websocket,
		Verbose:                          exch.Verbose,
		AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
		WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
		DefaultURL:                       poloniexWebsocketAddress,
		ExchangeName:                     exch.Name,
		RunningURL:                       wsRunningURL,
		Connector:                        p.WsConnect,
		Subscriber:                       p.Subscribe,
		UnSubscriber:                     p.Unsubscribe,
		GenerateSubscriptions:            p.GenerateDefaultSubscriptions,
		Features:                         &p.Features.Supports.WebsocketCapabilities,
		OrderbookBufferLimit:             exch.OrderbookConfig.WebsocketBufferLimit,
		BufferEnabled:                    exch.OrderbookConfig.WebsocketBufferEnabled,
		SortBuffer:                       true,
		SortBufferByUpdateIDs:            true,
	})
	if err != nil {
		return err
	}

	return p.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the Poloniex go routine
func (p *Poloniex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		p.Run()
		wg.Done()
	}()
}

// Run implements the Poloniex wrapper
func (p *Poloniex) Run() {
	if p.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s (url: %s).\n",
			p.Name,
			common.IsEnabled(p.Websocket.IsEnabled()),
			poloniexWebsocketAddress)
		p.PrintEnabledPairs()
	}

	forceUpdate := false

	avail, err := p.GetAvailablePairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			p.Name,
			err)
		return
	}

	if common.StringDataCompare(avail.Strings(), "BTC_USDT") {
		log.Warnf(log.ExchangeSys,
			"%s contains invalid pair, forcing upgrade of available currencies.\n",
			p.Name)
		forceUpdate = true
	}

	if !p.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err = p.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			p.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (p *Poloniex) FetchTradablePairs(asset asset.Item) ([]string, error) {
	resp, err := p.GetTicker()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range resp {
		currencies = append(currencies, x)
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (p *Poloniex) UpdateTradablePairs(forceUpgrade bool) error {
	pairs, err := p.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}
	ps, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return p.UpdatePairs(ps, asset.Spot, false, forceUpgrade)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (p *Poloniex) UpdateTicker(currencyPair currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := p.GetTicker()
	if err != nil {
		return nil, err
	}

	enabledPairs, err := p.GetEnabledPairs(assetType)
	if err != nil {
		return nil, err
	}
	for i := range enabledPairs {
		fpair, err := p.FormatExchangeCurrency(enabledPairs[i], assetType)
		if err != nil {
			return nil, err
		}
		curr := fpair.String()
		if _, ok := tick[curr]; !ok {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Pair:         enabledPairs[i],
			Ask:          tick[curr].LowestAsk,
			Bid:          tick[curr].HighestBid,
			High:         tick[curr].High24Hr,
			Last:         tick[curr].Last,
			Low:          tick[curr].Low24Hr,
			Volume:       tick[curr].BaseVolume,
			QuoteVolume:  tick[curr].QuoteVolume,
			ExchangeName: p.Name,
			AssetType:    assetType})
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(p.Name, currencyPair, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (p *Poloniex) FetchTicker(currencyPair currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(p.Name, currencyPair, assetType)
	if err != nil {
		return p.UpdateTicker(currencyPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (p *Poloniex) FetchOrderbook(currencyPair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(p.Name, currencyPair, assetType)
	if err != nil {
		return p.UpdateOrderbook(currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (p *Poloniex) UpdateOrderbook(c currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	callingBook := &orderbook.Base{
		Exchange:        p.Name,
		Pair:            c,
		Asset:           assetType,
		VerifyOrderbook: p.CanVerifyOrderbook,
	}
	orderbookNew, err := p.GetOrderbook("", poloniexMaxOrderbookDepth)
	if err != nil {
		return callingBook, err
	}

	enabledPairs, err := p.GetEnabledPairs(assetType)
	if err != nil {
		return callingBook, err
	}
	for i := range enabledPairs {
		book := &orderbook.Base{
			Exchange:        p.Name,
			Pair:            enabledPairs[i],
			Asset:           assetType,
			VerifyOrderbook: p.CanVerifyOrderbook,
		}

		fpair, err := p.FormatExchangeCurrency(enabledPairs[i], assetType)
		if err != nil {
			return book, err
		}
		data, ok := orderbookNew.Data[fpair.String()]
		if !ok {
			continue
		}

		for y := range data.Bids {
			book.Bids = append(book.Bids, orderbook.Item{
				Amount: data.Bids[y].Amount,
				Price:  data.Bids[y].Price,
			})
		}

		for y := range data.Asks {
			book.Asks = append(book.Asks, orderbook.Item{
				Amount: data.Asks[y].Amount,
				Price:  data.Asks[y].Price,
			})
		}
		err = book.Process()
		if err != nil {
			return book, err
		}
	}
	return orderbook.Get(p.Name, c, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Poloniex exchange
func (p *Poloniex) UpdateAccountInfo() (account.FullSnapshot, error) {
	accountBalance, err := p.GetCompleteBalances()
	if err != nil {
		return nil, err
	}

	m := make(account.HoldingsSnapshot)
	for code, balance := range accountBalance {
		m[currency.NewCode(code)] = account.Balance{
			Total:  balance.Available + balance.OnOrders,
			Locked: balance.OnOrders,
		}
	}

	err = p.LoadHoldings("exchange", asset.Spot, m)
	if err != nil {
		return nil, err
	}
	return p.GetFullSnapshot()
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (p *Poloniex) FetchAccountInfo() (account.FullSnapshot, error) {
	acc, err := p.GetFullSnapshot()
	if err != nil {
		return p.UpdateAccountInfo()
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (p *Poloniex) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (p *Poloniex) GetWithdrawalsHistory(_ currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (p *Poloniex) GetRecentTrades(currencyPair currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return p.GetHistoricTrades(currencyPair, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (p *Poloniex) GetHistoricTrades(currencyPair currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if timestampEnd.After(time.Now()) || timestampEnd.Before(timestampStart) {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v", timestampStart, timestampEnd)
	}
	var err error
	currencyPair, err = p.FormatExchangeCurrency(currencyPair, assetType)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	ts := timestampStart
allTrades:
	for {
		var tradeData []TradeHistory
		tradeData, err = p.GetTradeHistory(currencyPair.String(), ts.Unix(), timestampEnd.Unix())
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			var tt time.Time
			tt, err = time.Parse(common.SimpleTimeFormat, tradeData[i].Date)
			if err != nil {
				return nil, err
			}
			if (tt.Before(timestampStart) && !timestampStart.IsZero()) || (tt.After(timestampEnd) && !timestampEnd.IsZero()) {
				break allTrades
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Type)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     p.Name,
				TID:          strconv.FormatInt(tradeData[i].TradeID, 10),
				CurrencyPair: currencyPair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Rate,
				Amount:       tradeData[i].Amount,
				Timestamp:    tt,
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tt) {
					// reached end of trades to crawl
					break allTrades
				}
				if timestampStart.IsZero() {
					break allTrades
				}
				ts = tt
			}
		}
	}

	err = p.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}
	resp = trade.FilterTradesByTime(resp, timestampStart, timestampEnd)

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (p *Poloniex) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	fPair, err := p.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	fillOrKill := s.Type == order.Market
	isBuyOrder := s.Side == order.Buy
	response, err := p.PlaceOrder(fPair.String(),
		s.Price,
		s.Amount,
		false,
		fillOrKill,
		isBuyOrder)
	if err != nil {
		return submitOrderResponse, err
	}
	if response.OrderNumber > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response.OrderNumber, 10)
	}

	submitOrderResponse.IsOrderPlaced = true
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (p *Poloniex) ModifyOrder(action *order.Modify) (string, error) {
	if err := action.Validate(); err != nil {
		return "", err
	}

	oID, err := strconv.ParseInt(action.ID, 10, 64)
	if err != nil {
		return "", err
	}

	resp, err := p.MoveOrder(oID,
		action.Price,
		action.Amount,
		action.PostOnly,
		action.ImmediateOrCancel)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(resp.OrderNumber, 10), nil
}

// CancelOrder cancels an order by its corresponding ID number
func (p *Poloniex) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(p.Name, o.OrderIDRequired()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	return p.CancelExistingOrder(orderIDInt)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (p *Poloniex) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (p *Poloniex) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := p.GetOpenOrdersForAllCurrencies()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for key := range openOrders.Data {
		for i := range openOrders.Data[key] {
			err = p.CancelExistingOrder(openOrders.Data[key][i].OrderNumber)
			if err != nil {
				id := strconv.FormatInt(openOrders.Data[key][i].OrderNumber, 10)
				cancelAllOrdersResponse.Status[id] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (p *Poloniex) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	orderInfo := order.Detail{
		Exchange: p.Name,
		Pair:     pair,
	}

	trades, err := p.GetAuthenticatedOrderTrades(orderID)
	if err != nil && !strings.Contains(err.Error(), "Order not found") {
		return orderInfo, err
	}

	for i := range trades {
		var tradeHistory order.TradeHistory
		tradeHistory.Exchange = p.Name
		tradeHistory.Side, err = order.StringToOrderSide(trades[i].Type)
		if err != nil {
			return orderInfo, err
		}
		tradeHistory.TID = strconv.FormatInt(trades[i].GlobalTradeID, 10)
		tradeHistory.Timestamp, err = time.Parse(common.SimpleTimeFormat, trades[i].Date)
		if err != nil {
			return orderInfo, err
		}
		tradeHistory.Price = trades[i].Rate
		tradeHistory.Amount = trades[i].Amount
		tradeHistory.Total = trades[i].Total
		tradeHistory.Fee = trades[i].Fee
		orderInfo.Trades = append(orderInfo.Trades, tradeHistory)
	}

	resp, err := p.GetAuthenticatedOrderStatus(orderID)
	if err != nil {
		if len(orderInfo.Trades) > 0 { // on closed orders return trades only
			if strings.Contains(err.Error(), "Order not found") {
				orderInfo.Status = order.Closed
			}
			return orderInfo, nil
		}
		return orderInfo, err
	}

	orderInfo.Status, _ = order.StringToOrderStatus(resp.Status)
	orderInfo.Price = resp.Rate
	orderInfo.Amount = resp.Amount
	orderInfo.Cost = resp.Total
	orderInfo.Fee = resp.Fee
	orderInfo.TargetAmount = resp.StartingAmount

	orderInfo.Side, err = order.StringToOrderSide(resp.Type)
	if err != nil {
		return orderInfo, err
	}

	orderInfo.Date, err = time.Parse(common.SimpleTimeFormat, resp.Date)
	if err != nil {
		return orderInfo, err
	}

	return orderInfo, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (p *Poloniex) GetDepositAddress(cryptocurrency currency.Code, _ string) (exchange.DepositAddress, error) {
	a, err := p.GetDepositAddressesAll()
	if err != nil {
		return exchange.DepositAddress{}, err
	}

	address, ok := a.Addresses[cryptocurrency.Upper().String()]
	if !ok {
		return exchange.DepositAddress{}, fmt.Errorf("cannot find deposit address for %s",
			cryptocurrency)
	}
	return exchange.DepositAddress{Address: address}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (p *Poloniex) GetDepositAddresses(_ string) ([]exchange.DepositAddress, error) {
	p.Verbose = true
	a, err := p.GetDepositAddressesAll()
	if err != nil {
		return nil, err
	}

	var addrs []exchange.DepositAddress
	for key, val := range a.Addresses {
		addrs = append(addrs, exchange.DepositAddress{
			Currency: currency.NewCode(key),
			Address:  val,
		})
	}
	p.Verbose = false
	return addrs, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (p *Poloniex) WithdrawCryptocurrencyFunds(req *withdraw.Request) (*withdraw.Response, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	v, err := p.Withdraw(req.Currency.String(),
		req.Crypto.Address,
		req.Crypto.AddressTag,
		req.Amount)
	if err != nil {
		return nil, err
	}

	return &withdraw.Response{
		Status:       v.Response,
		WithdrawalID: strconv.FormatInt(v.WithdrawalNumber, 10),
		// Poloniex does not send an update on amount through websocket, so
		// immediately draw down on holdings for a realistic reflection of
		// account balance.
		ReduceAccountHoldings: true,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when as
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFunds(_ *withdraw.Request) (*withdraw.Response, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFundsToInternationalBank(_ *withdraw.Request) (*withdraw.Response, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (p *Poloniex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (!p.AllowAuthenticatedRequest() || p.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return p.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (p *Poloniex) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := p.GetOpenOrdersForAllCurrencies()
	if err != nil {
		return nil, err
	}

	format, err := p.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for key, values := range resp.Data {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(key, format.Delimiter)
		if err != nil {
			return nil, err
		}
		for i := range values {
			orderSide := order.Side(strings.ToUpper(values[i].Type))
			orderDate, err := time.Parse(common.SimpleTimeFormat, resp.Data[key][i].Date)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					p.Name,
					"GetActiveOrders",
					values[i].OrderNumber,
					values[i].Date)
			}

			oType := order.Limit
			if values[i].Margin == 1 {
				oType = order.Margin
			}

			orders = append(orders, order.Detail{
				ID:       strconv.FormatInt(resp.Data[key][i].OrderNumber, 10),
				Side:     orderSide,
				Amount:   values[i].Amount,
				Date:     orderDate,
				Price:    values[i].Rate,
				Pair:     symbol,
				Exchange: p.Name,
				Type:     oType,
			})
		}
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	order.FilterOrdersBySide(&orders, req.Side)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (p *Poloniex) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := p.GetAuthenticatedTradeHistory(req.StartTime.Unix(),
		req.EndTime.Unix(),
		10000)
	if err != nil {
		return nil, err
	}

	format, err := p.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for key := range resp.Data {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(key, format.Delimiter)
		if err != nil {
			return nil, err
		}

		for i := range resp.Data[key] {
			orderSide := order.Side(strings.ToUpper(resp.Data[key][i].Type))
			orderDate, err := time.Parse(common.SimpleTimeFormat,
				resp.Data[key][i].Date)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					p.Name,
					"GetActiveOrders",
					resp.Data[key][i].OrderNumber,
					resp.Data[key][i].Date)
			}

			orders = append(orders, order.Detail{
				ID:       strconv.FormatInt(resp.Data[key][i].GlobalTradeID, 10),
				Side:     orderSide,
				Amount:   resp.Data[key][i].Amount,
				Date:     orderDate,
				Price:    resp.Data[key][i].Rate,
				Pair:     symbol,
				Exchange: p.Name,
			})
		}
	}

	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	order.FilterOrdersBySide(&orders, req.Side)

	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (p *Poloniex) ValidateCredentials(assetType asset.Item) error {
	// _, err := p.UpdateAccountInfo(assetType)
	// return p.CheckTransientError(err)
	return nil
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (p *Poloniex) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := p.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	formattedPair, err := p.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	// we use Truncate here to round star to the nearest even number that matches the requested interval
	// example 10:17 with an interval of 15 minutes will go down 10:15
	// this is due to poloniex returning a non-complete candle if the time does not match
	candles, err := p.GetChartData(formattedPair.String(),
		start.Truncate(interval.Duration()), end,
		p.FormatExchangeKlineInterval(interval))
	if err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: p.Name,
		Interval: interval,
		Pair:     pair,
		Asset:    a,
	}

	for x := range candles {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   time.Unix(candles[x].Date, 0),
			Open:   candles[x].Open,
			High:   candles[x].High,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].Volume,
		})
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (p *Poloniex) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return p.GetHistoricCandles(pair, a, start, end, interval)
}

// GetAccounts returns the exchange accounts
func (p *Poloniex) GetAccounts() ([]string, error) {
	// There are 3 account types supported by the API but there is not a
	// consistant way to determine each individual levels for exchange, margin
	// and lending. Opted to amalgamate all into the exchange account and
	// individual levels will need to be tracked internally rather than by
	// external calls.
	return []string{"exchange"}, nil
}
