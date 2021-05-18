package withdraw

import (
	"errors"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// Validate takes interface and passes to asset type to check the request meets
// requirements to submit
func (r *Request) Validate(opt ...validate.Checker) (err error) {
	if r == nil {
		return ErrRequestCannotBeNil
	}

	if r.Exchange == "" {
		return ErrExchangeNameUnset
	}

	var allErrors []string
	if r.Amount <= 0 {
		allErrors = append(allErrors, ErrStrAmountMustBeGreaterThanZero)
	}

	if (r.Currency == currency.Code{}) {
		allErrors = append(allErrors, ErrStrNoCurrencySet)
	}

	switch r.Type {
	case Fiat:
		if (r.Currency != currency.Code{}) && !r.Currency.IsFiatCurrency() {
			allErrors = append(allErrors, ErrStrCurrencyNotFiat)
		}
		allErrors = append(allErrors, r.validateFiat()...)
	case Crypto:
		if (r.Currency != currency.Code{}) && !r.Currency.IsCryptocurrency() {
			allErrors = append(allErrors, ErrStrCurrencyNotCrypto)
		}
		allErrors = append(allErrors, r.validateCrypto()...)
	default:
		allErrors = append(allErrors, "invalid request type")
	}

	for _, o := range opt {
		if o == nil {
			continue
		}
		err := o.Check()
		if err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if len(allErrors) > 0 {
		return errors.New(strings.Join(allErrors, ", "))
	}
	return nil
}

// validateFiat takes interface and passes to asset type to check the request
// meets requirements to submit
func (r *Request) validateFiat() (err []string) {
	errBank := r.Fiat.Bank.ValidateForWithdrawal(r.Exchange, r.Currency)
	if errBank != nil {
		err = append(err, errBank...)
	}
	return err
}

// validateCrypto checks if Crypto request is valid and meets the minimum
// requirements to submit a crypto withdrawal request
func (r *Request) validateCrypto() (err []string) {
	if !portfolio.IsWhiteListed(r.Crypto.Address, r.Currency) {
		err = append(err, ErrStrAddressNotWhiteListed)
	}

	if !portfolio.IsExchangeSupported(r.Exchange, r.Crypto.Address, r.Currency) {
		err = append(err, ErrStrExchangeNotSupportedByAddress)
	}

	required, errR := portfolio.IsTagOrMemoRequired(r.Crypto.Address, r.Currency)
	if errR != nil {
		err = append(err, errR.Error())
	}

	if required && r.Crypto.AddressTag == "" {
		err = append(err, "crypto currency withdrawal address tag needs to be populated")
	}

	if r.Crypto.Address == "" {
		err = append(err, ErrStrAddressNotSet)
	}

	if r.Crypto.FeeAmount < 0 {
		err = append(err, ErrStrFeeCannotBeNegative)
	}
	return
}
