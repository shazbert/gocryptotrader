package engine

// // DepositAddressStore stores a list of exchange deposit addresses
// type DepositAddressStore struct {
// 	m     sync.Mutex
// 	Store map[string]map[string]string
// }

// // DepositAddressManager manages the exchange deposit address store
// type DepositAddressManager struct {
// 	Store DepositAddressStore
// }

// // vars related to the deposit address helpers
// var (
// 	ErrDepositAddressStoreIsNil = errors.New("deposit address store is nil")
// 	ErrDepositAddressNotFound   = errors.New("deposit address does not exist")
// )

// // Seed seeds the deposit address store
// func (d *DepositAddressStore) Seed(coinData map[string]map[string]string) {
// 	d.m.Lock()
// 	defer d.m.Unlock()
// 	if d.Store == nil {
// 		d.Store = make(map[string]map[string]string)
// 	}

// 	for k, v := range coinData {
// 		r := make(map[string]string)
// 		for w, x := range v {
// 			r[strings.ToUpper(w)] = x
// 		}
// 		d.Store[strings.ToUpper(k)] = r
// 	}
// }

// // GetDepositAddress returns a deposit address based on the specified item
// func (d *DepositAddressStore) GetDepositAddress(exchName string, item currency.Code) (string, error) {
// 	d.m.Lock()
// 	defer d.m.Unlock()

// 	if len(d.Store) == 0 {
// 		return "", ErrDepositAddressStoreIsNil
// 	}

// 	r, ok := d.Store[strings.ToUpper(exchName)]
// 	if !ok {
// 		return "", ErrExchangeNotFound
// 	}

// 	addr, ok := r[strings.ToUpper(item.String())]
// 	if !ok {
// 		return "", ErrDepositAddressNotFound
// 	}

// 	return addr, nil
// }

// // GetDepositAddresses returns a list of stored deposit addresses
// func (d *DepositAddressStore) GetDepositAddresses(exchName string) (map[string]string, error) {
// 	d.m.Lock()
// 	defer d.m.Unlock()

// 	if len(d.Store) == 0 {
// 		return nil, ErrDepositAddressStoreIsNil
// 	}

// 	r, ok := d.Store[strings.ToUpper(exchName)]
// 	if !ok {
// 		return nil, ErrDepositAddressNotFound
// 	}

// 	return r, nil
// }
