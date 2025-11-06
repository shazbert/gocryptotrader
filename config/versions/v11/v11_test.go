package v11_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	v11 "github.com/thrasher-corp/gocryptotrader/config/versions/v11"
)

func TestVersion11UpgradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&v11.Version{}).UpgradeExchange(t.Context(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test"}`)
	expected := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	got, err = (&v11.Version{}).UpgradeExchange(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)

	payload = []byte(`{"name":"test","websocketMetricsLogging":true}`)
	got, err = (&v11.Version{}).UpgradeExchange(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestVersion11DowngradeExchange(t *testing.T) {
	t.Parallel()

	got, err := (&v11.Version{}).DowngradeExchange(t.Context(), nil)
	require.NoError(t, err)
	require.Nil(t, got)

	payload := []byte(`{"name":"test","websocketMetricsLogging":false}`)
	expected := []byte(`{"name":"test"}`)
	got, err = (&v11.Version{}).DowngradeExchange(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestVersion11Exchanges(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	assert.Equal([]string{"*"}, (&v11.Version{}).Exchanges())
}
