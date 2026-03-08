# CSV Auto Template Configuration Design

## Problem

CSV import profiles must be configured manually by typing column names, separator, and date format. Users must know the exact header names in their CSV. Additionally, the current parser only supports a single amount column where sign determines income vs expense — but many bank exports use separate credit and debit columns.

## Solution

Replace the profile settings form with an interactive wizard. The user uploads a sample CSV, the system extracts headers, and the user maps columns via dropdowns. A live preview of the first 10 processed rows updates as mappings change. The sample CSV is throwaway — only used during the wizard session.

## Data Model Changes

Add three fields to `ImportProfile`:

- `AmountMode string` — `"single"` (default) or `"split"`
- `CreditColumn string` — header for credit/income column (split mode)
- `DebitColumn string` — header for debit/expense column (split mode)

Validation:
- `single` mode: `AmountColumn` required, `CreditColumn`/`DebitColumn` ignored
- `split` mode: `CreditColumn` and `DebitColumn` required, `AmountColumn` ignored
- Empty `AmountMode` treated as `"single"` for backward compatibility

## New Preview Endpoint

`POST /api/v0/import/preview` — multipart form.

Input fields: `file`, `csvSeparator`, `skipRows`, `dateColumn`, `dateFormat`, `descriptionColumn`, `amountMode`, `amountColumn`, `creditColumn`, `debitColumn`.

Response:
```json
{
  "headers": ["Fecha", "Concepto", "Cargo", "Abono"],
  "rows": [
    {
      "rowNumber": 1,
      "date": "2026-03-01",
      "description": "Salary",
      "amount": 1500.00,
      "type": "income",
      "error": ""
    }
  ],
  "totalRows": 45
}
```

- `headers` always returned — extracted after applying `skipRows` and `csvSeparator`
- `rows` only populated when column mappings are provided, capped at 10
- `totalRows` — total data rows excluding skipped/header rows

## Parser Changes

### Amount resolution in split mode

- Read both `CreditColumn` and `DebitColumn` per row
- Non-empty credit value -> income
- Non-empty debit value -> expense
- Both populated -> row error: "both credit and debit have values"
- Both empty -> row error: "no amount found"

### Preview function

Add `ParsePreview()` that skips category matching and duplicate detection. Caps output at 10 rows. Reuses existing `parseAmount()` and date parsing logic.

### Backward compatibility

Existing `Parse()` extended to handle split mode. When `AmountMode` is empty or `"single"`, behavior is unchanged.

## Frontend: Profile Wizard

Replaces the create/edit form in `CsvImportProfileView.vue`.

### Step 1: Upload & Basic Settings
- File upload input
- Separator dropdown (`,`, `;`, `\t`)
- Skip rows number input
- "Load Headers" button -> calls preview endpoint with file + separator + skipRows
- On response: stores headers, advances to step 2

### Step 2: Column Mapping & Preview
- Dropdowns populated from headers: Date Column, Date Format, Description Column
- Amount mode toggle: "Single column" vs "Split credit/debit"
  - Single: Amount Column dropdown
  - Split: Credit Column + Debit Column dropdowns
- Preview table (auto-refreshes on mapping change, debounced):
  - Columns: Row #, Date, Description, Amount, Type, Error
  - Up to 10 rows
- "Save Profile" button saves the mapping config via existing profile CRUD endpoint

### Editing existing profiles
Opens wizard at step 2 with fields pre-filled, no sample loaded. User can optionally upload a sample to re-test.

## Error Handling

- Empty CSV / no data after skip: `headers: []`, `rows: []`, `totalRows: 0`. Frontend shows "No data found" message.
- Column not found in headers: error per affected row.
- Non-numeric amount values: existing `parseAmount()` error handling applies.
- No migration needed — GORM auto-migrates new columns with zero values.

## Testing

- **Parser tests:** Split credit/debit cases (one filled, both filled, both empty). `ParsePreview()` caps at 10 rows, skips category/duplicate logic.
- **Profile validation tests:** Single mode requires `AmountColumn`, split mode requires `CreditColumn` + `DebitColumn`.
- **Preview endpoint test:** Upload sample CSV, call with mappings, verify response.
- **No new e2e tests** — manual testing for the UI wizard flow.
