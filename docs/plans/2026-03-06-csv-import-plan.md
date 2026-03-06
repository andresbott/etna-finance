# CSV Import Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to import bank CSV exports as income/expense transactions into specific accounts, with automatic category matching, duplicate detection, and a full preview/edit flow before committing.

**Architecture:** New `internal/csvimport` package for profiles, category rules, and CSV parsing logic. New API handlers in `app/router/handlers/csvimport/`. New Vue route `/import` with upload + preview states reusing the existing account entries table and edit dialogs. Import profiles are stored in DB and linked to accounts via a nullable FK.

**Tech Stack:** Go (GORM/SQLite), gorilla/mux, Vue 3 (PrimeVue, TanStack Vue Query, Axios)

**Design doc:** `docs/plans/2026-03-06-csv-import-design.md`

---

## Task 1: Import Profile — DB Model & Store

**Files:**
- Create: `internal/csvimport/profile.go`
- Create: `internal/csvimport/profile_test.go`
- Create: `internal/csvimport/csvimport.go` (store initialization)

**Step 1: Create the store and profile model**

Create `internal/csvimport/csvimport.go`:

```go
package csvimport

import (
	"fmt"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	s := &Store{db: db}
	err := db.AutoMigrate(&dbImportProfile{}, &dbCategoryRule{})
	if err != nil {
		return nil, err
	}
	return s, nil
}
```

Create `internal/csvimport/profile.go`:

```go
package csvimport

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrProfileNotFound = errors.New("import profile not found")

type ImportProfile struct {
	ID                uint
	Name              string
	CsvSeparator      string
	SkipRows          int
	DateColumn        string
	DateFormat        string
	DescriptionColumn string
	AmountColumn      string
}

type dbImportProfile struct {
	ID                uint   `gorm:"primarykey"`
	Name              string `gorm:"not null"`
	CsvSeparator      string `gorm:"default:','"`
	SkipRows          int    `gorm:"default:0"`
	DateColumn        string `gorm:"not null"`
	DateFormat        string `gorm:"not null"`
	DescriptionColumn string `gorm:"not null"`
	AmountColumn      string `gorm:"not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func dbToProfile(in dbImportProfile) ImportProfile {
	return ImportProfile{
		ID:                in.ID,
		Name:              in.Name,
		CsvSeparator:      in.CsvSeparator,
		SkipRows:          in.SkipRows,
		DateColumn:        in.DateColumn,
		DateFormat:        in.DateFormat,
		DescriptionColumn: in.DescriptionColumn,
		AmountColumn:      in.AmountColumn,
	}
}

func (s *Store) CreateProfile(ctx context.Context, p ImportProfile) (uint, error) {
	if p.Name == "" {
		return 0, ErrValidation("name cannot be empty")
	}
	if p.DateColumn == "" || p.DateFormat == "" || p.DescriptionColumn == "" || p.AmountColumn == "" {
		return 0, ErrValidation("date_column, date_format, description_column and amount_column are required")
	}
	sep := p.CsvSeparator
	if sep == "" {
		sep = ","
	}
	row := dbImportProfile{
		Name:              p.Name,
		CsvSeparator:      sep,
		SkipRows:          p.SkipRows,
		DateColumn:        p.DateColumn,
		DateFormat:        p.DateFormat,
		DescriptionColumn: p.DescriptionColumn,
		AmountColumn:      p.AmountColumn,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *Store) GetProfile(ctx context.Context, id uint) (ImportProfile, error) {
	var row dbImportProfile
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ImportProfile{}, ErrProfileNotFound
		}
		return ImportProfile{}, err
	}
	return dbToProfile(row), nil
}

func (s *Store) ListProfiles(ctx context.Context) ([]ImportProfile, error) {
	var rows []dbImportProfile
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ImportProfile, len(rows))
	for i, r := range rows {
		out[i] = dbToProfile(r)
	}
	return out, nil
}

