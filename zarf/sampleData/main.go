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
		slog.Error("Login failed", "error", err)
	}
	slog.Info("Login success")

	// ======================================================================
	// Create account provider
	// ======================================================================

	for i, provider := range Accounts {
		providerID, err := createProvider(apiBase, provider)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to create provider: %v", err))
		}
		slog.Info(fmt.Sprintf("✅ Provider '%s' created successfully with ID: %d", provider.Name, providerID))
		Accounts[i].ID = providerID

		for j, acc := range provider.Accounts {
			acc.ProviderID = providerID
			accoundId, err := createAccount(apiBase, acc)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to create account: %v", err))
			}
			Accounts[i].Accounts[j].ID = accoundId
			slog.Info(fmt.Sprintf("✅ Account '%s' created successfully\n", acc.Name))
		}
	}

	// ======================================================================
	// Create categories
	// ======================================================================

	// Create expense categories from nested structure
	expenseCategoryIDs, err := createCategoriesRecursive(apiBase, "expense", ExpenseCategories, 0)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create expense categories: %v", err))
	}

	// Create income categories from nested structure
	incomeCategoryIDs, err := createCategoriesRecursive(apiBase, "income", IncomeCategories, 0)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create income categories: %v", err))
	}

	// ======================================================================
	// Create sample entries
	// ======================================================================

	// Create entries from the Entries package variable
	for _, entryDef := range Entries {
		entry, err := convertEntryDefinitionToEntry(entryDef, expenseCategoryIDs, incomeCategoryIDs)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to convert entry definition: %v", err))
			continue
		}

		entryID, err := createEntry(apiBase, entry)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to create entry '%s': %v", entry.Description, err))
		} else {
			slog.Info(fmt.Sprintf("✅ Entry '%s' created with ID: %d", entry.Description, entryID))
		}
	}

	slog.Info("Sample data creation completed successfully")
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
