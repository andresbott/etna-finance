package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Category represents an expense or income category
type Category struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentID    int    `json:"parentId,omitempty"`
}

// CategoryResponse represents the API response
type CategoryResponse struct {
	ID          int    `json:"id"`
	ParentID    int    `json:"parentId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// createCategory posts a category to the API and returns the generated ID
func createCategory(baseURL string, categoryType string, category Category) (int, error) {
	var url string
	if categoryType == "expense" {
		url = fmt.Sprintf("%s/api/v0/fin/category/expense", baseURL)
	} else if categoryType == "income" {
		url = fmt.Sprintf("%s/api/v0/fin/category/income", baseURL)
	} else {
		return 0, fmt.Errorf("invalid category type: %s", categoryType)
	}

	body, _ := json.Marshal(category)
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
		return 0, fmt.Errorf("createCategory failed: %s", data)
	}

	var catResp CategoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&catResp); err != nil {
		return 0, err
	}

	return catResp.ID, nil
}