func (s *Store) UpdateProfile(ctx context.Context, id uint, p ImportProfile) error {
	q := s.db.WithContext(ctx).Model(&dbImportProfile{}).Where("id = ?", id).Updates(dbImportProfile{
		Name:              p.Name,
		CsvSeparator:      p.CsvSeparator,
		SkipRows:          p.SkipRows,
		DateColumn:        p.DateColumn,
		DateFormat:        p.DateFormat,
		DescriptionColumn: p.DescriptionColumn,
		AmountColumn:      p.AmountColumn,
	})
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (s *Store) DeleteProfile(ctx context.Context, id uint) error {
	q := s.db.WithContext(ctx).Where("id = ?", id).Delete(&dbImportProfile{})
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return ErrProfileNotFound
	}
	return nil
}
```

Create `internal/csvimport/errors.go`:

```go
package csvimport

type ErrValidation string

func (e ErrValidation) Error() string { return string(e) }
```

**Step 2: Write tests**

Create `internal/csvimport/profile_test.go`. Follow the project test pattern using `testdbs`. Tests should cover: create, get, list, update, delete, validation errors, not-found errors.

**Step 3: Run tests**

Run: `go test ./internal/csvimport/... -v`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/csvimport/
git commit -m "feat(csvimport): add import profile DB model and store CRUD"
```

---

## Task 2: Category Rules — DB Model & Store

**Files:**
- Create: `internal/csvimport/category_rule.go`
- Create: `internal/csvimport/category_rule_test.go`
- Modify: `internal/csvimport/csvimport.go` (already migrates `dbCategoryRule`)

**Step 1: Create the category rule model and store methods**

Create `internal/csvimport/category_rule.go`:

```go
package csvimport

import (
	"context"
	"errors"
	"regexp"
	"time"

	"gorm.io/gorm"
)

var ErrCategoryRuleNotFound = errors.New("category rule not found")

type CategoryRule struct {
	ID         uint
	Pattern    string
	IsRegex    bool
	CategoryID uint
	Position   int
}

type dbCategoryRule struct {
	ID         uint `gorm:"primarykey"`
	Pattern    string `gorm:"not null"`
	IsRegex    bool   `gorm:"default:false"`
	CategoryID uint   `gorm:"not null;index"`
	Position   int    `gorm:"not null;index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func dbToCategoryRule(in dbCategoryRule) CategoryRule {
	return CategoryRule{
		ID:         in.ID,
		Pattern:    in.Pattern,
		IsRegex:    in.IsRegex,
		CategoryID: in.CategoryID,
		Position:   in.Position,
	}
}

func (s *Store) CreateCategoryRule(ctx context.Context, r CategoryRule) (uint, error) {
	if r.Pattern == "" {
		return 0, ErrValidation("pattern cannot be empty")
	}
	if r.CategoryID == 0 {
		return 0, ErrValidation("category_id is required")
	}
	if r.IsRegex {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			return 0, ErrValidation("invalid regex pattern: " + err.Error())
		}
	}
	row := dbCategoryRule{
		Pattern:    r.Pattern,
		IsRegex:    r.IsRegex,
		CategoryID: r.CategoryID,
		Position:   r.Position,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (s *Store) GetCategoryRule(ctx context.Context, id uint) (CategoryRule, error) {
	var row dbCategoryRule
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return CategoryRule{}, ErrCategoryRuleNotFound
		}
		return CategoryRule{}, err
	}
	return dbToCategoryRule(row), nil
}

func (s *Store) ListCategoryRules(ctx context.Context) ([]CategoryRule, error) {
	var rows []dbCategoryRule
	if err := s.db.WithContext(ctx).Order("position ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]CategoryRule, len(rows))
	for i, r := range rows {
		out[i] = dbToCategoryRule(r)
	}
	return out, nil
}

func (s *Store) UpdateCategoryRule(ctx context.Context, id uint, r CategoryRule) error {
	if r.IsRegex && r.Pattern != "" {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			return ErrValidation("invalid regex pattern: " + err.Error())
		}
	}
	q := s.db.WithContext(ctx).Model(&dbCategoryRule{}).Where("id = ?", id).Updates(dbCategoryRule{
		Pattern:    r.Pattern,
		IsRegex:    r.IsRegex,
		CategoryID: r.CategoryID,
		Position:   r.Position,
	})
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return ErrCategoryRuleNotFound
	}
	return nil
}

