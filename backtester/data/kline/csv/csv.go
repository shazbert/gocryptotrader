package csv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errNoUSDData = errors.New("could not retrieve USD CSV candle data")
	errNoCandles = errors.New("no candles from data type")
)

// LoadData is a basic csv reader which converts the found CSV file into a kline
// item. WARNING: When fetching candles with a misconfigured interval this may
// add padding (empty candles) between candles for that time period.
func LoadData(dataType int64, filepath, exchangeName string, in gctkline.Interval, pair currency.Pair, a asset.Item, isUSDTrackingPair bool) (*kline.DataFromKline, error) {
	csvFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	records, err := csv.NewReader(csvFile).ReadAll()

	closeErr := csvFile.Close()
	if closeErr != nil {
		return nil, closeErr
	}

	if err != nil {
		return nil, err
	}

	var timeSeries []gctkline.Candle
	switch dataType {
	case common.DataCandle:
		timeSeries = make([]gctkline.Candle, len(records))
		for x := range records {
			var seconds int64
			seconds, err = strconv.ParseInt(records[x][0], 10, 64)
			if err != nil {
				return nil, err
			}

			timeSeries[x].Time = time.Unix(seconds, 0).UTC()
			if timeSeries[x].Time.Unix() == 0 {
				return nil, fmt.Errorf("invalid timestamp received on row %v", records[x][0])
			}

			timeSeries[x].Volume, err = strconv.ParseFloat(records[x][1], 64)
			if err != nil {
				return nil, fmt.Errorf("could not process candle volume %v %w", records[x][1], err)
			}

			timeSeries[x].Open, err = strconv.ParseFloat(records[x][2], 64)
			if err != nil {
				return nil, fmt.Errorf("could not process candle volume %v %w", records[x][2], err)
			}

			timeSeries[x].High, err = strconv.ParseFloat(records[x][3], 64)
			if err != nil {
				return nil, fmt.Errorf("could not process candle high %v %w", records[x][3], err)
			}

			timeSeries[x].Low, err = strconv.ParseFloat(records[x][4], 64)
			if err != nil {
				return nil, fmt.Errorf("could not process candle low %v %w", records[x][4], err)
			}

			timeSeries[x].Close, err = strconv.ParseFloat(records[x][5], 64)
			if err != nil {
				return nil, fmt.Errorf("could not process candle close %v %w", records[x][5], err)
			}
		}
	case common.DataTrade:
		trades := make([]order.TradeHistory, len(records))
		for x := range records {
			var seconds int64
			seconds, err = strconv.ParseInt(records[x][0], 10, 64)
			if err != nil {
				return nil, err
			}
			trades[x].Timestamp = time.Unix(seconds, 0).UTC()
			if trades[x].Timestamp.Unix() == 0 {
				return nil, fmt.Errorf("invalid timestamp received on row %v", records[x][0])
			}

			trades[x].Price, err = strconv.ParseFloat(records[x][1], 64)
			if err != nil {
				return nil, fmt.Errorf("could not process trade price %v, %w", records[x][1], err)
			}

			trades[x].Amount, err = strconv.ParseFloat(records[x][2], 64)
			if err != nil {
				return nil, fmt.Errorf("could not process trade amount %v, %w", records[x][2], err)
			}

			trades[x].Side, err = order.StringToOrderSide(records[x][3])
			if err != nil {
				return nil, fmt.Errorf("could not process trade side %v, %w", records[x][3], err)
			}
		}

		var temp *gctkline.Item
		temp, err = gctkline.CreateKline(trades, in, pair, a, exchangeName)
		if err != nil {
			return nil, err
		}
		timeSeries = temp.Candles
	default:
		if isUSDTrackingPair {
			return nil, fmt.Errorf("%w for %v %v %v. Please add USD pair data to your CSV or set `disable-usd-tracking` to `true` in your config. %v",
				errNoUSDData, exchangeName, a, pair, err)
		}
		return nil, fmt.Errorf("could not process csv data for %v %v %v, %w",
			exchangeName, a, pair, common.ErrInvalidDataType)
	}

	if len(timeSeries) == 0 {
		return nil, errNoCandles
	}

	// NOTE: This infers the start and end date specifically from the stored
	// data.
	start := timeSeries[0].Time
	end := timeSeries[len(timeSeries)-1].Time.Add(in.Duration())

	klineRequest, err := gctkline.CreateKlineRequest(exchangeName, pair, pair, a, in, in, start, end)
	if err != nil {
		return nil, err
	}

	// NOTE: This is a process to correctly manage, filter and structure the
	// data and to ensure it is not invalid.
	klineItem, err := klineRequest.ProcessResponse(timeSeries)
	if err != nil {
		return nil, err
	}

	return kline.NewDataFromKline(klineItem, start, end)
}
