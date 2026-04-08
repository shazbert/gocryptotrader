package v12_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	v12 "github.com/thrasher-corp/gocryptotrader/config/versions/v12"
)

func TestVersion12UpgradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&v12.Version{}).UpgradeExchange(t.Context(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test"}`)
	expected := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	got, err = (&v12.Version{}).UpgradeExchange(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)

	payload = []byte(`{"name":"test","websocketMetricsLogging":true}`)
	got, err = (&v12.Version{}).UpgradeExchange(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestVersion12DowngradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&v12.Version{}).DowngradeExchange(t.Context(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	expected := []byte(`{"name":"test"}`)
	got, err = (&v12.Version{}).DowngradeExchange(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestVersion12Exchanges(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	assert.Equal([]string{"*"}, (&v12.Version{}).Exchanges())
}