func (s *Store) DeleteCategoryRule(ctx context.Context, id uint) error {
	q := s.db.WithContext(ctx).Where("id = ?", id).Delete(&dbCategoryRule{})
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return ErrCategoryRuleNotFound
	}
	return nil
}
```

**Step 2: Write tests**

Create `internal/csvimport/category_rule_test.go`. Test: CRUD, ordering by position, regex validation on create/update, not-found, empty pattern validation.

**Step 3: Run tests**

Run: `go test ./internal/csvimport/... -v`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/csvimport/category_rule.go internal/csvimport/category_rule_test.go
git commit -m "feat(csvimport): add category rule DB model and store CRUD"
```

---

## Task 3: CSV Parser — Parse CSV with Profile & Match Categories

**Files:**
- Create: `internal/csvimport/parser.go`
- Create: `internal/csvimport/parser_test.go`

**Step 1: Define parsed row type and parser**

Create `internal/csvimport/parser.go`:

```go
package csvimport

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ParsedRow struct {
	RowNumber   int     `json:"rowNumber"`
	Date        string  `json:"date"`        // YYYY-MM-DD
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`        // "income" or "expense"
	CategoryID  uint    `json:"categoryId"`  // 0 if no match
	IsDuplicate bool    `json:"isDuplicate"`
	Error       string  `json:"error,omitempty"`
}

// ExistingTx represents a minimal existing transaction for duplicate detection.
type ExistingTx struct {
	Date   string  // YYYY-MM-DD
	Amount float64
}

// Parse reads a CSV from r using the given profile, applies category rules,
// and checks for duplicates against existing transactions.
func Parse(r io.Reader, profile ImportProfile, rules []CategoryRule, existing []ExistingTx) ([]ParsedRow, error) {
	sep := ','
	if profile.CsvSeparator != "" {
		runes := []rune(profile.CsvSeparator)
		sep = runes[0]
	}

	reader := csv.NewReader(r)
	reader.Comma = sep
	reader.LazyQuotes = true

	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	// Skip preamble rows
	skip := profile.SkipRows
	if skip >= len(allRows) {
		return nil, fmt.Errorf("skip_rows (%d) exceeds total rows (%d)", skip, len(allRows))
	}
	allRows = allRows[skip:]

	if len(allRows) < 2 { // header + at least one data row
		return nil, fmt.Errorf("CSV has no data rows after skipping %d rows", skip)
	}

	// Build header index
	header := allRows[0]
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	dateIdx, ok := colIdx[profile.DateColumn]
	if !ok {
		return nil, fmt.Errorf("date column %q not found in CSV headers", profile.DateColumn)
	}
	descIdx, ok := colIdx[profile.DescriptionColumn]
	if !ok {
		return nil, fmt.Errorf("description column %q not found in CSV headers", profile.DescriptionColumn)
	}
	amtIdx, ok := colIdx[profile.AmountColumn]
	if !ok {
		return nil, fmt.Errorf("amount column %q not found in CSV headers", profile.AmountColumn)
	}

	// Build duplicate lookup: key = "date|amount"
	dupSet := map[string]bool{}
	for _, tx := range existing {
		key := fmt.Sprintf("%s|%.2f", tx.Date, tx.Amount)
		dupSet[key] = true
	}

	// Compile regex rules once
	type compiledRule struct {
		rule  CategoryRule
		regex *regexp.Regexp
	}
	compiled := make([]compiledRule, len(rules))
	for i, rule := range rules {
		var re *regexp.Regexp
		if rule.IsRegex {
			re, _ = regexp.Compile(rule.Pattern) // already validated on create
		}
		compiled[i] = compiledRule{rule: rule, regex: re}
	}

	dataRows := allRows[1:]
	result := make([]ParsedRow, 0, len(dataRows))

	for i, row := range dataRows {
		parsed := ParsedRow{RowNumber: i + 1}

		// Date
		if dateIdx >= len(row) {
			parsed.Error = "row too short for date column"
			result = append(result, parsed)
			continue
		}
		dateStr := strings.TrimSpace(row[dateIdx])
		t, err := time.Parse(profile.DateFormat, dateStr)
		if err != nil {
			parsed.Error = fmt.Sprintf("invalid date %q: %v", dateStr, err)
			result = append(result, parsed)
			continue
		}
		parsed.Date = t.Format("2006-01-02")

		// Description
		if descIdx < len(row) {
			parsed.Description = strings.TrimSpace(row[descIdx])
		}

		// Amount
		if amtIdx >= len(row) {
			parsed.Error = "row too short for amount column"
			result = append(result, parsed)
			continue
		}
		amtStr := strings.TrimSpace(row[amtIdx])
		amt, err := parseAmount(amtStr)
		if err != nil {
			parsed.Error = fmt.Sprintf("invalid amount %q: %v", amtStr, err)
			result = append(result, parsed)
			continue
		}
		parsed.Amount = amt

		// Type from sign
		if amt >= 0 {
			parsed.Type = "income"
		} else {
			parsed.Type = "expense"
		}

		// Category matching
		parsed.CategoryID = matchCategory(parsed.Description, parsed.Type, compiled)

		// Duplicate detection
		key := fmt.Sprintf("%s|%.2f", parsed.Date, parsed.Amount)
		parsed.IsDuplicate = dupSet[key]

		result = append(result, parsed)
	}

	return result, nil
}

