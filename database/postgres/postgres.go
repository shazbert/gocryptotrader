package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/database/base"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"

	// // dont tell me how to live my life linter!
	// _ "github.com/volatiletech/sqlboiler/drivers/sqlboiler-psql/driver"
	_ "github.com/lib/pq"
)

const (
	conn = "user=%s password=%s dbname=%s sslmode=require host=%s"
)

// Postgres defines a connection to a SQLite3 database
type Postgres struct {
	base.RelationalMap
}

var ctx = context.Background
var host = "gocryptotrader-db.postgres.database.azure.com"
var port = 5432
var user = "gctbot@gocryptotrader-db"
var password = "1234"
var database = "test"

//port=%d sslmode=require

// Setup creates and sets database directory, folders and supplementary files
// that works in conjunction with PosgreSQL to regenerate models
func (p *Postgres) Setup(c base.ConnDetails) error {
	if c.DirectoryPath == "" {
		return errors.New("directory path for database not set")
	}

	if c.Host == "" {
		return errors.New("host not set for postgres connection, please set in flag -dbhost")
	}

	if c.User == "" {
		return errors.New("username not set for postgres connection, please set in flag -dbuser")
	}

	if c.DBName == "" {
		return errors.New("name not set for postgres connection, please set in flag -dbname")
	}

	if c.Pass == "" {
		log.Warnf("Password not set for the postgreSQL connection, please set in flag -dbpass")
	}

	p.InstanceName = base.Postgres
	p.PathDBDir = c.DirectoryPath
	p.DatabaseName = c.DBName
	p.Host = c.Host
	p.Password = c.Pass
	p.User = c.User
	p.Verbose = c.Verbose

	err := p.SetupHelperFiles()
	if err != nil {
		return err
	}

	fullPathToSchema := p.PathDBDir + base.SQLite3Schema
	// Creates a schema file for informational deployment
	_, err = common.ReadFile(fullPathToSchema)
	if err != nil {
		var fullSchema string

		fullSchema += postgresSchema["client"] + "\n\n"
		fullSchema += postgresSchema["exchange"] + "\n\n"
		fullSchema += postgresSchema["client_order_history"] + "\n\n"
		fullSchema += postgresSchema["exchange_platform_trade_history"]

		err = common.WriteFile(fullPathToSchema, []byte(fullSchema))
		if err != nil {
			return err
		}
		if p.Verbose {
			log.Debugf("Created schema file for database update and SQLBoiler model deployment %s",
				fullPathToSchema)
		}
	} else {
		if p.Verbose {
			log.Debugf("Schema file found at %s", fullPathToSchema)
		}
	}
	return nil
}

// Connect initiates a connection to a SQLite database
func (p *Postgres) Connect() error {
	if p.Host == "" {
		return fmt.Errorf("connect error host not set for %s", p.InstanceName)
	}

	if p.DatabaseName == "" {
		return fmt.Errorf("connect error database not set for %s",
			p.InstanceName)
	}

	if p.User == "" {
		return fmt.Errorf("connect error user not set for %s", p.InstanceName)
	}

	if p.Verbose {
		log.Debugf(base.DBConnecting, p.InstanceName, p.DatabaseName)
	}

	newCon := fmt.Sprintf(conn, p.User, p.Password, p.DatabaseName, p.Host)

	log.Debug(newCon)

	var err error
	p.C, err = sql.Open(base.Postgres, newCon)
	if err != nil {
		return err
	}

	log.Debug("Connected")

	err = p.C.Ping()
	if err != nil {
		return err
	}

	log.Debug("Pinged")

	// Instantiate tables in new postgres database
	for name, query := range postgresSchema {
		rows, err := p.C.Query(
			fmt.Sprintf("SELECT %s FROM information_schema.tables WHERE table_schema='public'",
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

		stmt, err := p.C.Prepare(query)
		if err != nil {
			return err
		}

		_, err = stmt.Exec()
		if err != nil {
			return err
		}
	}

	p.Connected = true

	return nil
}

// ClientLogin creates or logs in to a saved user profile
func (p *Postgres) ClientLogin() error {
	return nil
}

// InsertNewClient inserts a new client by username and password
func (p *Postgres) InsertNewClient(username string, password []byte) error {
	return nil
}

// SetSessionData sets user data for handling user/database connection
func (p *Postgres) SetSessionData(username string, cred []byte) error {
	return nil
}

// InsertPlatformTrade inserts platform matched trades
func (p *Postgres) InsertPlatformTrade(orderID,
	exchangeName,
	currencyPair,
	assetType,
	orderType string,
	amount,
	rate float64,
	fulfilledOn time.Time) error {
	return nil
}

// InsertAndRetrieveExchange returns exchange bra stuff
func (p *Postgres) InsertAndRetrieveExchange(exchName string) error {
	return nil
}

// GetPlatformTradeLast returns the last updated time.Time and tradeID values
// for the most recent trade history data in the set
func (p *Postgres) GetPlatformTradeLast(exchangeName, currencyPair, assetType string) (time.Time, string, error) {
	return time.Time{}, "", nil
}

// GetFullPlatformHistory returns the full matched trade history on the
// exchange platform by exchange name, currency pair and asset class
func (p *Postgres) GetFullPlatformHistory(exchangeName, currencyPair, assetType string) ([]exchange.PlatformTrade, error) {
	return nil, nil
}

// GetClientDetails returns a string of current user details
func (p *Postgres) GetClientDetails() (string, error) {
	return "", nil
}

// Disconnect closes the database connection
func (p *Postgres) Disconnect() error {
	return nil
}
