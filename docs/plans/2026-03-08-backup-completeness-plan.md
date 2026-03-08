# Backup Completeness Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend the backup system to cover all domain data: instruments, price history, FX rates, CSV import profiles, category rules, and missing fields (Icon, ImportProfileID).

**Architecture:** The backup package exports/imports a zip of JSON files, one per entity type. We extend the existing V1 schema with new files and fields. Export/Import accept three stores (`accounting.Store`, `marketdata.Store`, `csvimport.Store`). Each store gets a `WipeData` method for restore.

**Tech Stack:** Go, GORM, archive/zip, JSON, go-bumbu/timeseries, go-cmp (tests)

---

### Task 1: Add WipeData to csvimport.Store

**Files:**
- Modify: `internal/csvimport/csvimport.go`
- Test: `internal/csvimport/profile_test.go` (or create a new `csvimport_test.go`)

**Step 1: Write the failing test**

Add a test in `internal/csvimport/` that creates a profile and a category rule, calls `WipeData`, then asserts both lists are empty.

```go
func TestWipeData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:wipeDb?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to open db: %v", err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.CreateProfile(t.Context(), ImportProfile{
		Name: "test", DateColumn: "date", DateFormat: "2006-01-02",
		DescriptionColumn: "desc", AmountColumn: "amount",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateCategoryRule(t.Context(), CategoryRule{
		Pattern: "grocery", CategoryID: 1, Position: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = store.WipeData(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	profiles, err := store.ListProfiles(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(profiles))
	}

	rules, err := store.ListCategoryRules(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/csvimport/ -run TestWipeData -v`
Expected: FAIL — `WipeData` not defined.

**Step 3: Write minimal implementation**

In `internal/csvimport/csvimport.go`, add:

```go
func (s *Store) WipeData(ctx context.Context) error {
	tables := []string{"db_category_rules", "db_import_profiles"}
	for _, table := range tables {
		if err := s.db.WithContext(ctx).Table(table).Where("1 = 1").Delete(nil).Error; err != nil {
			return fmt.Errorf("failed to delete data in table '%s': %w", table, err)
		}
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/csvimport/ -run TestWipeData -v`
Expected: PASS

**Step 5: Commit**

```
feat(csvimport): add WipeData method for backup restore
```

---

### Task 2: Add WipeData to marketdata.Store

**Files:**
- Modify: `internal/marketdata/marketdata.go`
- Test: create `internal/marketdata/marketdata_test.go` or add to existing test file

**Step 1: Write the failing test**

Create a test that creates an instrument, ingests a price and an FX rate, calls `WipeData`, then asserts everything is empty.

```go
func TestWipeData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:mdWipeDb?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.CreateInstrument(t.Context(), Instrument{
		Symbol: "AAPL", Name: "Apple", Currency: currency.USD,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = store.IngestPrice(t.Context(), "AAPL", time.Now(), 150.0)
	if err != nil {
		t.Fatal(err)
	}
	err = store.IngestRate(t.Context(), "USD", "EUR", time.Now(), 0.85)
	if err != nil {
		t.Fatal(err)
	}

	err = store.WipeData(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	instruments, err := store.ListInstruments(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 0 {
		t.Errorf("expected 0 instruments, got %d", len(instruments))
	}

	symbols, err := store.ListPriceSymbols()
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 0 {
		t.Errorf("expected 0 price symbols, got %d", len(symbols))
	}

	pairs, err := store.ListFXPairs()
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 0 {
		t.Errorf("expected 0 FX pairs, got %d", len(pairs))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/marketdata/ -run TestWipeData -v`
Expected: FAIL — `WipeData` not defined.

**Step 3: Write minimal implementation**

In `internal/marketdata/marketdata.go`, add:

```go
func (s *Store) WipeData(ctx context.Context) error {
	// Order matters: records reference policies, policies reference series, instruments standalone
	tables := []string{"db_records", "db_sampling_policies", "db_time_series", "db_instruments"}
	for _, table := range tables {
		if err := s.db.WithContext(ctx).Table(table).Where("1 = 1").Delete(nil).Error; err != nil {
			return fmt.Errorf("failed to delete data in table '%s': %w", table, err)
		}
	}
	return nil
}
```

