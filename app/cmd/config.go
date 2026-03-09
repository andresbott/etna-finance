package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-bumbu/config"
	"github.com/gorilla/securecookie"
	"golang.org/x/text/currency"
)

type AppCfg struct {
	Server              serverCfg
	Obs                 serverCfg `config:"Observability"`
	Auth                authConfig
	Env                 Env
	Settings            AppSettings
	MarketDataImporters MarketDataImportersCfg
	Msgs                []Msg
	DataDir             string
}

// MarketDataImportersCfg holds named importer configs. Supported importers: Massive.
type MarketDataImportersCfg struct {
	Massive MarketDataImporterConfig
}

// MarketDataImporterConfig holds per-importer settings (e.g. API keys).
type MarketDataImporterConfig struct {
	ApiKeys []string
}

type AppSettings struct {
	DateFormat           string
	MainCurrency         string
	AdditionalCurrencies []string // currencies to track; main currency is implicit, do not repeat
	Instruments          bool
	MaxAttachmentSizeMB  float64 // max upload size in MB; 0 = default (10 MB)
}

// AllCurrencies returns MainCurrency plus AdditionalCurrencies (main is implicit, not repeated in config).
func (s AppSettings) AllCurrencies() []string {
	out := []string{s.MainCurrency}
	for _, c := range s.AdditionalCurrencies {
		if c != s.MainCurrency {
			out = append(out, c)
		}
	}
	return out
}

type Env struct {
	LogLevel   string
	Production bool
}

type serverCfg struct {
	BindIp string
	Port   int
}

func (c serverCfg) Addr() string {
	if c.BindIp == "" {
		return ":" + strconv.Itoa(c.Port)
	}
	return c.BindIp + ":" + strconv.Itoa(c.Port)
}

type authConfig struct {
	Enabled       bool   // when false, no login required; all operations use DefaultUser
	DefaultUser   string `config:"DefaultUser"` // used when Enabled=false; default "default"
	HashKeyBytes  []byte
	BlockKeyBytes []byte
	HashKey       string
	BlockKey      string
	UserStore     userStore
}

type userStore struct {
	StoreType string `config:"Type"` // can be static | file
	FilePath  string `config:"Path"`
	Users     []User
}
type User struct {
	Name string
	Pw   string
}

// Default represents the basic set of sensible defaults
var defaultCfg = AppCfg{
	DataDir: "./data",
	Server: serverCfg{
		BindIp: "",
		Port:   8085,
	},
	Obs: serverCfg{
		BindIp: "",
		Port:   9090,
	},
	Auth: authConfig{
		Enabled:     false, // no login required; all operations use DefaultUser
		DefaultUser: "default",
		HashKey:     "",
		BlockKey:    "",
		UserStore: userStore{
			StoreType: "static",
			Users: []User{
				{
					Name: "demo",
					Pw:   "demo",
				},
				{
					Name: "admin",
					Pw:   "admin",
				},
			},
		},
	},
	Env: Env{
		LogLevel:   "info",
		Production: false,
	},
	Settings: AppSettings{
		DateFormat:           "YYYY-MM-DD",
		MainCurrency:         "CHF",
		AdditionalCurrencies: nil, // empty = main currency only
		Instruments:          false,
	},
	MarketDataImporters: MarketDataImportersCfg{}, // set Massive with ApiKeys in YAML to enable backfill
}

type Msg struct {
	Level string
	Msg   string
}

const EnvBarPrefix = "ETNA"

// dateFormatRegex matches date formats composed of tokens YYYY, YY, MM, DD
// separated by one of: - / .
var dateFormatRegex = regexp.MustCompile(`^(YYYY|YY|MM|DD)([-/.](YYYY|YY|MM|DD)){2}$`)

// validateCurrency checks that the given string is a recognized ISO 4217 currency code
// in its canonical uppercase form.
func validateCurrency(code string) error {
	unit, err := currency.ParseISO(code)
	if err != nil {
		return fmt.Errorf("invalid currency %q: not a recognized ISO 4217 code (e.g. CHF, USD, EUR)", code)
	}
	if unit.String() != code {
		return fmt.Errorf("invalid currency %q: must be uppercase %q", code, unit.String())
	}
	return nil
}

