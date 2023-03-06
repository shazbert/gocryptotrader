package api

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// LoadData retrieves data from a GoCryptoTrader exchange wrapper which calls
// the exchange's API.
func LoadData(ctx context.Context, dataType int64, startDate, endDate time.Time, interval gctkline.Interval, exch exchange.IBotExchange, pair currency.Pair, a asset.Item) (*gctkline.Item, error) {
	var candles *gctkline.Item
	var err error
	switch dataType {
	case common.DataCandle:
		fmt.Println(exch.GetName())
		candles, err = exch.GetHistoricCandlesExtended(ctx,
			pair,
			a,
			interval,
			startDate,
			endDate)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve candle data for %v %v %v, %w",
				exch.GetName(), a, pair, err)
		}
	case common.DataTrade:
		var trades []trade.Data
		// TODO: Implement GetHistoricTrades extended functionality as this will
		// only fetch a set amount of trades dependant on the end point. This
		// will not align, or create candles correctly if there are missing
		// trades, due to that limitation.
		trades, err = exch.GetHistoricTrades(ctx, pair, a, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve trade data for %v %v %v, %w",
				exch.GetName(), a, pair, err)
		}

		// Converting to order.TradeHistory because it is more performant and
		// consistant when using gctkline.CreateKline() functionality.
		tradeHistory := make([]order.TradeHistory, len(trades))
		for x := range trades {
			tradeHistory[x] = order.TradeHistory{
				Price:     trades[x].Price,
				Amount:    trades[x].Amount,
				Timestamp: trades[x].Timestamp,
			}
		}

		// CreateKlineRequest is used to add padding when tradeHistory is
		// converted.
		request, err := gctkline.CreateKlineRequest(exch.GetName(), pair, pair, a, interval, interval, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("could not convert trade data to candles for %v %v %v, %w",
				exch.GetName(), a, pair, err)
		}

		candles, err = gctkline.CreateKline(tradeHistory, interval, pair, a, exch.GetName())
		if err != nil {
			return nil, fmt.Errorf("could not convert trade data to candles for %v %v %v, %w",
				exch.GetName(), a, pair, err)
		}

		candles, err = request.ProcessResponse(candles.Candles)
		if err != nil {
			return nil, fmt.Errorf("could not convert trade data to candles for %v %v %v, %w",
				exch.GetName(), a, pair, err)
		}
	default:
		return nil, fmt.Errorf("could not retrieve data for %v %v %v, %w",
			exch.GetName(), a, pair, common.ErrInvalidDataType)
	}
	// candles.Exchange = strings.ToLower(candles.Exchange)
	return candles, nil
}
