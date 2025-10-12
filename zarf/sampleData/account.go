package main

import (
	"fmt"
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
	var providerResp ProviderResponse
	err := postJSON(url, provider, &providerResp)
	if err != nil {
		return 0, err
	}
	return providerResp.ID, nil
}

// createAccount sends a POST request to create an account and returns the generated ID
func createAccount(baseURL string, account Account) (int, error) {
	url := fmt.Sprintf("%s/api/v0/fin/account", baseURL)
	var accountResp Account
	err := postJSON(url, account, &accountResp)
	if err != nil {
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
