# Timeseries upstream follow-ups

Workarounds in etna's `internal/marketdata` that exist because the
`github.com/go-bumbu/timeseries` library is missing an API. The goal is to fix
the library **before merging** the `replace` directive away and pinning a real
release.

- **Library location:** `/home/bott/datos/edit/programacion-privado/bumbu/timeseries` (branch `optimize-ts`)
- **Wired in via:** `go.mod` `replace github.com/go-bumbu/timeseries => /home/bott/datos/edit/programacion-privado/bumbu/timeseries`
- **Library public API:** `store.go`, `series.go`, `record.go`, `field.go`, `maintain.go`, `unixmilli.go`

Each item below is self-contained — pick one up in its own session. Suggested
order: #1, #2, #3 (each removes a documented hazard), then #4. #5 and #6 are
etna-only cleanups, no library change (#6 complements #4 — see #4's note).

---

## 1. Add a bulk-wipe API (`Store.Wipe`) — ✅ DONE

**Done.** `Store.Wipe(ctx)` added upstream (`series.go`, beside `DropSeries`):
deletes records/fields/series via the model types in one transaction under
`s.mu.Lock`, so the table names stay tied to their `TableName()` methods.
Covered by `TestWipe` in `store_test.go`. etna's `WipeData`
(`marketdata.go`) now calls `s.store.Wipe(ctx)` and then deletes only its own
`db_instruments` table; the hardcoded internal-table loop and its hazard
comment are gone. Existing `TestWipeData` still passes.

**Problem (original).** `WipeData` reaches past the abstraction and hardcodes the
library's internal table names because the Store exposes no bulk-wipe API.

- `internal/marketdata/marketdata.go:77-89`

```go
// "records"/"fields"/"series" are go-bumbu/timeseries v0.2 internal table names — the Store
// exposes no bulk-wipe API, so we delete directly. Revisit if the library adds one or renames
// its tables ...
tables := []string{"records", "fields", "series", "db_instruments"}
```

A rename upstream silently breaks this — no compile error, no test failure
unless asserted. Used by backup/restore (`internal/backup/import.go`
`wipeStores`).

**Upstream fix.** Add `Store.Wipe(ctx) error` that truncates `records`,
`fields`, `series` in one transaction under the exclusive lock (`s.mu.Lock`).
Mirror the table list from the `dbRecord`/`dbField`/`dbSeries` `TableName()`
methods so a rename stays internally consistent.

**etna follow-up.** Replace the raw-table loop in `WipeData` with
`s.store.Wipe(ctx)`, then delete only etna's own `db_instruments` table.

---

## 2. Add an atomic record move (`Store.Move`) — ✅ DONE

**Done.** `Store.Move(ctx, series, oldTime, p Point)` added upstream
(`record.go`, beside `Write`/`Delete`): deletes every field at `oldTime` and
upserts `p` (at `p.Time`) in one transaction under the shared lock, so a crash
can't drop the record. It carries the new point's values (not just a
timestamp) because both call sites edit values while moving — `oldTime ==
p.Time` degenerates to a clean replace, a zero `oldTime` to a plain upsert.
Covered by `TestMove_Relocate` / `TestMove_ZeroTime`. etna's `EditPrice`
(`price.go`) and `UpdateRate` (`fx.go`) now route the time-change path through
`s.store.Move`; the duplicated delete+write warning comments are gone. Existing
`TestEditPrice` / `TestUpdateRate` still pass.

**Problem (original).** Moving a record to a new timestamp is implemented as a
non-atomic delete + write in two places, because the library wraps each op in
its own transaction and exposes no way to span two. Both sites carry the same
warning comment.

- `internal/marketdata/price.go:146-160` (`EditPrice`)
- `internal/marketdata/fx.go:200-208` (`UpdateRate`)

```go
// Moving a record to a new time is a delete + write: the timeseries store has no
// transaction spanning both, so a crash between them can drop the record. ...
// revisit if the store gains atomic move support.
```

**Upstream fix.** Add `Store.Move(ctx, series string, oldTime, newTime time.Time) error`
that performs the delete-old + upsert-new in a single transaction under the
lock. (Alternatively a more general `UpsertPoint`/`UpdatePoint`.) The library
already holds an `RWMutex` and uses transactions internally, so it is well
positioned to do this safely.

**etna follow-up.** Replace the delete+write sequences in `EditPrice` and
`UpdateRate` with the new atomic call and drop the duplicated warning comments.

---

## 3. Add "latest record" + "count" APIs — ✅ DONE

**Done.** Three read methods added upstream (`record.go`, beside `FieldAt`/`At`,
all under `RLock`):
- `Latest(ctx, series) (Point, bool, error)` — point at the series' max timestamp
  with the real time preserved (newest row, then its fields; ~5 rows, not the
  whole series). `found=false` for an empty series, `ErrSeriesNotFound` for an
  unknown one.
- `LatestField(ctx, series, field) (Sample, bool, error)` — newest `(time, value)`
  for one field, a single-row read.
- `Count(ctx, series) (int, error)` — server-side `COUNT(DISTINCT time)`, no row
  transfer.

Covered by `TestLatest` / `TestLatestField` / `TestCount`. etna rewired:
`LatestPrice` → `Latest`, `LatestRate` → `LatestField`, `Stats` → `Count` (which
also dropped `Stats`'s hardcoded `"close"`/`fxField` names and its `time`
import). Existing `TestLatestPrice` / `TestLatestRate` / `TestStats` still pass.

**Problem (original).** There is no way to get the most-recent *(time, value)* pair, or a
row count, without loading the entire series. `FieldAt` returns only the value
(not its source timestamp) and `At` overwrites `Point.Time` with the query
time — so callers that need the real timestamp must scan everything.

- `internal/marketdata/price.go:108-124` (`LatestPrice`) — `Range(zero, zero)` then `[len-1]`
- `internal/marketdata/fx.go:153-169` (`LatestRate`) — `FieldRange(zero, zero)` then `[len-1]`
- `internal/marketdata/stats.go:30-42` (`Stats`) — `FieldRange(zero, zero)` just to call `len()`

```go
// LatestPrice: loads ALL history just to take the last element
points, err := s.store.Range(ctx, seriesName(symbol), time.Time{}, time.Time{})
...
rec := pointToPriceRecord(symbol, points[len(points)-1])
```

These scans grow unbounded with retention (10 years of daily candles per
series).

**Upstream fix.** Add:
- `Store.Latest(ctx, series) (Point, bool, error)` — `ORDER BY time DESC LIMIT 1`, pivoted to a `Point` with the real timestamp.
- `Store.LatestField(ctx, series, field) (Sample, bool, error)` — same for one field, returning `(time, value)`.
- `Store.Count(ctx, series) (int, error)` (and/or per-field) for the stats path.

**etna follow-up.** Rewrite `LatestPrice` via `Latest`, `LatestRate` via
`LatestField`, and `Stats` via `Count`.

---

## 4. Avoid full `DefineSeries` on every write — contention smell — ✅ DONE

**Done.** `DefineSeries` now early-returns on a no-op define
(`series.go`). After the cheap arg checks it calls a new
`definitionUnchanged(ctx, cfg)` helper that takes only `RLock`, reads the stored
series + fields, and reports whether precision, retention, and the field set
(via `sameFieldSet`, order-insensitive) all match. If so, `DefineSeries` returns
`nil` without taking the exclusive lock or opening a write transaction; only an
actual change (new series, or differing precision/retention/fields) escalates to
the original define-and-reconcile path. Invalid configs (duplicate field names,
unknown aggregates) can't match a stored definition, so they still fall through
to `validateFields` and error as before. Covered by
`TestDefineSeries_UnchangedSkipsWriteLock` (white-box: holds `RLock` and
requires a redundant define to still complete) and
`TestDefineSeries_ChangeStillApplies` (escalation still reconciles). Full suite
passes under `-race`; etna needs no change — `RegisterInstrument`/`RegisterPair`
on the write path just became cheap on the steady-state path.

**Problem (original).** Every ingest auto-registers the series via the full
`DefineSeries`, which takes the **exclusive** lock, opens a transaction,
upserts the series row, and reconciles all fields — even when nothing changed.
On bulk import this serializes a heavy structural op ahead of each batch.

- `internal/marketdata/price.go:57,74` (`IngestPrice`, `IngestPricesBulk` → `RegisterInstrument`)
- `internal/marketdata/fx.go:90,104` (`IngestRate`, `IngestRatesBulk` → `RegisterPair`)

Correct, just costly. Lower priority than #1–#3.

**Upstream fix (recommended).** Make `DefineSeries` early-return when the stored
definition already matches the requested one: take `RLock`, read the existing
series + fields, and if precision, retention, and the field set are identical,
return `nil` without opening a write transaction or taking the exclusive lock.
Only escalate to the current define-and-reconcile path when something actually
differs (or the series is missing). This is good practice on its own — a
structural "define" call should be idempotent and cheap on the no-op path
regardless of who calls it — so it's worth doing even if #6 removes the
hot-path callers. (A separate lightweight `EnsureSeries` is an alternative, but
folding the early-exit into `DefineSeries` keeps one code path and benefits
every caller.)

**etna follow-up.** No change required — the existing
`RegisterInstrument`/`RegisterPair` calls simply become cheap on the
steady-state path.

> **Note.** #6 is a stronger, etna-side alternative that takes `DefineSeries`
> off the write path entirely by defining the series once at creation time.
> The two are complementary, not mutually exclusive: do this early-exit
> regardless (it's good library hygiene), and #6 on top if the hot-path call is
> worth eliminating.

---

## 5. FX synthetic time-as-ID — etna-only cleanup (no library change) — ✅ DONE

**Done.** The FX path is now time-addressed end-to-end, exactly like the price
path. `fxID`/`fxTime` and their two `gosec` G115 nolints are gone, along with
`RateRecord.ID`. The store exposes `EditRate(main, secondary, oldTime, RatePoint)`
(mirrors `EditPrice`) and `DeleteRateAt(main, secondary, t)` (mirrors
`DeletePriceAt`); the `RateUpdate` partial-update struct was dropped in favour of
a full upsert/move like prices. The REST API edits/deletes by date
(`PUT`/`DELETE /fin/fx/{main}/{secondary}/rates/{date}`) instead of `{id}`, and
the handlers (`EditFXRate`/`DeleteFXRate`) plus route handlers no longer parse a
numeric id. Frontend follows suit: `RateRecord.id` and `UpdateRateDTO` removed,
`updateRate(origDate, CreateRateDTO)`/`deleteRate(date)` in `CurrencyRates.ts`,
`useFXMutations` keyed on `origDate`/`time`, and `CurrencyDetailView`'s table
`dataKey="time"`. Covered by `TestEditRate`/`TestDeleteRateAt` and the updated
`CurrencyRates.test.ts`.

**Problem (original).** `fx.go:33-36` fabricated a `uint` id from a record's
UNIX-seconds time (with two `gosec` G115 nolints) so the FX REST API had a
stable record id. `price.go` does **not** do this — it edits by time directly
(`oldTime`). So this was an internal inconsistency in etna, not a library gap.

---

## 6. Define series at creation time, not on every write — etna refactor (no library change)

**Not a library concern.** This complements #4: where #4 makes `DefineSeries`
cheap on the no-op path, this takes it *off* the write path entirely by moving
series definition to a real creation point. Needs no upstream change. Do #4
regardless (it's good library hygiene); do this on top if eliminating the
hot-path call is worth the etna refactor.

**Problem.** The library's `Write`/`WriteMany` does **not** auto-create series
or fields — `WriteMany` resolves the series id (`ErrSeriesNotFound` if absent)
and every field id before writing. So a series must already be defined. Today
that definition happens *lazily*, inside each ingest call via
`RegisterInstrument`/`RegisterPair` → `DefineSeries`, and the design has two
seams:

- **Prices:** `CreateInstrument` (`instrument.go:73`) writes only etna's own
  `db_instruments` metadata row — it never calls `DefineSeries`. The OHLCV
  series is created lazily on first ingest. So the register call on the write
  path is currently load-bearing: drop it and the first `IngestPrice` for a new
  instrument fails with `ErrSeriesNotFound`.
- **FX:** there is no pair-creation step at all. `ListFXPairs` derives pairs
  from existing series names — the series *is* the pair — so `RegisterPair` on
  first ingest is the only place an FX series ever comes into being.

Two of the register calls are already **dead weight**, though: `EditPrice`
(`price.go:157`) and `UpdateRate` (`fx.go:170`) call register only on the
*move* branch (`oldTime != newTime`), which by definition means a record — and
therefore the series — already exists.

**etna refactor.**
- **Prices:** call `DefineSeries` (via `RegisterInstrument`) inside
  `CreateInstrument`, then drop it from `IngestPrice`/`IngestPricesBulk`.
- **FX:** add an explicit pair-creation step and call it where pairs are first
  introduced (including restore), then drop register from
  `IngestRate`/`IngestRatesBulk`. If a standalone pair-creation step isn't
  wanted, keep lazy register on the FX ingest path only.
- **Both edit paths:** remove the redundant register calls in `EditPrice` and
  `UpdateRate` regardless — they're already guaranteed a live series.

**Watch out.** The restore path (`internal/backup/import.go`) calls
`CreateInstrument` then `IngestPricesBulk` for prices (lines 558/584), but for
FX it calls only `IngestRatesBulk` (606). So FX restore currently *relies* on
lazy register — any refactor must define the FX series during restore (or keep
lazy register on the FX ingest path). Also consider series that could be
missing for an existing instrument (e.g. after a partial wipe): writes would
start failing instead of self-healing.
