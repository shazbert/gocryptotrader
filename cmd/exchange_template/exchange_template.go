package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	exchangeConfigPath = "../../testdata/configtest.json"
	targetPath         = "../../exchanges"
	toolName           = "GoCryptoTrader: Exchange templating tool"
)

var (
	errInvalidExchangeName   = errors.New("invalid exchange name")
	errExchangeNameIsEmpty   = errors.New("exchange name is empty")
	errExchangeAlreadyExists = errors.New("exchange already exists")
	errNoProtocolsSpecified  = errors.New("no supported protocols specified, atleast one is required")
	errBreakingChange        = errors.New("breaking change detected on current branch, please update templates")
)

func main() {
	settings := &TemplateSettings{}

	flag.StringVar(&settings.Name, "name", "", "the exchange name")
	flag.BoolVar(&settings.WS, "ws", false, "whether the exchange supports websocket")
	flag.BoolVar(&settings.REST, "rest", false, "whether the exchange supports REST")
	flag.BoolVar(&settings.FIX, "fix", false, "whether the exchange supports FIX")
	flag.StringVar(&settings.config, "config", config.DefaultFilePath(), "the config at which this will deploy a new exchange config")
	flag.BoolVar(&settings.deploy, "deploy", true, "whether the exchange supports FIX")

	flag.Parse()

	fmt.Println(toolName)
	fmt.Println(core.Copyright)
	fmt.Println()

	if len(os.Args) == 1 {
		fmt.Println("Invalid arguments supplied, please see application usage below:")
		flag.Usage()
		os.Exit(0)
	}

	// Verifies all supplied settings and cross references exchanges with
	// application configurations.
	err := settings.Verify()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	fmt.Println("Exchange Name:       ", settings.CapitalName)
	fmt.Println("Protocols Supported: ")
	fmt.Println("Websocket            ", settings.WS)
	fmt.Println("REST                 ", settings.REST)
	fmt.Println("FIX                  ", settings.FIX)
	fmt.Println("Output Directory     ", settings.directory)
	fmt.Println("Config               ", settings.config)
	fmt.Println("Deploy to Config     ", settings.deploy)
	fmt.Println()
	fmt.Println("Please check if everything is correct, type 'y' and then press enter to continue.")

	var choice []byte
	_, err = fmt.Scanln(&choice)
	if err != nil {
		fmt.Println("fmt.Scanln", err)
		os.Exit(1)
	}

	if !common.YesOrNo(string(choice)) {
		fmt.Println("templating tool stopped...")
		os.Exit(0)
	}

	// Creates a new directory for the new exchange
	err = settings.createDirectory()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// Sets the exchange files for from the templates
	err = settings.deployExchangeTemplates()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// Checks current files deployed if testing can occur and then formats them
	err = settings.qualifyExchange()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// If everything works print out JSON for old config and config test
	// directory

	// if deployment is set to true, then save a new config

	// newConfig, err = makeExchange(exchangeDirectory, configTestFile, &exch)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = saveConfig(exchangeDirectory, configTestFile, newConfig)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println("GoCryptoTrader: Exchange templating tool service complete")
	// fmt.Println("When the exchange code implementation has been completed (REST/Websocket/wrappers and tests), please add the exchange to engine/exchange.go")
	// fmt.Println("Add the exchange config settings to config_example.json (it will automatically be added to testdata/configtest.json)")
	// fmt.Println("Increment the available exchanges counter in config/config_test.go")
	// fmt.Println("Add the exchange name to exchanges/support.go")
	// fmt.Println("Ensure go test ./... -race passes")
	// fmt.Println("Open a pull request")
	// fmt.Println("If help is needed, please post a message in Slack.")
}

// TemplateSettings define exported variables used within the template
// deployment and sub variables used for verfication and upgrading of
// configurations.
type TemplateSettings struct {
	Name        string
	CapitalName string
	Variable    string
	REST        bool
	WS          bool
	FIX         bool

	directory string
	config    string
	deploy    bool
}

