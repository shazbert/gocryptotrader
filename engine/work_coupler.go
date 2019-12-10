package engine

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

const (
	fetchTicker = iota
	updateTicker
	fetchOrderbook
	updateOrderbook
	getAccountInfo
	getExchangeHistory
	getFeeByType
	getFundingHistory
	submitOrder
	modifyOrder
	cancelOrder
	cancelAllOrders
	getOrderDetail
	getDepositAddress
	getOrderHistory
	withdraw
)

var (
	errFunctionalityNotSupported = errors.New("function not supported")
	errFunctionalityNotFound     = errors.New("function not found")
)

// TODO:
// ClientPermissions: Determine if calling system can execute, set API KEYS
// DATABASE: Insert intention into audit table
// TRADE Heuristics: trade/account security
// DATABASE: Insert event in audit table

// Execute implements the command interface for the exchange coupler
func (f *FetchTicker) Execute() {
	f.Price, f.Error = f.FetchTicker(f.Pair, f.Asset)
}

// FetchTicker initiates a call to an exchange through the priority job queue
func (i Exchange) FetchTicker(p currency.Pair, a asset.Item) (ticker.Price, error) {
	if i.e == nil {
		return ticker.Price{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, fetchTicker)
	if err != nil {
		return ticker.Price{}, err
	}

	t := &FetchTicker{Pair: p, Asset: a, IBotExchange: i.e}

	err = i.wm.ExecuteJob(t, low)
	if err != nil {
		return t.Price, err
	}

	return t.Price, t.Error
}

// Execute implements the command interface for the exchange coupler
func (f *UpdateTicker) Execute() {
	f.Price, f.Error = f.UpdateTicker(f.Pair, f.Asset)
}

// UpdateTicker initiates a call to an exchange through the priority job queue
func (i Exchange) UpdateTicker(p currency.Pair, a asset.Item) (ticker.Price, error) {
	if i.e == nil {
		return ticker.Price{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, updateTicker)
	if err != nil {
		return ticker.Price{}, err
	}

	t := &UpdateTicker{Pair: p, Asset: a, IBotExchange: i.e}

	err = i.wm.ExecuteJob(t, low)
	if err != nil {
		return t.Price, err
	}

	return t.Price, t.Error
}

// Execute implements the command interface for the exchange coupler
func (o *FetchOrderbook) Execute() {
	o.Orderbook, o.Error = o.FetchOrderbook(o.Pair, o.Asset)
}

// FetchOrderbook initiates a call to an exchange through the priority job queue
func (i Exchange) FetchOrderbook(p currency.Pair, a asset.Item) (orderbook.Base, error) {
	if i.e == nil {
		return orderbook.Base{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, fetchOrderbook)
	if err != nil {
		return orderbook.Base{}, err
	}

	o := &FetchOrderbook{Pair: p, Asset: a, IBotExchange: i.e}

	err = i.wm.ExecuteJob(o, medium)
	if err != nil {
		return o.Orderbook, err
	}

	return o.Orderbook, o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *UpdateOrderbook) Execute() {
	o.Orderbook, o.Error = o.UpdateOrderbook(o.Pair, o.Asset)
}

// UpdateOrderbook initiates a call to an exchange through the priority job
// queue
func (i Exchange) UpdateOrderbook(p currency.Pair, a asset.Item) (orderbook.Base, error) {
	if i.e == nil {
		return orderbook.Base{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, updateOrderbook)
	if err != nil {
		return orderbook.Base{}, err
	}

	o := &UpdateOrderbook{Pair: p, Asset: a, IBotExchange: i.e}

	err = i.wm.ExecuteJob(o, medium)
	if err != nil {
		return o.Orderbook, err
	}

	return o.Orderbook, o.Error
}

// Execute implements the command interface for the exchange coupler
func (g *GetAccountInfo) Execute() {
	g.AccountInfo, g.Error = g.GetAccountInfo()
}

// GetAccountInfo initiates a call to an exchange through the priority job queue
func (i Exchange) GetAccountInfo() (exchange.AccountInfo, error) {
	if i.e == nil {
		return exchange.AccountInfo{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getAccountInfo)
	if err != nil {
		return exchange.AccountInfo{}, err
	}

	acc := &GetAccountInfo{IBotExchange: i.e}

	err = i.wm.ExecuteJob(acc, high)
	if err != nil {
		return acc.AccountInfo, err
	}

	return acc.AccountInfo, acc.Error
}

// Execute implements the command interface for the exchange coupler
func (e *GetExchangeHistory) Execute() {
	e.Response, e.Error = e.GetExchangeHistory(e.Request)
}

// GetExchangeHistory initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetExchangeHistory(r *exchange.TradeHistoryRequest) ([]exchange.TradeHistory, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getExchangeHistory)
	if err != nil {
		return nil, err
	}

	h := &GetExchangeHistory{Request: r, IBotExchange: i.e}

	err = i.wm.ExecuteJob(h, low)
	if err != nil {
		return nil, err
	}

	return h.Response, h.Error
}

// Execute implements the command interface for the exchange coupler
func (f *GetFeeByType) Execute() {
	f.Response, f.Error = f.GetFeeByType(f.Request)
}

// GetFeeByType initiates a call to an exchange through the priority job queue
func (i Exchange) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if i.e == nil {
		return 0, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getFeeByType)
	if err != nil {
		return 0, err
	}

	f := &GetFeeByType{Request: feeBuilder, IBotExchange: i.e}

	err = i.wm.ExecuteJob(f, high)
	if err != nil {
		return 0, err
	}

	return f.Response, f.Error
}

// Execute implements the command interface for the exchange coupler
func (f *GetFundingHistory) Execute() {
	f.Response, f.Error = f.GetFundingHistory()
}

// GetFundingHistory initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetFundingHistory() ([]exchange.FundHistory, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getFundingHistory)
	if err != nil {
		return nil, err
	}

	f := &GetFundingHistory{IBotExchange: i.e}

	err = i.wm.ExecuteJob(f, medium)
	if err != nil {
		return nil, err
	}

	return f.Response, f.Error
}

// Execute implements the command interface for the exchange coupler
func (o *SubmitOrder) Execute() {
	o.Response, o.Error = o.SubmitOrder(o.Request)
}

// SubmitOrder initiates a call to an exchange through the priority job queue
func (i Exchange) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	if i.e == nil {
		return order.SubmitResponse{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, submitOrder)
	if err != nil {
		return order.SubmitResponse{}, err
	}

	o := &SubmitOrder{Request: s, IBotExchange: i.e}

	err = i.wm.ExecuteJob(o, extreme)
	if err != nil {
		return o.Response, err
	}

	return o.Response, o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *ModifyOrder) Execute() {
	o.Response, o.Error = o.ModifyOrder(o.Request)
}

// ModifyOrder initiates a call to an exchange through the priority job queue
func (i Exchange) ModifyOrder(action *order.Modify) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, modifyOrder)
	if err != nil {
		return "", err
	}

	o := &ModifyOrder{Request: action, IBotExchange: i.e}

	err = i.wm.ExecuteJob(o, extreme)
	if err != nil {
		return o.Response, err
	}

	return o.Response, o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *CancelOrder) Execute() {
	o.Error = o.CancelOrder(o.Request)
}