Note: The timeseries library uses `db_records`, `db_sampling_policies`, and `db_time_series` tables (from `dbRecord`, `dbSamplingPolicy`, `dbTimeSeries` structs in `vendor/github.com/go-bumbu/timeseries/`). Instruments use `db_instruments` (see `internal/marketdata/instrument.go:33`).

**Step 4: Run test to verify it passes**

Run: `go test ./internal/marketdata/ -run TestWipeData -v`
Expected: PASS

**Step 5: Commit**

```
feat(marketdata): add WipeData method for backup restore
```

---

### Task 3: Extend V1 data types in dataV1.go

**Files:**
- Modify: `internal/backup/dataV1.go`

**Step 1: Update existing types and add new ones**

Update `accountProviderV1`, `accountV1`, `categoryV1` to include `Icon`. Update `accountV1` to include `ImportProfileID`. Add new consts and types for instruments, prices, FX rates, import profiles, and category rules.

Add to `internal/backup/dataV1.go`:

```go
// --- Updated fields ---
// accountProviderV1: add Icon string `json:"icon"`
// accountV1: add Icon string `json:"icon"`, ImportProfileID uint `json:"importProfileId"`
// categoryV1: add Icon string `json:"icon"`

// --- New file constants ---
const instrumentsFile = "instruments.json"
const priceHistoryFile = "price_history.json"
const fxRatesFile = "fx_rates.json"
const importProfilesFile = "import_profiles.json"
const categoryRulesFile = "category_rules.json"

// --- New types ---
type instrumentV1 struct {
	ID                   uint   `json:"id"`
	InstrumentProviderID uint   `json:"instrumentProviderId"`
	Symbol               string `json:"symbol"`
	Name                 string `json:"name"`
	Currency             string `json:"currency"`
}

type priceRecordV1 struct {
	Symbol string    `json:"symbol"`
	Time   time.Time `json:"time"`
	Price  float64   `json:"price"`
}

type fxRateRecordV1 struct {
	Main      string    `json:"main"`
	Secondary string    `json:"secondary"`
	Time      time.Time `json:"time"`
	Rate      float64   `json:"rate"`
}

type importProfileV1 struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	CsvSeparator      string `json:"csvSeparator"`
	SkipRows          int    `json:"skipRows"`
	DateColumn        string `json:"dateColumn"`
	DateFormat        string `json:"dateFormat"`
	DescriptionColumn string `json:"descriptionColumn"`
	AmountColumn      string `json:"amountColumn"`
	AmountMode        string `json:"amountMode"`
	CreditColumn      string `json:"creditColumn"`
	DebitColumn       string `json:"debitColumn"`
}

type categoryRuleV1 struct {
	ID         uint   `json:"id"`
	Pattern    string `json:"pattern"`
	IsRegex    bool   `json:"isRegex"`
	CategoryID uint   `json:"categoryId"`
	Position   int    `json:"position"`
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/backup/...`
Expected: SUCCESS

**Step 3: Commit**

```
feat(backup): extend V1 data types with new entities and missing fields
```

---

### Task 4: Update export functions

**Files:**
- Modify: `internal/backup/export.go`

**Step 1: Update function signatures**

Change `ExportToFile` and `export` to accept all three stores:

```go
func ExportToFile(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, zipFile string) error {
	err := verifyZipPath(zipFile)
	if err != nil {
		return err
	}
	return export(ctx, store, mdStore, csvStore, zipFile)
}

func export(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, fullPath string) error {
```

**Step 2: Update writeAccountProviders to include Icon**

```go
jsonData[i] = accountProviderV1{
	ID:          provider.ID,
	Name:        provider.Name,
	Description: provider.Description,
	Icon:        provider.Icon,
}
```

**Step 3: Update writeAccounts to include Icon and ImportProfileID**

