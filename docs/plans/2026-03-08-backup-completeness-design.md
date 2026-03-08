# Backup Completeness Design

**Date:** 2026-03-08
**Status:** Approved

## Problem

The backup system only covers a subset of domain data. Several entity types added since the original implementation are not included in backup/restore: instruments, price history, FX rates, CSV import profiles, category rules. Additionally, recently added fields (Icon on providers/accounts/categories, ImportProfileID on accounts) are not exported.

## Decisions

- Keep schema version `"1.0.0"` — no backward compatibility needed
- Accept all three stores directly as parameters (`accounting.Store`, `marketdata.Store`, `csvimport.Store`)
- Wipe ALL stores atomically before restore (same approach as today, broader scope)
- Keep both round-trip test and static fixture file

## New Zip Archive Files

| File | Type | Content |
|---|---|---|
| `instruments.json` | `[]instrumentV1` | ID, Symbol, Name, Currency, InstrumentProviderID |
| `price_history.json` | `[]priceRecordV1` | Symbol, Time, Price |
| `fx_rates.json` | `[]fxRateRecordV1` | Main, Secondary, Time, Rate |
| `import_profiles.json` | `[]importProfileV1` | All ImportProfile fields |
| `category_rules.json` | `[]categoryRuleV1` | All CategoryRule fields |

## Updated Existing Types

- `accountProviderV1` — add `Icon string`
- `accountV1` — add `Icon string`, `ImportProfileID uint`
- `categoryV1` — add `Icon string`

## Function Signatures

```go
func ExportToFile(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, zipFile string) error
func Import(ctx context.Context, store *accounting.Store, mdStore *marketdata.Store, csvStore *csvimport.Store, file string) error
```

## Callers to Update

- `app/router/handlers/backup/backup.go` — Handler struct gets MdStore and CsvStore fields
- `app/tasks/backup.go` — BackupTaskCfg and NewBackupTaskFn get extra stores
- Wherever the handler/task are wired up in router/server setup

## Import Order

1. Wipe all three stores
2. Account providers -> Accounts
3. Categories (income + expense)
4. Instruments
5. Transactions (rebuilds trades/lots/positions automatically)
6. Price history + FX rates
7. Import profiles + Category rules

## Wipe Strategy

Each store needs a WipeData method:

- `accounting.Store.WipeData()` — already exists
- `marketdata.Store.WipeData()` — new, deletes all instruments + price series + FX series
- `csvimport.Store.WipeData()` — new, deletes all import profiles + category rules

All three wipes happen before any data is restored. If any wipe fails, import returns early.

## Testing Strategy

### Export test (export_test.go)

- Extend `sampleData()` to create instruments, price history, FX rates, import profiles, category rules, and icons
- Extend `backupPayload` and `readFromZip()` to parse new JSON files
- Compare exported zip against expected data
- Regenerate static `testdata/backup-v1.zip` fixture

### Import test (import_test.go)

- Load updated fixture
- Add assertion sub-tests: instruments, price history, FX rates, import profiles, category rules
- Verify icons and ImportProfileID on existing entity assertions

### Round-trip test

- Create sample data -> export -> wipe -> import -> export again -> compare both exports structurally
- Catches asymmetries between export and import
