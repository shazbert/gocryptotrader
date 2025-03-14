package okx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWsPlaceOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsPlaceOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	out := &PlaceOrderRequestParam{
		InstrumentID: "BTC-USDT",
		TradeMode:    TradeModeCash,
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       0.0001,
		Price:        -1,
		Currency:     "USDT",
	}

	got, err := ok.WsPlaceOrder(context.Background(), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsPlaceMultipleOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsPlaceMultipleOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ok.WsPlaceMultipleOrder(context.Background(), []PlaceOrderRequestParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	out := PlaceOrderRequestParam{
		InstrumentID: "BTC-USDT",
		TradeMode:    TradeModeCash,
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       0.0001,
		Price:        -1, // Intentional fail
		Currency:     "USDT",
	}

	got, err := ok.WsPlaceMultipleOrder(context.Background(), []PlaceOrderRequestParam{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsCancelOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsCancelOrder(context.Background(), CancelOrderRequestParam{})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.WsCancelOrder(context.Background(), CancelOrderRequestParam{InstrumentID: "BTC-USDT"})
	require.ErrorIs(t, err, errMissingClientOrderIDOrOrderID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	c := CancelOrderRequestParam{InstrumentID: "BTC-USDT", OrderID: "1680136326338387968"}
	got, err := ok.WsCancelOrder(context.Background(), c)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsCancleMultipleOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsCancelMultipleOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ok.WsCancelMultipleOrder(context.Background(), []CancelOrderRequestParam{{}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.WsCancelMultipleOrder(context.Background(), []CancelOrderRequestParam{{InstrumentID: "BTC-USDT"}})
	require.ErrorIs(t, err, errMissingClientOrderIDOrOrderID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	c := CancelOrderRequestParam{InstrumentID: "BTC-USDT", OrderID: "1680136326338387968"}
	got, err := ok.WsCancelMultipleOrder(context.Background(), []CancelOrderRequestParam{c})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsAmendOrder(t *testing.T) {
	t.Parallel()

	_, err := ok.WsAmendOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	out := &AmendOrderRequestParams{}
	_, err = ok.WsAmendOrder(context.Background(), out)
	require.ErrorIs(t, err, errMissingInstrumentID)

	out.InstrumentID = "BTC-USDT"
	_, err = ok.WsAmendOrder(context.Background(), out)
	require.ErrorIs(t, err, errMissingClientOrderIDOrOrderID)

	out.OrderID = "1680136326338387968"
	_, err = ok.WsAmendOrder(context.Background(), out)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	out.NewPrice = 20
	got, err := ok.WsAmendOrder(context.Background(), out)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsAmendMultipleOrders(t *testing.T) {
	t.Parallel()

	_, err := ok.WsAmendMultipleOrders(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	out := AmendOrderRequestParams{}
	_, err = ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errMissingInstrumentID)

	out.InstrumentID = "BTC-USDT"
	_, err = ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errMissingClientOrderIDOrOrderID)

	out.OrderID = "1680136326338387968"
	_, err = ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{out})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	ok := getWebsocketInstance(t, ok) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	out.NewPrice = 20

	got, err := ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{out})
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestWsMassCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.WsMassCancelOrders(contextGenerate(), []CancelMassReqParam{{}})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = ok.WsMassCancelOrders(contextGenerate(), []CancelMassReqParam{{InstrumentFamily: "BTC-USD"}})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = ok.WsMassCancelOrders(contextGenerate(), []CancelMassReqParam{{InstrumentType: "OPTION"}})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsMassCancelOrders(contextGenerate(), []CancelMassReqParam{
		{
			InstrumentType:   "OPTION",
			InstrumentFamily: "BTC-USD",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WsPlaceSpreadOrder(contextGenerate(), &SpreadOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsPlaceSpreadOrder(contextGenerate(), &SpreadOrderParam{
		SpreadID:      "BTC-USDT_BTC-USDT-SWAP",
		ClientOrderID: "b15",
		Side:          order.Buy.Lower(),
		OrderType:     "limit",
		Price:         2.15,
		Size:          2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAmandSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WsAmandSpreadOrder(contextGenerate(), &AmendSpreadOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.WsAmandSpreadOrder(contextGenerate(), &AmendSpreadOrderParam{NewSize: 2})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.WsAmandSpreadOrder(contextGenerate(), &AmendSpreadOrderParam{OrderID: "2510789768709120"})
	require.ErrorIs(t, err, errSizeOrPriceIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsAmandSpreadOrder(contextGenerate(), &AmendSpreadOrderParam{
		OrderID: "2510789768709120",
		NewSize: 2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// newExchangeWithWebsocket returns a websocket instance copy for testing.
// This restricts the pairs to a single pair per asset type to reduce test time.
func newExchangeWithWebsocket(t *testing.T) *Okx {
	t.Helper()

	if apiKey == "" || apiSecret == "" || passphrase == "" {
		t.Skip()
	}

	ok := new(Okx)
	require.NoError(t, testexch.Setup(ok), "Test instance Setup must not error")
	testexch.UpdatePairsOnce(t, ok)
	ok.API.AuthenticatedSupport = true
	ok.API.AuthenticatedWebsocketSupport = true
	ok.SetCredentials(apiKey, apiSecret, passphrase, "", "", "")

	for _, a := range ok.GetAssetTypes(true) {
		if a != asset.Spot {
			require.NoError(t, ok.CurrencyPairs.SetAssetEnabled(a, false))
			continue
		}
		avail, err := ok.GetAvailablePairs(a)
		require.NoError(t, err)
		if len(avail) > 1 {
			avail = avail[:1] // reduce pairs to 1 to speed up tests
		}
		require.NoError(t, ok.SetPairs(avail, a, true))
	}
	require.NoError(t, ok.Websocket.Connect())
	return ok
}
