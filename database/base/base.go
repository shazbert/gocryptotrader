package base

import (
	"database/sql"
	"sync"

	"github.com/naoina/toml"
	"github.com/thrasher-/gocryptotrader/common"
	log "github.com/thrasher-/gocryptotrader/logger"

	_ "github.com/lib/pq"
)

// Exported strings for database packages
const (
	SQLBoilerToml  = "sqlboiler.toml"
	SQLite3Schema  = "sqlite3.schema"
	PostGresSchema = "postgres.schema"

	SQLite   = "sqlite3"
	Postgres = "postgres"

	CreatedLog = "Created helper file for SQLBoiler model deployment %s"
	FoundLog   = "SQLBoiler file found at %s, verifying contents.."

	DBPathNotSet = "path to %s database not set"
	DBConnecting = "Opening connection to %s database using PATH: %s"
)

// RelationalMap defines a mapping of variables specific to an individual
// database
type RelationalMap struct {
	C            *sql.DB
	InstanceName string
	Enabled      bool
	Connected    bool
	Verbose      bool
	SessionID    int64
	SessionCred  []byte

	// Pathways to folders and instances
	PathToDB  string
	PathDBDir string

	// Connection fields
	DatabaseName string
	Host         string
	User         string
	Password     string

	// Super duper locking mechanism
	sync.Mutex
}

// GetName returns name of database
func (r *RelationalMap) GetName() string {
	r.Lock()
	defer r.Unlock()
	return r.InstanceName
}

// IsEnabled returns if the database is enabled
func (r *RelationalMap) IsEnabled() bool {
	r.Lock()
	defer r.Unlock()
	return r.Enabled
}

// IsConnected returns if the database has established a connection
func (r *RelationalMap) IsConnected() bool {
	r.Lock()
	defer r.Unlock()
	return r.Connected
}

// SetupHelperFiles sets up helper files for SQLBoiler model generation
func (r *RelationalMap) SetupHelperFiles() error {
	// Checks to see if default directory is made
	err := common.CheckDir(r.PathDBDir, true)
	if err != nil {
		return err
	}

	var sqlBoilerFile RelativeDbPaths
	fullPathToTomlFile := r.PathDBDir + SQLBoilerToml

	// Creates a configuration file that points to a database for generating new
	// database models, located in the database folder
	file, err := common.ReadFile(fullPathToTomlFile)
	switch r.InstanceName {
	case SQLite:
		if err != nil {
			sqlBoilerFile.Sqlite.DBName = r.PathToDB

			e, err := toml.Marshal(sqlBoilerFile)
			if err != nil {
				return err
			}

			err = common.WriteFile(fullPathToTomlFile, e)
			if err != nil {
				return err
			}

			if r.Verbose {
				log.Debugf(CreatedLog, fullPathToTomlFile)
			}
		} else {
			if r.Verbose {
				log.Debugf(FoundLog, fullPathToTomlFile)
			}

			err = toml.Unmarshal(file, &sqlBoilerFile)
			if err != nil {
				return err
			}

			if sqlBoilerFile.Sqlite.DBName == "" {
				sqlBoilerFile.Sqlite.DBName = r.PathToDB

				e, err := toml.Marshal(sqlBoilerFile)
				if err != nil {
					return err
				}

				err = common.WriteFile(fullPathToTomlFile, e)
				if err != nil {
					return err
				}
			}
		}

	case Postgres:
		if err != nil {
			sqlBoilerFile.Postgress.DBName = r.DatabaseName
			sqlBoilerFile.Postgress.Host = r.Host
			sqlBoilerFile.Postgress.Pass = r.Password
			sqlBoilerFile.Postgress.User = r.User
			sqlBoilerFile.Postgress.SSLMode = "require"

			e, err := toml.Marshal(sqlBoilerFile)
			if err != nil {
				return err
			}

			err = common.WriteFile(fullPathToTomlFile, e)
			if err != nil {
				return err
			}

			if r.Verbose {
				log.Debugf(CreatedLog, fullPathToTomlFile)
			}
		} else {
			if r.Verbose {
				log.Debugf(FoundLog, fullPathToTomlFile)
			}

			err = toml.Unmarshal(file, &sqlBoilerFile)
			if err != nil {
				return err
			}

			if sqlBoilerFile.Postgress.DBName == r.DatabaseName ||
				sqlBoilerFile.Postgress.Host == r.Host ||
				sqlBoilerFile.Postgress.User == r.User ||
				sqlBoilerFile.Postgress.Pass == r.Password {
				return nil
			}

			sqlBoilerFile.Postgress.DBName = r.DatabaseName
			sqlBoilerFile.Postgress.Host = r.Host
			sqlBoilerFile.Postgress.User = r.User
			sqlBoilerFile.Postgress.Pass = r.Password

			e, err := toml.Marshal(sqlBoilerFile)
			if err != nil {
				return err
			}

			err = common.WriteFile(fullPathToTomlFile, e)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Disconnect closes the database connection
func (r *RelationalMap) Disconnect() error {
	r.Lock()
	defer r.Unlock()
	r.Connected = false
	return r.C.Close()
}
