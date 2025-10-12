package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Provider struct {
	ID          int       `json:"ID,omitempty"`
	Name        string    `json:"Name"`
	Description string    `json:"Description"`
	Accounts    []Account `json:"Accounts,omitempty"`
}

type Account struct {
	ID          int    `json:"ID,omitempty"`
	ProviderID  int    `json:"providerId"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Currency    string `json:"Currency"`
	Type        string `json:"Type"`
}

type ProviderResponse struct {
	ID int `json:"ID"`
}

// createProvider sends a POST request and returns the generated provider ID
func createProvider(baseURL string, provider Provider) (int, error) {
	url := fmt.Sprintf("%s/api/v0/fin/provider", baseURL)

	body, _ := json.Marshal(provider)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if authCookie != nil {
		req.AddCookie(authCookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("createProvider failed: %s", data)
	}

	var providerResp ProviderResponse
	if err := json.NewDecoder(resp.Body).Decode(&providerResp); err != nil {
		return 0, err
	}

	return providerResp.ID, nil
}

// createAccount sends a POST request to create an account and returns the generated ID
func createAccount(baseURL string, account Account) (int, error) {
	url := fmt.Sprintf("%s/api/v0/fin/account", baseURL)

	body, _ := json.Marshal(account)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if authCookie != nil {
		req.AddCookie(authCookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("createAccount failed: %s", data)
	}

	var accountResp Account
	if err := json.NewDecoder(resp.Body).Decode(&accountResp); err != nil {
		return 0, err
	}

	return accountResp.ID, nil
}

// findAccountID searches for an account by provider name + account name and returns the account ID
func findAccountID(providerName, accountName string) (int, error) {
	for _, provider := range Accounts {
		if provider.Name == providerName {
			for _, account := range provider.Accounts {
				if account.Name == accountName {
					if account.ID == 0 {
						return 0, fmt.Errorf("account '%s' in provider '%s' has no ID (not yet created)", accountName, providerName)
					}
					return account.ID, nil
				}
			}
			return 0, fmt.Errorf("account '%s' not found in provider '%s'", accountName, providerName)
		}
	}
	return 0, fmt.Errorf("provider '%s' not found", providerName)
}