// matchCategory returns the category ID of the first matching rule, or 0.
func matchCategory(description, txType string, rules []compiledRule) uint {
	descLower := strings.ToLower(description)
	for _, cr := range rules {
		matched := false
		if cr.rule.IsRegex && cr.regex != nil {
			matched = cr.regex.MatchString(description)
		} else {
			matched = strings.Contains(descLower, strings.ToLower(cr.rule.Pattern))
		}
		if matched {
			return cr.rule.CategoryID
		}
	}
	return 0
}

// parseAmount handles amounts with comma as decimal separator (e.g. "1.234,56" or "1234,56")
// and standard dot notation ("1234.56").
func parseAmount(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}

	// If the string contains both comma and dot, determine which is decimal
	hasComma := strings.Contains(s, ",")
	hasDot := strings.Contains(s, ".")

	if hasComma && hasDot {
		// e.g. "1.234,56" → comma is decimal
		if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			// e.g. "1,234.56" → dot is decimal
			s = strings.ReplaceAll(s, ",", "")
		}
	} else if hasComma {
		// Only comma: treat as decimal separator
		s = strings.ReplaceAll(s, ",", ".")
	}

	return strconv.ParseFloat(s, 64)
}
```

**Step 2: Write tests**

Create `internal/csvimport/parser_test.go`. Test cases:
- Parse a valid CSV with known profile (comma-separated, semicolon-separated)
- Amount sign → correct type (income/expense)
- Category matching: substring match, regex match, first-match-wins, no match → 0
- Duplicate detection: matching date+amount flagged
- Error rows: bad date, bad amount, short row
- Skip rows
- Comma-as-decimal amount parsing (`1.234,56`)
- Missing columns in CSV → error

**Step 3: Run tests**

Run: `go test ./internal/csvimport/... -v`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/csvimport/parser.go internal/csvimport/parser_test.go
git commit -m "feat(csvimport): add CSV parser with category matching and duplicate detection"
```

---

## Task 4: Account Import Profile Link

**Files:**
- Modify: `internal/accounting/account.go` — add `ImportProfileID` to `Account` and `dbAccount`
- Modify: `internal/accounting/account.go` — update `CreateAccount`, `UpdateAccount`, `dbToAccount`
- Modify: existing account tests if needed

**Step 1: Add the field**

In `internal/accounting/account.go`, add to `Account`:
```go
type Account struct {
	ID                uint
	AccountProviderID uint
	Name              string
	Description       string
	Icon              string
	Currency          currency.Unit
	Type              AccountType
	ImportProfileID   uint // 0 means no linked profile
}
```

