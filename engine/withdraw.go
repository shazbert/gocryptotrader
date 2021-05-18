package engine

import (
	"fmt"
	"time"

	withdrawDataStore "github.com/thrasher-corp/gocryptotrader/database/repository/withdraw"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// ErrWithdrawRequestNotFound message to display when no record is found
	ErrWithdrawRequestNotFound = "%v not found %w"
	// ErrRequestCannotbeNil message to display when request is nil
	ErrRequestCannotbeNil = "request cannot be nil"
	// StatusError const for for "error" string
	StatusError = "error"
)

// SubmitWithdrawal performs validation and submits a new withdraw request to
// exchange
func (bot *Engine) SubmitWithdrawal(req *withdraw.Request) (*withdraw.Details, error) {
	if req == nil {
		return nil, withdraw.ErrRequestCannotBeNil
	}

	exch := bot.GetExchangeByName(req.Exchange)
	if exch == nil {
		return nil, ErrExchangeNotFound
	}

	info := &withdraw.Details{Request: req}
	if bot.Settings.EnableDryRun {
		log.Warnln(log.Global, "Dry run enabled, no withdrawal request will be submitted or have an event created")
		info.InternalWithdrawalID = withdraw.DryRunID
		info.Response.Status = "dryrun"
		info.Response.WithdrawalID = withdraw.DryRunID.String()
		// Opted for dry run information to not be deployed into database as we
		// will have conflicting UUIDS
		return info, nil
	}

	claim, err := exch.Claim(req.Account, req.Asset, req.Currency, req.Amount, true)
	if err != nil {
		return nil, err
	}
	var resp *withdraw.Response
	if req.Type == withdraw.Fiat {
		resp, err = exch.WithdrawFiatFunds(req)
	} else if req.Type == withdraw.Crypto {
		resp, err = exch.WithdrawCryptocurrencyFunds(req)
	}
	if err != nil {
		info.Response.Status = err.Error()
		err = claim.Release()
		if err != nil {
			log.Errorln(log.Global, err)
		}
	} else {
		info.Response = *resp
		if resp.ReduceAccountHoldings {
			// In the event we cannot wait for pending to be reduced
			// (i.e. exchange does not support this feature) we can
			// immediately reduce holdings now.
			err = claim.ReleaseAndReduce()
			if err != nil {
				log.Errorln(log.Global, err)
			}
		} else {
			err = claim.ReleaseToPending()
			if err != nil {
				log.Errorln(log.Global, err)
			}
		}
	}
	// Even on error, all withdrawal events will be saved to database.
	withdrawDataStore.Event(info)

	// Is an LRU cache neccessary for withdrawal information? TODO: might remove
	// this feature as historical events can be fetched from DB.
	withdraw.Cache.Add(info.InternalWithdrawalID, resp)

	return info, nil
}

// WithdrawalEventByID returns a withdrawal request by ID
func WithdrawalEventByID(id string) (*withdraw.Details, error) {
	v := withdraw.Cache.Get(id)
	if v != nil {
		info, ok := v.(*withdraw.Details)
		if !ok {
			return nil, fmt.Errorf("type assertion failure when retrieving from the LRU cache")
		}
		return info, nil
	}

	l, err := withdrawDataStore.GetEventByUUID(id)
	if err != nil {
		return nil, fmt.Errorf(ErrWithdrawRequestNotFound, id, err)
	}
	withdraw.Cache.Add(id, l)
	return l, nil
}

// WithdrawalEventByExchange returns a withdrawal request by ID
func WithdrawalEventByExchange(exchange string, limit int) ([]*withdraw.Details, error) {
	return withdrawDataStore.GetEventsByExchange(exchange, limit)
}

// WithdrawEventByDate returns a withdrawal request by ID
func WithdrawEventByDate(exchange string, start, end time.Time, limit int) ([]*withdraw.Details, error) {
	return withdrawDataStore.GetEventsByDate(exchange, start, end, limit)
}

// WithdrawalEventByExchangeID returns a withdrawal request by Exchange ID
func WithdrawalEventByExchangeID(exchange, id string) (*withdraw.Details, error) {
	return withdrawDataStore.GetEventByExchangeID(exchange, id)
}

