package sqlite3

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/database/base"
	"github.com/thrasher-/gocryptotrader/database/sqlite3/models"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"

	// External package for SQL queries
	_ "github.com/volatiletech/sqlboiler-sqlite3/driver"
)

// SQLite3 defines a connection to a SQLite3 database
type SQLite3 struct {
	base.RelationalMap
}

// Setup creates and sets database directory, folders and supplementary files
// that works in conjunction with SQLBoiler to regenerate models
func (s *SQLite3) Setup(c base.ConnDetails) error {
	if c.DirectoryPath == "" {
		return base.ErrDirectoryNotSet
	}

	if c.SQLPath == "" {
		return errors.New("full path to SQLite3 database not set")
	}

	s.PathToDB = c.SQLPath
	s.Verbose = c.Verbose
	s.InstanceName = base.SQLite
	s.PathDBDir = c.DirectoryPath

	if c.MemCacheSize == 0 {
		s.MaxSizeOfCache = base.DefaultMemCache
	} else {
		log.Warnf("Database write buffer size %d is not default", c.MemCacheSize)
		s.MaxSizeOfCache = c.MemCacheSize
	}

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
		err = common.WriteFile(fullPathToSchema, []byte(GetSchema()))
		if err != nil {
			return err
		} else if s.Verbose {
			log.Debugf(base.DebugSchemaFileCreated, fullPathToSchema)
		}
	} else if s.Verbose {
		log.Debugf(base.DebugSchemaFileFound, fullPathToSchema)
	}
	return nil
}

// GetSchema returns the full schema ready for file use
func GetSchema() string {
	var fullSchema string
	for i, s := range sqliteSchema {
		fullSchema += s
		if len(sqliteSchema)-1 != i {
			fullSchema += "\n\n"
		}
	}
	return fullSchema
}

// Connect initiates a connection to a SQLite database
func (s *SQLite3) Connect() error {
	if s.PathToDB == "" {
		return fmt.Errorf(base.DBPathNotSet, s.InstanceName)
	}

	if s.Verbose {
		log.Debugf(base.DebugDBConnecting, s.InstanceName, s.PathToDB)
	}

	var err error
	s.C, err = sql.Open(base.SQLite, s.PathToDB)
	if err != nil {
		return err
	}

	err = s.C.Ping()
	if err != nil {
		err = s.Disconnect()
		if err != nil {
			log.Error("Disconnection from sqlite3 db error", err)
		}
		return err
	}

	rows, err := s.C.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var supername string
		rows.Scan(&supername)

		if !common.StringContains(supername, "sqlite3_") {
			log.Warnf(base.WarnTablesExist)
			s.Connected = true
			return nil
		}
	}

	// Instantiate tables in new SQLite3 database if no random tables found
	for _, query := range sqliteSchema {
		stmt, err := s.C.Prepare(query)
		if err != nil {
			return err
		}

		_, err = stmt.Exec()
		if err != nil {
			return err
		}
	}

	err = s.InsertAccessControl(base.GetAccessLevels())
	if err != nil {
		return err
	}

	err = s.InsertExchanges(base.GetSupportedExchanges())
	if err != nil {
		return err
	}

	s.Connected = true
	return nil
}

// ClientLogin creates or logs in to a saved user profile
func (s *SQLite3) ClientLogin(newclient bool) error {
	fmt.Println()
	if newclient {
		log.Info(base.InfoInsertClient)
		return s.InsertNewClientByPrompt()
	}

	clients, err := models.Clients().All(base.Ctx, s.C)
	if err != nil {
		return err
	}

	if len(clients) == 0 {
		log.Info(base.InfoNoClients)
		return s.InsertNewClientByPrompt()
	}

	if len(clients) == 1 {
		log.Info(base.InfoSingleClient)
		return s.CheckClientPassword(clients[0])
	}

	log.Info(base.InfoMultiClient)
	return s.CheckClientUserPassword(clients)
}