```go
jsonData[i] = accountV1{
	ID:                acc.ID,
	AccountProviderID: acc.AccountProviderID,
	Name:              acc.Name,
	Description:       acc.Description,
	Icon:              acc.Icon,
	Currency:          acc.Currency.String(),
	Type:              acc.Type.String(),
	ImportProfileID:   acc.ImportProfileID,
}
```

**Step 4: Update writeCategories to include Icon**

```go
jsonData[i] = categoryV1{
	ID:          income.Id,
	ParentId:    income.ParentId,
	Name:        income.Name,
	Description: income.Description,
	Icon:        income.Icon,
}
```
(Same for expenses loop.)

**Step 5: Add new write functions**

```go
func writeInstruments(ctx context.Context, zw *zipWriter, mdStore *marketdata.Store) error {
	instruments, err := mdStore.ListInstruments(ctx)
	if err != nil {
		return err
	}
	jsonData := make([]instrumentV1, len(instruments))
	for i, inst := range instruments {
		jsonData[i] = instrumentV1{
			ID:                   inst.ID,
			InstrumentProviderID: inst.InstrumentProviderID,
			Symbol:               inst.Symbol,
			Name:                 inst.Name,
			Currency:             inst.Currency.String(),
		}
	}
	return zw.writeJsonFile(instrumentsFile, jsonData)
}

func writePriceHistory(ctx context.Context, zw *zipWriter, mdStore *marketdata.Store) error {
	symbols, err := mdStore.ListPriceSymbols()
	if err != nil {
		return err
	}
	var jsonData []priceRecordV1
	for _, symbol := range symbols {
		records, err := mdStore.PriceHistory(ctx, symbol, time.Time{}, time.Time{})
		if err != nil {
			return fmt.Errorf("failed to get price history for %s: %w", symbol, err)
		}
		for _, rec := range records {
			jsonData = append(jsonData, priceRecordV1{
				Symbol: rec.Symbol,
				Time:   rec.Time,
				Price:  rec.Price,
			})
		}
	}
	if jsonData == nil {
		jsonData = []priceRecordV1{}
	}
	return zw.writeJsonFile(priceHistoryFile, jsonData)
}

func writeFXRates(ctx context.Context, zw *zipWriter, mdStore *marketdata.Store) error {
	pairs, err := mdStore.ListFXPairs()
	if err != nil {
		return err
	}
	var jsonData []fxRateRecordV1
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "/", 2)
		if len(parts) != 2 {
			continue
		}
		records, err := mdStore.RateHistory(ctx, parts[0], parts[1], time.Time{}, time.Time{})
		if err != nil {
			return fmt.Errorf("failed to get FX history for %s: %w", pair, err)
		}
		for _, rec := range records {
			jsonData = append(jsonData, fxRateRecordV1{
				Main:      rec.Main,
				Secondary: rec.Secondary,
				Time:      rec.Time,
				Rate:      rec.Rate,
			})
		}
	}
	if jsonData == nil {
		jsonData = []fxRateRecordV1{}
	}
	return zw.writeJsonFile(fxRatesFile, jsonData)
}

func writeImportProfiles(ctx context.Context, zw *zipWriter, csvStore *csvimport.Store) error {
	profiles, err := csvStore.ListProfiles(ctx)
	if err != nil {
		return err
	}
	jsonData := make([]importProfileV1, len(profiles))
	for i, p := range profiles {
		jsonData[i] = importProfileV1{
			ID:                p.ID,
			Name:              p.Name,
			CsvSeparator:      p.CsvSeparator,
			SkipRows:          p.SkipRows,
			DateColumn:        p.DateColumn,
			DateFormat:        p.DateFormat,
			DescriptionColumn: p.DescriptionColumn,
			AmountColumn:      p.AmountColumn,
			AmountMode:        p.AmountMode,
			CreditColumn:      p.CreditColumn,
			DebitColumn:       p.DebitColumn,
		}
	}
	return zw.writeJsonFile(importProfilesFile, jsonData)
}

func writeCategoryRules(ctx context.Context, zw *zipWriter, csvStore *csvimport.Store) error {
	rules, err := csvStore.ListCategoryRules(ctx)
	if err != nil {
		return err
	}
	jsonData := make([]categoryRuleV1, len(rules))
	for i, r := range rules {
		jsonData[i] = categoryRuleV1{
			ID:         r.ID,
			Pattern:    r.Pattern,
			IsRegex:    r.IsRegex,
			CategoryID: r.CategoryID,
			Position:   r.Position,
		}
	}
	return zw.writeJsonFile(categoryRulesFile, jsonData)
}
```

