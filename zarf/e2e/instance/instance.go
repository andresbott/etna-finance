package instance

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// EnvCfg mirrors the basic etna config that e2e can override via ETNA_* env vars.
type EnvCfg struct {
	Settings            EnvSettings
	MarketDataImporters EnvMarketDataImporters
}

type EnvSettings struct {
	DateFormat            string   // e.g. "YYYY-MM-DD"
	MainCurrency          string   // e.g. "CHF"
	AdditionalCurrencies  []string // e.g. ["EUR", "USD"]
	InvestmentInstruments bool
}

type EnvMarketDataImporters struct {
	Massive EnvMassiveConfig
}

type EnvMassiveConfig struct {
	ApiKeys []string
}

// DefaultEnvCfg returns a minimal config (no extra currencies, no API keys).
func DefaultEnvCfg() EnvCfg {
	return EnvCfg{
		Settings: EnvSettings{
			DateFormat:            "YYYY-MM-DD",
			MainCurrency:          "CHF",
			AdditionalCurrencies:  nil,
			InvestmentInstruments: false,
		},
		MarketDataImporters: EnvMarketDataImporters{
			Massive: EnvMassiveConfig{ApiKeys: nil},
		},
	}
}

// ApplyEnv unsets existing ETNA_* vars for our config, then sets them from cfg.
func ApplyEnv(cfg EnvCfg) {
	unset := []string{
		"ETNA_SETTINGS_DATEFORMAT",
		"ETNA_SETTINGS_MAINCURRENCY",
		"ETNA_SETTINGS_INVESTMENTINSTRUMENTS",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_0",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_1",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_2",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_3",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_4",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_5",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_6",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_7",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_8",
		"ETNA_SETTINGS_ADDITIONALCURRENCIES_9",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_0",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_1",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_2",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_3",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_4",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_5",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_6",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_7",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_8",
		"ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_9",
	}
	for _, k := range unset {
		_ = os.Unsetenv(k)
	}

	if cfg.Settings.DateFormat != "" {
		_ = os.Setenv("ETNA_SETTINGS_DATEFORMAT", cfg.Settings.DateFormat)
	}
	if cfg.Settings.MainCurrency != "" {
		_ = os.Setenv("ETNA_SETTINGS_MAINCURRENCY", cfg.Settings.MainCurrency)
	}
	_ = os.Setenv("ETNA_SETTINGS_INVESTMENTINSTRUMENTS", strconv.FormatBool(cfg.Settings.InvestmentInstruments))
	for i, c := range cfg.Settings.AdditionalCurrencies {
		_ = os.Setenv(fmt.Sprintf("ETNA_SETTINGS_ADDITIONALCURRENCIES_%d", i), c)
	}
	for i, k := range cfg.MarketDataImporters.Massive.ApiKeys {
		_ = os.Setenv(fmt.Sprintf("ETNA_MARKETDATAIMPORTERS_MASSIVE_APIKEYS_%d", i), k)
	}
}

// Instance holds a running etna-finance app and its URLs.
type Instance struct {
	BaseURL    string // root for browser navigation; with built UI: BackendURL
	BackendURL string // for API calls
	DataDir    string
	stopOnce   sync.Once
	stop       func()
}

// Stop kills the backend subprocess and removes the temp data dir.
func (i *Instance) Stop() {
	i.stopOnce.Do(i.stop)
}

// InitInstance creates a temp DataDir, starts the backend with env overrides, and polls until ready.
func InitInstance(cfg *EnvCfg) (*Instance, error) {
	if cfg != nil {
		ApplyEnv(*cfg)
	}
	dataDir, err := os.MkdirTemp("", "etna-e2e-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	fmt.Println(dataDir)

	// Allocate a free port for the main server
	listener, err := net.Listen("tcp", ":0") //nolint:gosec // binding to all interfaces is intentional for test server port allocation
	if err != nil {
		_ = os.RemoveAll(dataDir)
		return nil, fmt.Errorf("allocate port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	// Allocate a free port for the observability server
	obsListener, err := net.Listen("tcp", ":0") //nolint:gosec // binding to all interfaces is intentional for test server port allocation
	if err != nil {
		_ = os.RemoveAll(dataDir)
		return nil, fmt.Errorf("allocate obs port: %w", err)
	}
	obsPort := obsListener.Addr().(*net.TCPAddr).Port
	_ = obsListener.Close()

	backendURL := "http://127.0.0.1:" + strconv.Itoa(port)
	baseURL := backendURL

	// Project root: go test sets cwd to package dir (zarf/e2e), so walk up to find main.go
	cwd, err := os.Getwd()
	if err != nil {
		_ = os.RemoveAll(dataDir)
		return nil, fmt.Errorf("getwd: %w", err)
	}
	projectRoot := ""
	for _, root := range []string{cwd, filepath.Clean(filepath.Join(cwd, "..", ".."))} {
		if _, err := os.Stat(filepath.Join(root, "main.go")); err == nil {
			projectRoot = root
			break
		}
	}
	if projectRoot == "" {
		_ = os.RemoveAll(dataDir)
		return nil, fmt.Errorf("main.go not found (cwd=%s)", cwd)
	}

	cmd := exec.Command("go", "run", "main.go", "start", "-c", dataDir+"/config.yaml") //nolint:gosec // test helper, arguments are controlled
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(),
		"ETNA_DATADIR="+dataDir,
		"ETNA_SERVER_PORT="+strconv.Itoa(port),
		"ETNA_SERVER_BINDIP=127.0.0.1",
		"ETNA_OBSERVABILITY_PORT="+strconv.Itoa(obsPort),
		"ETNA_OBSERVABILITY_BINDIP=127.0.0.1",
		"ETNA_AUTH_ENABLED=false",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Place "go run" and its child process in a new process group so we can
	// kill the entire tree on cleanup (go run spawns a child binary).
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(dataDir)
		return nil, fmt.Errorf("start backend: %w", err)
	}

	stopFn := func() {
		// Kill the entire process group (negative PID) so the child binary
		// spawned by "go run" is also terminated.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Wait()
		_ = os.RemoveAll(dataDir)
	}

	inst := &Instance{
		BaseURL:    baseURL,
		BackendURL: backendURL,
		DataDir:    dataDir,
		stop:       stopFn,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := waitHealthy(ctx, backendURL); err != nil {
		inst.Stop()
		_ = os.RemoveAll(dataDir)
		return nil, fmt.Errorf("health check: %w", err)
	}
	return inst, nil
}

func waitHealthy(ctx context.Context, baseURL string) error {
	healthURL := baseURL + "/api/v0/settings"
	client := &http.Client{Timeout: 2 * time.Second}
	interval := 50 * time.Millisecond
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return nil
		}
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(interval)
		if interval < 500*time.Millisecond {
			interval *= 2
		}
	}
}
