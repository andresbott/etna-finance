package e2e

import (
	"fmt"
	"os"
	"testing"

	"github.com/andresbott/etna/zarf/e2e/browser"
	"github.com/andresbott/etna/zarf/e2e/instance"
)

// SetupE2E starts a fresh instance and browser for each test. Call at the start of each test.
// If cfg is non-nil, ApplyEnv(cfg) is called before starting the instance; nil uses DefaultEnvCfg().
// Registers t.Cleanup to stop the instance and close the browser.
func SetupE2E(t *testing.T, cfg *instance.EnvCfg) (*instance.Instance, *browser.Browser) {
	t.Helper()
	if os.Getenv("E2E") == "" {
		t.Skip("e2e tests skipped (E2E not set)")
	}

	in, err := instance.InitInstance(cfg)
	if err != nil {
		t.Fatalf("InitInstance: %v", err)
	}
	headless := os.Getenv("HEADLESS") != "false" && os.Getenv("HEADLESS") != "0"
	br, err := browser.New(browser.Cfg{
		Headless: headless,
		WindowW:  1400,
		WindowH:  1200,
	})
	if err != nil {
		in.Stop()
		t.Fatalf("browser.New: %v", err)
	}
	if err := br.Start(); err != nil {
		in.Stop()
		t.Fatalf("browser.Start: %v", err)
	}

	t.Cleanup(func() {
		in.Stop()
		_ = br.Instance.Close()
	})
	return in, br
}

// TestMain runs once per package; when E2E is set, runs all tests.
func TestMain(m *testing.M) {
	if os.Getenv("E2E") == "" {
		fmt.Println("Skipping e2e tests (E2E not set)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
