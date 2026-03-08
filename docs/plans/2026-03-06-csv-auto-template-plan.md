# CSV Auto Template Configuration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the manual CSV profile form with an interactive wizard that extracts headers from a sample CSV, lets the user map columns via dropdowns with a live 10-row preview, and supports split credit/debit columns.

**Architecture:** Add `AmountMode`, `CreditColumn`, `DebitColumn` fields to the ImportProfile model. Add a `ParsePreview()` function that parses CSV without DB dependencies (no category matching or duplicate detection), capped at 10 rows. Add a `POST /import/preview` endpoint. Replace the frontend profile dialog with a two-step wizard.

**Tech Stack:** Go (backend), Vue 3 + PrimeVue (frontend), GORM (ORM), gorilla/mux (router)

---

### Task 1: Add split-column fields to ImportProfile model

**Files:**
- Modify: `internal/csvimport/profile.go:14-38` (both db and public structs)
- Modify: `internal/csvimport/profile.go:41-54` (dbToProfile mapper)

**Step 1: Write the failing test**

Add to `internal/csvimport/profile_test.go`:

```go
func TestCreateProfile_SplitMode(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	p := ImportProfile{
		Name:              "Split Bank",
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "2006-01-02",
		DescriptionColumn: "Desc",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}
	id, err := store.CreateProfile(ctx, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := store.GetProfile(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AmountMode != "split" {
		t.Errorf("expected AmountMode=split, got %s", got.AmountMode)
	}
	if got.CreditColumn != "Credit" {
		t.Errorf("expected CreditColumn=Credit, got %s", got.CreditColumn)
	}
	if got.DebitColumn != "Debit" {
		t.Errorf("expected DebitColumn=Debit, got %s", got.DebitColumn)
	}
}

func TestCreateProfile_SplitMode_MissingColumns(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	p := ImportProfile{
		Name:              "Split Bank",
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "2006-01-02",
		DescriptionColumn: "Desc",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		// DebitColumn missing
	}
	_, err := store.CreateProfile(ctx, p)
	if err == nil {
		t.Fatal("expected validation error for missing debit column")
	}
}

func TestCreateProfile_SingleMode_BackwardCompat(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// No AmountMode set — should default to single and require AmountColumn
	p := validProfile()
	p.AmountMode = ""
	id, err := store.CreateProfile(ctx, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := store.GetProfile(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AmountMode != "single" {
		t.Errorf("expected AmountMode=single, got %s", got.AmountMode)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && go test ./internal/csvimport/ -run TestCreateProfile_Split -v`
Expected: Compilation error — `AmountMode` field doesn't exist.

**Step 3: Write minimal implementation**

In `internal/csvimport/profile.go`, add fields to both structs:

```go
// dbImportProfile — add after AmountColumn field (line 22):
AmountMode   string `gorm:"default:'single'"`
CreditColumn string
DebitColumn  string

// ImportProfile — add after AmountColumn field (line 37):
AmountMode   string
CreditColumn string
DebitColumn  string
```

Update `dbToProfile` to map the new fields.

Update `CreateProfile` validation (lines 56-71): Replace the unconditional `AmountColumn` check with mode-aware validation:

```go
// Default amount mode
mode := p.AmountMode
if mode == "" {
	mode = "single"
}

switch mode {
case "single":
	if p.AmountColumn == "" {
		return 0, ErrValidation("amount_column cannot be empty when amount_mode is single")
	}
case "split":
	if p.CreditColumn == "" {
		return 0, ErrValidation("credit_column cannot be empty when amount_mode is split")
	}
	if p.DebitColumn == "" {
		return 0, ErrValidation("debit_column cannot be empty when amount_mode is split")
	}
default:
	return 0, ErrValidation("amount_mode must be 'single' or 'split'")
}
```

Set `mode` on the db row before create. Apply the same validation logic to `UpdateProfile`.

Update the `Select` list in `UpdateProfile` (line 144) to include `"AmountMode", "CreditColumn", "DebitColumn"`.

