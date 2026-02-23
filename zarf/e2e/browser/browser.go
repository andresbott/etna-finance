package browser

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// Cfg configures the browser.
type Cfg struct {
	Headless bool
	WindowW  int // window width (0 = default 1280)
	WindowH  int // window height (0 = default 900)
}

// Browser wraps a ROD browser instance.
type Browser struct {
	Path     string
	Instance *rod.Browser
	Page     *rod.Page
	headless bool
	windowW  int
	windowH  int
}

const defaultWindowW = 1280
const defaultWindowH = 900

// New creates a new browser (use Start to launch it).
func New(cfg Cfg) (*Browser, error) {
	path, _ := launcher.LookPath()
	if path == "" {
		return nil, fmt.Errorf("unable to find browsers to launch")
	}
	w, h := cfg.WindowW, cfg.WindowH
	if w <= 0 {
		w = defaultWindowW
	}
	if h <= 0 {
		h = defaultWindowH
	}
	return &Browser{
		Path:     path,
		headless: cfg.Headless,
		windowW:  w,
		windowH:  h,
	}, nil
}

// Start launches the browser and connects to it.
func (br *Browser) Start() error {
	u, err := launcher.New().Headless(br.headless).Bin(br.Path).Launch()
	if err != nil {
		return fmt.Errorf("unable to launch browser: %w", err)
	}

	br.Instance = rod.New().ControlURL(u)
	if err := br.Instance.Connect(); err != nil {
		return fmt.Errorf("unable to connect to browser: %w", err)
	}
	br.Page = br.Instance.MustPage()
	br.Page.MustSetWindow(0, 0, br.windowW, br.windowH)
	return nil
}

// Navigate navigates to url and waits for load.
func (br *Browser) Navigate(url string) (*rod.Page, error) {
	if err := br.Page.Navigate(url); err != nil {
		return nil, err
	}
	if err := br.Page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait for page: %w", err)
	}
	return br.Page, nil
}