**Step 6: Wire new write functions into export()**

After existing `writeTransactions` call, add:

```go
err = writeInstruments(ctx, zw, mdStore)
if err != nil {
	return err
}
err = writePriceHistory(ctx, zw, mdStore)
if err != nil {
	return err
}
err = writeFXRates(ctx, zw, mdStore)
if err != nil {
	return err
}
err = writeImportProfiles(ctx, zw, csvStore)
if err != nil {
	return err
}
err = writeCategoryRules(ctx, zw, csvStore)
if err != nil {
	return err
}
```

**Step 7: Verify it compiles**

Run: `go build ./internal/backup/...`
Expected: May fail because callers pass wrong number of args — that's OK, fixed in Task 7.

**Step 8: Commit**

```
feat(backup): export instruments, prices, FX rates, import profiles, category rules
```

---

### Task 5: Update import functions

**Files:**
- Modify: `internal/backup/import.go`

**Step 1: Update Import signature and add wipe calls**

```go
func Import(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, file string) error {
```

After schema version check, wipe all three stores:

```go
err = store.WipeData(ctx)
if err != nil {
	return err
}
err = mdStore.WipeData(ctx)
if err != nil {
	return err
}
err = csvStore.WipeData(ctx)
if err != nil {
	return err
}
```

**Step 2: Update importAccounts to restore Icon and ImportProfileID**

In the `importAccounts` function, update the provider creation:

```go
item := accounting.AccountProvider{Name: provider.Name, Description: provider.Description, Icon: provider.Icon}
```

Update the account creation:

```go
item := accounting.Account{Name: account.Name, Description: account.Description, Icon: account.Icon}
// ... existing fields ...
item.ImportProfileID = account.ImportProfileID
```

Note: `ImportProfileID` references a profile that may not exist yet (profiles are imported later). Since the field is just a uint FK stored on the account (not enforced by DB constraint), it can be set now. If you need the mapped profile ID, import profiles before accounts — but since we're doing a full wipe+restore and IDs in the backup are the original IDs, the profile ID mapping should still work. **However**, since IDs may be remapped on import, we need a `profilesMap` similar to `accountsMap`. This means import profiles must happen BEFORE accounts. Update the import order in `Import()`:

```go
// 1. Wipe all stores (done above)
// 2. Import profiles + get profilesMap
profilesMap, err := importProfiles(ctx, csvStore, r)
if err != nil {
	return err
}
// 3. Import category rules (depends on category IDs, but rules have raw categoryID — import after categories)
// 4. Import accounts (depends on providers + profilesMap)
accountsMap, err := importAccounts(ctx, store, r, profilesMap)
if err != nil {
	return err
}
// 5. Import categories
inMap, exMap, err := importCategories(ctx, store, r)
if err != nil {
	return err
}
// 6. Import instruments + get instrumentsMap
instrumentsMap, err := importInstruments(ctx, mdStore, r)
if err != nil {
	return err
}
// 7. Import transactions (needs accountsMap, inMap, exMap; instrument IDs in transactions reference original IDs — need instrumentsMap)
err = importTransactions(ctx, store, r, accountsMap, inMap, exMap, instrumentsMap)
if err != nil {
	return err
}
// 8. Import price history + FX rates
err = importPriceHistory(ctx, mdStore, r)
if err != nil {
	return err
}
err = importFXRates(ctx, mdStore, r)
if err != nil {
	return err
}
// 9. Import category rules (needs category ID mapping)
err = importCategoryRules(ctx, csvStore, r, inMap, exMap)
if err != nil {
	return err
}
```

**Step 3: Update importAccounts to accept profilesMap**