**Step 4: Run test to verify it passes**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && go test ./internal/csvimport/ -run TestCreateProfile -v`
Expected: All profile tests PASS, including existing ones.

**Step 5: Commit**

```bash
git add internal/csvimport/profile.go internal/csvimport/profile_test.go
git commit -m "feat(csvimport): add split credit/debit column fields to ImportProfile"
```

---

### Task 2: Add ParsePreview function with split-column support

**Files:**
- Modify: `internal/csvimport/parser.go` (add `ParsePreview`, add `resolveAmount` helper, update `Parse`)
- Modify: `internal/csvimport/parser_test.go` (add tests)

**Step 1: Write the failing tests**

Add to `internal/csvimport/parser_test.go`:

```go
func TestParsePreview_ReturnsHeadersAndRows(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Salary,1500.00
02/03/2026,Grocery,-45.30
`
	profile := defaultProfile()

	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
	if result.Headers[0] != "Date" || result.Headers[1] != "Description" || result.Headers[2] != "Amount" {
		t.Errorf("unexpected headers: %v", result.Headers)
	}
	if len(result.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(result.Rows))
	}
	if result.TotalRows != 2 {
		t.Errorf("expected TotalRows=2, got %d", result.TotalRows)
	}
}

func TestParsePreview_CapsAt10Rows(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("Date,Description,Amount\n")
	for i := 0; i < 20; i++ {
		sb.WriteString(fmt.Sprintf("01/03/2026,Item %d,%.2f\n", i, float64(i+1)))
	}
	profile := defaultProfile()

	result, err := ParsePreview(strings.NewReader(sb.String()), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 10 {
		t.Errorf("expected 10 preview rows, got %d", len(result.Rows))
	}
	if result.TotalRows != 20 {
		t.Errorf("expected TotalRows=20, got %d", result.TotalRows)
	}
}

func TestParsePreview_HeadersOnlyWhenNoMapping(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Salary,1500.00
`
	profile := ImportProfile{
		CsvSeparator: ",",
		SkipRows:     0,
		// No column mappings set
	}

	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
	if len(result.Rows) != 0 {
		t.Errorf("expected 0 rows when no mapping, got %d", len(result.Rows))
	}
	if result.TotalRows != 1 {
		t.Errorf("expected TotalRows=1, got %d", result.TotalRows)
	}
}

func TestParsePreview_SplitColumns(t *testing.T) {
	csv := `Date,Description,Credit,Debit
01/03/2026,Salary,1500.00,
02/03/2026,Grocery,,45.30
`
	profile := ImportProfile{
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}

	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result.Rows))
	}

	// Credit row → income, positive amount
	if result.Rows[0].Type != "income" {
		t.Errorf("row 0: expected type=income, got %s", result.Rows[0].Type)
	}
	if result.Rows[0].Amount != 1500.00 {
		t.Errorf("row 0: expected amount=1500.00, got %f", result.Rows[0].Amount)
	}

	// Debit row → expense, negative amount
	if result.Rows[1].Type != "expense" {
		t.Errorf("row 1: expected type=expense, got %s", result.Rows[1].Type)
	}
	if result.Rows[1].Amount != -45.30 {
		t.Errorf("row 1: expected amount=-45.30, got %f", result.Rows[1].Amount)
	}
}

func TestParsePreview_SplitBothPopulated(t *testing.T) {
	csv := `Date,Description,Credit,Debit
01/03/2026,Weird,100.00,50.00
`
	profile := ImportProfile{
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}

	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rows[0].Error == "" {
		t.Error("expected error when both credit and debit are populated")
	}
}

func TestParsePreview_SplitBothEmpty(t *testing.T) {
	csv := `Date,Description,Credit,Debit
01/03/2026,Empty,,
`
	profile := ImportProfile{
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}

	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rows[0].Error == "" {
		t.Error("expected error when both credit and debit are empty")
	}
}