// CancelOrder initiates a call to an exchange through the priority job queue
func (i Exchange) CancelOrder(cancel *order.Cancel) error {
	if i.e == nil {
		return errExchangNotFound
	}

	err := i.checkFunctionality(i.e, cancelOrder)
	if err != nil {
		return err
	}

	o := &CancelOrder{Request: cancel, IBotExchange: i.e}

	err = i.wm.ExecuteJob(o, extreme)
	if err != nil {
		return err
	}

	return o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *CancelAllOrders) Execute() {
	o.Response, o.Error = o.CancelAllOrders(o.Request)
}

// CancelAllOrders initiates a call to an exchange through the priority job
// queue
func (i Exchange) CancelAllOrders(cancel *order.Cancel) (order.CancelAllResponse, error) {
	if i.e == nil {
		return order.CancelAllResponse{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, cancelAllOrders)
	if err != nil {
		return order.CancelAllResponse{}, err
	}

	o := &CancelAllOrders{Request: cancel, IBotExchange: i.e}

	err = i.wm.ExecuteJob(o, extreme)
	if err != nil {
		return o.Response, err
	}

	return o.Response, o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *GetOrderInfo) Execute() {
	o.Response, o.Error = o.GetOrderInfo(o.Request)
}

// GetOrderInfo initiates a call to an exchange through the priority job queue
func (i Exchange) GetOrderInfo(orderID string) (order.Detail, error) {
	if i.e == nil {
		return order.Detail{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getOrderDetail)
	if err != nil {
		return order.Detail{}, err
	}

	o := &GetOrderInfo{Request: orderID, IBotExchange: i.e}

	err = i.wm.ExecuteJob(o, high)
	if err != nil {
		return o.Response, err
	}

	return o.Response, o.Error
}

// Execute implements the command interface for the exchange coupler
func (a *GetDepositAddress) Execute() {
	a.Response, a.Error = a.GetDepositAddress(a.Crypto, a.AccountID)
}

// GetDepositAddress initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetDepositAddress(crypto currency.Code, accountID string) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getDepositAddress)
	if err != nil {
		return "", err
	}

	a := &GetDepositAddress{Crypto: crypto, AccountID: accountID, IBotExchange: i.e}

	err = i.wm.ExecuteJob(a, medium)
	if err != nil {
		return a.Response, err
	}

	return a.Response, a.Error
}

// Execute implements the command interface for the exchange coupler
func (h *GetOrderHistory) Execute() {
	h.Response, h.Error = h.GetOrderHistory(h.Request)
}

// GetOrderHistory initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getOrderDetail)
	if err != nil {
		return nil, err
	}

	h := &GetOrderHistory{Request: req, IBotExchange: i.e}

	err = i.wm.ExecuteJob(h, medium)
	if err != nil {
		return h.Response, err
	}

	return h.Response, h.Error
}

