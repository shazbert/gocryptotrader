package orderbook

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Get checks and returns the orderbook given an exchange name and currency pair
func Get(i Identifier) (*Depth, error) {
	if err := i.verify(); err != nil {
		return nil, err
	}
	return service.Retrieve(i.Exchange, i.Pair, i.Asset)
}

func LoadSnapshot(i Identifier, o Options, bids, asks Items) {

}

func (i *Identifier) verify() error {
	if i.Exchange == "" {
		return errExchangeNameUnset
	}

	if i.Pair.IsEmpty() {
		return errPairNotSet
	}

	if i.Asset.String() == "" {
		return errAssetTypeNotSet
	}
	return nil
}

// Retrieve returns the pointer to the orderbook depth values
func (s *Service) Retrieve(exchange string, p currency.Pair, a asset.Item) (*Depth, error) {
	s.Lock()
	defer s.Unlock()
	m1, ok := s.Books[strings.ToLower(exchange)]
	if !ok {
		return nil, fmt.Errorf("no orderbooks for %s exchange", exchange)
	}

	m2, ok := m1[a]
	if !ok {
		return nil, fmt.Errorf("no orderbooks associated with asset type %s",
			a)
	}

	m3, ok := m2[p.Base.Item]
	if !ok {
		return nil, fmt.Errorf("no orderbooks associated with base currency %s",
			p.Base)
	}

	book, ok := m3[p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("no orderbooks associated with quote currency %s",
			p.Quote)
	}

	return book, nil
}

func (elements *Items) verifyAsks(isFunding, notAggregated bool) error {
	return elements.verify(func(i, j float64) bool {
		return i < j
	}, isFunding, notAggregated)
}

func (elements *Items) verifyBids(isFunding, notAggregated bool) error {
	return elements.verify(func(i, j float64) bool {
		return i > j
	}, isFunding, notAggregated)
}

// Verify ensures that the orderbook items are correctly sorted prior to being
// set and will reject any book with incorrect values.
// Bids should always go from a high price to a low price and
// Asks should always go from a low price to a higher price
func (elements *Items) verify(fn outOfOrder, funding, notAggregated bool) error {
	// Checking for both ask and bid lengths being zero has been removed and
	// a warning has been put in place some exchanges e.g. LakeBTC return zero
	// level books. In the event that there is a massive liquidity change where
	// a book dries up, this will still update so we do not traverse potential
	// incorrect old data.
	if len(*elements) == 0 {
		// log.Warnf(log.OrderBook,
		// 	bookLengthIssue,
		// 	b.ExchangeName,
		// 	b.Pair,
		// 	b.AssetType,
		// 	len(b.Bids),
		// 	len(b.Asks))
	}
	for i := range *elements {
		if (*elements)[i].Price == 0 {
			return errPriceNotSet
		}
		if (*elements)[i].Amount <= 0 {
			return errAmountInvalid
		}
		if funding && (*elements)[i].Period == 0 {
			return errPeriodUnset
		}
		if i != 0 {
			if fn((*elements)[i].Price, (*elements)[i-1].Price) {
				return errOutOfOrder
			}

			if !notAggregated && (*elements)[i].Price == (*elements)[i-1].Price {
				return errDuplication
			}

			if (*elements)[i].ID != 0 && (*elements)[i].ID == (*elements)[i-1].ID {
				return errIDDuplication
			}
		}
	}
	return nil
}

// Reverse reverses the order of orderbook items; some bid/asks are
// returned in either ascending or descending order. One bid or ask slice
// depending on whats received can be reversed. This is usually faster than
// using a sort algorithm as the algorithm could be impeded by a worst case time
// complexity when elements are shifted as opposed to just swapping element
// values.
func (elements *Items) Reverse() {
	eLen := len(*elements)
	var target int
	for i := eLen/2 - 1; i >= 0; i-- {
		target = eLen - 1 - i
		(*elements)[i], (*elements)[target] = (*elements)[target], (*elements)[i]
	}
}

// SortToAsks sorts ask items to the correct ascending order if pricing values are
// scattered. If order from exchange is descending consider using the Reverse
// function.
func (elements *Items) SortToAsks() {
	sort.Slice(elements, func(i, j int) bool {
		return (*elements)[i].Price < (*elements)[j].Price
	})
}

// SortToBids sorts bid items to the correct descending order if pricing values
// are scattered. If order from exchange is ascending consider using the Reverse
// function.
func (elements *Items) SortToBids() {
	sort.Slice(elements, func(i, j int) bool {
		return (*elements)[i].Price > (*elements)[j].Price
	})
}