func TestParsePreview_SkipRows(t *testing.T) {
	csv := `Preamble line
Date,Description,Amount
01/03/2026,Test,50.00
`
	profile := defaultProfile()
	profile.SkipRows = 1

	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
	if result.Rows[0].Amount != 50.00 {
		t.Errorf("expected amount=50.00, got %f", result.Rows[0].Amount)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && go test ./internal/csvimport/ -run TestParsePreview -v`
Expected: Compilation error — `ParsePreview` doesn't exist.

**Step 3: Write implementation**

Add to `internal/csvimport/parser.go`:

```go
// PreviewResult holds the result of a CSV preview parse.
type PreviewResult struct {
	Headers   []string   `json:"headers"`
	Rows      []ParsedRow `json:"rows"`
	TotalRows int        `json:"totalRows"`
}

// ParsePreview reads a CSV, extracts headers, and optionally parses up to 10
// rows using the profile's column mappings. It does NOT do category matching
// or duplicate detection. If column mappings (DateColumn, etc.) are empty,
// only headers and totalRows are returned.
func ParsePreview(r io.Reader, profile ImportProfile) (PreviewResult, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	sep := profile.CsvSeparator
	if sep == "" {
		sep = ","
	}
	if len(sep) > 0 {
		reader.Comma = rune(sep[0])
	}

	allRows, err := reader.ReadAll()
	if err != nil {
		return PreviewResult{}, fmt.Errorf("error reading CSV: %w", err)
	}

	skip := profile.SkipRows
	if skip >= len(allRows) {
		return PreviewResult{}, nil
	}
	allRows = allRows[skip:]

	if len(allRows) == 0 {
		return PreviewResult{}, nil
	}

	// Extract headers
	header := allRows[0]
	headers := make([]string, len(header))
	for i, h := range header {
		headers[i] = strings.TrimSpace(h)
	}

	dataRows := allRows[1:]
	totalRows := len(dataRows)

	// If no column mappings, return headers only
	if profile.DateColumn == "" && profile.DescriptionColumn == "" &&
		profile.AmountColumn == "" && profile.CreditColumn == "" && profile.DebitColumn == "" {
		return PreviewResult{Headers: headers, TotalRows: totalRows}, nil
	}

	// Build column index
	colIndex := make(map[string]int, len(headers))
	for i, h := range headers {
		colIndex[h] = i
	}

	// Resolve amount mode
	mode := profile.AmountMode
	if mode == "" {
		mode = "single"
	}

	// Cap preview at 10 rows
	previewLimit := 10
	if len(dataRows) < previewLimit {
		previewLimit = len(dataRows)
	}

	result := make([]ParsedRow, 0, previewLimit)
	for i := 0; i < previewLimit; i++ {
		row := dataRows[i]
		rowNum := skip + 1 + i + 1
		parsed := ParsedRow{RowNumber: rowNum}

		// Parse date
		if idx, ok := colIndex[profile.DateColumn]; ok && idx < len(row) {
			rawDate := strings.TrimSpace(row[idx])
			t, err := time.Parse(profile.DateFormat, rawDate)
			if err != nil {
				parsed.Error = fmt.Sprintf("invalid date %q: %v", rawDate, err)
				result = append(result, parsed)
				continue
			}
			parsed.Date = t.Format("2006-01-02")
		} else if profile.DateColumn != "" {
			parsed.Error = fmt.Sprintf("date column %q not found", profile.DateColumn)
			result = append(result, parsed)
			continue
		}

		// Parse description
		if idx, ok := colIndex[profile.DescriptionColumn]; ok && idx < len(row) {
			parsed.Description = strings.TrimSpace(row[idx])
		}

		// Parse amount based on mode
		amount, amtType, amtErr := resolveAmount(row, colIndex, mode, profile)
		if amtErr != "" {
			parsed.Error = amtErr
			result = append(result, parsed)
			continue
		}
		parsed.Amount = amount
		parsed.Type = amtType

		result = append(result, parsed)
	}

	return PreviewResult{Headers: headers, Rows: result, TotalRows: totalRows}, nil
}

// resolveAmount extracts the amount and transaction type from a CSV row
// based on the amount mode (single or split).
func resolveAmount(row []string, colIndex map[string]int, mode string, profile ImportProfile) (float64, string, string) {
	switch mode {
	case "split":
		creditIdx, creditOk := colIndex[profile.CreditColumn]
		debitIdx, debitOk := colIndex[profile.DebitColumn]

		var creditRaw, debitRaw string
		if creditOk && creditIdx < len(row) {
			creditRaw = strings.TrimSpace(row[creditIdx])
		}
		if debitOk && debitIdx < len(row) {
			debitRaw = strings.TrimSpace(row[debitIdx])
		}

		hasCredit := creditRaw != ""
		hasDebit := debitRaw != ""

		if hasCredit && hasDebit {
			return 0, "", "both credit and debit have values"
		}
		if !hasCredit && !hasDebit {
			return 0, "", "no amount found"
		}

		if hasCredit {
			amt, err := parseAmount(creditRaw)
			if err != nil {
				return 0, "", fmt.Sprintf("invalid credit amount %q: %v", creditRaw, err)
			}
			return amt, "income", ""
		}
		// hasDebit
		amt, err := parseAmount(debitRaw)
		if err != nil {
			return 0, "", fmt.Sprintf("invalid debit amount %q: %v", debitRaw, err)
		}
		return -math.Abs(amt), "expense", ""

	default: // "single"
		amtIdx, ok := colIndex[profile.AmountColumn]
		if !ok {
			return 0, "", fmt.Sprintf("amount column %q not found", profile.AmountColumn)
		}
		if amtIdx >= len(row) {
			return 0, "", "row has fewer columns than expected"
		}
		rawAmount := strings.TrimSpace(row[amtIdx])
		amount, err := parseAmount(rawAmount)
		if err != nil {
			return 0, "", fmt.Sprintf("invalid amount %q: %v", rawAmount, err)
		}
		if amount >= 0 {
			return amount, "income", ""
		}
		return amount, "expense", ""
	}
}
```

Also refactor the existing `Parse()` function to use `resolveAmount()` instead of its inline amount logic (lines 81-136). Replace the `amtIdx` lookup and the amount parsing + type determination block with a call to `resolveAmount`. The `Parse()` function needs to resolve the amount mode from the profile and build `colIndex` (which it already does). Replace:

```go
// Old lines 81-84 (amtIdx lookup) and 121-136 (amount parsing + type)
```

With usage of `resolveAmount(row, colIndex, mode, profile)` following the same pattern as in `ParsePreview`.

**Step 4: Run all parser tests**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && go test ./internal/csvimport/ -run "TestParse" -v`
Expected: ALL tests pass — both new `ParsePreview` tests and existing `Parse` tests.

**Step 5: Commit**

```bash
git add internal/csvimport/parser.go internal/csvimport/parser_test.go
git commit -m "feat(csvimport): add ParsePreview and split credit/debit support"
```

---

### Task 3: Update existing Parse() to support split columns

**Files:**
- Modify: `internal/csvimport/parser.go:34-151` (the `Parse` function)
- Modify: `internal/csvimport/parser_test.go`

**Step 1: Write the failing test**

Add to `internal/csvimport/parser_test.go`:

```go
func TestParse_SplitColumns(t *testing.T) {
	csv := `Date,Description,Credit,Debit
01/03/2026,Salary,1500.00,
02/03/2026,Grocery,,45.30
`
	profile := ImportProfile{
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}

	rows, err := Parse(strings.NewReader(csv), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].Type != "income" || rows[0].Amount != 1500.00 {
		t.Errorf("row 0: expected income/1500.00, got %s/%f", rows[0].Type, rows[0].Amount)
	}
	if rows[1].Type != "expense" || rows[1].Amount != -45.30 {
		t.Errorf("row 1: expected expense/-45.30, got %s/%f", rows[1].Type, rows[1].Amount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && go test ./internal/csvimport/ -run TestParse_SplitColumns -v`
Expected: FAIL — `Parse()` still requires `AmountColumn` and doesn't know about split mode.

**Step 3: Refactor Parse() to use resolveAmount**

In `internal/csvimport/parser.go`, modify `Parse()`:

1. Replace the `amtIdx` column validation (lines 81-84) with mode-aware validation:

```go
mode := profile.AmountMode
if mode == "" {
	mode = "single"
}

// Validate required columns based on mode
switch mode {
case "single":
	if _, ok := colIndex[profile.AmountColumn]; !ok {
		return nil, fmt.Errorf("required column %q not found in headers", profile.AmountColumn)
	}
case "split":
	if _, ok := colIndex[profile.CreditColumn]; !ok {
		return nil, fmt.Errorf("required column %q not found in headers", profile.CreditColumn)
	}
	if _, ok := colIndex[profile.DebitColumn]; !ok {
		return nil, fmt.Errorf("required column %q not found in headers", profile.DebitColumn)
	}
}
```

2. Replace the amount parsing and type determination block (lines 102, 121-136) with:

```go
// Remove: amtIdx bounds check from the "if dateIdx >= len(row) || ..." line
// Replace the amount parsing + type block with:
amount, amtType, amtErr := resolveAmount(row, colIndex, mode, profile)
if amtErr != "" {
	parsed.Error = amtErr
	result = append(result, parsed)
	continue
}
parsed.Amount = amount
parsed.Type = amtType
```

**Step 4: Run all tests**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && go test ./internal/csvimport/ -v`
Expected: ALL tests pass.

**Step 5: Commit**

```bash
git add internal/csvimport/parser.go internal/csvimport/parser_test.go
git commit -m "refactor(csvimport): use resolveAmount in Parse for split column support"
```

---

### Task 4: Add preview API endpoint

**Files:**
- Create: `app/router/handlers/csvimport/preview.go`
- Modify: `app/router/api_v0.go:607-611` (add route constant)
- Modify: `app/router/api_v0.go:709-719` (register route)

**Step 1: Create the preview handler**

Create `app/router/handlers/csvimport/preview.go`:

```go
package csvimport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/andresbott/etna/internal/csvimport"
)

func (h *ImportHandler) PreviewCSV() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, fmt.Sprintf("unable to parse multipart form: %s", err.Error()), http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to get uploaded file: %s", err.Error()), http.StatusBadRequest)
			return
		}
		defer file.Close()

		skipRows := 0
		if v := r.FormValue("skipRows"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				skipRows = n
			}
		}

		profile := csvimport.ImportProfile{
			CsvSeparator:      r.FormValue("csvSeparator"),
			SkipRows:          skipRows,
			DateColumn:        r.FormValue("dateColumn"),
			DateFormat:        r.FormValue("dateFormat"),
			DescriptionColumn: r.FormValue("descriptionColumn"),
			AmountMode:        r.FormValue("amountMode"),
			AmountColumn:      r.FormValue("amountColumn"),
			CreditColumn:      r.FormValue("creditColumn"),
			DebitColumn:       r.FormValue("debitColumn"),
		}

		result, err := csvimport.ParsePreview(file, profile)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to preview CSV: %s", err.Error()), http.StatusBadRequest)
			return
		}

		respJSON, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}