```go
func importAccounts(ctx context.Context, store *accounting.Store, r *zip.ReadCloser, profilesMap map[uint]uint) (map[uint]uint, error) {
```

When creating accounts, map the ImportProfileID:

```go
item.ImportProfileID = profilesMap[account.ImportProfileID]
```

**Step 4: Update importTransactions to accept instrumentsMap**

```go
func importTransactions(ctx context.Context, store *accounting.Store, r *zip.ReadCloser, accountsMap, incomeMap, expenseMap, instrumentsMap map[uint]uint) error {
```

In the stock transaction cases, map `InstrumentID`:

```go
// For StockBuy:
InstrumentID: instrumentsMap[tx.InstrumentID],
// Same for StockSell, StockGrant, StockTransfer
```

**Step 5: Add new import functions**

```go
func importInstruments(ctx context.Context, mdStore *marketdata.Store, r *zip.ReadCloser) (map[uint]uint, error) {
	instruments, err := loadV1Json[[]instrumentV1](r, instrumentsFile)
	if err != nil {
		return nil, err
	}
	instrumentsMap := map[uint]uint{}
	for _, inst := range instruments {
		cur, err := currency.ParseISO(inst.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to parse currency %s: %w", inst.Currency, err)
		}
		item := marketdata.Instrument{
			InstrumentProviderID: inst.InstrumentProviderID,
			Symbol:               inst.Symbol,
			Name:                 inst.Name,
			Currency:             cur,
		}
		newID, err := mdStore.CreateInstrument(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("failed to create instrument: %w", err)
		}
		instrumentsMap[inst.ID] = newID
	}
	return instrumentsMap, nil
}

func importPriceHistory(ctx context.Context, mdStore *marketdata.Store, r *zip.ReadCloser) error {
	records, err := loadV1Json[[]priceRecordV1](r, priceHistoryFile)
	if err != nil {
		return err
	}
	// Group by symbol for bulk ingest
	bySymbol := map[string][]marketdata.PricePoint{}
	for _, rec := range records {
		bySymbol[rec.Symbol] = append(bySymbol[rec.Symbol], marketdata.PricePoint{
			Time:  rec.Time,
			Price: rec.Price,
		})
	}
	for symbol, points := range bySymbol {
		if err := mdStore.IngestPricesBulk(ctx, symbol, points); err != nil {
			return fmt.Errorf("failed to ingest prices for %s: %w", symbol, err)
		}
	}
	return nil
}

func importFXRates(ctx context.Context, mdStore *marketdata.Store, r *zip.ReadCloser) error {
	records, err := loadV1Json[[]fxRateRecordV1](r, fxRatesFile)
	if err != nil {
		return err
	}
	// Group by pair for bulk ingest
	type pair struct{ main, secondary string }
	byPair := map[pair][]marketdata.RatePoint{}
	for _, rec := range records {
		key := pair{rec.Main, rec.Secondary}
		byPair[key] = append(byPair[key], marketdata.RatePoint{
			Time: rec.Time,
			Rate: rec.Rate,
		})
	}
	for p, points := range byPair {
		if err := mdStore.IngestRatesBulk(ctx, p.main, p.secondary, points); err != nil {
			return fmt.Errorf("failed to ingest FX rates for %s/%s: %w", p.main, p.secondary, err)
		}
	}
	return nil
}

func importProfiles(ctx context.Context, csvStore *csvimport.Store, r *zip.ReadCloser) (map[uint]uint, error) {
	profiles, err := loadV1Json[[]importProfileV1](r, importProfilesFile)
	if err != nil {
		return nil, err
	}
	profilesMap := map[uint]uint{}
	for _, p := range profiles {
		item := csvimport.ImportProfile{
			Name:              p.Name,
			CsvSeparator:      p.CsvSeparator,
			SkipRows:          p.SkipRows,
			DateColumn:        p.DateColumn,
			DateFormat:        p.DateFormat,
			DescriptionColumn: p.DescriptionColumn,
			AmountColumn:      p.AmountColumn,
			AmountMode:        p.AmountMode,
			CreditColumn:      p.CreditColumn,
			DebitColumn:       p.DebitColumn,
		}
		newID, err := csvStore.CreateProfile(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("failed to create import profile: %w", err)
		}
		profilesMap[p.ID] = newID
	}
	return profilesMap, nil
}

func importCategoryRules(ctx context.Context, csvStore *csvimport.Store, r *zip.ReadCloser, incomeMap, expenseMap map[uint]uint) error {
	rules, err := loadV1Json[[]categoryRuleV1](r, categoryRulesFile)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		// Try mapping category ID from income map first, then expense
		catID := incomeMap[rule.CategoryID]
		if catID == 0 {
			catID = expenseMap[rule.CategoryID]
		}
		item := csvimport.CategoryRule{
			Pattern:    rule.Pattern,
			IsRegex:    rule.IsRegex,
			CategoryID: catID,
			Position:   rule.Position,
		}
		_, err := csvStore.CreateCategoryRule(ctx, item)
		if err != nil {
			return fmt.Errorf("failed to create category rule: %w", err)
		}
	}
	return nil
}
```

