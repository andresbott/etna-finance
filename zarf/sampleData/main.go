package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

// Environment variable key
const envAPIBase = "API_BASE_URL"
const envAPIUser = "API_USER"
const envAPIPass = "API_PASS"

// Global HTTP client and cookie storage
var client = &http.Client{}
var authCookie *http.Cookie

func main() {
	apiBase := os.Getenv(envAPIBase)
	if apiBase == "" {
		apiBase = "http://localhost:8085"
	}
	user := os.Getenv(envAPIUser)
	if user == "" {
		user = "demo"
	}
	pass := os.Getenv(envAPIPass)
	if pass == "" {
		pass = "demo"
	}

	if err := login(apiBase, user, pass); err != nil {
		slog.Error("Login failed: %v", err)
	}
	slog.Info("Login success")

	// ======================================================================
	// Create account provider
	// ======================================================================
	provider := Provider{
		Name:        "Banana",
		Description: "Sample provider",
	}
	providerID, err := createProvider(apiBase, provider)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create provider: %v", err))
	}
	slog.Info(fmt.Sprintf("✅ Provider '%s' created successfully with ID: %d", provider.Name, providerID))

	// ======================================================================
	// Create accounts
	// ======================================================================
	// 3️⃣ Create accounts associated with provider
	accounts := []Account{
		{
			ProviderID:  providerID,
			Name:        "Checking Account",
			Description: "Primary checking",
			Currency:    "USD",
			Type:        "cash",
		},
		{
			ProviderID:  providerID,
			Name:        "Savings Account",
			Description: "Savings",
			Currency:    "EUR",
			Type:        "cash",
		},
	}

	for _, acc := range accounts {
		if err := createAccount(apiBase, acc); err != nil {
			slog.Error(fmt.Sprintf("Failed to create account %s: %v", acc.Name, err))
		}
		slog.Info(fmt.Sprintf("✅ Account '%s' created successfully\n", acc.Name))
	}

	// ======================================================================
	// Create categories
	// ======================================================================

	// Expense category tree
	expRootID, err := createCategory(apiBase, "expense", Category{
		Name:        "Office Expenses",
		Description: "All office related expenses",
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create expense root: %v", err))
	}
	slog.Info(fmt.Sprintf("✅ Expense root category created with ID: %d\n", expRootID))

	_, _ = createCategory(apiBase, "expense", Category{
		Name:        "Stationery",
		Description: "Pens, papers, etc.",
		ParentID:    expRootID,
	})
	_, _ = createCategory(apiBase, "expense", Category{
		Name:        "Software",
		Description: "SaaS subscriptions",
		ParentID:    expRootID,
	})

	// Income category tree
	incRootID, err := createCategory(apiBase, "income", Category{
		Name:        "Sales",
		Description: "Revenue from sales",
	})
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create income root: %v", err))
	}
	slog.Info(fmt.Sprintf("✅ Income root category created with ID: %d\n", incRootID))

	_, _ = createCategory(apiBase, "income", Category{
		Name:        "Online Sales",
		Description: "Revenue from online store",
		ParentID:    incRootID,
	})
	_, _ = createCategory(apiBase, "income", Category{
		Name:        "Retail Sales",
		Description: "Revenue from physical store",
		ParentID:    incRootID,
	})

	slog.Info("Create account successfully")
}

type LoginRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	SessionRenew bool   `json:"sessionRenew"`
}

// login authenticates and stores session cookie
func login(baseURL, username, password string) error {
	url := fmt.Sprintf("%s/auth/login", baseURL)

	body, _ := json.Marshal(LoginRequest{
		Username:     username,
		Password:     password,
		SessionRenew: false,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s", data)
	}

	// Extract and store the authentication cookie
	for _, c := range resp.Cookies() {
		if c.Name == "_c_auth" {
			authCookie = c
			return nil
		}
	}
	return fmt.Errorf("no session cookie received")
}

// postJSON is a helper that performs authenticated JSON POST requests
func postJSON(url string, payload interface{}) error {
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if authCookie != nil {
		req.AddCookie(authCookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s failed: %s", url, data)
	}

	return nil
}
