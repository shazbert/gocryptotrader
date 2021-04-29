package order

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
)

var (
	errOrderIDNotSet            = errors.New("ID not set")
	errPairNotSet               = errors.New("pair not set")
	errAssetNotSet              = errors.New("asset not set")
	errWalletAddressNotSet      = errors.New("wallet address not set")
	errClientIDNotSet           = errors.New("client ID not set")
	errClientOrderIDNotSet      = errors.New("client order ID not set")
	errSymbolNotSet             = errors.New("symbol not set")
	errAccountIDNotSet          = errors.New("account ID is not set")
	errIDAndClientOrderIDNotSet = errors.New("order ID and client order ID not set")
	errOrderSideNotSet          = errors.New("order side not set")
)

// Cancel contains all properties that may be required
// to cancel an order on an exchange
// Each exchange has their own requirements, so not all fields
// are required to be populated
type Cancel struct {
	ID            string
	ClientOrderID string
	AccountID     string
	ClientID      string
	WalletAddress string
	Side          Side
	AssetType     asset.Item
	Pair          currency.Pair
	Symbol        string

	// Validator method attach for specific struct checks
	validate.Validator
}

// OrderIDRequired defines an option in the validator to make sure an ID is set
// for a standard cancel
func (c *Cancel) OrderIDRequired() validate.Checker {
	return validate.Check(func() error {
		if c.ID == "" {
			return fmt.Errorf("cannot cancel: %w", errOrderIDNotSet)
		}
		return nil
	})
}

// PairRequired defines an option in the validator to make sure a pair is set
func (c *Cancel) PairRequired() validate.Checker {
	return validate.Check(func() error {
		if c.Pair.IsEmpty() {
			return fmt.Errorf("cannot cancel: %w", errPairNotSet)
		}
		return nil
	})
}

// AssetRequired defines an option in the validator to make sure a asset is set
func (c *Cancel) AssetRequired() validate.Checker {
	return validate.Check(func() error {
		if c.AssetType == "" {
			return fmt.Errorf("cannot cancel: %w", errAssetNotSet)
		}
		return nil
	})
}

// WalletAddressRequired defines an option in the validator to make sure a
// wallet address is set
func (c *Cancel) WalletAddressRequired() validate.Checker {
	return validate.Check(func() error {
		if c.WalletAddress == "" {
			return fmt.Errorf("cannot cancel: %w", errWalletAddressNotSet)
		}
		return nil
	})
}

// ClientIDRequired defines an option in the validator to make sure a
// wallet address is set
func (c *Cancel) ClientIDRequired() validate.Checker {
	return validate.Check(func() error {
		if c.ClientID == "" {
			return fmt.Errorf("cannot cancel: %w", errClientIDNotSet)
		}
		return nil
	})
}

// ClientOrderIDRequired defines an option in the validator to make sure a
// client order ID is set
func (c *Cancel) ClientOrderIDRequired() validate.Checker {
	return validate.Check(func() error {
		if c.ClientOrderID == "" {
			return fmt.Errorf("cannot cancel: %w", errClientOrderIDNotSet)
		}
		return nil
	})
}

// SymbolRequired defines an option in the validator to make sure a
// client order ID is set
func (c *Cancel) SymbolRequired() validate.Checker {
	return validate.Check(func() error {
		if c.Symbol == "" {
			return fmt.Errorf("cannot cancel: %w", errSymbolNotSet)
		}
		return nil
	})
}

// AccountIDRequired defines an option in the validator to make sure an
// account ID is set
func (c *Cancel) AccountIDRequired() validate.Checker {
	return validate.Check(func() error {
		if c.AccountID == "" {
			return fmt.Errorf("cannot cancel: %w", errAccountIDNotSet)
		}
		return nil
	})
}

// IDOrClientIDRequired defines an option in the validator to make sure an
// ID or a client ID is set
func (c *Cancel) IDOrClientIDRequired() validate.Checker {
	return validate.Check(func() error {
		if c.ClientOrderID == "" && c.ID == "" {
			return fmt.Errorf("cannot cancel: %w", errIDAndClientOrderIDNotSet)
		}
		return nil
	})
}

// OrderSideRequired defines an option in the validator to make sure an
// order side is set
func (c *Cancel) OrderSideRequired() validate.Checker {
	return validate.Check(func() error {
		if c.Side == "" {
			return fmt.Errorf("cannot cancel: %w", errOrderSideNotSet)
		}
		return nil
	})
}