Add to `dbAccount`:
```go
type dbAccount struct {
	// ... existing fields ...
	ImportProfileID uint `gorm:"default:null"`
}
```

Update `dbToAccount` to copy the field. Update `AccountUpdatePayload` to include `ImportProfileID *uint`. Update `UpdateAccount` to handle the new field. Update `CreateAccount` to set it.

**Step 2: Run existing tests to confirm no regressions**

Run: `go test ./internal/accounting/... -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/accounting/account.go
git commit -m "feat(accounting): add ImportProfileID field to Account model"
```

---

## Task 5: Import API Handlers — Profiles & Category Rules CRUD

**Files:**
- Create: `app/router/handlers/csvimport/profile.go`
- Create: `app/router/handlers/csvimport/category_rule.go`
- Modify: `app/router/api_v0.go` — register new routes
- Modify: `app/router/router.go` — add csvimport store to `Cfg` and `MainAppHandler`

**Step 1: Create profile handler**

Create `app/router/handlers/csvimport/profile.go`. Follow the same handler pattern as `finance/category.go`:
- `type ProfileHandler struct { Store *csvimport.Store }`
- `ListProfiles() http.Handler` — GET, returns JSON array
- `CreateProfile() http.Handler` — POST, decode JSON body, return created ID
- `GetProfile(id uint) http.Handler` — GET by ID
- `UpdateProfile(id uint) http.Handler` — PUT, decode body, update
- `DeleteProfile(id uint) http.Handler` — DELETE by ID

JSON payload struct:
```go
type profilePayload struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	CsvSeparator      string `json:"csvSeparator"`
	SkipRows          int    `json:"skipRows"`
	DateColumn        string `json:"dateColumn"`
	DateFormat        string `json:"dateFormat"`
	DescriptionColumn string `json:"descriptionColumn"`
	AmountColumn      string `json:"amountColumn"`
}
```

**Step 2: Create category rule handler**

Create `app/router/handlers/csvimport/category_rule.go`. Same pattern:
- `type CategoryRuleHandler struct { Store *csvimport.Store }`
- CRUD handlers for category rules

JSON payload struct:
```go
type categoryRulePayload struct {
	ID         uint   `json:"id"`
	Pattern    string `json:"pattern"`
	IsRegex    bool   `json:"isRegex"`
	CategoryID uint   `json:"categoryId"`
	Position   int    `json:"position"`
}
```

**Step 3: Wire into router**

Modify `app/router/router.go` — add `CsvImportStore *csvimport.Store` to `Cfg` struct and store it in `MainAppHandler`.

Modify `app/router/api_v0.go` — add a new method `csvImportAPI(r *mux.Router)` following the `accountingAPI` pattern. Register routes:
```
/api/v0/import/profiles       GET, POST
/api/v0/import/profiles/{id}  GET, PUT, DELETE
/api/v0/import/category-rules       GET, POST
/api/v0/import/category-rules/{id}  GET, PUT, DELETE
```

Call `h.csvImportAPI(apiV0)` from the same place `h.accountingAPI(apiV0)` is called.

**Step 4: Initialize csvimport.Store in server.go**

Modify `app/cmd/server.go` — after `accounting.NewStore`, create `csvimportStore, err := csvimport.NewStore(db)` and pass it to `router.Cfg.CsvImportStore`.

**Step 5: Build and verify**

Run: `go build ./...`
Expected: Compiles cleanly

**Step 6: Commit**

```bash
git add app/router/handlers/csvimport/ app/router/api_v0.go app/router/router.go app/cmd/server.go
git commit -m "feat(csvimport): add profile and category rule API endpoints"
```

---

## Task 6: Import API Handlers — Parse & Submit

**Files:**
- Create: `app/router/handlers/csvimport/import.go`
- Modify: `app/router/api_v0.go` — register parse/submit routes

**Step 1: Create the import handler**

Create `app/router/handlers/csvimport/import.go`:

```go
type ImportHandler struct {
	CsvStore *csvimport.Store
	FinStore *accounting.Store
}
```

