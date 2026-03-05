# Evaluation: `db_entries` as a Unified Model for Cash + Portfolio

## Context

The current model uses a single `db_entries` table for both cash transactions (income, expense, transfer) and stock operations (buy, sell, grant, transfer). The user is finding it hard to build portfolio management features on top of this.

---

## Current Model Assessment

### What works well

- **Cash operations are clean.** Income/expense/transfer entries use Amount + AccountID + CategoryID in a straightforward way. Aggregation is a simple `SUM(amount)` filtered by entry type and category.
- **Double-entry integrity is maintained.** Every transaction produces balanced entries across accounts.
- **The entry_type enum clearly separates concerns** at the data level (11 distinct types).

### Where the model breaks down for portfolio management

#### 1. Semantic overload on `Amount` and `Quantity`

The same columns mean completely different things depending on entry type:

| entry_type | Amount means | Quantity means |
|---|---|---|
| income/expense | cash value | unused |
| stockBuy (position) | stock value in instrument currency | shares |
| stockSell (position) | negative cost basis (computed) | shares sold |
| stockCashOut | negative cash paid | unused |
| stockGrant | FMV x quantity | shares |
| stockTransfer | 0 | shares |

This makes the table hard to reason about and query. You can't `SUM(amount)` across entry types without CASE statements to interpret what each amount means.

#### 2. No materialized positions — everything is derived from replay

There is no `positions` or `holdings` table. Current holdings and cost basis are computed by:
- **Backend** (`transaction.go:604-666`): Full chronological replay of all position transactions for an (account, instrument) pair on every sell.
- **Frontend** (`useHoldings.ts:40-200`): Reimplements the same averaging logic client-side, fetching raw entries paginated at 500/page over 20 years.

This means:
- Every stock sell triggers an O(n) replay of all prior transactions for that instrument
- The frontend duplicates backend logic and could diverge
- There's no server-side holdings endpoint — the API only returns raw entries

#### 3. Query complexity

`ListTransactions` (`transaction.go:2040-2159`) uses ~50 lines of `CASE/MAX/SUM` conditional SQL to reconstruct transaction shapes from flat entries. Each transaction type needs its own extraction logic because the same columns store different things.

#### 4. No lot tracking

The cost basis computation uses **weighted-average cost** (proportional allocation), not individual lot tracking. This prevents:
- Tax-lot optimization (specific identification for tax-efficient selling)
- True FIFO/LIFO cost basis methods
- Per-lot gain/loss reporting
- Lot-level holding period tracking (short-term vs long-term capital gains)

#### 5. No clean separation between cash leg and position leg

A stock buy creates 2 entries in the same table: one for the position change and one for the cash change. These are fundamentally different events (position movement vs cash movement) sharing a schema designed for neither perfectly.

#### 6. Currency handling gaps

`entry.go:55-57` explicitly warns that `sumEntries` mixes currencies. Stock operations frequently involve instruments priced in a different currency than the cash account. There's no exchange rate captured per entry.

---

## Is this model suitable for both? — Verdict

**For cash accounting: Yes.** The model is adequate and simple.

**For portfolio management: No, not as-is.** The fundamental issue is that `db_entries` is an **accounting journal** being asked to also serve as a **position ledger**. These have different query patterns:

| Need | Accounting query | Portfolio query |
|---|---|---|
| "Total income in category X" | `SUM(amount) WHERE category_id = X` | N/A |
| "Current AAPL holdings" | N/A | Replay all buy/sell/grant/transfer for AAPL |
| "Cost basis for AAPL" | N/A | Chronological replay with averaging |
| "Unrealized P&L" | N/A | (holdings x current price) - cost basis |
| "What did I sell at a loss?" | N/A | Compare sale proceeds vs allocated cost per lot |

The accounting journal pattern (append entries, aggregate with SUM) is fundamentally mismatched with portfolio queries (stateful replay, per-instrument position tracking, lot management).

---

## Recommended Direction

Separate the concerns into distinct tables while keeping `db_entries` for what it does well:

### Keep `db_entries` for cash-side accounting
- Income, expense, transfer entries stay as-is
- Stock cash legs (stockCashOut/stockCashIn) stay here — they are real cash movements
- Remove position-tracking entries (stockBuy, stockSell, stockGrant, stockTransfer entry types) from this table

### Add new portfolio tables

```
db_trades                          db_lots
├── id           PK                ├── id              PK
├── transaction_id  FK → txn       ├── trade_id        FK → db_trades (the buy/grant)
├── account_id   FK → accounts     ├── account_id      FK → accounts
├── instrument_id FK → instruments ├── instrument_id   FK → instruments
├── trade_type   (buy/sell/grant)  ├── open_date       time
├── quantity     float64           ├── quantity         float64 (remaining)
├── price_per_share float64        ├── original_qty    float64
├── total_amount float64           ├── cost_per_share  float64
├── fees         float64           ├── cost_basis      float64
├── currency     string            ├── status          (open/closed/partial)
├── fx_rate      float64           ├── closed_date     time (nullable)
├── date         time              └── created_at      time
├── created_at   time
└── updated_at   time              db_lot_disposals
                                   ├── id              PK
db_positions (materialized)        ├── lot_id          FK → db_lots
├── id           PK                ├── sell_trade_id   FK → db_trades (the sell)
├── account_id   FK → accounts     ├── quantity        float64
├── instrument_id FK → instruments ├── proceeds        float64
├── quantity     float64           ├── realized_gl     float64
├── cost_basis   float64           └── date            time
├── avg_cost     float64
├── updated_at   time
└── UNIQUE(account_id, instrument_id)
```

**Benefits:**
- `db_entries` stays clean for cash accounting
- `db_trades` captures stock operations with unambiguous fields (price_per_share, fees, fx_rate)
- `db_lots` enables FIFO/LIFO/specific-id cost basis methods
- `db_positions` eliminates the need for full replay on every query
- `db_lot_disposals` tracks which lots were sold in each sell, enabling per-lot gain/loss
- Queries become simple: `SELECT * FROM db_positions WHERE instrument_id = ?`

**The cash leg link:** A stock buy would create both a `db_trades` record AND a cash entry in `db_entries` (the stockCashOut), linked via `transaction_id`. This preserves the double-entry accounting integrity while giving portfolio queries their own clean model.
