package end2endTests

import (
	"fmt"
	"github.com/andresbott/etna/zarf/end2endTests/browser"
	"os"
	"strings"
	"testing"
	"time"
)

var nav *browser.Browser
var username = "test"
var password = "test"

func TestMain(m *testing.M) {
	// e2e actually enabled

	e2eEnabled := os.Getenv("E2E")
	if e2eEnabled == "" {
		fmt.Println("Skipping end 2 end tests because E2E environment variable is not set.")
		os.Exit(0)
	}

	// setup the browser
	headless := false
	headlessEnv := os.Getenv("HEADLESS")
	if headlessEnv != "" {
		headless = true
	}

	br, err := browser.New(browser.Cfg{
		Headless: headless,
	})
	if err != nil {
		fmt.Printf("unexpected start error %v\n", err)
		os.Exit(1)
	}

	err = br.Start()
	if err != nil {
		fmt.Printf("unexpected start error %v\n", err)
		os.Exit(1)
	}
	nav = br

	// Run all tests
	exitCode := m.Run()

	// --- Global teardown ---

	// Exit with the test code
	os.Exit(exitCode)
}

const frontPort = 5173

func getUrl(path string) string {
	sanitizedPath := strings.TrimPrefix(path, "/")
	return fmt.Sprintf("http://localhost:%d/%s", frontPort, sanitizedPath)
}

var isLoggedIn = false

func logIn(t *testing.T) {
	if isLoggedIn {
		return
	}
	loginPage, err := nav.Navigate(getUrl("/login"))
	if err != nil {
		t.Errorf("unable to get login page: %v", err)
	}

	loginPage.MustWaitLoad()

	// input user and password
	loginPage.MustElement("#username").MustInput(username)
	loginPage.MustElement("#password input").MustInput(password)

	// Click the login button
	loginPage.MustElement("#login-submit").MustClick() // change selector

	// Optionally, wait for navigation or a success element
	err = loginPage.WaitStable(300 * time.Millisecond)
	if err != nil {
		t.Errorf("unable to wait login: %v", err)
	}
	success := loginPage.MustHas("#sidebar-menu")
	if !success {
		t.Errorf("Login did not succeed")
	}
	isLoggedIn = true
}

func TestLogin(t *testing.T) {
	// since login is a pre-requisite for any other test, we extracted it into a function
	logIn(t)
}

func TestLogin2(t *testing.T) {
	t.Log("here goes the login test")
	logIn(t)
	time.Sleep(30 * time.Minute)
}
