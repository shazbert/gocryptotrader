package withdraw

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	modelPSQL "github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	modelSQLite "github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	exchangeDB "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
	"github.com/volatiletech/null"
)

var (
	// ErrNoResults is the error returned if no results are found
	ErrNoResults = errors.New("no results found")
)

// Event stores Withdrawal Response details in database
func Event(info *withdraw.Details) {
	if database.DB.SQL == nil {
		return
	}

	ctx := context.Background()
	ctx = boil.SkipTimestamps(ctx)

	var err error
	info.InternalExchangeID, err = exchangeDB.UUIDByName(info.Request.Exchange)
	if err != nil {
		log.Error(log.DatabaseMgr, err)
		return
	}

	tx, err := database.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event transaction being failed: %v", err)
		return
	}

	if repository.GetSQLDialect() == database.DBSQLite3 {
		err = addSQLiteEvent(ctx, tx, info)
	} else {
		err = addPSQLEvent(ctx, tx, info)
	}
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Transaction rollback failed: %v", err)
		}
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event Transaction commit failed: %v", err)
		return
	}
}

func addPSQLEvent(ctx context.Context, tx *sql.Tx, info *withdraw.Details) (err error) {
	var tempEvent = modelPSQL.WithdrawalHistory{
		ExchangeNameID: info.InternalExchangeID.String(), // UUID to exchange in DB
		ExchangeID:     info.Request.Exchange,            // String name of exchange
		Status:         info.Response.Status,
		Currency:       info.Request.Currency.String(),
		Amount:         info.Request.Amount,
		WithdrawType:   int(info.Request.Type),
		Description:    null.NewString(info.Request.Description, info.Request.Description != ""),
	}

	err = tempEvent.Insert(ctx, tx, boil.Infer())
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
		}
		return
	}

	switch info.Request.Type {
	case withdraw.Fiat:
		fiatEvent := &modelPSQL.WithdrawalFiat{
			BankName:          info.Request.Fiat.Bank.BankName,
			BankAddress:       info.Request.Fiat.Bank.BankAddress,
			BankAccountName:   info.Request.Fiat.Bank.AccountName,
			BankAccountNumber: info.Request.Fiat.Bank.AccountNumber,
			BSB:               info.Request.Fiat.Bank.BSBNumber,
			SwiftCode:         info.Request.Fiat.Bank.SWIFTCode,
			Iban:              info.Request.Fiat.Bank.IBAN,
		}
		err = tempEvent.SetWithdrawalFiatWithdrawalFiats(ctx, tx, true, fiatEvent)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
			}
			return
		}
	case withdraw.Crypto:
		cryptoEvent := &modelPSQL.WithdrawalCrypto{
			Address:    info.Request.Crypto.Address,
			Fee:        info.Request.Crypto.FeeAmount,
			AddressTag: null.NewString(info.Request.Crypto.AddressTag, info.Request.Crypto.AddressTag != ""),
		}

		err = tempEvent.AddWithdrawalCryptoWithdrawalCryptos(ctx, tx, true, cryptoEvent)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
			}
			return
		}
	}

	realID, err := uuid.FromString(tempEvent.ID)
	if err != nil {
		log.Errorf(log.DatabaseMgr,
			"Parsing UUID from inserted withdraw event error: %v",
			err)
	}
	info.InternalWithdrawalID = realID
	return nil
}

