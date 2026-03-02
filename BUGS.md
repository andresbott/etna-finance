# Known Bugs

---

## [BUG] Stock sell overstates cash balance by realized gain amount

**Discovered:** 2026-03-02
**Status:** Open
**Branch:** stocks

### Symptom

After selling stocks, the cash account balance shown in the overview is higher than the net amount entered by the user.

**Example:**
- Sell 5 shares @ 248.344, net amount entered = 1,280.00, fees = 0
- Expected cash reserve: **1,280.00**
- Actual cash reserve shown: **1,318.28**
- Discrepancy: **38.28**

### Root Cause

`CreateStockSell` in `internal/accounting/transaction.go` creates **two entries against the cash account**:

1. `stockCashInEntry` → `TotalAmount - fees` = 1,280.00 (the net proceeds)
2. `incomeEntry` → `realizedGainLoss` = `TotalAmount - fees - costBasis` = 38.28

`balanceEntryTypes` in `internal/accounting/report.go:171` includes **both** `stockCashInEntry` and `incomeEntry`, so they are summed together into the displayed cash balance → **1,280.00 + 38.28 = 1,318.28**.

The realized gain is double-counted: the net proceeds (1,280.00) already embed any gain over cost basis, yet the system also books the gain as a separate credit to the same cash account.

The 38.28 arises because:
- `TotalAmount` = the user-entered net amount (1,280.00) — see `doSave()` in `BuySellInstrumentDialog.vue:349`: `total = netAmount + fees`
- `costBasis` = 1,241.72 (from prior buy lot at 248.344 × 5)
- `realizedGainLoss` = 1,280.00 − 0 − 1,241.72 = **38.28** → incorrectly posted to `CashAccountID`

### Relevant Files

| File | Location | Notes |
|---|---|---|
| `internal/accounting/transaction.go` | `CreateStockSell`, around line 628–650 | Creates the double entry |
| `internal/accounting/report.go` | `balanceEntryTypes`, line 171 | Sums both entry types into cash balance |
| `webui/src/views/entries/dialogs/BuySellInstrumentDialog.vue` | `doSave()`, line 346–349 | `total = netAmount + fees` (net, not price×qty) |

### Intended Fix

The `incomeEntry` (realized gain/loss) should **not** be posted to `CashAccountID`. It should go to a separate P&L / equity account. The cash account should only receive the one `stockCashInEntry` for the net proceeds.

If no separate income account exists yet, the simplest correct fix is to remove the `incomeEntry` from the cash account entries entirely — the net cash effect is already fully captured by `stockCashInEntry`. Realized gain/loss tracking can be handled via the trades table separately.
