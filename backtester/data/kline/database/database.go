package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var errNoUSDData = errors.New("could not retrieve USD database candle data")

// LoadData retrieves data from an existing database using GoCryptoTrader's database handling implementation
func LoadData(startDate, endDate time.Time, in gctkline.Interval, exchangeName string, dataType int64, fPair currency.Pair, a asset.Item, isUSDTrackingPair bool) (*kline.DataFromKline, error) {
	var timeSeries *gctkline.Item
	var err error
	switch dataType {
	case common.DataCandle:
		timeSeries, err = gctkline.LoadFromDatabase(exchangeName, fPair, a, in, startDate, endDate)
	case common.DataTrade:
		// TODO: Update param inputs
		trades, err := trade.GetTradesInRange(exchangeName,
			a.String(),
			fPair.Base.String(),
			fPair.Quote.String(),
			startDate,
			endDate)
		if err != nil {
			return nil, err
		}
		// TODO: Shift to gctkline.Conversion function
		timeSeries, err = trade.ConvertTradesToCandles(in, trades...)
	default:
		if isUSDTrackingPair {
			return nil, fmt.Errorf("%w for %v %v %v. Please add USD pair data to your CSV or set `disable-usd-tracking` to `true` in your config",
				errNoUSDData,
				exchangeName,
				a,
				fPair)
		}
		return nil, fmt.Errorf("could not retrieve database data for %v %v %v, %w",
			exchangeName,
			a,
			fPair,
			common.ErrInvalidDataType)
	}
	if err != nil {
		if isUSDTrackingPair {
			return nil, fmt.Errorf("%w for %v %v %v. Please save USD candle pair data to the database or set `disable-usd-tracking` to `true` in your config. %v",
				errNoUSDData,
				exchangeName,
				a,
				fPair,
				err)
		}
		return nil, fmt.Errorf("could not retrieve database candle data for %v %v %v, %w",
			exchangeName,
			a,
			fPair,
			err)
	}

	// TODO: Shift check.
	for x := range timeSeries.Candles {
		if timeSeries.Candles[x].ValidationIssues != "" {
			log.Warnf(common.Data, "Candle validation issue for %v %v %v: %v",
				timeSeries.Exchange,
				timeSeries.Asset,
				timeSeries.Pair,
				timeSeries.Candles[x].ValidationIssues)
		}
	}

	return kline.NewDataFromKline(timeSeries, startDate, endDate)
}