// Verify returns validates and sets running settings used in template
// deployment
func (t *TemplateSettings) Verify() error {
	if strings.Contains(t.Name, " ") || len(t.Name) < 3 {
		return fmt.Errorf("%w, should not have blank space or have a length less than 3",
			errInvalidExchangeName)
	}

	t.Name = strings.ToLower(t.Name)
	if !t.WS && !t.REST && !t.FIX {
		return fmt.Errorf("for exchange %s, %w",
			t.Name,
			errNoProtocolsSpecified)
	}

	var checkConfig config.Config
	err := checkConfig.LoadConfig(exchangeConfigPath, true, false)
	if err != nil {
		return err
	}

	_, err = checkConfig.GetExchangeConfig(t.Name)
	if err == nil {
		return fmt.Errorf("%w in test configuration", errExchangeAlreadyExists)
	}

	if t.deploy {
		err = checkConfig.LoadConfig(t.config, true, false)
		if err != nil {
			return err
		}

		_, err = checkConfig.GetExchangeConfig(t.Name)
		if err == nil {
			return fmt.Errorf("%w in main configuration", errExchangeAlreadyExists)
		}
	}

	t.directory, err = filepath.Abs(filepath.Join(targetPath, t.Name))
	if err != nil {
		return fmt.Errorf("for exchange %s, %w", t.Name, err)
	}

	_, err = os.Stat(t.directory)
	if !os.IsNotExist(err) {
		return fmt.Errorf("%w in directory %s",
			errExchangeAlreadyExists,
			t.directory)
	}

	t.CapitalName = strings.Title(t.Name)
	t.Variable = t.Name[0:2]

	return nil
}

// createDirectory creates a new directory for the exchange
func (t *TemplateSettings) createDirectory() error {
	_, err := os.Stat(t.directory)
	if !os.IsNotExist(err) {
		return fmt.Errorf("cannot create directory: %w", err)
	}
	err = os.MkdirAll(t.directory, 0770)
	if err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}
	return nil
}

// // saveConfig inserts a new exchange config into the test file
// func saveConfig(exchangeDirectory string, configTestFile *config.Config, newExchConfig *config.ExchangeConfig) error {

// 	configTestFile.Exchanges = append(configTestFile.Exchanges, *newExchConfig)
// 	err := configTestFile.SaveConfigToFile(exchangeConfigPath)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// Write implements the io writer interface
func (t *TemplateSettings) Write(p []byte) (n int, err error) {
	return fmt.Println("Go Test output:", string(p))
}

// qualifyExchange determines if the current implementation is testable with
// standard unit tests and that it is fully formatted with go fmt.
func (t *TemplateSettings) qualifyExchange() error {
	cmd := exec.Command("go", "test", "-json", "./...", "-v")
	cmd.Dir = t.directory
	cmd.Stdout = t
	err := cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		if strings.ContainsAny(err.Error(), "[build failed]") {
			return errBreakingChange
		}
		return err
	}
	// // out, err := cmd.Output()
	// if err != nil {
	// 	return err
	// 	// return fmt.Errorf(
	// 	// 	"unable to validate exchange at %s with 'go test'. output: %s err: %w",
	// 	// 	t.directory,
	// 	// 	out,
	// 	// 	err)
	// }

	cmd = exec.Command("go", "fmt")
	cmd.Dir = t.directory
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("unable to go fmt. output: %s err: %w", out, err)
	}
	return nil
}

// outputFile defines core file structure for an exchange implementation
type outputFile struct {
	Name         string
	Filename     string
	FilePostfix  string
	TemplateFile string
}

// deployExchangeTemplates deploys the exchange templates to new exchange
// directory
func (t *TemplateSettings) deployExchangeTemplates() error {
	outputFiles := []outputFile{
		{
			Name:         "readme",
			Filename:     "README.md",
			TemplateFile: "readme_file.tmpl",
		},
		{
			Name:         "main",
			Filename:     "main_file.tmpl",
			FilePostfix:  ".go",
			TemplateFile: "main_file.tmpl",
		},
		{
			Name:         "test",
			Filename:     "test_file.tmpl",
			FilePostfix:  "_test.go",
			TemplateFile: "test_file.tmpl",
		},
		{
			Name:         "type",
			Filename:     "type_file.tmpl",
			FilePostfix:  "_types.go",
			TemplateFile: "type_file.tmpl",
		},
		{
			Name:         "wrapper",
			Filename:     "wrapper_file.tmpl",
			FilePostfix:  "_wrapper.go",
			TemplateFile: "wrapper_file.tmpl",
		},
	}

	for x := range outputFiles {
		tmpl, err := template.New(outputFiles[x].Name).ParseFiles(outputFiles[x].TemplateFile)
		if err != nil {
			return fmt.Errorf("%s template error: %w", outputFiles[x].Name, err)
		}

		filename := outputFiles[x].Filename
		if outputFiles[x].FilePostfix != "" {
			filename = t.Name + outputFiles[x].FilePostfix
		}

		outputFile := filepath.Join(t.directory, filename)
		file, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE, 0770)
		if err != nil {
			return fmt.Errorf("%s template error: %w", outputFiles[x].Name, err)
		}
		defer file.Close()
		if err = tmpl.Execute(file, t); err != nil {
			return fmt.Errorf("%s template error: %w", outputFiles[x].Name, err)
		}
	}
	return nil
}