// CheckClientUserPassword matches username and checks client password with
// account
func (s *SQLite3) CheckClientUserPassword(c models.ClientSlice) error {
	username, err := common.PromptForUsername()
	if err != nil {
		return err
	}

	for i := range c {
		if c[i].UserName == username {
			log.Infof(base.InfoUserNameFound, username)
			return s.CheckClientPassword(c[i])
		}
	}

	return fmt.Errorf(base.UsernameNotFound, username)
}

// CheckClientPassword matches password and sets user client
func (s *SQLite3) CheckClientPassword(c *models.Client) error {
	for tries := 3; tries > 0; tries-- {
		_, err := common.ComparePassword([]byte(c.Password))
		if err != nil {
			if tries != 1 {
				log.Warnf(base.WarnWrongPassword, tries-1)
			}
			continue
		}

		s.Client = c
		return nil
	}
	return fmt.Errorf(base.LoginFailure, c.UserName)
}

// InsertNewClientByPrompt inserts a new client by username and password
// prompt when starting a new gocryptotrader instance
func (s *SQLite3) InsertNewClientByPrompt() error {
	username, err := common.PromptForUsername()
	if err != nil {
		return err
	}

	e, err := models.Clients(qm.Where(base.QueryUserName,
		username)).Exists(base.Ctx, s.C)
	if err != nil {
		return err
	}

	if e {
		return fmt.Errorf(base.UsernameAlreadyUsed, username)
	}

	pw, err := common.PromptForPassword(true)
	if err != nil {
		return err
	}

	hashPw, err := common.HashPassword(pw)
	if err != nil {
		return err
	}

	newuser := &models.Client{
		UserName:          username,
		Password:          hashPw,
		PasswordCreatedAt: time.Now(),
		LastLoggedIn:      time.Now(),
		Enabled:           true,
	}

	basicAccess := &models.AccessControl{
		Level: int64(base.Basic),
	}

	err = basicAccess.AddAccessLevelClients(base.Ctx, s.C, true, newuser)
	if err != nil {
		return err
	}

	err = newuser.Reload(base.Ctx, s.C)
	if err != nil {
		return err
	}

	s.Client = newuser
	return nil
}

// InsertPlatformTrades inserts platform matched trades
func (s *SQLite3) InsertPlatformTrades(exchangeName string, trades []*base.PlatformTrades) error {
	s.Lock()
	defer s.Unlock()

	if !s.Connected {
		return base.ErrDatabaseConnection
	}

	e, err := s.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return err
	}

	tx, err := s.NewTx()
	if err != nil {
		return err
	}

	for i := range trades {
		newStatement := &models.ExchangePlatformTradeHistory{
			FulfilledOn:  trades[i].FullfilledOn,
			CurrencyPair: trades[i].Pair,
			AssetType:    trades[i].AssetType,
			OrderType:    trades[i].OrderType,
			Amount:       trades[i].Amount,
			Rate:         trades[i].Rate,
			OrderID:      trades[i].OrderID,
			ExchangeID:   e.ID,
		}
		stmErr := newStatement.Insert(base.Ctx, tx, boil.Infer())
		if stmErr != nil {
			return stmErr
		}
	}

	return s.CommitTx(len(trades))
}

// InsertAndRetrieveExchange returns the pointer to an exchange model to
// minimise database queries for future insertion, used in conjunction with
// lockable funcs
func (s *SQLite3) insertAndRetrieveExchange(exchName string) (*models.Exchange, error) {
	if s.Exchanges == nil {
		s.Exchanges = make(map[string]interface{})
	}

	e, ok := s.Exchanges[exchName].(*models.Exchange)
	if !ok {
		var err error
		e, err = models.Exchanges(qm.Where(base.QueryExchangeName, exchName)).One(base.Ctx, s.C)
		if err != nil {
			i := &models.Exchange{
				ExchangeName: exchName,
			}

			err = i.Insert(base.Ctx, s.C, boil.Infer())
			if err != nil {
				return nil, err
			}

			err = i.Reload(base.Ctx, s.C)
			if err != nil {
				return nil, err
			}

			e = i
		}
	}

	s.Exchanges[exchName] = e
	return e, nil
}

