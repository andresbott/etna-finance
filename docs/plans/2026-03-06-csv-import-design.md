# CSV Import Feature — Design Document

Date: 2026-03-06

## Overview

Import bank CSV exports into etna-finance as income/expense transactions. The feature is scoped to cash transaction types (income, expense). Transfers are out of scope for v1.

## Components

### 1. Import Profiles

Stored in DB. Each profile defines how to map CSV columns to transaction fields.

**Table `db_import_profiles`:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint (PK) | Auto-increment |
| `name` | string | Human-readable name, e.g. "UBS Checking CSV" |
| `csv_separator` | string | Column delimiter: `,`, `;`, `\t` (default `,`) |
| `skip_rows` | int | Rows to skip before the header row (default `0`) |
| `date_column` | string | CSV header mapped to transaction date |
| `date_format` | string | Go time layout for parsing, e.g. `02.01.2006` |
| `description_column` | string | CSV header mapped to description |
| `amount_column` | string | CSV header mapped to amount |

**Account linkage:** `db_accounts` gets a nullable `import_profile_id` FK. Set via account edit UI.

**Type detection:** Amount sign determines transaction type — positive = income, negative = expense.

### 2. Category Matching Rules

Global rules stored in DB that auto-assign categories to imported transactions based on description matching.

**Table `db_category_rules`:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint (PK) | Auto-increment |
| `pattern` | string | The match pattern |
| `is_regex` | bool | `false` = case-insensitive substring, `true` = regex |
| `category_id` | uint (FK) | Target category |
| `position` | int | Evaluation order — first match wins |

**Matching behavior:**
- Rules evaluated in `position` order against the transaction description.
- Plain patterns: case-insensitive substring match.
- Regex patterns: full regex match against the description.
- First match wins.
- If the matched category type conflicts with the amount sign, the match is ignored (category left empty).
- If no rule matches, category is left empty for user to assign in preview.

### 3. Import Flow

Two-step backend process: parse then submit.

**Parse — `POST /api/v0/import/parse`**

Request: multipart form with CSV file + account ID.

Backend steps:
1. Look up the account and its linked import profile (400 if none).
2. Read CSV using profile's separator and skip_rows.
3. Map columns using profile's header mappings.
4. Parse dates using profile's date_format.
5. Parse amounts (handle comma-as-decimal).
6. Determine type: positive = income, negative = expense.
7. Run category matching rules (ordered by position, first match wins).
8. Detect duplicates: existing transaction in same account with same date + amount.
9. Return parsed rows with duplicate flags and matched categories.

Response:
```json
{
  "rows": [
    {
      "rowNumber": 1,
      "date": "2024-03-15",
      "description": "MIGROS ZURICH",
      "amount": -45.60,
      "type": "expense",
      "categoryId": 12,
      "isDuplicate": false
    }
  ]
}
```

Rows with parse errors include an `error` field instead of failing the whole request.

**Submit — `POST /api/v0/import/submit`**

Request: account ID + array of edited rows (user may have changed categories, descriptions, excluded rows).

Backend steps:
1. Validate each row (valid date, category type matches amount sign, category exists).
2. Create transactions in bulk within a single DB transaction (all-or-nothing).
3. Return count of created transactions.

### 4. API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v0/import/profiles` | List all import profiles |
| POST | `/api/v0/import/profiles` | Create import profile |
| PUT | `/api/v0/import/profiles/{id}` | Update import profile |
| DELETE | `/api/v0/import/profiles/{id}` | Delete import profile |
| GET | `/api/v0/import/category-rules` | List category rules (ordered) |
| POST | `/api/v0/import/category-rules` | Create category rule |
| PUT | `/api/v0/import/category-rules/{id}` | Update category rule |
| DELETE | `/api/v0/import/category-rules/{id}` | Delete category rule |
| POST | `/api/v0/import/parse` | Parse CSV with profile |
| POST | `/api/v0/import/submit` | Submit edited rows as transactions |

### 5. Frontend

**Import page** — new route: `/import?accountId=X`

Two states:

1. **Upload state:** Shows account info + file upload area. User picks CSV, clicks parse.

2. **Preview state:** Reuses the same transaction table component from the account view — same columns, balance calculation, styling. New (imported) rows are interleaved with existing transactions. Existing rows are read-only/dimmed. New rows have:
   - Checkbox to include/exclude (duplicates unchecked by default)
   - All fields editable via the same transaction edit dialog used elsewhere
   - Duplicate rows visually flagged
   - Error rows highlighted with message
   - Summary bar: "X new, Y duplicates, Z errors"
   - Submit button creates checked transactions, navigates back to account view.

**Entry point:** "Import CSV" button on the account transaction list view. Only visible if account has a linked import profile.

**Settings pages:**
- Import profiles: table with CRUD (builds on existing mock `CsvImportProfileView.vue`).
- Category rules: sortable table with drag-to-reorder, pattern field, regex toggle, category dropdown.
- Account edit form: new "Import Profile" dropdown.

## Out of Scope (v1)

- Transfer transaction detection/creation from CSV
- Multi-file import in a single session
- Scheduled/automatic imports