func addSQLiteEvent(ctx context.Context, tx *sql.Tx, info *withdraw.Details) (err error) {
	info.InternalWithdrawalID, err = uuid.NewV4()
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Failed to generate UUID: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
		}
		return
	}

	var tempEvent = modelSQLite.WithdrawalHistory{
		ID:             info.InternalWithdrawalID.String(),
		ExchangeNameID: info.InternalExchangeID.String(), // UUID to exchange in DB
		ExchangeID:     info.Request.Exchange,            // String name of exchange
		Status:         info.Response.Status,
		Currency:       info.Request.Currency.String(),
		Amount:         info.Request.Amount,
		WithdrawType:   int64(info.Request.Type),
		Description:    null.NewString(info.Request.Description, info.Request.Description != ""),
	}

	err = tempEvent.Insert(ctx, tx, boil.Infer())
	if err != nil {
		log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
		}
		return
	}

	switch info.Request.Type {
	case withdraw.Fiat:
		fiatEvent := &modelSQLite.WithdrawalFiat{
			BankName:          info.Request.Fiat.Bank.BankName,
			BankAddress:       info.Request.Fiat.Bank.BankAddress,
			BankAccountName:   info.Request.Fiat.Bank.AccountName,
			BankAccountNumber: info.Request.Fiat.Bank.AccountNumber,
			BSB:               info.Request.Fiat.Bank.BSBNumber,
			SwiftCode:         info.Request.Fiat.Bank.SWIFTCode,
			Iban:              info.Request.Fiat.Bank.IBAN,
		}

		err = tempEvent.AddWithdrawalFiats(ctx, tx, true, fiatEvent)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
			}
			return
		}
	case withdraw.Crypto:
		cryptoEvent := &modelSQLite.WithdrawalCrypto{
			Address:    info.Request.Crypto.Address,
			Fee:        info.Request.Crypto.FeeAmount,
			AddressTag: null.NewString(info.Request.Crypto.AddressTag, info.Request.Crypto.AddressTag != ""),
		}

		err = tempEvent.AddWithdrawalCryptos(ctx, tx, true, cryptoEvent)
		if err != nil {
			log.Errorf(log.DatabaseMgr, "Event Insert failed: %v", err)
			err = tx.Rollback()
			if err != nil {
				log.Errorf(log.DatabaseMgr, "Rollback failed: %v", err)
			}
			return
		}
	}

	return nil
}

// GetEventByUUID return requested withdraw information by ID
func GetEventByUUID(id string) (*withdraw.Details, error) {
	resp, err := getByColumns(generateWhereQuery([]string{"id"}, []string{id}, 1))
	if err != nil {
		log.Error(log.DatabaseMgr, err)
		return nil, err
	}
	return resp[0], nil
}

// GetEventsByExchange returns all withdrawal requests by exchange
func GetEventsByExchange(exchange string, limit int) ([]*withdraw.Details, error) {
	exch, err := exchangeDB.UUIDByName(exchange)
	if err != nil {
		log.Error(log.DatabaseMgr, err)
		return nil, err
	}
	return getByColumns(generateWhereQuery([]string{"exchange_name_id"}, []string{exch.String()}, limit))
}

// GetEventByExchangeID return requested withdraw information by Exchange ID
func GetEventByExchangeID(exchange, id string) (*withdraw.Details, error) {
	exch, err := exchangeDB.UUIDByName(exchange)
	if err != nil {
		log.Error(log.DatabaseMgr, err)
		return nil, err
	}
	resp, err := getByColumns(generateWhereQuery([]string{"exchange_name_id", "exchange_id"}, []string{exch.String(), id}, 1))
	if err != nil {
		return nil, err
	}
	return resp[0], err
}

// GetEventsByDate returns requested withdraw information by date range
func GetEventsByDate(exchange string, start, end time.Time, limit int) ([]*withdraw.Details, error) {
	betweenQuery := generateWhereBetweenQuery("created_at", start, end, limit)
	if exchange == "" {
		return getByColumns(betweenQuery)
	}
	exch, err := exchangeDB.UUIDByName(exchange)
	if err != nil {
		log.Error(log.DatabaseMgr, err)
		return nil, err
	}
	return getByColumns(append(generateWhereQuery([]string{"exchange_name_id"}, []string{exch.String()}, 0), betweenQuery...))
}

func generateWhereQuery(columns, id []string, limit int) []qm.QueryMod {
	var queries []qm.QueryMod
	if limit > 0 {
		queries = append(queries, qm.Limit(limit))
	}
	for x := range columns {
		queries = append(queries, qm.Where(columns[x]+"= ?", id[x]))
	}
	return queries
}

func generateWhereBetweenQuery(column string, start, end interface{}, limit int) []qm.QueryMod {
	return []qm.QueryMod{
		qm.Limit(limit),
		qm.Where(column+" BETWEEN ? AND ?", start, end),
	}
}

