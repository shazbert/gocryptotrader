package v14_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v14 "github.com/thrasher-corp/gocryptotrader/config/versions/v14"
)

func TestExchanges(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"GateIO"}, new(v14.Version).Exchanges())
}

func TestUpgradeExchange(t *testing.T) {
	t.Parallel()
	version := new(v14.Version)

	tests := []struct {
		name string
		in   string
		exp  string
	}{
		{
			name: "previous defaults",
			in:   `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
			exp:  `{"name":"GateIO","features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		},
		{
			name: "custom selection",
			in:   `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
			exp:  `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`,
		},
		{
			name: "custom settings",
			in:   `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"10ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":20}]}}`,
			exp:  `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"10ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":20}]}}`,
		},
		{
			name: "missing subscriptions",
			in:   `{"name":"GateIO"}`,
			exp:  `{"name":"GateIO"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := version.UpgradeExchange(t.Context(), []byte(tc.in))
			require.NoError(t, err, "UpgradeExchange must not error")
			assert.JSONEq(t, tc.exp, string(got), "UpgradeExchange should migrate only previous defaults")
		})
	}
}

func TestDowngradeExchange(t *testing.T) {
	t.Parallel()
	in := []byte(`{"name":"GateIO","features":{"subscriptions":[{"enabled":false,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"spot.obu","asset":"spot","levels":50}]}}`)
	exp := `{"name":"GateIO","features":{"subscriptions":[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":false,"channel":"spot.obu","asset":"spot","levels":50}]}}`
	got, err := new(v14.Version).DowngradeExchange(t.Context(), in)
	require.NoError(t, err, "DowngradeExchange must not error")
	assert.JSONEq(t, exp, string(got), "DowngradeExchange should restore previous defaults")
}

func TestUpgradeExchangeRejectsMalformedSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := new(v14.Version).UpgradeExchange(t.Context(), []byte(`{"features":{"subscriptions":{}}}`))
	require.Error(t, err, "UpgradeExchange must reject malformed subscriptions")
}