`ParseCSV() http.Handler`:
- Accept multipart form: `file` field (the CSV) + `accountId` field
- Look up the account → get its `ImportProfileID` (400 if 0/none)
- Look up the profile from csvimport store
- Load all category rules (ordered by position)
- Load existing transactions for that account in a reasonable date range (or all) for duplicate detection — query the accounting store for transactions in the account, extract date+amount pairs as `ExistingTx`
- Call `csvimport.Parse(file, profile, rules, existing)`
- Return JSON `{ "rows": [...] }`

`SubmitImport() http.Handler`:
- Accept JSON body: `{ "accountId": uint, "rows": [{ date, description, amount, type, categoryId }] }`
- Validate each row
- Create transactions in bulk using `accounting.Store` methods (one `CreateTransaction` call per row, wrapped in a DB transaction via `store.db.Transaction()`)
- Return `{ "created": count }`

**Step 2: Register routes**

Add to `csvImportAPI` in `app/router/api_v0.go`:
```
POST /api/v0/import/parse
POST /api/v0/import/submit
```

Pass both `CsvStore` and `FinStore` to the `ImportHandler`.

**Step 3: Build and manually test with curl**

Run: `go build ./...`
Expected: Compiles cleanly

**Step 4: Commit**

```bash
git add app/router/handlers/csvimport/import.go app/router/api_v0.go
git commit -m "feat(csvimport): add CSV parse and submit API endpoints"
```

---

## Task 7: Frontend — API Client & Types

**Files:**
- Create: `webui/src/lib/api/CsvImport.ts`
- Create: `webui/src/types/csvimport.ts`

**Step 1: Define TypeScript types**

Create `webui/src/types/csvimport.ts`:

```ts
export interface ImportProfile {
  id: number
  name: string
  csvSeparator: string
  skipRows: number
  dateColumn: string
  dateFormat: string
  descriptionColumn: string
  amountColumn: string
}

export interface CategoryRule {
  id: number
  pattern: string
  isRegex: boolean
  categoryId: number
  position: number
}

export interface ParsedRow {
  rowNumber: number
  date: string
  description: string
  amount: number
  type: 'income' | 'expense'
  categoryId: number
  isDuplicate: boolean
  error?: string
}
```

**Step 2: Create API functions**

Create `webui/src/lib/api/CsvImport.ts` using the `apiClient` from `client.ts`:

```ts
import { apiClient } from './client'
import type { ImportProfile, CategoryRule, ParsedRow } from '@/types/csvimport'

// Profiles
export const getProfiles = () => apiClient.get<ImportProfile[]>('/import/profiles').then(r => r.data)
export const createProfile = (p: Omit<ImportProfile, 'id'>) => apiClient.post<{ id: number }>('/import/profiles', p).then(r => r.data)
export const updateProfile = (id: number, p: Partial<ImportProfile>) => apiClient.put(`/import/profiles/${id}`, p).then(r => r.data)
export const deleteProfile = (id: number) => apiClient.delete(`/import/profiles/${id}`).then(r => r.data)

// Category Rules
export const getCategoryRules = () => apiClient.get<CategoryRule[]>('/import/category-rules').then(r => r.data)
export const createCategoryRule = (r: Omit<CategoryRule, 'id'>) => apiClient.post<{ id: number }>('/import/category-rules', r).then(res => res.data)
export const updateCategoryRule = (id: number, r: Partial<CategoryRule>) => apiClient.put(`/import/category-rules/${id}`, r).then(res => res.data)
export const deleteCategoryRule = (id: number) => apiClient.delete(`/import/category-rules/${id}`).then(r => r.data)

// Import
export const parseCSV = (accountId: number, file: File) => {
  const form = new FormData()
  form.append('file', file)
  form.append('accountId', String(accountId))
  return apiClient.post<{ rows: ParsedRow[] }>('/import/parse', form, {
    headers: { 'Content-Type': 'multipart/form-data' }
  }).then(r => r.data)
}

export const submitImport = (accountId: number, rows: ParsedRow[]) =>
  apiClient.post<{ created: number }>('/import/submit', { accountId, rows }).then(r => r.data)
```