// GetPlatformTradeLast returns the last updated time.Time and tradeID values
// for the most recent trade history data in the set
func (s *SQLite3) GetPlatformTradeLast(exchangeName, currencyPair, assetType string) (time.Time, string, error) {
	s.Lock()
	err := s.NewQuery()
	if err != nil {
		return time.Time{}, "", err
	}

	defer func() { s.FinishQuery(); s.Unlock() }()

	if !s.Connected {
		return time.Time{}, "", base.ErrDatabaseConnection
	}

	e, err := s.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return time.Time{}, "", err
	}

	th, err := e.ExchangePlatformTradeHistory(qm.Where(base.QueryCurrencyPair, currencyPair),
		qm.And(base.QueryAssetType, assetType),
		qm.OrderBy(base.OrderByFulfilledDesc),
		qm.Limit(1)).One(base.Ctx, s.C)
	if err != nil {
		return time.Time{}, "", err
	}

	return th.FulfilledOn, th.OrderID, nil
}

// GetPlatformTradeFirst returns the first updated time.Time and tradeID values
// for the initial entry boundary points
func (s *SQLite3) GetPlatformTradeFirst(exchangeName, currencyPair, assetType string) (time.Time, string, error) {
	s.Lock()
	err := s.NewQuery()
	if err != nil {
		return time.Time{}, "", err
	}

	defer func() { s.Unlock(); s.FinishQuery() }()

	if !s.Connected {
		return time.Time{}, "", base.ErrDatabaseConnection
	}

	e, err := s.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return time.Time{}, "", err
	}

	th, err := e.ExchangePlatformTradeHistory(qm.Where(base.QueryCurrencyPair, currencyPair),
		qm.And(base.QueryAssetType, assetType),
		qm.OrderBy(base.OrderByFullfilledAsc),
		qm.Limit(1)).One(base.Ctx, s.C)
	if err != nil {
		return time.Time{}, "", err
	}

	return th.FulfilledOn, th.OrderID, nil
}

// GetFullPlatformHistory returns the full matched trade history on the
// exchange platform by exchange name, currency pair and asset class
func (s *SQLite3) GetFullPlatformHistory(exchangeName, currencyPair, assetType string) ([]exchange.PlatformTrade, error) {
	s.Lock()
	err := s.NewQuery()
	if err != nil {
		return nil, err
	}

	defer func() { s.Unlock(); s.FinishQuery() }()

	if !s.Connected {
		return nil, base.ErrDatabaseConnection
	}

	e, err := s.insertAndRetrieveExchange(exchangeName)
	if err != nil {
		return nil, err
	}

	h, err := e.ExchangePlatformTradeHistory(qm.Where(base.QueryCurrencyPair, currencyPair),
		qm.And(base.QueryAssetType, assetType)).All(base.Ctx, s.C)
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
	s.Lock()
	defer s.Unlock()
	return s.Client.(*models.Client).UserName, nil
}

// InsertAccessControl inserts bot access list
func (s *SQLite3) InsertAccessControl(m map[string]int) error {
	s.Lock()
	defer s.Unlock()

	for k, v := range m {
		control := &models.AccessControl{
			Level: int64(v),
			Name:  k,
		}

		err := control.Insert(base.Ctx, s.C, boil.Infer())
		if err != nil {
			return err
		}
	}
	return nil
}

// InsertExchanges inserts exchange data
func (s *SQLite3) InsertExchanges(e []string) error {
	s.Lock()
	defer s.Unlock()

	for i := range e {
		exchange := &models.Exchange{
			ExchangeName: e[i],
		}

		err := exchange.Insert(base.Ctx, s.C, boil.Infer())
		if err != nil {
			return err
		}
	}
	return nil
}