```

**Step 2: Register the route**

In `app/router/api_v0.go`:

Add constant (after line 610):
```go
const importPreviewPath = "/import/preview"
```

Add route registration (after the parse route, around line 719):
```go
r.Path(importPreviewPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if _, err := sessionauth.CtxGetUserData(r); err != nil {
		http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	importHndlr.PreviewCSV().ServeHTTP(w, r)
})
```

**Step 3: Verify it compiles**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && go build ./...`
Expected: No errors.

**Step 4: Commit**

```bash
git add app/router/handlers/csvimport/preview.go app/router/api_v0.go
git commit -m "feat(csvimport): add POST /import/preview endpoint"
```

---

### Task 5: Update TypeScript types and API client

**Files:**
- Modify: `webui/src/types/csvimport.ts`
- Modify: `webui/src/lib/api/CsvImport.ts`

**Step 1: Update types**

In `webui/src/types/csvimport.ts`, add `amountMode`, `creditColumn`, `debitColumn` to `ImportProfile`:

```typescript
export interface ImportProfile {
  id: number
  name: string
  csvSeparator: string
  skipRows: number
  dateColumn: string
  dateFormat: string
  descriptionColumn: string
  amountColumn: string
  amountMode: 'single' | 'split'
  creditColumn: string
  debitColumn: string
}

export interface PreviewResult {
  headers: string[]
  rows: ParsedRow[]
  totalRows: number
}
```