**Step 3: Commit**

```bash
git add webui/src/lib/api/CsvImport.ts webui/src/types/csvimport.ts
git commit -m "feat(csvimport): add frontend API client and TypeScript types"
```

---

## Task 8: Frontend — Import Page (Upload + Preview)

**Files:**
- Create: `webui/src/views/csvimport/ImportView.vue`
- Modify: `webui/src/router/index.js` — add `/import` route

**Step 1: Add the route**

In `webui/src/router/index.js`, add:
```js
{
  path: '/import',
  name: 'csv-import',
  meta: { requiresAuth: true },
  component: () => import('@/views/csvimport/ImportView.vue')
}
```

**Step 2: Create the ImportView component**

Create `webui/src/views/csvimport/ImportView.vue` with two states:

**Upload state:**
- Read `accountId` from `route.query.accountId`
- Show account name (reuse `useAccounts` composable)
- File input (PrimeVue `FileUpload` or simple `<input type="file" accept=".csv">`)
- "Parse" button → calls `parseCSV(accountId, file)`
- On success, transitions to preview state

**Preview state:**
- Reuse `AccountEntriesTable` component with the parsed rows interleaved with existing entries
- Parsed rows get a checkbox (default checked, except duplicates)
- Each new row is editable via the existing `IncomeExpenseDialog`
- Summary bar at top: "X new, Y duplicates, Z errors"
- "Import Selected" button → filters checked rows, calls `submitImport`
- On success, navigates to `/entries/:accountId`

The parsed rows need to be transformed to match the entry shape expected by `AccountEntriesTable`. Map `ParsedRow` fields to the entry format: `{ id: 'import-N', date, description, Amount: Math.abs(amount), type, categoryId, accountId, isImportRow: true }`.

**Step 3: Build frontend**

Run: `cd webui && npm run build`
Expected: Compiles cleanly

**Step 4: Commit**

```bash
git add webui/src/views/csvimport/ImportView.vue webui/src/router/index.js
git commit -m "feat(csvimport): add import page with upload and preview"
```

---

## Task 9: Frontend — "Import CSV" Button on Account View

**Files:**
- Modify: `webui/src/views/entries/AccountEntriesView.vue`

**Step 1: Add the import button**

In `AccountEntriesView.vue`, in the toolbar section, add a router-link button next to the `AddEntryMenu`:

```vue
<router-link
  v-if="accountHasImportProfile"
  :to="{ name: 'csv-import', query: { accountId: accountId } }"
>
  <Button label="Import CSV" icon="pi pi-upload" severity="secondary" />
</router-link>
```

Compute `accountHasImportProfile` from the account data (check if `importProfileId` is set and > 0). This requires the account API to return the `importProfileId` field — update the account API handler to include it in the response.

**Step 2: Update account API handler**

Modify `app/router/handlers/finance/account.go` — include `importProfileId` in the account JSON response. Also accept it in create/update payloads.

**Step 3: Build and verify**

Run: `go build ./... && cd webui && npm run build`
Expected: Both compile

**Step 4: Commit**

```bash
git add webui/src/views/entries/AccountEntriesView.vue app/router/handlers/finance/account.go
git commit -m "feat(csvimport): add Import CSV button to account view"
```

---

## Task 10: Frontend — Profile Management Settings Page

**Files:**
- Modify: `webui/src/views/csvimport/CsvImportProfileView.vue` — replace mock with real API calls
- Modify: `webui/src/components/CsvHeaderEditor.vue` — adapt to match profile fields

**Step 1: Wire the existing mock UI to real API**

The existing `CsvImportProfileView.vue` already has the UI structure. Replace the mock data and simulated delays with real API calls using the functions from `CsvImport.ts`. Use TanStack Vue Query (`useQuery`, `useMutation`) following the pattern in `useEntries.ts`.

Update `CsvHeaderEditor.vue` to match the simplified profile fields (date_column, description_column, amount_column, date_format, csv_separator, skip_rows) instead of the generic header mapping it currently has.

**Step 2: Build frontend**

Run: `cd webui && npm run build`
Expected: Compiles cleanly

