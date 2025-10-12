package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// Category represents an expense or income category
type Category struct {
	ID          int        `json:"id,omitempty"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ParentID    int        `json:"parentId,omitempty"`
	Children    []Category `json:"_,omitempty"`
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

	switch categoryType {
	case "expense":
		url = fmt.Sprintf("%s/api/v0/fin/category/expense", baseURL)
	case "income":
		url = fmt.Sprintf("%s/api/v0/fin/category/income", baseURL)
	default:
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
	defer func() {
		e := resp.Body.Close()
		if e != nil {
			panic(e)
		}
	}()

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

// createCategoriesRecursive creates categories from a nested structure
func createCategoriesRecursive(baseURL string, categoryType string, categories []Category, parentID int) (map[string]int, error) {
	categoryIDs := make(map[string]int)

	for _, category := range categories {
		// Set parent ID if provided
		if parentID > 0 {
			category.ParentID = parentID
		}

		// Create the category
		categoryID, err := createCategory(baseURL, categoryType, category)
		if err != nil {
			return nil, fmt.Errorf("failed to create category '%s': %v", category.Name, err)
		}

		// Store the ID with the category name as key
		categoryIDs[category.Name] = categoryID
		slog.Info(fmt.Sprintf("✅ %s category '%s' created with ID: %d", categoryType, category.Name, categoryID))

		// Recursively create children if they exist
		if len(category.Children) > 0 {
			childIDs, err := createCategoriesRecursive(baseURL, categoryType, category.Children, categoryID)
			if err != nil {
				return nil, err
			}

			// Merge child IDs into the main map
			for name, id := range childIDs {
				categoryIDs[name] = id
			}
		}
	}

	return categoryIDs, nil
}
