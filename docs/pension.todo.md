# Pension Account - Revaluation Transaction

## Problem

Pension accounts receive transfers (contributions), but their value changes over time due to investment strategies.
Using income/expense transactions to capture gains/losses pollutes category reports.

## Decision

Add a **Revaluation Transaction** type that:
- Adjusts account balance (gains/losses)
- Is excluded from income/expense category reports
- Works alongside existing transfers for contributions

## Design Details

### Backend changes

#### 1. New entry type in `internal/accounting/entry.go`
- Add `revaluationEntry` after `balanceStatusEntry`

#### 2. New transaction type in `internal/accounting/transaction.go`
- Add `RevaluationTransaction` after `BalanceStatusTransaction`

#### 3. New struct
```go
type Revaluation struct {
    Id           uint
    Description  string
    Notes        string
    Amount       float64    // positive = gain, negative = loss
    AccountID    uint
    Date         time.Time
    AttachmentID *uint
    baseTx
}
```

#### 4. CreateRevaluation
- Creates a single `dbEntry` with `revaluationEntry` type
- Similar to `CreateBalanceStatus` but the entry affects balance

#### 5. Add `revaluationEntry` to `balanceEntryTypes` in `report.go:172`
```go
var balanceEntryTypes = []entryType{
    incomeEntry, expenseEntry, transferInEntry, transferOutEntry,
    stockCashOutEntry, stockCashInEntry,
    revaluationEntry,
}
```

#### 6. Allowed account types
- Decide: new `PensionAccountType` or allow revaluations on existing types (e.g. `SavingsAccountType`)
- Need a separate `allowedRevaluationAccountTypes` list

### What stays untouched
- **Category reports**: only query `incomeEntry`/`expenseEntry` - revaluation is invisible
- **PriorPageBalance**: uses `balanceEntryTypes` - includes revaluation automatically
- **Balance charts**: uses `balanceEntryTypes` - includes revaluation automatically
- **Transfer logic**: completely separate

### Frontend
- Handler name: `revaluation` (backend) / `revaluation` (frontend operation)
- Simple form: pick account, enter date, enter amount (gain/loss), optional description
- Similar to BalanceStatus form

### UX: Amount vs Target
- **Delta-based** (Amount = +20): simpler, matches entry model
- **Target-based** (new balance = 520, system calculates delta): friendlier UX
- Option: frontend computes `delta = target - currentBalance`, backend always stores delta

## Open Questions
- [ ] New `PensionAccountType` or reuse existing account types?
- [ ] Which account types should allow revaluation transactions?
- [ ] Delta-based vs target-based UX (or both)?

## Implementation Checklist
- [ ] Add `revaluationEntry` to entry types
- [ ] Add `RevaluationTransaction` to transaction types
- [ ] Add `Revaluation` struct
- [ ] Implement `CreateRevaluation`
- [ ] Implement `UpdateRevaluation`
- [ ] Add `revaluationEntry` to `balanceEntryTypes`
- [ ] Define `allowedRevaluationAccountTypes`
- [ ] Add handler endpoint (REST)
- [ ] Add backup import/export support
- [ ] Add frontend form and operation
- [ ] Tests
