package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const defaultConfigYAML = `# =============================================================================
# etna-finance configuration file
# =============================================================================
# Generated with: etna generate-config
#
# Configuration is loaded in this order (last wins):
#   1. Built-in defaults
#   2. .env file (optional)
#   3. This YAML config file (optional, default: ./config.yaml)
#   4. Environment variables (prefix: ETNA_)
#
# Environment variable format: ETNA_<SECTION>_<KEY>, e.g.:
#   ETNA_SERVER_PORT=8085
#   ETNA_SETTINGS_MAINCURRENCY=USD
#   ETNA_SETTINGS_ADDITIONALCURRENCIES_0=EUR
# =============================================================================

# -----------------------------------------------------------------------------
# Server — main application HTTP server
# -----------------------------------------------------------------------------
Server:
  # IP address to bind to. Empty string means listen on all interfaces.
  BindIp: ""
  # Port to listen on.
  Port: 8085

# -----------------------------------------------------------------------------
# Observability — metrics / health-check HTTP server
# -----------------------------------------------------------------------------
Observability:
  # IP address to bind to. Empty string means listen on all interfaces.
  BindIp: ""
  # Port to listen on.
  Port: 9090

# -----------------------------------------------------------------------------
# DataDir — directory for database, sessions, backups, and task logs
# -----------------------------------------------------------------------------
# Relative paths are resolved from the working directory.
DataDir: "./data"

# -----------------------------------------------------------------------------
# Env — runtime environment settings
# -----------------------------------------------------------------------------
Env:
  # Log level: "debug", "info", "warn", or "error".
  LogLevel: "info"
  # Production mode. When true, debug tasks are hidden and stricter checks apply.
  Production: false

# -----------------------------------------------------------------------------
# Settings — application settings exposed to the frontend
# -----------------------------------------------------------------------------
Settings:
  # Date display format.
  # Tokens: YYYY (4-digit year), YY (2-digit year), MM (month), DD (day).
  # Separators: - / .
  # Examples: "YYYY-MM-DD", "DD/MM/YYYY", "MM.DD.YY"
  DateFormat: "YYYY-MM-DD"

  # Main currency (ISO 4217 uppercase code, e.g. CHF, USD, EUR).
  # This currency is always included; do not repeat it in AdditionalCurrencies.
  MainCurrency: "CHF"

  # Extra currencies to track. Each must be a valid ISO 4217 uppercase code.
  # The main currency is implicit and must not appear here.
  # Example: ["USD", "EUR"]
  AdditionalCurrencies: []

  # Enable investment / instruments tracking (portfolio, stocks, unvested assets).
  # Automatically enabled if the database already contains investment accounts.
  Instruments: false

# -----------------------------------------------------------------------------
# Auth — authentication and session management
# -----------------------------------------------------------------------------
Auth:
  # Set to true to require login. When false, all operations use DefaultUser.
  Enabled: false

  # Username used for all operations when auth is disabled.
  DefaultUser: "default"

  # --- The fields below only apply when Enabled: true ---

  # Session encryption keys. Required when auth is enabled.
  # HashKey must be exactly 64 characters; BlockKey must be exactly 32 characters.
  # If left empty when auth is enabled, random keys are generated (sessions won't
  # survive restarts).
  # HashKey:  "your-64-character-secret-key-here............................"
  # BlockKey: "your-32-character-secret-key...."

  # User store configuration.
  UserStore:
    # Type: "static" (users defined below) or "file" (users loaded from a file).
    Type: "static"

    # Path to the users file (only used when Type: "file").
    # Path: "/path/to/users.yaml"

    # Static users (only used when Type: "static").
    Users:
      - Name: "demo"
        Pw:   "demo"
      - Name: "admin"
        Pw:   "admin"

# -----------------------------------------------------------------------------
# MarketDataImporters — external market data sources
# -----------------------------------------------------------------------------
# Configure API keys for market data providers.
# Keys can also be set via environment variables:
#   ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_0=your_api_key
MarketDataImporters:
  Massive:
    ApiKeys: []
`

func generateConfigCmd() *cobra.Command {
	var outputFile = "./config.yaml"

	cmd := &cobra.Command{
		Use:   "config",
		Short: "generate a default configuration file",
		Long:  "generate a YAML configuration file with all default values and comments explaining each option",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(outputFile); err == nil {
				return fmt.Errorf("file %s already exists, not overwriting", outputFile)
			}
			if err := os.WriteFile(outputFile, []byte(defaultConfigYAML), 0644); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}
			fmt.Printf("Configuration written to %s\n", outputFile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", outputFile, "output file path")
	return cmd
}
