// Package v14 migrates GateIO's default spot orderbook websocket subscription to V2.
package v14

import (
	"context"
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"fmt"
	"strconv"

	"github.com/buger/jsonparser"
)

const (
	legacyOrderbookChannel = "orderbook"
	spotAsset              = "spot"
	spotOrderbookV2Channel = "spot.obu"
)

// Version implements ExchangeVersion for GateIO's spot orderbook subscription migration.
type Version struct{}

// Exchanges returns just GateIO.
func (*Version) Exchanges() []string { return []string{"GateIO"} }

// UpgradeExchange replaces the previous default spot orderbook subscription with V2.
func (*Version) UpgradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return migrateSubscriptions(exchange, true)
}

// DowngradeExchange restores the previous default spot orderbook subscription.
func (*Version) DowngradeExchange(_ context.Context, exchange []byte) ([]byte, error) {
	return migrateSubscriptions(exchange, false)
}

func migrateSubscriptions(exchange []byte, upgrade bool) ([]byte, error) {
	raw, _, _, err := jsonparser.Get(exchange, "features", "subscriptions")
	if err != nil {
		if err == jsonparser.KeyPathNotFoundError {
			return exchange, nil
		}
		return exchange, fmt.Errorf("error getting GateIO subscriptions: %w", err)
	}

	var subscriptions []struct {
		Enabled  *bool  `json:"enabled"`
		Channel  string `json:"channel"`
		Asset    string `json:"asset"`
		Interval string `json:"interval"`
		Levels   int    `json:"levels"`
	}
	if err := json.Unmarshal(raw, &subscriptions); err != nil {
		return exchange, fmt.Errorf("error decoding GateIO subscriptions: %w", err)
	}

	legacyIndex, v2Index := -1, -1
	for i := range subscriptions {
		if subscriptions[i].Asset != spotAsset || subscriptions[i].Enabled == nil {
			continue
		}
		switch subscriptions[i].Channel {
		case legacyOrderbookChannel:
			if legacyIndex != -1 {
				return exchange, nil
			}
			legacyIndex = i
		case spotOrderbookV2Channel:
			if v2Index != -1 {
				return exchange, nil
			}
			v2Index = i
		}
	}
	if legacyIndex == -1 || v2Index == -1 ||
		subscriptions[legacyIndex].Interval != "100ms" || subscriptions[v2Index].Levels != 50 ||
		*subscriptions[legacyIndex].Enabled != upgrade || *subscriptions[v2Index].Enabled == upgrade {
		return exchange, nil
	}

	exchange, err = jsonparser.Set(exchange, []byte(strconv.FormatBool(!upgrade)), "features", "subscriptions", "["+strconv.Itoa(legacyIndex)+"]", "enabled")
	if err != nil {
		return exchange, fmt.Errorf("error setting GateIO legacy spot orderbook subscription: %w", err)
	}
	exchange, err = jsonparser.Set(exchange, []byte(strconv.FormatBool(upgrade)), "features", "subscriptions", "["+strconv.Itoa(v2Index)+"]", "enabled")
	if err != nil {
		return exchange, fmt.Errorf("error setting GateIO V2 spot orderbook subscription: %w", err)
	}
	return exchange, nil
}