**Step 2: Add preview API function**

In `webui/src/lib/api/CsvImport.ts`, add:

```typescript
export const previewCSV = (file: File, config: {
  csvSeparator: string
  skipRows: number
  dateColumn?: string
  dateFormat?: string
  descriptionColumn?: string
  amountMode?: string
  amountColumn?: string
  creditColumn?: string
  debitColumn?: string
}) => {
  const form = new FormData()
  form.append('file', file)
  form.append('csvSeparator', config.csvSeparator)
  form.append('skipRows', String(config.skipRows))
  if (config.dateColumn) form.append('dateColumn', config.dateColumn)
  if (config.dateFormat) form.append('dateFormat', config.dateFormat)
  if (config.descriptionColumn) form.append('descriptionColumn', config.descriptionColumn)
  if (config.amountMode) form.append('amountMode', config.amountMode)
  if (config.amountColumn) form.append('amountColumn', config.amountColumn)
  if (config.creditColumn) form.append('creditColumn', config.creditColumn)
  if (config.debitColumn) form.append('debitColumn', config.debitColumn)
  return apiClient.post<PreviewResult>('/import/preview', form, {
    headers: { 'Content-Type': 'multipart/form-data' }
  }).then(r => r.data)
}
```

**Step 3: Verify frontend builds**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance/webui && npm run build`
Expected: Build succeeds (there may be warnings about unused imports in existing code, but no errors).

**Step 4: Commit**

```bash
git add webui/src/types/csvimport.ts webui/src/lib/api/CsvImport.ts
git commit -m "feat(csvimport): add preview API client and split-column types"
```

---

### Task 6: Replace profile dialog with interactive wizard

**Files:**
- Modify: `webui/src/views/csvimport/CsvImportProfileView.vue` (major rewrite of the dialog)

**Step 1: Rewrite the profile dialog**

Replace the `<Dialog>` section (lines 261-359) and the form state/logic in `<script setup>` with the wizard. Key changes:

**Script additions** — add these refs and functions:

```javascript
import { ref, onMounted, watch } from 'vue'
import FileUpload from 'primevue/fileupload'
import RadioButton from 'primevue/radiobutton'
import { previewCSV } from '@/lib/api/CsvImport'