// newExchangeConfig generates a new  exchange config
func newExchangeConfig(name string) (*config.Exchange, error) {
	if name == "" {
		return nil, errExchangeNameIsEmpty
	}
	return &config.Exchange{
		Name:    name,
		Enabled: true,
		API: config.APIConfig{
			Credentials: config.APICredentialsConfig{
				Key:    "Key",
				Secret: "Secret",
			},
		},
		CurrencyPairs: &currency.PairsManager{
			UseGlobalFormat: true,
			RequestFormat: &currency.PairFormat{
				Uppercase: true,
			},
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
			},
		},
	}, nil
}

// ExchangeConfigTest defines the old config for manual insertion
type ExchangeConfigTest struct {
	API struct {
		AuthenticatedSupport             bool `json:"authenticatedSupport"`
		AuthenticatedWebsocketAPISupport bool `json:"authenticatedWebsocketApiSupport"`
		Credentials                      struct {
			Key    string `json:"key"`
			Secret string `json:"secret"`
		} `json:"credentials"`
		CredentialsValidator struct {
			RequiresKey    bool `json:"requiresKey"`
			RequiresSecret bool `json:"requiresSecret"`
		} `json:"credentialsValidator"`
		Endpoints struct {
			URL          string `json:"url"`
			URLSecondary string `json:"urlSecondary"`
			WebsocketURL string `json:"websocketURL"`
		} `json:"endpoints"`
	} `json:"api"`
	BankAccounts []struct {
		AccountName         string `json:"accountName"`
		AccountNumber       string `json:"accountNumber"`
		BankAddress         string `json:"bankAddress"`
		BankCountry         string `json:"bankCountry"`
		BankName            string `json:"bankName"`
		BankPostalCity      string `json:"bankPostalCity"`
		BankPostalCode      string `json:"bankPostalCode"`
		Enabled             bool   `json:"enabled"`
		Iban                string `json:"iban"`
		SupportedCurrencies string `json:"supportedCurrencies"`
		SwiftCode           string `json:"swiftCode"`
	} `json:"bankAccounts"`
	BaseCurrencies string `json:"baseCurrencies"`
	CurrencyPairs  struct {
		AssetTypes   []string `json:"assetTypes"`
		ConfigFormat struct {
			Delimiter string `json:"delimiter"`
			Uppercase bool   `json:"uppercase"`
		} `json:"configFormat"`
		Pairs struct {
			Spot struct {
				Available string `json:"available"`
				Enabled   string `json:"enabled"`
			} `json:"spot"`
		} `json:"pairs"`
		RequestFormat struct {
			Uppercase bool `json:"uppercase"`
		} `json:"requestFormat"`
		UseGlobalFormat bool `json:"useGlobalFormat"`
	} `json:"currencyPairs"`
	Enabled  bool `json:"enabled"`
	Features struct {
		Enabled struct {
			AutoPairUpdates bool `json:"autoPairUpdates"`
			WebsocketAPI    bool `json:"websocketAPI"`
		} `json:"enabled"`
		Supports struct {
			RestAPI          bool `json:"restAPI"`
			RestCapabilities struct {
				AutoPairUpdates bool `json:"autoPairUpdates"`
				TickerBatching  bool `json:"tickerBatching"`
			} `json:"restCapabilities"`
			WebsocketAPI          bool `json:"websocketAPI"`
			WebsocketCapabilities struct {
			} `json:"websocketCapabilities"`
		} `json:"supports"`
	} `json:"features"`
	HTTPTimeout                   int64  `json:"httpTimeout"`
	Name                          string `json:"name"`
	Verbose                       bool   `json:"verbose"`
	WebsocketOrderbookBufferLimit int    `json:"websocketOrderbookBufferLimit"`
	WebsocketResponseCheckTimeout int    `json:"websocketResponseCheckTimeout"`
	WebsocketResponseMaxLimit     int64  `json:"websocketResponseMaxLimit"`
	WebsocketTrafficTimeout       int64  `json:"websocketTrafficTimeout"`
}
