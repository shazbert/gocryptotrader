package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/communications"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/coinmarketcap"
	"github.com/thrasher-/gocryptotrader/database"
	"github.com/thrasher-/gocryptotrader/database/base"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

// Bot contains configuration, portfolio, exchange & ticker data and is the
// overarching type across this code base.
type Bot struct {
	config     *config.Config
	portfolio  *portfolio.Base
	exchanges  []exchange.IBotExchange
	comms      *communications.Communications
	db         database.Databaser
	shutdown   chan bool
	dryRun     bool
	configFile string
	dataDir    string
}

const (
	banner = `
   ______        ______                     __        ______                  __
  / ____/____   / ____/_____ __  __ ____   / /_ ____ /_  __/_____ ______ ____/ /___   _____
 / / __ / __ \ / /    / ___// / / // __ \ / __// __ \ / /  / ___// __  // __  // _ \ / ___/
/ /_/ // /_/ // /___ / /   / /_/ // /_/ // /_ / /_/ // /  / /   / /_/ // /_/ //  __// /
\____/ \____/ \____//_/    \__, // .___/ \__/ \____//_/  /_/    \__,_/ \__,_/ \___//_/
                          /____//_/
`
)

var bot Bot

func main() {
	bot.shutdown = make(chan bool)
	HandleInterrupt()

	defaultPath, err := config.GetFilePath("")
	if err != nil {
		log.Fatal(err)
	}

	// Handle flags
	flag.StringVar(&bot.configFile, "config", defaultPath, "config file to load")
	flag.StringVar(&bot.dataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")
	dryrun := flag.Bool("dryrun", false, "dry runs bot, doesn't save config file")
	version := flag.Bool("version", false, "retrieves current GoCryptoTrader version")
	verbosity := flag.Bool("verbose", false, "increases logging verbosity for GoCryptoTrader")

	// Database flags
	dbDir := flag.String("dbdirectory", common.GetDefaultDatabaseDir(), "Sets a non default path to a database data directory")
	sqlite := flag.Bool("sqlite", false, "initiates and connects to a sqlite3 database")
	sqlitePath := flag.String("sqlitepath", common.GetDefaultSQLitePath(), "Sets a non default path to a SQLite3 database")
	postgres := flag.Bool("postgres", false, "initiates a postgres connection")
	dbHost := flag.String("dbhost", "", "Sets database host")
	dbUser := flag.String("dbuser", "", "Sets database user")
	dbpass := flag.String("dbpass", "", "Sets database password")
	dbName := flag.String("dbname", "", "Sets database name")
	dbPort := flag.String("dbport", "", "Sets database port, does not need to be set")
	dbSSLMode := flag.String("dbsslmode", "", "Sets database SSL mode")
	newclient := flag.Bool("newclient", false, "Creates a new client to load in database")

	Coinmarketcap := flag.Bool("c", false, "overrides config and runs currency analaysis")
	FxCurrencyConverter := flag.Bool("fxa", false, "overrides config and sets up foreign exchange Currency Converter")
	FxCurrencyLayer := flag.Bool("fxb", false, "overrides config and sets up foreign exchange Currency Layer")
	FxFixer := flag.Bool("fxc", false, "overrides config and sets up foreign exchange Fixer.io")
	FxOpenExchangeRates := flag.Bool("fxd", false, "overrides config and sets up foreign exchange Open Exchange Rates")

	flag.Parse()

	if *version {
		fmt.Print(BuildVersion(true))
		os.Exit(0)
	}

	if *dryrun {
		bot.dryRun = true
	}

	fmt.Println(banner)
	fmt.Println(BuildVersion(false))

	bot.config = &config.Cfg
	log.Debugf("Loading config file %s..\n", bot.configFile)
	err = bot.config.LoadConfig(bot.configFile)
	if err != nil {
		log.Fatalf("Failed to open/create data directory: %s. Err: %s",
			bot.dataDir,
			err)
	}

	var dbOn bool
	if *postgres || *sqlite || bot.config.Databases.Postgres.Enabled || bot.config.Databases.Sqlite3.Enabled {
		if (*postgres || bot.config.Databases.Postgres.Enabled) &&
			(*sqlite || bot.config.Databases.Sqlite3.Enabled) {
			log.Fatal("Can only run one database at a time, please check config and flags")
		}

		log.Debugf("Using data directory: %s.\n", bot.dataDir)
		log.Debugf("Setting up database directory with supplementary files at %s",
			*dbDir)

		if *postgres {
			bot.db = database.GetPostgresInstance()
			if *dbHost == "" || *dbUser == "" || *dbpass == "" || *dbName == "" || *dbSSLMode == "" {
				log.Warn("Database PostgreSQL command line flags incorrectly set, defaulting to config.json")
				*dbHost = bot.config.Databases.Postgres.Host
				*dbUser = bot.config.Databases.Postgres.Username
				*dbpass = bot.config.Databases.Postgres.Password
				*dbName = bot.config.Databases.Postgres.DatabaseName
				*dbPort = bot.config.Databases.Postgres.Port
				*dbSSLMode = bot.config.Databases.Postgres.SSLMode
			}

			err = bot.db.Setup(base.ConnDetails{Verbose: *verbosity,
				DirectoryPath: *dbDir,
				Host:          *dbHost,
				User:          *dbUser,
				Pass:          *dbpass,
				DBName:        *dbName,
				Port:          *dbPort,
				SSLMode:       *dbSSLMode,
				MemCacheSize:  bot.config.Databases.MemoryAllocationInBytes,
			})
			if err != nil {
				log.Fatal("Postgres instance failed ", err)
			}

		}

		if *sqlite {
			bot.db = database.GetSQLite3Instance()
			err = bot.db.Setup(base.ConnDetails{
				DirectoryPath: *dbDir,
				SQLPath:       *sqlitePath,
				Verbose:       *verbosity,
				MemCacheSize:  bot.config.Databases.MemoryAllocationInBytes,
			})
			if err != nil {
				log.Fatal("default database 'SQLite3' failed to setup reason:", err)
			}
		}

		log.Debug("Initial setup complete, establishing connection to database")
		err = bot.db.Connect()
		if err != nil {
			disconnectErr := bot.db.Disconnect()
			if disconnectErr != nil {
				log.Error(disconnectErr)
			}
			log.Error("Database connection failure reason:", err)
		} else {
			err = bot.db.ClientLogin(*newclient)
			if err != nil {
				log.Fatal("User failed to log into database reason:", err)
			}
		}

		if bot.db.IsConnected() {
			log.Debugf("Bot is now connected to a %s database", bot.db.GetName())
			var client string
			client, err = bot.db.GetClientDetails()
			if err != nil {
				log.Error("Retrieving client data from database failure reason:",
					err)
			} else {
				log.Debugf("Database credentials set for client %s", client)
			}

			dbOn = true
		}
	}

	err = bot.config.CheckLoggerConfig()
	if err != nil {
		log.Errorf("Failed to configure logger reason: %s", err)
	}

	err = log.SetupLogger()
	if err != nil {
		log.Errorf("Failed to setup logger reason: %s", err)
	}

	AdjustGoMaxProcs()
	log.Debugf("Bot '%s' started.\n", bot.config.Name)
	log.Debugf("Bot dry run mode: %v.\n", common.IsEnabled(bot.dryRun))

	log.Debugf("Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(bot.config.Exchanges),
		bot.config.CountEnabledExchanges())

	common.HTTPClient = common.NewHTTPClientWithTimeout(bot.config.GlobalHTTPTimeout)
	log.Debugf("Global HTTP request timeout: %v.\n", common.HTTPClient.Timeout)

	SetupExchanges()
	if len(bot.exchanges) == 0 {
		log.Fatalf("No exchanges were able to be loaded. Exiting")
	}

	log.Debugf("Starting communication mediums..")
	cfg := bot.config.GetCommunicationsConfig()
	bot.comms = communications.NewComm(&cfg)
	bot.comms.GetEnabledCommunicationMediums()

	var newFxSettings []currency.FXSettings
	for _, d := range bot.config.Currency.ForexProviders {
		newFxSettings = append(newFxSettings, currency.FXSettings(d))
	}

	err = currency.RunStorageUpdater(currency.BotOverrides{
		Coinmarketcap:       *Coinmarketcap,
		FxCurrencyConverter: *FxCurrencyConverter,
		FxCurrencyLayer:     *FxCurrencyLayer,
		FxFixer:             *FxFixer,
		FxOpenExchangeRates: *FxOpenExchangeRates,
	},
		currency.MainConfiguration{
			ForexProviders:         newFxSettings,
			CryptocurrencyProvider: coinmarketcap.Settings(bot.config.Currency.CryptocurrencyProvider),
			Cryptocurrencies:       bot.config.Currency.Cryptocurrencies,
			FiatDisplayCurrency:    bot.config.Currency.FiatDisplayCurrency,
			CurrencyDelay:          bot.config.Currency.CurrencyFileUpdateDuration,
			FxRateDelay:            bot.config.Currency.ForeignExchangeUpdateDuration,
		},
		bot.dataDir,
		*verbosity)
	if err != nil {
		log.Warn("currency updater system failed to start", err)
	}

	bot.portfolio = &portfolio.Portfolio
	bot.portfolio.SeedPortfolio(bot.config.Portfolio)
	SeedExchangeAccountInfo(GetAllEnabledExchangeAccountInfo().Data)

	if bot.config.Webserver.Enabled {
		listenAddr := bot.config.Webserver.ListenAddress
		log.Debugf(
			"HTTP Webserver support enabled. Listen URL: http://%s:%d/\n",
			common.ExtractHost(listenAddr), common.ExtractPort(listenAddr),
		)

		router := NewRouter()
		go func() {
			err = http.ListenAndServe(listenAddr, router)
			if err != nil {
				log.Fatal(err)
			}
		}()

		log.Debugln("HTTP Webserver started successfully.")
		log.Debugln("Starting websocket handler.")
		StartWebsocketHandler()
	} else {
		log.Debugln("HTTP RESTful Webserver support disabled.")
	}

	go portfolio.StartPortfolioWatcher()

	go TickerUpdaterRoutine()
	go OrderbookUpdaterRoutine()
	go WebsocketRoutine(*verbosity)

	if dbOn {
		go PlatformTradeUpdaterRoutine()
	}

	<-bot.shutdown
	Shutdown()
}

// AdjustGoMaxProcs adjusts the maximum processes that the CPU can handle.
func AdjustGoMaxProcs() {
	log.Debugln("Adjusting bot runtime performance..")
	maxProcsEnv := os.Getenv("GOMAXPROCS")
	maxProcs := runtime.NumCPU()
	log.Debugln("Number of CPU's detected:", maxProcs)

	if maxProcsEnv != "" {
		log.Debugln("GOMAXPROCS env =", maxProcsEnv)
		env, err := strconv.Atoi(maxProcsEnv)
		if err != nil {
			log.Debugf("Unable to convert GOMAXPROCS to int, using %d", maxProcs)
		} else {
			maxProcs = env
		}
	}
	if i := runtime.GOMAXPROCS(maxProcs); i != maxProcs {
		log.Error("Go Max Procs were not set correctly.")
	}
	log.Debugln("Set GOMAXPROCS to:", maxProcs)
}

// HandleInterrupt monitors and captures the SIGTERM in a new goroutine then
// shuts down bot
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Debugf("Captured %v, shutdown requested.", sig)
		bot.shutdown <- true
	}()
}

// Shutdown correctly shuts down bot saving configuration files
func Shutdown() {
	log.Debugln("Bot shutting down..")

	if len(portfolio.Portfolio.Addresses) != 0 {
		bot.config.Portfolio = portfolio.Portfolio
	}

	err := bot.db.Disconnect()
	if err != nil {
		log.Debug("Unable to disconnect from database.", err)
	} else {
		log.Debug("Succesfully shutdown database.")
	}

	if !bot.dryRun {
		err := bot.config.SaveConfig(bot.configFile)

		if err != nil {
			log.Warn("Unable to save config.")
		} else {
			log.Debugln("Config file saved successfully.")
		}
	}

	log.Debugln("Exiting.")

	log.CloseLogFile()
	os.Exit(0)
}