// Execute implements the command interface for the exchange coupler
func (h *GetActiveOrders) Execute() {
	h.Response, h.Error = h.GetActiveOrders(h.Request)
}

// GetActiveOrders initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getOrderDetail)
	if err != nil {
		return nil, err
	}

	h := &GetActiveOrders{Request: req, IBotExchange: i.e}

	err = i.wm.ExecuteJob(h, medium)
	if err != nil {
		return h.Response, err
	}

	return h.Response, h.Error
}

// Execute implements the command interface for the exchange coupler
func (w *WithdrawCryptocurrencyFunds) Execute() {
	w.Response, w.Error = w.WithdrawCryptocurrencyFunds(w.Request)
}

// WithdrawCryptocurrencyFunds initiates a call to an exchange through the
// priority job queue
func (i Exchange) WithdrawCryptocurrencyFunds(req *exchange.CryptoWithdrawRequest) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, withdraw)
	if err != nil {
		return "", err
	}

	w := &WithdrawCryptocurrencyFunds{Request: req, IBotExchange: i.e}

	err = i.wm.ExecuteJob(w, high)
	if err != nil {
		return w.Response, err
	}

	return w.Response, w.Error
}

// Execute implements the command interface for the exchange coupler
func (w *WithdrawFiatFunds) Execute() {
	w.Response, w.Error = w.WithdrawFiatFunds(w.Request)
}

// WithdrawFiatFunds initiates a call to an exchange through the priority job
// queue
func (i Exchange) WithdrawFiatFunds(req *exchange.FiatWithdrawRequest) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, withdraw)
	if err != nil {
		return "", err
	}

	w := &WithdrawFiatFunds{Request: req, IBotExchange: i.e}

	err = i.wm.ExecuteJob(w, high)
	if err != nil {
		return w.Response, err
	}

	return w.Response, w.Error
}

// Execute implements the command interface for the exchange coupler
func (w *WithdrawFiatFundsToInternationalBank) Execute() {
	w.Response, w.Error = w.WithdrawFiatFundsToInternationalBank(w.Request)
}

// WithdrawFiatFundsToInternationalBank initiates a call to an exchange through
// the priority job queue
func (i Exchange) WithdrawFiatFundsToInternationalBank(req *exchange.FiatWithdrawRequest) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, withdraw)
	if err != nil {
		return "", err
	}

	w := &WithdrawFiatFundsToInternationalBank{Request: req, IBotExchange: i.e}

	err = i.wm.ExecuteJob(w, high)
	if err != nil {
		return w.Response, err
	}

	return w.Response, w.Error
}

func (i Exchange) checkFunctionality(e exchange.IBotExchange, function int) error {
	b := e.GetBase()

	switch function {
	case fetchTicker, updateTicker:
		if !b.Features.REST.TickerFetching.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case fetchOrderbook, updateOrderbook:
		if !b.Features.REST.OrderbookFetching.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case getAccountInfo:
		if !b.Features.REST.AccountInfo.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case getExchangeHistory:
		if !b.Features.REST.ExchangeTradeHistory.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case getFeeByType:
		// need to fix this
		if !b.Features.REST.TradeFee.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case getFundingHistory:
		// fix this
		return errFunctionalityNotSupported

	case submitOrder:
		if !b.Features.REST.SubmitOrder.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case modifyOrder:
		if !b.Features.REST.ModifyOrder.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case cancelOrder:
		if !b.Features.REST.CancelOrder.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case cancelAllOrders:
		if !b.Features.REST.CancelOrders.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case getOrderDetail:
		if !b.Features.REST.GetOrder.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case getOrderHistory:
		if !b.Features.REST.GetOrders.IsEnabled() {
			return errFunctionalityNotSupported
		}
	case withdraw:
		if *b.Features.REST.Withdraw == 0 {
			return errFunctionalityNotSupported
		}
	default:
		return errFunctionalityNotFound
	}
	return nil
}
