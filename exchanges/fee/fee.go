package fee

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bank"
)

var (
	// ErrDefinitionsAreNil defines if the exchange specific fee definitions
	// have bot been loaded or set up.
	ErrDefinitionsAreNil = errors.New("fee definitions are nil")

	errCurrencyIsEmpty         = errors.New("currency is empty")
	errTransferFeeNotFound     = errors.New("transfer fee not found")
	errBankTransferFeeNotFound = errors.New("bank transfer fee not found")
	errPriceIsZero             = errors.New("price is zero")
	errAmountIsZero            = errors.New("amount is zero")
	errFeeTypeMismatch         = errors.New("fee type mismatch")
	errRateNotFound            = errors.New("rate not found")
	errCommissionRateNotFound  = errors.New("commission rate not found")
	errTakerInvalid            = errors.New("taker is invalid")
	errMakerInvalid            = errors.New("maker is invalid")
	errMakerBiggerThanTaker    = errors.New("maker cannot be bigger than taker")
	errNoTransferFees          = errors.New("missing transfer fees to load")

	// OmitPair is a an empty pair designation for unused pair variables
	OmitPair = currency.Pair{}

	// AllAllowed defines a potential bank transfer when all foreign exchange
	// currencies are allowed to operate.
	AllAllowed = currency.NewCode("ALLALLOWED")
)

// NewFeeDefinitions generates a new fee struct for exchange usage
func NewFeeDefinitions() *Definitions {
	return &Definitions{
		globalCommissions: make(map[asset.Item]*CommissionInternal),
		pairCommissions:   make(map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal),
		chainTransfer:     make(map[*currency.Item]map[string]*transfer),
		bankTransfer:      make(map[bank.Transfer]map[*currency.Item]*transfer),
	}
}

// Definitions defines the full fee definitions for different currencies
// TODO: Eventually upgrade with key manager for different fees associated
// with different accounts/keys.
type Definitions struct {
	// Commission is the holder for the up to date comission rates for the assets.
	globalCommissions map[asset.Item]*CommissionInternal
	// pairCommissions is the holder for the up to date commissions rates for
	// the specific trading pairs.
	pairCommissions map[asset.Item]map[*currency.Item]map[*currency.Item]*CommissionInternal
	// transfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions. These will commonly be real values.
	chainTransfer map[*currency.Item]map[string]*transfer
	// bankTransfer defines a map of currencies with differing withdrawal and
	// deposit fee definitions for banking. These will commonly be fixed real
	// values.
	bankTransfer map[bank.Transfer]map[*currency.Item]*transfer
	mtx          sync.RWMutex
}

// LoadDynamic loads the current dynamic account fee structure for maker and
// taker values. The pair is an optional paramater if ommited will designate
// global/exchange maker, taker fees irrespective of individual trading
// operations.
func (d *Definitions) LoadDynamic(maker, taker float64, a asset.Item, pair currency.Pair) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}
	if taker < 0 {
		return errTakerInvalid
	}
	if maker > taker {
		return errMakerBiggerThanTaker
	}
	if !a.IsValid() {
		return fmt.Errorf("%s: %w", a, asset.ErrNotSupported)
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	var c *CommissionInternal
	if !pair.IsEmpty() {
		// NOTE: These will create maps, as we can initially start out as global
		// commission rates and update ad-hoc.
		m1, ok := d.pairCommissions[a]
		if !ok {
			m1 = make(map[*currency.Item]map[*currency.Item]*CommissionInternal)
			d.pairCommissions[a] = m1
		}
		m2, ok := m1[pair.Base.Item]
		if !ok {
			m2 = make(map[*currency.Item]*CommissionInternal)
			m1[pair.Base.Item] = m2
		}
		c, ok = m2[pair.Quote.Item]
		if !ok {
			c = new(CommissionInternal)
			m2[pair.Quote.Item] = c
		}
	} else {
		var ok bool
		c, ok = d.globalCommissions[a]
		if !ok {
			return fmt.Errorf("global %w", errCommissionRateNotFound)
		}
	}
	c.load(maker, taker)
	return nil
}