**Step 6: Update loadV1Json type constraint**

The generic `loadV1Json` function has a type union constraint. Add the new types:

```go
func loadV1Json[T metaInfoV1 | []accountProviderV1 | []accountV1 | []categoryV1 | []TransactionV1 | []instrumentV1 | []priceRecordV1 | []fxRateRecordV1 | []importProfileV1 | []categoryRuleV1](r *zip.ReadCloser, fileName string) (T, error) {
```

**Step 7: Verify it compiles**

Run: `go build ./internal/backup/...`
Expected: May fail until callers updated.

**Step 8: Commit**

```
feat(backup): import instruments, prices, FX rates, import profiles, category rules
```

---

### Task 6: Update callers (handler, tasks, wiring)

**Files:**
- Modify: `app/router/handlers/backup/backup.go` — lines 20-23 (Handler struct), lines 169, 268, 306
- Modify: `app/tasks/backup.go` — lines 28-37 (BackupTaskCfg), lines 41-42 (NewBackupTaskFn), line 56
- Modify: `app/router/api_v0.go` — line 744-747 (backupApi wiring)
- Modify: `app/cmd/server.go` — line 287 (task registration)

**Step 1: Update Handler struct**

In `app/router/handlers/backup/backup.go`:

```go
type Handler struct {
	Destination string
	Store       *accounting.Store
	MdStore     *marketdata.Store
	CsvStore    *csvimport.Store
}
```

Update all calls to `backup.ExportToFile` and `backup.Import` to pass `h.MdStore, h.CsvStore`.

**Step 2: Update task functions**

In `app/tasks/backup.go`:

```go
type BackupTaskCfg struct {
	Store       *accounting.Store
	MdStore     *marketdata.Store
	CsvStore    *csvimport.Store
	Destination string
	Interval    time.Duration
	Logger      *slog.Logger
}

func NewBackupTaskFn(store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, destination string, l *slog.Logger) func(ctx context.Context) error {
	return newBackupFunc(store, mdStore, csvStore, destination, l)
}

func newBackupFunc(store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, destination string, l *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// ... same body but pass mdStore, csvStore to ExportToFile
		err := backup.ExportToFile(ctx, store, mdStore, csvStore, zipFile)
		// ...
	}
}
```

**Step 3: Update wiring in api_v0.go**

At `app/router/api_v0.go:744`:

```go
backupHndl := backup.Handler{
	Destination: h.backupDestination,
	Store:       h.finStore,
	MdStore:     h.marketStore,
	CsvStore:    h.csvImportStore,
}
```

**Step 4: Update wiring in server.go**

At `app/cmd/server.go:287`:

```go
runner.RegisterTask(tasks.NewBackupTaskFn(finStore, marketStore, csvImportStore, backupDest, l), tasks.BackupTaskName, 0)
```

**Step 5: Verify it compiles**

Run: `go build ./...`
Expected: SUCCESS

**Step 6: Commit**

```
feat(backup): update callers to pass all three stores
```