// Wizard state
const wizardStep = ref(1)
const sampleFile = ref(null)
const detectedHeaders = ref([])
const previewRows = ref([])
const previewTotalRows = ref(0)
const isLoadingPreview = ref(false)

// Form state — add to existing:
const formAmountMode = ref('single')
const formCreditColumn = ref('')
const formDebitColumn = ref('')

// Header options for dropdowns (computed from detectedHeaders)
const headerOptions = computed(() =>
  detectedHeaders.value.map(h => ({ label: h, value: h }))
)
```

**Step 1 wizard template** — file upload + separator + skip rows + "Load Headers" button:

When "Load Headers" is clicked, call `previewCSV(file, { csvSeparator, skipRows })` — this returns headers and totalRows. Store them, advance to step 2.

**Step 2 wizard template** — column mapping dropdowns + amount mode radio + preview table:

- Date Column: `<Select>` with `headerOptions`
- Date Format: `<Select>` with `dateFormatOptions` (existing)
- Description Column: `<Select>` with `headerOptions`
- Amount Mode: `<RadioButton>` toggle — "Single column" / "Split credit/debit"
- Conditionally show Amount Column dropdown (single) OR Credit + Debit Column dropdowns (split)
- Preview table: `<DataTable>` with columns: Row#, Date, Description, Amount, Type, Error

**Preview refresh** — use a `watch` with debounce on the mapping fields. When any mapping changes, call `previewCSV(file, allCurrentSettings)` and update `previewRows`:

```javascript
let previewTimeout = null
const refreshPreview = () => {
  if (previewTimeout) clearTimeout(previewTimeout)
  previewTimeout = setTimeout(async () => {
    if (!sampleFile.value) return
    // Need at least one mapping to preview
    if (!formDateColumn.value && !formDescriptionColumn.value) return
    isLoadingPreview.value = true
    try {
      const result = await previewCSV(sampleFile.value, {
        csvSeparator: formCsvSeparator.value,
        skipRows: formSkipRows.value ?? 0,
        dateColumn: formDateColumn.value,
        dateFormat: formDateFormat.value,
        descriptionColumn: formDescriptionColumn.value,
        amountMode: formAmountMode.value,
        amountColumn: formAmountMode.value === 'single' ? formAmountColumn.value : undefined,
        creditColumn: formAmountMode.value === 'split' ? formCreditColumn.value : undefined,
        debitColumn: formAmountMode.value === 'split' ? formDebitColumn.value : undefined,
      })
      previewRows.value = result.rows
      previewTotalRows.value = result.totalRows
    } catch (e) {
      // Show error toast
    } finally {
      isLoadingPreview.value = false
    }
  }, 500)
}