// LoadStatic loads predefined custom long term fee structures for items like
// worst case scenario values, transfer fees to and from exchanges, and
// international bank transfer rates.
func (d *Definitions) LoadStatic(o Options) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if err := o.validate(); err != nil {
		return err
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	// Loads global commission rates based on asset item
	for a, value := range o.GlobalCommissions {
		d.globalCommissions[a] = value.convert()
	}

	// Loads pair specific commission rates
	for a, incoming := range o.PairCommissions {
		for pair, value := range incoming {
			m1, ok := d.pairCommissions[a]
			if !ok {
				m1 = make(map[*currency.Item]map[*currency.Item]*CommissionInternal)
				d.pairCommissions[a] = m1
			}
			m2, ok := m1[pair.Base.Item]
			if !ok {
				m2 = make(map[*currency.Item]*CommissionInternal)
			}
			m2[pair.Quote.Item] = value.convert()
		}
	}

	// Loads exchange withdrawal and deposit fees
	for x := range o.ChainTransfer {
		chainTransfer, ok := d.chainTransfer[o.ChainTransfer[x].Currency.Item]
		if !ok {
			chainTransfer = make(map[string]*transfer)
			d.chainTransfer[o.ChainTransfer[x].Currency.Item] = chainTransfer
		}
		chainTransfer[o.ChainTransfer[x].Chain] = o.ChainTransfer[x].convert()
	}

	// Loads international banking withdrawal and deposit fees
	for x := range o.BankTransfer {
		transferFees, ok := d.bankTransfer[o.BankTransfer[x].BankTransfer]
		if !ok {
			transferFees = make(map[*currency.Item]*transfer)
			d.bankTransfer[o.BankTransfer[x].BankTransfer] = transferFees
		}
		transferFees[o.BankTransfer[x].Currency.Item] = o.BankTransfer[x].convert()
	}
	return nil
}

// GetCommissionFee returns a pointer of the current commission rate for the
// asset type.
func (d *Definitions) GetCommissionFee(a asset.Item, pair currency.Pair) (*CommissionInternal, error) {
	if d == nil {
		return nil, ErrDefinitionsAreNil
	}
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return d.getCommission(a, pair)
}

// getCommission returns the internal commission rate based on provided params
func (d *Definitions) getCommission(a asset.Item, pair currency.Pair) (*CommissionInternal, error) {
	if len(d.pairCommissions) != 0 && !pair.IsEmpty() {
		m1, ok := d.pairCommissions[a]
		if !ok {
			return nil, fmt.Errorf("pair %w", errCommissionRateNotFound)
		}

		m2, ok := m1[pair.Base.Item]
		if !ok {
			return nil, fmt.Errorf("pair %w", errCommissionRateNotFound)
		}

		c, ok := m2[pair.Quote.Item]
		if !ok {
			return nil, fmt.Errorf("pair %w", errCommissionRateNotFound)
		}
		return c, nil
	}
	c, ok := d.globalCommissions[a]
	if !ok {
		return nil, fmt.Errorf("global %w", errCommissionRateNotFound)
	}
	return c, nil
}

// CalculateMaker returns the fee amount derived from the price, amount and fee
// percentage.
func (d *Definitions) CalculateMaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, err
	}
	return c.CalculateMaker(price, amount)
}

// CalculateWorstCaseMaker returns the fee amount derived from the price, amount
// and fee percentage using the worst-case scenario trading fee.
func (d *Definitions) CalculateWorstCaseMaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, err
	}
	return c.CalculateWorstCaseMaker(price, amount)
}

// GetMaker returns the maker fee value and if it is a percentage or whole
// number
func (d *Definitions) GetMaker(a asset.Item, pair currency.Pair) (fee float64, isSetAmount bool, err error) {
	if d == nil {
		return 0, false, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, false, err
	}
	fee, isSetAmount = c.GetMaker()
	return
}

// CalculateTaker returns the fee amount derived from the price, amount and fee
// percentage.
func (d *Definitions) CalculateTaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, err
	}
	return c.CalculateTaker(price, amount)
}

// CalculateWorstCaseTaker returns the fee amount derived from the price, amount
// and fee percentage using the worst-case scenario trading fee.
func (d *Definitions) CalculateWorstCaseTaker(price, amount float64, a asset.Item, pair currency.Pair) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, err
	}
	return c.CalculateWorstCaseTaker(price, amount)
}