---

### Task 7: Update export test

**Files:**
- Modify: `internal/backup/export_test.go`

**Step 1: Update sampleData to include new entities**

The `sampleData` function needs to create instruments, prices, FX rates, import profiles, category rules, and set icons. Since the test currently only uses `*accounting.Store`, change it to also accept `*marketdata.Store` and `*csvimport.Store`.

Update the test setup and `sampleData` function to:
- Set icons on providers, accounts, categories
- Create instruments via `mdStore.CreateInstrument`
- Ingest prices via `mdStore.IngestPrice`
- Ingest FX rates via `mdStore.IngestRate`
- Create import profiles via `csvStore.CreateProfile`
- Create category rules via `csvStore.CreateCategoryRule`
- Set `ImportProfileID` on at least one account

**Step 2: Update backupPayload and readFromZip**

Add new fields to `backupPayload`:

```go
type backupPayload struct {
	Meta              metaInfoV1
	Providers         []accountProviderV1
	Accounts          []accountV1
	IncomeCategories  []categoryV1
	ExpenseCategories []categoryV1
	Transactions      []TransactionV1
	Instruments       []instrumentV1
	PriceHistory      []priceRecordV1
	FXRates           []fxRateRecordV1
	ImportProfiles    []importProfileV1
	CategoryRules     []categoryRuleV1
}
```

Update `readFromZip` to parse the new files in its switch statement.

**Step 3: Update expected data in TestExport**

Add expected values for new fields (icons, ImportProfileID) and new entity types. Update the `export()` call to pass all three stores.

**Step 4: Run test**

Run: `go test ./internal/backup/ -run TestExport -v`
Expected: PASS

**Step 5: Commit**

```
test(backup): extend export test with new entity types and fields
```

---

### Task 8: Update import test

**Files:**
- Modify: `internal/backup/import_test.go`
- Modify: `internal/backup/export_test.go` — uncomment `copyFile(target)` temporarily to regenerate `testdata/backup-v1.zip`

**Step 1: Regenerate testdata/backup-v1.zip**

In `export_test.go`, temporarily uncomment `copyFile(target)` at line 46, run the export test, then re-comment it. This creates a new fixture with all entity types.

Run: `go test ./internal/backup/ -run TestExport -v`

Copy the generated `backup.zip` to `testdata/backup-v1.zip`.

**Step 2: Update TestImportV1 setup**

The import test needs all three stores. Create `marketdata.Store` and `csvimport.Store` in the test setup, similar to the accounting store.

Update the `Import` call:

```go
err = Import(t.Context(), store, mdStore, csvStore, backupFile)
```

Update `sampleDataNoise` to also create noise data in the new stores.

**Step 3: Add assertion sub-tests**

Add new sub-tests after the existing ones:

```go
t.Run("assert instruments", func(t *testing.T) {
	instruments, err := mdStore.ListInstruments(t.Context())
	// assert expected instruments with symbol, name, currency
})

t.Run("assert price history", func(t *testing.T) {
	records, err := mdStore.PriceHistory(t.Context(), "AAPL", time.Time{}, time.Time{})
	// assert expected price records
})

t.Run("assert fx rates", func(t *testing.T) {
	records, err := mdStore.RateHistory(t.Context(), "USD", "EUR", time.Time{}, time.Time{})
	// assert expected rate records
})

t.Run("assert import profiles", func(t *testing.T) {
	profiles, err := csvStore.ListProfiles(t.Context())
	// assert expected profiles
})

t.Run("assert category rules", func(t *testing.T) {
	rules, err := csvStore.ListCategoryRules(t.Context())
	// assert expected rules with mapped category IDs
})
```

**Step 4: Update existing assertions for new fields**

In `assert accounts`, verify Icon fields on providers and accounts, and ImportProfileID.

In `assert categories`, verify Icon fields.

**Step 5: Run test**

Run: `go test ./internal/backup/ -run TestImportV1 -v`
Expected: PASS

**Step 6: Commit**

```
test(backup): extend import test with new entity types and fixture
```

---

### Task 9: Add round-trip test

