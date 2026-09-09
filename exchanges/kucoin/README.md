# GoCryptoTrader package Kucoin

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/kucoin)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)


This kucoin package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Kucoin Exchange

### Current Features

+ REST Support
+ Websocket Support

### Subscriptions

Default Public Subscriptions:
- Ticker for spot, margin and futures
- Orderbook for spot, margin and futures
- All trades for spot and margin

When authenticated websocket support is enabled, the default orderbook subscription uses the realtime spot and futures feeds. Realtime spot snapshots require authentication; KuCoin also uses the authenticated futures snapshot endpoint to avoid dynamic rate limits on its public equivalent.

Legacy `/market/level2` and `/contractMarket/level2` subscription entries are removed during setup. If any removed entry was enabled, existing enabled generic orderbook subscriptions are retained and generic subscriptions are added for any uncovered assets from those entries. Legacy entries with no asset or all assets cover all enabled assets. If no generic orderbook subscription is enabled, the generic `orderbook` subscription for all assets is enabled or added instead. This fallback can broaden a previous per-asset choice to all enabled assets and pairs, including margin pairs merged into spot. Pin the generic subscription to the desired assets and pairs to restrict coverage. Explicit depth-5 channel subscriptions remain available and are not upgraded by authentication.

Default Authenticated Subscriptions:
- All trades for futures
- Stop Order Lifecycle events for futures
- Account Balance events for spot, margin and futures
- Margin Position updates
- Margin Loan updates

Subscriptions are subject to enabled assets and pairs.

Margin subscriptions for ticker, orderbook and All trades are merged into Spot subscriptions because duplicates are not allowed,
unless Spot subscription does not exist, i.e. Spot asset not enabled, or subscription configured only for Margin

Limitations:
- 100 symbols per subscription
- 300 symbols per connection

Due to these limitations, if more than 10 symbols are enabled, ticker will subscribe to ticker:all.

Unimplemented subscriptions:
- Candles for Futures
- Market snapshot for currency

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
