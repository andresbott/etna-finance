package browser

import (
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"os"
)

// About
// https://go-rod.github.io/#/selectors/README

type Browser struct {
	Path     string
	Instance *rod.Browser
	Page     *rod.Page
	headles  bool
}
type Cfg struct {
	Headless bool
}

// New will start a new browser based on the system one
// eventually we can configure it to download the browser to use
func New(cfg Cfg) (*Browser, error) {

	headless := os.Getenv("HEADLESS")
	if headless != "" {
		cfg.Headless = false
	}

	path, _ := launcher.LookPath()
	if path == "" {
		return nil, fmt.Errorf("unable to find browsers to launch")
	}

	br := Browser{
		Path:    path,
		headles: cfg.Headless,
	}
	return &br, nil
}

// Start will start the browser and connect to it, this needs to be called only once
func (br *Browser) Start() error {

	u, err := launcher.New().Headless(br.headles).Bin(br.Path).Launch()
	if err != nil {
		return fmt.Errorf("unable to launch browser")
	}

	br.Instance = rod.New().ControlURL(u)
	err = br.Instance.Connect()
	if err != nil {
		return fmt.Errorf("unable to connect to browser")
	}
	br.Page = br.Instance.MustPage()
	return nil

}

// Navigate directs the browser to go to a specific page
func (br *Browser) Navigate(url string) (*rod.Page, error) {

	err := br.Page.Navigate(url)
	if err != nil {
		return nil, err
	}

	err = br.Page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("error waiting for page: %v", err)
	}

	return br.Page, nil

}