**Files:**
- Modify: `internal/backup/export_test.go` (add new test function)

**Step 1: Write the round-trip test**

```go
func TestRoundTrip(t *testing.T) {
	// Setup: three stores (source)
	db1, _ := gorm.Open(sqlite.Open("file:rtSource?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	store1, _ := accounting.NewStore(db1, nil)
	mdStore1, _ := marketdata.NewStore(db1)
	csvStore1, _ := csvimport.NewStore(db1)

	// Populate with sample data
	sampleData(t, store1, mdStore1, csvStore1)

	// Export
	tmpdir := t.TempDir()
	target1 := filepath.Join(tmpdir, "export1.zip")
	err := export(t.Context(), store1, mdStore1, csvStore1, target1)
	if err != nil {
		t.Fatalf("export1 failed: %v", err)
	}

	// Setup: three stores (destination)
	db2, _ := gorm.Open(sqlite.Open("file:rtDest?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	store2, _ := accounting.NewStore(db2, nil)
	mdStore2, _ := marketdata.NewStore(db2)
	csvStore2, _ := csvimport.NewStore(db2)

	// Import into destination
	err = Import(t.Context(), store2, mdStore2, csvStore2, target1)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// Export again from destination
	target2 := filepath.Join(tmpdir, "export2.zip")
	err = export(t.Context(), store2, mdStore2, csvStore2, target2)
	if err != nil {
		t.Fatalf("export2 failed: %v", err)
	}

	// Compare both exports structurally
	got1, err := readFromZip(target1)
	if err != nil {
		t.Fatalf("readFromZip1 failed: %v", err)
	}
	got2, err := readFromZip(target2)
	if err != nil {
		t.Fatalf("readFromZip2 failed: %v", err)
	}

	// IDs will differ between exports, so ignore ID fields
	if diff := cmp.Diff(got1, got2,
		cmpopts.IgnoreFields(metaInfoV1{}, "Date"),
		cmpopts.SortSlices(func(a, b categoryV1) bool { return a.Name < b.Name }),
		cmpopts.SortSlices(func(a, b TransactionV1) bool { return a.Date.Before(b.Date) }),
		cmpopts.SortSlices(func(a, b priceRecordV1) bool { return a.Symbol < b.Symbol || (a.Symbol == b.Symbol && a.Time.Before(b.Time)) }),
		cmpopts.SortSlices(func(a, b fxRateRecordV1) bool { return a.Main < b.Main || (a.Main == b.Main && a.Time.Before(b.Time)) }),
	); diff != "" {
		t.Errorf("round-trip mismatch (-first +second):\n%s", diff)
	}
}
```

Note: The IDs will be remapped on import so direct ID comparison won't work. The comparison should focus on data fields (names, amounts, dates, symbols) and ignore or normalize IDs. Adjust `cmpopts.IgnoreFields` as needed for all ID fields across all V1 types.

**Step 2: Run test**

Run: `go test ./internal/backup/ -run TestRoundTrip -v`
Expected: PASS

**Step 3: Commit**

```
test(backup): add round-trip export-import-export test
```

---

### Task 10: Update handler test

**Files:**
- Modify: `app/router/handlers/backup/backup_test.go`

**Step 1: Review and update**

The handler test at `app/router/handlers/backup/backup_test.go` needs the Handler struct updated with the new store fields. Update test setup to create and pass `marketdata.Store` and `csvimport.Store`. The test likely mocks or uses in-memory stores — match the pattern.

**Step 2: Run test**

Run: `go test ./app/router/handlers/backup/ -v`
Expected: PASS

**Step 3: Run full test suite**

Run: `go test ./...`
Expected: ALL PASS

**Step 4: Commit**

```
test(backup): update handler test for new store parameters
```

---

### Task 11: Final verification

**Step 1: Run full test suite**

Run: `go test ./...`
Expected: ALL PASS

**Step 2: Run linter if available**

Run: `golangci-lint run ./...` (if configured)
Expected: No new issues

**Step 3: Final commit with any cleanup**

If any adjustments needed, commit them.
