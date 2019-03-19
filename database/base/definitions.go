package base

// Define database client access levels
const (
	Basic AccessLevel = iota + 1
	Trader
	Sales
	Manager
	SuperUser
)

// AccessLevel defines a database access level for clients
type AccessLevel int

// GetAccessLevels returns database access levels
func GetAccessLevels() map[string]int {
	return map[string]int{
		"Basic":     int(Basic),
		"Trader":    int(Trader),
		"Sales":     int(Sales),
		"Manager":   int(Manager),
		"SuperUser": int(SuperUser),
	}
}

// GetSupportedExchanges returns database supported exchanges
func GetSupportedExchanges() []string {
	return []string{
		"ANX",
		"Binance",
		"Bitfinex",
		"Bitflyer",
		"Bithumb",
		"Bitmex",
		"Bitstamp",
		"Bittrex",
		"BTCC",
		"BTC Markets",
		"COINUT",
		"EXMO",
		"CoinbasePro",
		"GateIO",
		"Gemini",
		"HitBTC",
		"Huobi",
		"HuobiHadax",
		"ITBIT",
		"Kraken",
		"LakeBTC",
		"Liqui",
		"LocalBitcoins",
		"OKCOIN International",
		"OKEX",
		"Poloniex",
		"WEX",
		"Yobit",
		"ZB",
	}
}
