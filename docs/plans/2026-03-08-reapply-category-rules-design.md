# Re-apply Category Rules to Existing Transactions

## Problem

When a user creates or modifies category matching rules, existing transactions are not updated. Transactions imported before the rule existed (or with outdated rule assignments) keep their old or missing categories.

## Solution

Add a "Re-apply Rules" flow that evaluates all category rules (in position order) against all existing transactions across all accounts, shows a preview of proposed changes, and lets the user select which transactions to update.

## Backend

### New endpoints

**`POST /api/v0/import/reapply-preview`**

No request body. Returns all transactions that match at least one rule.

Response shape:

```json
[
  {
    "transactionId": 42,
    "transactionType": "expense",
    "description": "GROCERY STORE",
    "date": "2025-12-01T00:00:00Z",
    "amount": 45.60,
    "accountId": 1,
    "accountName": "Checking",
    "currentCategoryId": 0,
    "currentCategoryName": "",
    "newCategoryId": 5,
    "newCategoryName": "Food > Groceries",
    "changed": true
  }
]
```

- Loads all category rules in position order
- Loads all transactions (income + expense) across all accounts
- Runs `matchCategory()` on each transaction description
- Excludes transactions that don't match any rule
- `changed` is true when `currentCategoryId != newCategoryId`

**`POST /api/v0/import/reapply-submit`**

Request body:

```json
[
  { "transactionId": 42, "transactionType": "expense", "newCategoryId": 5 }
]
```

- Updates each transaction's CategoryID
- Validates category type compatibility (income tx -> income category, expense tx -> expense category)
- Returns `{ "updated": 3 }`

### Store changes

**accounting store:**
- `ListAllTransactions(ctx)` — returns all income and expense entries with account name, without account filtering
- `UpdateTransactionCategory(ctx, id, txType, categoryID)` — updates CategoryID on a transaction with type validation

### Reused logic

- `matchCategory()` from `internal/csvimport/parser.go` — same rule matching logic used during CSV import
- Category tree loading and name resolution — same patterns as import handler

## Frontend

### CategoryRulesView changes

- Add "Re-apply Rules" button in toolbar next to "New Rule"
- Button navigates to `/import/reapply`

### New route

- `/import/reapply` -> `ReapplyRulesView.vue`

### ReapplyRulesView.vue

DataTable with checkboxes, similar to CSV import preview.

Columns: checkbox, date, description, amount, account name, current category, new category.

Behavior:
- On mount, calls reapply-preview endpoint; shows loading state
- Rows with `changed=true` are checked by default
- Rows with `changed=false` (already correct) are shown but unchecked, with reduced opacity
- Summary bar: "X transactions will be updated, Y already correct"
- "Apply Selected" button submits checked rows via reapply-submit
- Success toast with count, navigates back to category rules page

### API client

Add to `CsvImport.ts`:
- `reapplyPreview()` — POST to `/import/reapply-preview`
- `reapplySubmit(rows)` — POST to `/import/reapply-submit`

## Testing

### Backend tests

- **reapply-preview**: create transactions with no category, wrong category, and correct category; create rules; call preview; assert correct rows returned with correct `changed` flags
- **reapply-submit**: submit selection; verify DB updates; verify category type validation rejects mismatched types
- Follow existing patterns in `import_test.go`

### Frontend

No e2e tests planned (consistent with existing CSV import coverage).