// GetTaker returns the taker fee value and if it is a percentage or real number
func (d *Definitions) GetTaker(a asset.Item, pair currency.Pair) (fee float64, isSetAmount bool, err error) {
	if d == nil {
		return 0, false, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return 0, false, err
	}
	fee, isSetAmount = c.GetTaker()
	return
}

// CalculateDeposit returns calculated fee from the amount
func (d *Definitions) CalculateDeposit(c currency.Code, chain string, amount float64) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, chain)
	if err != nil {
		return 0, err
	}
	return t.calculate(t.Deposit, amount)
}

// GetDeposit returns the deposit fee associated with the currency
func (d *Definitions) GetDeposit(c currency.Code, chain string) (fee Value, isPercentage bool, err error) {
	if d == nil {
		return nil, false, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, chain)
	if err != nil {
		return nil, false, err
	}
	return t.Deposit, t.Percentage, nil
}

// CalculateDeposit returns calculated fee from the amount
func (d *Definitions) CalculateWithdrawal(c currency.Code, chain string, amount float64) (float64, error) {
	if d == nil {
		return 0, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, chain)
	if err != nil {
		return 0, err
	}
	return t.calculate(t.Withdrawal, amount)
}

// GetWithdrawal returns the withdrawal fee associated with the currency
func (d *Definitions) GetWithdrawal(c currency.Code, chain string) (fee Value, isPercentage bool, err error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, err := d.get(c, chain)
	if err != nil {
		return nil, false, err
	}
	return t.Withdrawal, t.Percentage, nil
}

// get returns the fee structure by the currency and its chain type
func (d *Definitions) get(c currency.Code, chain string) (*transfer, error) {
	if c.String() == "" {
		return nil, errCurrencyIsEmpty
	}

	s, ok := d.chainTransfer[c.Item][chain]
	if !ok {
		return nil, errTransferFeeNotFound
	}
	return s, nil
}

