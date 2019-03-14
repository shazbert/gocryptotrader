package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/database/base"
	"github.com/thrasher-/gocryptotrader/database/sqlite3/models"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"

	// External package for SQL queries
	_ "github.com/volatiletech/sqlboiler-sqlite3/driver"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// consts defined here reproduce standard query strings used throughout this
// package
const (
	QueryExchangeName    = "exchange_name = ?"
	QueryCurrencyPair    = "currency_pair = ?"
	QueryAssetType       = "asset_type = ?"
	QueryUserName        = "user_name = ?"
	OrderByFulfilledDesc = "fulfilled_on DESC"
)

var (
	ctx = context.Background()

	// ErrDatabaseConnection defines a database connection failure error
	ErrDatabaseConnection = errors.New("database connection not established")
)

// SQLite3 defines a connection to a SQLite3 database
type SQLite3 struct {
	base.RelationalMap
	exch map[string]*models.Exchange
}

// Setup creates and sets database directory, folders and supplementary files
// that works in conjunction with SQLBoiler to regenerate models
func (s *SQLite3) Setup(c base.ConnDetails) error {
	if c.DirectoryPath == "" {
		return errors.New("directory path not set")
	}

	if c.SQLPath == "" {
		return errors.New("full path to SQLite3 database not set")
	}

	s.PathToDB = c.SQLPath
	s.Verbose = c.Verbose
	s.InstanceName = base.SQLite
	s.PathDBDir = c.DirectoryPath

	// Checks to see if default directory is made
	err := common.CheckDir(s.PathDBDir, true)
	if err != nil {
		return err
	}

	err = s.SetupHelperFiles()
	if err != nil {
		return err
	}

	fullPathToSchema := c.DirectoryPath + base.SQLite3Schema
	// Creates a schema file for informational deployment
	_, err = common.ReadFile(fullPathToSchema)
	if err != nil {
		var fullSchema string

		fullSchema += sqliteSchema["client"] + "\n\n"
		fullSchema += sqliteSchema["exchange"] + "\n\n"
		fullSchema += sqliteSchema["client_order_history"] + "\n\n"
		fullSchema += sqliteSchema["exchange_platform_trade_history"]

		err = common.WriteFile(fullPathToSchema, []byte(fullSchema))
		if err != nil {
			return err
		}
		if s.Verbose {
			log.Debugf("Created schema file for database update and SQLBoiler model deployment %s",
				fullPathToSchema)
		}
	} else {
		if s.Verbose {
			log.Debugf("Schema file found at %s",
				fullPathToSchema)
		}
	}
	return nil
}

// Connect initiates a connection to a SQLite database
func (s *SQLite3) Connect() error {
	if s.PathToDB == "" {
		return fmt.Errorf(base.DBPathNotSet, s.InstanceName)
	}

	if s.Verbose {
		log.Debugf(base.DBConnecting, s.InstanceName, s.PathToDB)
	}

	var err error
	s.C, err = sql.Open(base.SQLite, s.PathToDB)
	if err != nil {
		return err
	}

	err = s.C.Ping()
	if err != nil {
		return err
	}

	// Instantiate tables in new SQLite3 database
	for name, query := range sqliteSchema {
		rows, err := s.C.Query(
			fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name='%s'",
				name))
		if err != nil {
			return err
		}

		var returnedName string
		for rows.Next() {
			rows.Scan(&returnedName)
		}

		if returnedName == name {
			continue
		}

		stmt, err := s.C.Prepare(query)
		if err != nil {
			return err
		}

		_, err = stmt.Exec()
		if err != nil {
			return err
		}
	}

	s.Connected = true
	return nil
}

// ClientLogin creates or logs in to a saved user profile
func (s *SQLite3) ClientLogin() error {
	for {
		username, err := common.PromptForUsername()
		if err != nil {
			return err
		}

		users, err := models.Clients(qm.Where(QueryUserName, username)).All(ctx, s.C)
		if err != nil {
			return err
		}

		if len(users) > 1 {
			return errors.New("duplicate users found in database")
		}

		if len(users) == 1 {
			for tries := 3; tries > 0; tries-- {
				pw, err := common.ComparePassword([]byte(users[0].Password))
				if err != nil {
					fmt.Println("Incorrect password, try again.")
					continue
				}
				return s.SetSessionData(username, pw)
			}

			return fmt.Errorf("Failed to authenticate using password for username %s",
				username)
		}

		var decision string
		fmt.Printf("Username %s not found in database, would you like to create a new user, enter [y,n],\nthen press enter to continue.\n",
			username)
		fmt.Scanln(&decision)

		if common.YesOrNo(decision) {
			pw, err := common.PromptForPassword(true)
			if err != nil {
				return err
			}

			err = s.InsertNewClient(username, pw)
			if err != nil {
				return err
			}

			return s.SetSessionData(username, pw)
		}
	}
}

