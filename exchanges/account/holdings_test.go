package account

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const AccountTest = "test"

var one = decimal.NewFromInt(1)
var twenty = decimal.NewFromInt(20)

func TestGetHolding(t *testing.T) {
	h, err := DeployHoldings("getHolding", false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	_, err = h.GetHolding("", "", currency.Code{})
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}

	_, err = h.GetHolding(AccountTest, "", currency.Code{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected: %v but received: %v", asset.ErrNotSupported, err)
	}

	_, err = h.GetHolding(AccountTest, asset.Spot, currency.Code{})
	if !errors.Is(err, errCurrencyIsEmpty) {
		t.Fatalf("expected: %v but received: %v", errCurrencyIsEmpty, err)
	}

	values := HoldingsSnapshot{
		currency.BTC: {Total: 1},
		currency.LTC: {Total: 20},
	}

	err = h.LoadHoldings(AccountTest, asset.Spot, values)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	btcHolding, err := h.GetHolding(AccountTest, asset.Spot, currency.BTC)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !btcHolding.free.Equal(one) {
		t.Fatalf("expected free holdings: %s, but received %s", one, btcHolding.free)
	}

	if !btcHolding.locked.Equal(decimal.Zero) {
		t.Fatalf("expected free holdings: %s, but received %s", decimal.Zero, btcHolding.locked)
	}

	ltcHolding, err := h.GetHolding(AccountTest, asset.Spot, currency.LTC)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !ltcHolding.free.Equal(twenty) {
		t.Fatalf("expected free holdings: %s, but received %s", twenty, ltcHolding.free)
	}

	if !ltcHolding.locked.Equal(decimal.Zero) {
		t.Fatalf("expected free holdings: %s, but received %s", decimal.Zero, ltcHolding.locked)
	}

	ethHolding, err := h.GetHolding("subAccount", asset.Spot, currency.ETH)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if !ethHolding.free.Equal(decimal.Zero) {
		t.Fatalf("expected free holdings: %s, but received %s", decimal.Zero, ethHolding.free)
	}

	if !ethHolding.locked.Equal(decimal.Zero) {
		t.Fatalf("expected free holdings: %s, but received %s", decimal.Zero, ethHolding.locked)
	}
}

func TestLoad(t *testing.T) {
	h, err := DeployHoldings("load", false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = h.LoadHoldings("", "", nil)
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}

	err = h.LoadHoldings(AccountTest, "", nil)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected: %v but received: %v", asset.ErrNotSupported, err)
	}

	err = h.LoadHoldings(AccountTest, asset.Spot, nil)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	values := HoldingsSnapshot{
		currency.BTC: {Total: 1},
		currency.LTC: {Total: 20},
	}

	err = h.LoadHoldings(AccountTest, asset.Spot, values)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	values = HoldingsSnapshot{
		currency.BTC: {Total: 2, Locked: 0.5},
		currency.XRP: {Total: 60000},
	}

	err = h.LoadHoldings(AccountTest, asset.Spot, values)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	btcHolding, err := h.GetHolding(AccountTest, asset.Spot, currency.BTC)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if btcHolding.GetFree() != 1.5 {
		t.Fatal("unexpected amounts received")
	}
}

func TestAdjustHolding(t *testing.T) {
	h, err := DeployHoldings("adjustholdings", false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	// Initial start with limit orders already present on exchange
	values := HoldingsSnapshot{
		currency.XRP: {Total: 40.5, Locked: 6},
	}

	err = h.LoadHoldings("adjustholdings", asset.Spot, values)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = h.AdjustByBalance("", "", currency.Code{}, 0)
	if !errors.Is(err, errAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errAccountNameUnset, err)
	}

	err = h.AdjustByBalance("adjustholdings", "", currency.Code{}, 0)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected: %v but received: %v", asset.ErrNotSupported, err)
	}

	err = h.AdjustByBalance("adjustholdings", asset.Spot, currency.Code{}, 0)
	if !errors.Is(err, errCurrencyCodeEmpty) {
		t.Fatalf("expected: %v but received: %v", errCurrencyCodeEmpty, err)
	}

	err = h.AdjustByBalance("adjustholdings", asset.Spot, currency.XRP, 0)
	if !errors.Is(err, errAmountCannotBeZero) {
		t.Fatalf("expected: %v but received: %v", errAmountCannotBeZero, err)
	}

	err = h.AdjustByBalance("dummy", asset.Spot, currency.XRP, 1)
	if !errors.Is(err, errAccountNotFound) {
		t.Fatalf("expected: %v but received: %v", errAccountNotFound, err)
	}

	err = h.AdjustByBalance("adjustholdings", asset.Futures, currency.XRP, 1)
	if !errors.Is(err, errAssetTypeNotFound) {
		t.Fatalf("expected: %v but received: %v", errAssetTypeNotFound, err)
	}

	err = h.AdjustByBalance("adjustholdings", asset.Spot, currency.BTC, 1)
	if !errors.Is(err, errCurrencyItemNotFound) {
		t.Fatalf("expected: %v but received: %v", errCurrencyItemNotFound, err)
	}

	holding, err := h.GetHolding("adjustholdings", asset.Spot, currency.XRP)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(holding, 40.5, 6, 34.5, 0, 0, t)

	// Balance increased by one - limit order cancelled
	err = h.AdjustByBalance("adjustholdings", asset.Spot, currency.XRP, 1)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(holding, 40.5, 5, 35.5, 0, 0, t)

	// limit/market order executed claim by algo or rpc
	claim, err := holding.Claim(1, true)
	if err != nil {
		t.Fatal(err)
	}
	checkValues(holding, 40.5, 5, 34.5, 0, 1, t)

	// limit/market order accepted by exchange
	err = claim.ReleaseToPending()
	if err != nil {
		t.Fatal(err)
	}
	checkValues(holding, 40.5, 5, 34.5, 1, 0, t)

	// simulate balance change on pending - does not mean it was matched
	// this demonstrates Poloniex balance flow
	err = h.AdjustByBalance("adjustholdings", asset.Spot, currency.XRP, -1)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}
	checkValues(holding, 39.5, 5, 34.5, 0, 0, t)
}