// GetAllFees returns a snapshot of the full fee definitions, super cool.
func (d *Definitions) GetAllFees() (Options, error) {
	if d == nil {
		return Options{}, ErrDefinitionsAreNil
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	op := Options{
		GlobalCommissions: make(map[asset.Item]Commission),
		PairCommissions:   make(map[asset.Item]map[currency.Pair]Commission),
	}

	for a, value := range d.globalCommissions {
		op.GlobalCommissions[a] = value.convert()
	}

	for a, mInternal := range d.pairCommissions {
		for c1, mInternal2 := range mInternal {
			for c2, value := range mInternal2 {
				mOutgoing, ok := op.PairCommissions[a]
				if !ok {
					mOutgoing = make(map[currency.Pair]Commission)
					op.PairCommissions[a] = mOutgoing
				}
				p := currency.NewPair(currency.Code{Item: c1}, currency.Code{Item: c2})
				mOutgoing[p] = value.convert()
			}
		}
	}

	for currencyItem, m1 := range d.chainTransfer {
		for chain, val := range m1 {
			out := val.convert()
			out.Currency = currency.Code{Item: currencyItem, UpperCase: true}
			out.Chain = chain
			op.ChainTransfer = append(op.ChainTransfer, out)
		}
	}

	for bankProtocol, m1 := range d.bankTransfer {
		for currencyItem, val := range m1 {
			out := val.convert()
			out.Currency = currency.Code{Item: currencyItem, UpperCase: true}
			out.BankTransfer = bankProtocol
			op.BankTransfer = append(op.BankTransfer, out)
		}
	}
	return op, nil
}

// SetCommissionFee sets new global fees and forces custom control for that
// asset
func (d *Definitions) SetCommissionFee(a asset.Item, pair currency.Pair, maker, taker float64, setAmount bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if taker < 0 {
		return errTakerInvalid
	}

	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	c, err := d.getCommission(a, pair)
	if err != nil {
		return err
	}
	return c.set(maker, taker, setAmount)
}

// GetTransferFee returns a snapshot of the current Commission rate for the
// asset type.
func (d *Definitions) GetTransferFee(c currency.Code, chain string) (Transfer, error) {
	if d == nil {
		return Transfer{}, ErrDefinitionsAreNil
	}

	if c.String() == "" {
		return Transfer{}, errCurrencyIsEmpty
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, ok := d.chainTransfer[c.Item][chain]
	if !ok {
		return Transfer{}, errRateNotFound
	}
	return t.convert(), nil
}

// SetTransferFees sets new transfer fees
// TODO: need min and max settings might deprecate due to complexity of value
// types
func (d *Definitions) SetTransferFee(c currency.Code, chain string, withdraw, deposit float64, isPercentage bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if withdraw < 0 {
		return errWithdrawalIsInvalid
	}

	if deposit < 0 {
		return errDepositIsInvalid
	}

	if c.String() == "" {
		return errCurrencyIsEmpty
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	t, ok := d.chainTransfer[c.Item][chain]
	if !ok {
		return errTransferFeeNotFound
	}

	// These should not change, and a package update might need to occur.
	if t.Percentage != isPercentage {
		return errFeeTypeMismatch
	}

	t.Withdrawal = Convert(withdraw) // TODO: need min and max settings
	t.Deposit = Convert(deposit)     // TODO: need min and max settings
	return nil
}

// GetBankTransferFee returns a snapshot of the current bank transfer rate for the
// asset.
func (d *Definitions) GetBankTransferFee(c currency.Code, transType bank.Transfer) (Transfer, error) {
	if d == nil {
		return Transfer{}, ErrDefinitionsAreNil
	}

	if c.String() == "" {
		return Transfer{}, errCurrencyIsEmpty
	}

	err := transType.Validate()
	if err != nil {
		return Transfer{}, err
	}

	d.mtx.RLock()
	defer d.mtx.RUnlock()
	t, ok := d.bankTransfer[transType][c.Item]
	if !ok {
		return Transfer{}, errRateNotFound
	}
	return t.convert(), nil
}

// SetBankTransferFee sets new bank transfer fees
// TODO: need min and max settings might deprecate due to complexity of value
// types
func (d *Definitions) SetBankTransferFee(c currency.Code, transType bank.Transfer, withdraw, deposit float64, isPercentage bool) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if c.String() == "" {
		return errCurrencyIsEmpty
	}

	err := transType.Validate()
	if err != nil {
		return err
	}

	if withdraw < 0 {
		return errWithdrawalIsInvalid
	}

	if deposit < 0 {
		return errDepositIsInvalid
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	tFee, ok := d.bankTransfer[transType][c.Item]
	if !ok {
		return errBankTransferFeeNotFound
	}

	if tFee.Percentage != isPercentage {
		return errFeeTypeMismatch
	}

	tFee.Withdrawal = Convert(withdraw) // TODO: need min and max settings
	tFee.Deposit = Convert(deposit)     // TODO: need min and max settings
	return nil
}

// LoadTransferFees allows the loading of current transfer fees for
// cryptocurrency deposit and withdrawals
func (d *Definitions) LoadTransferFees(fees []Transfer) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if len(fees) == 0 {
		return errNoTransferFees
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	for x := range fees {
		err := fees[x].validate()
		if err != nil {
			return fmt.Errorf("loading crypto fees error: %w", err)
		}
		m1, ok := d.chainTransfer[fees[x].Currency.Item]
		if !ok {
			m1 = make(map[string]*transfer)
			d.chainTransfer[fees[x].Currency.Item] = m1
		}
		val, ok := m1[fees[x].Chain]
		if !ok {
			m1[fees[x].Chain] = fees[x].convert()
			continue
		}
		err = val.update(fees[x])
		if err != nil {
			return fmt.Errorf("loading crypto fees error: %w", err)
		}
	}
	return nil
}

// LoadBankTransferFees allows the loading of current banking transfer fees for
// banking deposit and withdrawals
func (d *Definitions) LoadBankTransferFees(fees map[bank.Transfer]map[currency.Code]Transfer) error {
	if d == nil {
		return ErrDefinitionsAreNil
	}

	if len(fees) == 0 {
		return errNoTransferFees
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	for bankType, m1 := range fees {
		for code, incomingVal := range m1 {
			trAssets, ok := d.bankTransfer[bankType]
			if !ok {
				trAssets = make(map[*currency.Item]*transfer)
				d.bankTransfer[bankType] = trAssets
			}
			trVal, ok := trAssets[code.Item]
			if !ok {
				trAssets[code.Item] = incomingVal.convert()
				continue
			}
			err := trVal.update(incomingVal)
			if err != nil {
				return fmt.Errorf("loading banking fees error: %w", err)
			}
		}
	}
	return nil
}