// InsertNewClient inserts a new client by username and password
func (s *SQLite3) InsertNewClient(username string, password []byte) error {
	exists, err := models.Clients(qm.Where(QueryUserName, username)).Exists(ctx, s.C)
	if err != nil {
		return err
	}

	if exists {
		return errors.New("username already found")
	}

	hashPw, err := common.HashPassword(password)
	if err != nil {
		return err
	}

	newuser := &models.Client{
		UserName:     username,
		Password:     hashPw,
		LastLoggedIn: time.Now(),
	}

	return newuser.Insert(ctx, s.C, boil.Infer())
}

// SetSessionData sets user data for handling client/database connection
func (s *SQLite3) SetSessionData(username string, cred []byte) error {
	user, err := models.Clients(qm.Where(QueryUserName, username)).One(ctx, s.C)
	if err != nil {
		return err
	}

	s.SessionID = user.ID
	s.SessionCred = cred

	user.LastLoggedIn = time.Now()

	_, err = user.Update(ctx, s.C, boil.Infer())
	if err != nil {
		return err
	}
	return nil
}

// InsertPlatformTrade inserts platform matched trades
func (s *SQLite3) InsertPlatformTrade(orderID, exchangeName, currencyPair, assetType, orderType string, amount, rate float64, fulfilledOn time.Time) error {
	s.Lock()
	defer s.Unlock()

	if !s.Connected {
		return ErrDatabaseConnection
	}

	e, err := s.InsertAndRetrieveExchange(exchangeName)
	if err != nil {
		return err
	}

	return e.SetExchangePlatformTradeHistory(ctx,
		s.C,
		true,
		&models.ExchangePlatformTradeHistory{
			FulfilledOn:  fulfilledOn,
			CurrencyPair: currencyPair,
			AssetType:    assetType,
			OrderType:    orderType,
			Amount:       amount,
			Rate:         rate,
			OrderID:      orderID,
		})
}

// InsertAndRetrieveExchange returns exchange bra stuff
func (s *SQLite3) InsertAndRetrieveExchange(exchName string) (*models.Exchange, error) {
	if s.exch == nil {
		s.exch = make(map[string]*models.Exchange)
	}

	e, ok := s.exch[exchName]
	if !ok {
		var err error
		e, err = models.Exchanges(qm.Where("exchange_name = ?", exchName)).One(ctx, s.C)
		if err != nil {
			i := &models.Exchange{
				ExchangeName: exchName,
			}

			err = i.Insert(ctx, s.C, boil.Infer())
			if err != nil {
				return nil, err
			}

			err = i.Reload(ctx, s.C)
			if err != nil {
				return nil, err
			}

			e = i
		}
	}

	s.exch[exchName] = e
	return e, nil
}

// GetPlatformTradeLast returns the last updated time.Time and tradeID values
// for the most recent trade history data in the set
func (s *SQLite3) GetPlatformTradeLast(exchangeName, currencyPair, assetType string) (time.Time, string, error) {
	s.Lock()
	defer s.Unlock()

	if !s.Connected {
		return time.Time{}, "", ErrDatabaseConnection
	}

	e, err := s.InsertAndRetrieveExchange(exchangeName)
	if err != nil {
		return time.Time{}, "", err
	}

	th, err := e.ExchangePlatformTradeHistory(qm.Where(QueryCurrencyPair, currencyPair),
		qm.And(QueryAssetType, assetType),
		qm.OrderBy(OrderByFulfilledDesc),
		qm.Limit(1)).One(ctx, s.C)
	if err != nil {
		return time.Time{}, "", err
	}

	return th.FulfilledOn, th.OrderID, nil
}

// GetFullPlatformHistory returns the full matched trade history on the
// exchange platform by exchange name, currency pair and asset class
func (s *SQLite3) GetFullPlatformHistory(exchangeName, currencyPair, assetType string) ([]exchange.PlatformTrade, error) {
	s.Lock()
	defer s.Unlock()

	if !s.Connected {
		return nil, ErrDatabaseConnection
	}

	e, err := s.InsertAndRetrieveExchange(exchangeName)
	if err != nil {
		return nil, err
	}

	h, err := e.ExchangePlatformTradeHistory(qm.Where(QueryCurrencyPair, currencyPair),
		qm.And(QueryAssetType, assetType)).All(ctx, s.C)
	if err != nil {
		return nil, err
	}

	var platformHistory []exchange.PlatformTrade
	for i := range h {
		platformHistory = append(platformHistory,
			exchange.PlatformTrade{
				Exchange:  e.ExchangeName,
				Timestamp: h[i].FulfilledOn,
				TID:       h[i].OrderID,
				Price:     h[i].Rate,
				Amount:    h[i].Amount,
				Type:      h[i].OrderType})
	}

	return platformHistory, nil
}

// GetClientDetails returns a string of current user details
func (s *SQLite3) GetClientDetails() (string, error) {
	if !s.Connected {
		return "", ErrDatabaseConnection
	}

	s.Lock()
	defer s.Unlock()

	q, err := models.Clients(qm.Where("id = ?", s.SessionID)).All(ctx, s.C)
	if err != nil {
		return "", err
	}

	if len(q) != 1 {
		return "", errors.New("query failure returned incorrect user information")
	}

	return q[0].UserName, nil
}