func parseMultipleEvents(ret []*withdraw.Details) *gctrpc.WithdrawalEventsByExchangeResponse {
	v := &gctrpc.WithdrawalEventsByExchangeResponse{}
	for x := range ret {
		v.Event = append(v.Event, getGRPCWithdrawalEventResponse(ret[x]))
	}
	return v
}

// todo: change ret
func parseWithdrawalsHistory(ret []exchange.WithdrawalHistory, exchName string, limit int) *gctrpc.WithdrawalEventsByExchangeResponse {
	v := &gctrpc.WithdrawalEventsByExchangeResponse{}
	for x := range ret {
		if limit > 0 && x >= limit {
			return v
		}

		updatedAt := timestamppb.New(ret[x].Timestamp)
		if err := updatedAt.CheckValid(); err != nil {
			log.Errorf(log.Global, "withdrawal parseWithdrawalsHistory UpdatedAt: %s", err)
		}

		v.Event = append(v.Event, &gctrpc.WithdrawalEventResponse{
			Exchange: &gctrpc.WithdrawlExchangeEvent{
				WithdrawalId:  ret[x].TransferID,
				TransactionId: ret[x].CryptoTxID,
				Status:        ret[x].Status,
			},
			Request: &gctrpc.WithdrawalRequestEvent{
				Currency:    ret[x].Currency,
				Description: ret[x].Description,
				Amount:      ret[x].Amount,
				Crypto: &gctrpc.CryptoWithdrawalEvent{
					Address: ret[x].CryptoToAddress,
					Fee:     ret[x].Fee,
				},
			},
			UpdatedAt: updatedAt,
		})
	}
	return v
}

func parseSingleEvents(ret *withdraw.Details) *gctrpc.WithdrawalEventsByExchangeResponse {
	return &gctrpc.WithdrawalEventsByExchangeResponse{
		Event: []*gctrpc.WithdrawalEventResponse{getGRPCWithdrawalEventResponse(ret)},
	}
}

func getGRPCWithdrawalEventResponse(info *withdraw.Details) *gctrpc.WithdrawalEventResponse {
	createdAt := timestamppb.New(info.CreatedAt)
	if err := createdAt.CheckValid(); err != nil {
		log.Errorf(log.Global, "withdrawal parseSingleEvents CreatedAt %s", err)
	}
	updatedAt := timestamppb.New(info.UpdatedAt)
	if err := updatedAt.CheckValid(); err != nil {
		log.Errorf(log.Global, "withdrawal parseSingleEvents UpdatedAt: %s", err)
	}

	grpcEvent := &gctrpc.WithdrawalEventResponse{
		GctWithdrawalUuid: info.InternalWithdrawalID.String(),
		Exchange: &gctrpc.WithdrawlExchangeEvent{
			WithdrawalId:  info.Response.WithdrawalID,
			TransactionId: info.Response.TransactionID,
			Status:        info.Response.Status,
		},
		Request: &gctrpc.WithdrawalRequestEvent{
			Exchange:    info.Request.Exchange,
			Currency:    info.Request.Currency.String(),
			Description: info.Request.Description,
			Amount:      info.Request.Amount,
			Type:        int32(info.Request.Type),
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	switch info.Request.Type {
	case withdraw.Crypto:
		grpcEvent.Request.Crypto = &gctrpc.CryptoWithdrawalEvent{
			Address:    info.Request.Crypto.Address,
			AddressTag: info.Request.Crypto.AddressTag,
			Fee:        info.Request.Crypto.FeeAmount,
		}
	case withdraw.Fiat:
		grpcEvent.Request.Fiat = &gctrpc.FiatWithdrawalEvent{
			BankName:      info.Request.Fiat.Bank.BankName,
			AccountName:   info.Request.Fiat.Bank.AccountName,
			AccountNumber: info.Request.Fiat.Bank.AccountNumber,
			Bsb:           info.Request.Fiat.Bank.BSBNumber,
			Swift:         info.Request.Fiat.Bank.SWIFTCode,
			Iban:          info.Request.Fiat.Bank.IBAN,
		}
	default:
		// TODO: Maybe we can add a warning but not a serious issue
	}
	return grpcEvent
}