func validateSettings(s AppSettings) error {
	// Validate date format
	if !dateFormatRegex.MatchString(s.DateFormat) {
		return fmt.Errorf("invalid date format %q: must use tokens YYYY, YY, MM, DD separated by - / or . (e.g. YYYY-MM-DD, DD/MM/YYYY)", s.DateFormat)
	}

	// Ensure all three components (year, month, day) are present
	hasYear := strings.Contains(s.DateFormat, "YYYY") || strings.Contains(s.DateFormat, "YY")
	hasMonth := strings.Contains(s.DateFormat, "MM")
	hasDay := strings.Contains(s.DateFormat, "DD")
	if !hasYear || !hasMonth || !hasDay {
		return fmt.Errorf("invalid date format %q: must contain year (YYYY or YY), month (MM) and day (DD)", s.DateFormat)
	}

	// Validate main currency
	if err := validateCurrency(s.MainCurrency); err != nil {
		return fmt.Errorf("main currency: %w", err)
	}

	// Validate additional currencies (main is implicit; do not repeat)
	for _, c := range s.AdditionalCurrencies {
		if err := validateCurrency(c); err != nil {
			return fmt.Errorf("additional currencies: %w", err)
		}
		if c == s.MainCurrency {
			return fmt.Errorf("additional currencies must not include main currency %q (it is implicit)", s.MainCurrency)
		}
	}

	return nil
}

// applyAuthDisabledDefaults sets default values when auth is disabled.
func applyAuthDisabledDefaults(cfg *AppCfg) {
	if cfg.Auth.DefaultUser == "" {
		cfg.Auth.DefaultUser = "default"
	}
}

// validateAuthEnabledConfig validates and populates auth key bytes when auth is enabled.
func validateAuthEnabledConfig(cfg *AppCfg) error {
	if cfg.Auth.HashKey == "" || cfg.Auth.BlockKey == "" {
		cfg.Auth.HashKeyBytes = securecookie.GenerateRandomKey(64)
		cfg.Auth.BlockKeyBytes = securecookie.GenerateRandomKey(32)
	}
	if len(cfg.Auth.HashKey) != 64 {
		return fmt.Errorf("hashkey must be 64 chars long")
	}
	if len(cfg.Auth.BlockKey) != 32 {
		return fmt.Errorf("blockKey must be 32 chars long")
	}
	cfg.Auth.HashKeyBytes = []byte(cfg.Auth.HashKey)
	cfg.Auth.BlockKeyBytes = []byte(cfg.Auth.BlockKey)
	return nil
}

func getAppCfg(file string) (AppCfg, error) {
	configMsg := []Msg{}
	cfg := AppCfg{}
	var err error
	_, err = config.Load(
		config.Defaults{Item: defaultCfg},
		config.EnvFile{Path: ".env", Mandatory: false},
		config.CfgFile{Path: file, Mandatory: false},
		config.EnvVar{Prefix: EnvBarPrefix},
		config.Unmarshal{Item: &cfg},
		config.Writer{Fn: func(level, msg string) {
			if level == config.InfoLevel {
				configMsg = append(configMsg, Msg{Level: "info", Msg: msg})
			}
			if level == config.DebugLevel {
				configMsg = append(configMsg, Msg{Level: "debug", Msg: msg})
			}
		}},
	)
	cfg.Msgs = configMsg
	if err != nil {
		return cfg, err
	}

	absPath, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		return cfg, fmt.Errorf("failed to get absolute path: %w", err)
	}
	cfg.DataDir = absPath

	// handle auth config
	if !cfg.Auth.Enabled {
		applyAuthDisabledDefaults(&cfg)
	} else {
		if err := validateAuthEnabledConfig(&cfg); err != nil {
			return cfg, err
		}
	}

	// Validate application settings
	if err := validateSettings(cfg.Settings); err != nil {
		return cfg, fmt.Errorf("settings validation: %w", err)
	}

	return cfg, nil
}