watch([formDateColumn, formDateFormat, formDescriptionColumn, formAmountMode,
       formAmountColumn, formCreditColumn, formDebitColumn], refreshPreview)
```

**Save profile** — update `handleSaveProfile` to include new fields in payload:

```javascript
const payload = {
  name: formName.value.trim(),
  csvSeparator: formCsvSeparator.value,
  skipRows: formSkipRows.value ?? 0,
  dateColumn: formDateColumn.value,
  dateFormat: formDateFormat.value,
  descriptionColumn: formDescriptionColumn.value,
  amountMode: formAmountMode.value,
  amountColumn: formAmountMode.value === 'single' ? formAmountColumn.value.trim() : '',
  creditColumn: formAmountMode.value === 'split' ? formCreditColumn.value.trim() : '',
  debitColumn: formAmountMode.value === 'split' ? formDebitColumn.value.trim() : '',
}
```

Update validation in `handleSaveProfile` to check mode-dependent fields.

**Edit existing profile** — `openEditDialog` should set all new fields, start at `wizardStep = 2`, with no sample file. The dropdowns won't be populated from headers (since there's no file), so when editing without a file, allow free-text InputText as fallback for column names. When a file IS uploaded, switch to Select dropdowns.

**Reset form** — update `resetForm` to include:
```javascript
formAmountMode.value = 'single'
formCreditColumn.value = ''
formDebitColumn.value = ''
wizardStep.value = 1
sampleFile.value = null
detectedHeaders.value = []
previewRows.value = []
```

**Step 2: Verify frontend builds**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance/webui && npm run build`
Expected: No errors.

**Step 3: Manual testing**

1. Start backend: `make run`
2. Open the CSV Import Profiles page
3. Click "New Profile" — should see step 1 with file upload
4. Upload a sample CSV, set separator, click "Load Headers"
5. Should advance to step 2 with dropdowns populated from CSV headers
6. Map columns, toggle between single/split mode
7. Preview table should update live as mappings change
8. Save and verify profile appears in the list
9. Edit existing profile — should open at step 2 with fields pre-filled

**Step 4: Commit**

```bash
git add webui/src/views/csvimport/CsvImportProfileView.vue
git commit -m "feat(csvimport): replace profile form with interactive wizard"
```

---

### Task 7: Update profile handler to accept new fields

**Files:**
- Modify: `app/router/handlers/csvimport/profile.go`

**Step 1: Check current handler**

Read `app/router/handlers/csvimport/profile.go` to see the create/update request structs. They need `amountMode`, `creditColumn`, `debitColumn` fields.

**Step 2: Add fields to request struct**

Add to the profile request struct (used for create/update):

```go
AmountMode   string `json:"amountMode"`
CreditColumn string `json:"creditColumn"`
DebitColumn  string `json:"debitColumn"`
```

Map these fields when constructing `csvimport.ImportProfile` in both create and update handlers.

**Step 3: Verify it compiles and existing tests pass**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && go build ./... && go test ./...`
Expected: Compiles and all tests pass.

**Step 4: Commit**

```bash
git add app/router/handlers/csvimport/profile.go
git commit -m "feat(csvimport): accept split-column fields in profile API"
```

---

### Task 8: Final integration test

**Files:** None (manual verification only)

**Step 1: Run full test suite**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance && go test ./...`
Expected: All tests pass.

**Step 2: Run frontend build**

Run: `cd /home/odo/.datos/edit/programacion/bumbu/etna-finance/webui && npm run build`
Expected: Build succeeds.

**Step 3: End-to-end manual test**

1. Start the app
2. Create a profile using the wizard with a sample CSV
3. Import transactions using the created profile — verify both single-column and split-column CSVs work
4. Verify existing profiles still work (backward compatibility)