**Step 3: Commit**

```bash
git add webui/src/views/csvimport/CsvImportProfileView.vue webui/src/components/CsvHeaderEditor.vue
git commit -m "feat(csvimport): wire profile management UI to real API"
```

---

## Task 11: Frontend — Category Rules Management Page

**Files:**
- Create: `webui/src/views/csvimport/CategoryRulesView.vue`
- Modify: `webui/src/router/index.js` — add route

**Step 1: Create the category rules view**

Build a settings page with:
- PrimeVue `DataTable` showing rules ordered by position
- Columns: Position (drag handle or up/down buttons), Pattern, Regex toggle, Category (dropdown from existing categories), Actions (edit/delete)
- Create/edit dialog with fields: pattern (InputText), isRegex (Checkbox), categoryId (Dropdown using categories from `useCategories` composable), position (InputNumber)
- Reorder support: when position changes, send update to API

Use `getCategoryRules`, `createCategoryRule`, `updateCategoryRule`, `deleteCategoryRule` from the API client.

**Step 2: Add route**

In `webui/src/router/index.js`:
```js
{
  path: '/settings/category-rules',
  name: 'category-rules',
  meta: { requiresAuth: true },
  component: () => import('@/views/csvimport/CategoryRulesView.vue')
}
```

**Step 3: Build frontend**

Run: `cd webui && npm run build`
Expected: Compiles cleanly

**Step 4: Commit**

```bash
git add webui/src/views/csvimport/CategoryRulesView.vue webui/src/router/index.js
git commit -m "feat(csvimport): add category rules management page"
```

---

## Task 12: Account Edit — Import Profile Dropdown

**Files:**
- Modify: the account create/edit dialog component (find in `webui/src/views/accounts/` or `webui/src/components/`)
- Modify: account API types to include `importProfileId`

**Step 1: Add import profile dropdown**

In the account edit dialog/form, add a PrimeVue `Dropdown` for selecting an import profile. Load profiles via `getProfiles()`. The value maps to `importProfileId` on the account. Allow "None" option (value 0/null).

**Step 2: Update account TypeScript types**

Add `importProfileId?: number` to the account interface in `webui/src/types/account.ts`.

**Step 3: Build and verify**

Run: `cd webui && npm run build`
Expected: Compiles cleanly

**Step 4: Commit**

```bash
git add webui/src/views/accounts/ webui/src/types/account.ts
git commit -m "feat(csvimport): add import profile dropdown to account edit form"
```

---

## Task 13: End-to-End Test

**Files:**
- Create: e2e test following existing patterns (check if there's an `e2e/` or `tests/` directory)

**Step 1: Write an integration test**

Test the full flow in Go:
1. Create a csvimport store
2. Create a profile
3. Create category rules
4. Parse a sample CSV string with the profile
5. Assert: correct rows, types, category matches, duplicates flagged
6. Submit the rows via the accounting store
7. Verify transactions were created

**Step 2: Run test**

Run: `go test ./internal/csvimport/... -v -run TestIntegration`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/csvimport/
git commit -m "test(csvimport): add integration test for full import flow"
```

---

## Task Order & Dependencies

```
Task 1 (Profile store) ──┐
Task 2 (Category rules) ─┤
                          ├─→ Task 3 (Parser) ─→ Task 6 (Parse/Submit API)
Task 4 (Account FK) ─────┘                              │
                                                         ▼
Task 5 (Profile/Rule API) ──────────────────→ Task 7 (Frontend types/API)
                                                         │
                                              ┌──────────┼──────────┐
                                              ▼          ▼          ▼
                                          Task 8     Task 10    Task 11
                                        (Import      (Profile   (Category
                                         page)        mgmt)      rules)
                                              │
                                              ▼
                                          Task 9 (Import button)
                                              │
                                              ▼
                                          Task 12 (Profile dropdown)
                                              │
                                              ▼
                                          Task 13 (E2E test)
```

**Parallelizable:** Tasks 1, 2, 4 can run in parallel. Tasks 8, 10, 11 can run in parallel.