func getByColumns(q []qm.QueryMod) ([]*withdraw.Details, error) {
	if database.DB.SQL == nil {
		return nil, database.ErrDatabaseSupportDisabled
	}

	var resp []*withdraw.Details
	var ctx = context.Background()
	if repository.GetSQLDialect() == database.DBSQLite3 {
		v, err := modelSQLite.WithdrawalHistories(q...).
			All(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}
		for x := range v {
			var info = &withdraw.Details{}
			info.InternalWithdrawalID, err = uuid.FromString(v[x].ID)
			if err != nil {
				return nil, err
			}

			info.InternalExchangeID, err = uuid.FromString(v[x].ExchangeNameID)
			if err != nil {
				log.Errorf(log.DatabaseMgr,
					"invalid exchange name UUID for record %v",
					v[x].ID)
				return nil, err
			}

			info.Response.Status = v[x].Status
			info.Request = &withdraw.Request{
				Exchange: v[x].ExchangeID,
				Currency: currency.NewCode(v[x].Currency),
				// Asset: , // TODO
				// Account: , // TODO
				Description: v[x].Description.String,
				Amount:      v[x].Amount,
				Type:        withdraw.RequestType(v[x].WithdrawType),
			}

			info.CreatedAt, err = time.Parse(time.RFC3339, v[x].CreatedAt)
			if err != nil {
				log.Errorf(log.DatabaseMgr,
					"record: %v has an incorrect time format ( %v ) - defaulting to empty time: %v",
					info.InternalWithdrawalID,
					v[x].CreatedAt,
					err)
			}

			info.UpdatedAt, err = time.Parse(time.RFC3339, v[x].UpdatedAt)
			if err != nil {
				log.Errorf(log.DatabaseMgr,
					"record: %v has an incorrect time format ( %v ) - defaulting to empty time: %v",
					info.InternalWithdrawalID,
					v[x].UpdatedAt,
					err)
			}

			switch withdraw.RequestType(v[x].WithdrawType) {
			case withdraw.Crypto:
				x, err := v[x].WithdrawalCryptos().One(ctx, database.DB.SQL)
				if err != nil {
					return nil, err
				}
				info.Request.Crypto.Address = x.Address
				info.Request.Crypto.AddressTag = x.AddressTag.String
				info.Request.Crypto.FeeAmount = x.Fee
			case withdraw.Fiat:
				x, err := v[x].WithdrawalFiats().One(ctx, database.DB.SQL)
				if err != nil {
					return nil, err
				}
				info.Request.Fiat.Bank.AccountName = x.BankAccountName
				info.Request.Fiat.Bank.AccountNumber = x.BankAccountNumber
				info.Request.Fiat.Bank.IBAN = x.Iban
				info.Request.Fiat.Bank.SWIFTCode = x.SwiftCode
				info.Request.Fiat.Bank.BSBNumber = x.BSB
			default:
			}
			resp = append(resp, info)
		}
	} else {
		v, err := modelPSQL.WithdrawalHistories(q...).All(ctx, database.DB.SQL)
		if err != nil {
			return nil, err
		}

		for x := range v {
			var info = &withdraw.Details{
				CreatedAt: v[x].CreatedAt,
				UpdatedAt: v[x].UpdatedAt,
			}
			info.InternalWithdrawalID, err = uuid.FromString(v[x].ID)
			if err != nil {
				return nil, err
			}

			info.InternalExchangeID, err = uuid.FromString(v[x].ExchangeNameID)
			if err != nil {
				return nil, err
			}

			info.Response.Status = v[x].Status
			info.Request = &withdraw.Request{
				Exchange: v[x].ExchangeID,
				Currency: currency.NewCode(v[x].Currency),
				// Asset: , TODO:
				// Account: , TODO:
				Description: v[x].Description.String,
				Amount:      v[x].Amount,
				Type:        withdraw.RequestType(v[x].WithdrawType),
			}

			switch withdraw.RequestType(v[x].WithdrawType) {
			case withdraw.Crypto:
				x, err := v[x].WithdrawalCryptoWithdrawalCryptos().
					One(ctx, database.DB.SQL)
				if err != nil {
					return nil, err
				}
				info.Request.Crypto.Address = x.Address
				info.Request.Crypto.AddressTag = x.AddressTag.String
				info.Request.Crypto.FeeAmount = x.Fee
			case withdraw.Fiat:
				x, err := v[x].WithdrawalFiatWithdrawalFiats().
					One(ctx, database.DB.SQL)
				if err != nil {
					return nil, err
				}
				info.Request.Fiat.Bank.AccountName = x.BankAccountName
				info.Request.Fiat.Bank.AccountNumber = x.BankAccountNumber
				info.Request.Fiat.Bank.IBAN = x.Iban
				info.Request.Fiat.Bank.SWIFTCode = x.SwiftCode
				info.Request.Fiat.Bank.BSBNumber = x.BSB
			default:
			}
			resp = append(resp, info)
		}
	}
	if len(resp) == 0 {
		return nil, ErrNoResults
	}
	return resp, nil
}
