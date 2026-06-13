# Timeseries upstream follow-ups — third pass (2026-06-13)

New findings from a third re-audit of etna's `internal/marketdata` against the
`github.com/go-bumbu/timeseries` library. Same goal and method as the prior
passes: find places where etna **reaches past the library abstraction** because
an API is missing, plus latent bugs hidden inside those workarounds — and fix the
library **before** removing the `replace` directive and pinning a real release.

This continues the numbering of the previous doc (`timeseries-upstream-followups.md`,
items #1–#11), which is now staged for deletion. Items #1–#6 (first pass) and #7–#10
(second pass) are ✅ DONE there; #11 (low-priority grab-bag) was the only item left
open, and its sub-points still stand unless noted below.

- **Library location:** `/home/bott/datos/edit/programacion-privado/bumbu/timeseries` (branch `optimize-ts`)
- **Wired in via:** `go.mod` `replace github.com/go-bumbu/timeseries => …/bumbu/timeseries`
- **Library public API:** `store.go`, `series.go`, `record.go`, `field.go`, `label.go`, `maintain.go`, `unixmilli.go`
- **Vendoring:** library edits are only visible to etna after `go mod vendor`.

## Status

Four parallel reviewers (price/instrument/stats, fx, eps, store-wiring + library
contracts) ran on 2026-06-13. The **fx** scope came back **clean** — #5 (FX
time-as-ID removed) and #9 (labels replace name-parsing) verified sound. The other
three scopes surfaced **five new items, #12–#16**, plus two trivia notes. Two of
the five are high-severity, REST-reachable, silent-data-loss paths and should be
fixed before pinning a release:

- **#13** is the most serious: a data-loss/outage regression that **#6 introduced**
  (removing lazy auto-register unmasked it). A symbol rename orphans the series.
- **#12** (the destination-collision half of #8) is now **resolved**: `Move` was
  removed from the library outright, so the silent-overwrite path no longer exists.
  Editing a record's date is rejected (400) instead.

The two trivia notes and the carried-over #11 sub-points are collected at the end.

---

## 12. `Move` silently clobbers an existing record at the destination timestamp — ✅ RESOLVED (Move removed)

> **Resolution (2026-06-13).** Rather than guard the destination, we removed
> `Store.Move` from the library entirely (and with it `ErrRecordNotFound`, whose
> only user was `Move`). A timestamp **is** a record's identity in a timeseries, so
> "moving" a point is just delete + create — not a primitive the library should own,
> and the source of this whole collision/identity hazard. etna's `EditPrice`/
> `EditEPS`/`EditRate` now upsert at the point's time and **reject a date change**
> with a new `ErrDateImmutable` sentinel → HTTP **400**; the webui date picker is
> read-only in edit mode. This also moots the #8 source-not-found guard and the
> `Move`-related notes under #16 and the carried-over #11 list below.

**Tags:** (library, correctness) · **Reachable from:** `EditPrice`, `EditEPS`, `EditRate`

**Anchors:** `internal/marketdata/price.go:160-176` (`EditPrice`),
`internal/marketdata/eps.go:123-137` (`EditEPS`), `internal/marketdata/fx.go:180-196`
(`EditRate`), all calling `timeseries.Store.Move` at `bumbu/timeseries/record.go:106-150`.

An edit that changes a record's date is resolved as
`Move(series, oldTime, Point{Time: newTime, …})`. `Move` deletes every field row
at `oldTime`, then upserts the new point at `newTime` via `OnConflict … DoUpdates`.
The not-found guard added by #8 protects only the **source** (`oldTime` must exist,
`record.go:137-139`). There is **no guard on the destination**: if a record already
exists at `newTime`, the upsert overwrites it and `Move` returns `nil`.

*Original problem.* A user edits the 2024-03-01 candle and changes its date to
2024-03-02, where a candle already exists. The Edit handler returns `200 OK`. The
pre-existing 2024-03-02 candle is destroyed and replaced by the edited values; the
2024-03-01 candle is deleted. Two records silently collapse into one, with no error
and no warning — a destructive merge masquerading as a rename. Same failure mode for
`EditEPS` and `EditRate`. `Move`'s contract (`record.go:95-105`) says nothing about
destination collisions, so the behavior is undocumented either way.

*Proposed fix (library).* The collision check must be inside the same transaction as
the delete+upsert to be race-free — etna cannot pre-check without reintroducing the
exact TOCTOU hazard `Move` was built to eliminate. Add a sentinel and check for an
existing row at `p.Time` when `!oldTime.Equal(p.Time)`, rolling back on collision:

```go
// ErrRecordExists is returned by Move when oldTime != p.Time and a record already
// exists at p.Time, so a move cannot silently overwrite the destination.
var ErrRecordExists = errors.New("record already exists")
```

Or, less invasively, a `MoveOpts{Overwrite bool}` so the caller opts in. etna would
translate the sentinel to a package-owned error at the boundary (as it does for
`ErrRecordNotFound`) and the three Edit handlers would map it to **409 Conflict**.

*Judgment.* **Fix before pinning.** Real silent-data-loss path reachable from the
public REST API; the fix is necessarily library-side.

---

## 13. Symbol rename orphans the price/EPS series — ingestion outage + history vanishes — ⬜ OPEN (high)

**Tags:** (etna + library, correctness) · **Regression introduced by #6**

**Anchors:** `internal/marketdata/instrument.go:160-233` (`UpdateInstrument`);
series names keyed on symbol at `marketdata.go:158` (`seriesName` → `price:SYM`) and
`marketdata.go:180` (`epsSeriesName` → `eps:SYM`); importer call sites
`app/tasks/financial_import.go:137`, `app/tasks/financial_backfill.go:109`.

`UpdateInstrument` allows changing `Symbol` (validated 164-169, uniqueness 198-208,
applied 210-213) but **never renames, redefines, or drops the name-keyed series** on
a rename. It only conditionally (re)defines the EPS series when the *type* becomes
stock (223-231); it never touches the price series at all.

*Original problem.* Before #6, the first `IngestPrice` after a rename lazily
auto-registered `price:NEWSYM`, self-healing. #6 deliberately removed auto-register
from the ingest path (`price.go:56-85` — "must already exist… does not auto-register").
The scheduled importer calls `IngestPricesBulk(ctx, inst.Symbol, …)` with the
*current* symbol. So after a rename:

- the next price import fails with `ErrSeriesNotFound` for `price:NEWSYM` (never
  defined) — a **silent ingestion outage** for that instrument;
- the entire prior history is stranded under `price:OLDSYM`; `PriceHistory`/
  `LatestPrice` query `price:NEWSYM` and return nil — **history vanishes** from the UI;
- `ListPriceSymbols`/`Stats` still surface `price:OLDSYM` via its (now stale) `symbol`
  label — a **ghost symbol** lingers.

EPS shares this fate for stock instruments (a pure symbol rename never re-defines
`eps:NEWSYM`). No test covers a successful rename — `instrument_test.go:230` only
checks empty/duplicate-symbol rejection.

*Proposed fix.* The principled fix is a library rename primitive that atomically
moves a series' name together with its records/fields/labels:

```go
// RenameSeries moves a series to newName (records, fields and labels follow) in one
// transaction under s.mu.Lock. Returns ErrSeriesNotFound if old is absent, and an
// error if newName already exists.
func (s *Store) RenameSeries(ctx context.Context, oldName, newName string) error
```

`UpdateInstrument` then calls `RenameSeries(ctx, seriesName(old), seriesName(new))`
(and the EPS equivalent for stocks) inside the symbol-change branch. Without a
library change, the only etna-side options are a copy-then-drop (no bulk record-copy
API exists — itself a reach-around) or simply **blocking symbol changes** as a
temporary guard.

*Judgment.* **Must-fix.** Data-loss/outage on a reachable REST path, and a
regression #6 unmasked. `RenameSeries` is the right boundary; blocking renames is an
acceptable stopgap only.

---

## 14. EPS create handlers lack the lazy-register safety net the FX handlers have — ✅ DONE (medium)

> **Resolution (2026-06-13).** Promoted `registerEPSSeries` →
> `Store.RegisterEPSSeries(ctx, symbol)` (with an empty-symbol guard mirroring
> `RegisterPair`) and called it before ingest in both `CreateEPS` and `CreateEPSBulk`,
> matching the FX create handlers. A first EPS point for a symbol with no `eps:` series
> now registers it and returns **201** instead of mapping `ErrSeriesNotFound` to a 500.
> Regression test `TestCreateEPS_FirstPointRegistersSeries` (single + bulk, non-stock
> instrument) covers it.

**Tags:** (etna, correctness)

**Anchors:** `app/router/handlers/marketdata/marketdata.go:382-411` (`CreateEPS`) and
`:413-450` (`CreateEPSBulk`) call `IngestEPS`/`IngestEPSBulk` directly; the FX create
handlers call `RegisterPair` first (`marketdata.go:659`, `:706`).

Since #6/#10, EPS ingest no longer auto-registers the series (`eps.go:50-79` — requires
the `eps:` series to already exist). The `eps:` series is created only at instrument
creation for stock-type instruments, on a type→stock switch in `UpdateInstrument`, and
by the startup migration. The FX create handlers handle the equivalent "first point
introduces the series" case by registering first; the EPS create handlers have **no
such step**.

*Original problem.* A `POST …/eps` (or `/eps/bulk`) for a symbol whose `eps:` series
doesn't yet exist — a non-stock instrument the user is manually annotating, or a stock
whose series creation was skipped — hits `ErrSeriesNotFound`, which the handler maps
to a generic **500** ("unable to ingest EPS"), where the FX-equivalent returns 201.
A legitimate "first EPS point for this symbol" is a client-fixable condition reported
as a server error, and it makes the manual-EPS-entry path unusable for any symbol that
wasn't auto-defined. The `EditEPS` zero/equal-time branch shares this (it degrades to
`IngestEPS`, `eps.go:127-129`).

*Proposed fix (etna-side, no library change).* Mirror the FX handlers: promote the
unexported `registerEPSSeries` to a public `Store.RegisterEPSSeries(ctx, symbol)` and
call it before ingest in both EPS create handlers. `DefineSeries` is already cheap on
the no-op path (#4), so registering before every create is free in steady state.

*Judgment.* Worth fixing — a real behavioral asymmetry with a 500-on-valid-input
symptom, entirely in etna's control, with the FX path already showing the pattern.

---

## 15. Library `Delete` discards `RowsAffected` — delete cannot report not-found — ✅ DONE (low)

> **Resolution (2026-06-13).** `Store.Delete` now returns `(deleted bool, err error)`
> (`res.RowsAffected > 0`), so a no-op delete is distinguishable from a real one.
> etna owns an `ErrRecordNotFound` sentinel (`marketdata.go`); `DeletePriceAt`/
> `DeleteEPSAt`/`DeleteRateAt` return it when nothing was removed, and the three
> delete handlers map it to **404** (mirroring the "no data found" 404s on the read
> paths). Covered by store tests (`{Price,Rate,EPS}` delete-missing → `ErrRecordNotFound`),
> a library `TestDeletes` assertion (deleted true then false), and a handler test
> (`TestDeleteEndpoints_MissingRecordReturns404`).

**Tags:** (library, correctness) · **Affects price/fx/eps delete paths identically**

**Anchors:** `bumbu/timeseries/record.go:381-389` (`Store.Delete`); etna callers
`eps.go:139-148` (`DeleteEPSAt`), and the price/fx equivalents.

`Store.Delete` issues `Where(series_id, time).Delete(&dbRecord{})` and returns
`.Error` only — it never inspects `RowsAffected`. So deleting a timestamp with no
record returns `nil`, and the handlers answer **200 OK** for a delete that removed
nothing. This is asymmetric with `Move`, which #8 gave an `ErrRecordNotFound` signal;
etna **cannot** return a 404 here because the count is thrown away before it reaches
the caller.

*Original problem.* A client deleting an already-removed record (double-click,
concurrent delete, stale UI) gets a success indistinguishable from a real delete — a
silent swallowed not-found. No data corrupts, but the API misreports the outcome.

*Proposed fix (library).* Report whether anything was removed:

```go
func (s *Store) Delete(ctx context.Context, series string, t time.Time) (bool, error)
```

(returning `res.RowsAffected > 0`). Then `DeleteEPSAt`/`DeletePriceAt`/`DeleteRateAt`
can return a not-found sentinel and the handlers can answer 404, matching the edit
path's 404 from #8.

*Judgment.* Low urgency — idempotent-delete-returns-200 is a defensible REST choice.
(Note: the `Move`/`ErrRecordNotFound` path this once paired with is gone as of #12;
`Delete` is now the only mutation that can't report not-found.)

---

## 16. `Latest` is a non-atomic two-query read under `RLock` — ⬜ OPEN (low)

**Tags:** (library, correctness/informational) · New facet of the #11(a)/(d) two-phase-lock family

**Anchors:** `bumbu/timeseries/record.go:300-333` (`Latest`); writers also take only
`RLock` (`WriteMany` `record.go:49`, `Move` `record.go:110`, `Delete` `record.go:382`).

`Latest` issues two queries under `RLock`: (1) `ORDER BY time DESC LIMIT 1` to find
the newest timestamp, then (2) `WHERE time = newest` to fetch its fields. The
`RWMutex` serializes reads/writes only against the *exclusive* structural ops
(`DefineSeries` escalation / `Maintain` / `Drop` / `Wipe`), never against each other
(`store.go:28-36`).

*Original problem.* A concurrent `WriteMany` inserting a newer timestamp can land
between query (1) and query (2): (1) picks T, a write then adds T+1 (and possibly more
fields at T), and (2) reads T's fields — so `Latest` returns T while a newer point
exists, or a torn view of T.

*Proposed fix (library).* Collapse to a single query —
`WHERE time = (SELECT MAX(time) WHERE series_id = ?)` — mirroring the
correlated-subquery style `At` already uses (`record.go:273-281`), so the
newest-timestamp pick and its field read are one consistent snapshot. Alternatively,
document `Latest` as best-effort under concurrent writes.

*Judgment.* Low impact given etna's single-writer importer reality. Fix-when-touched;
library hygiene, not urgent.

---

## Trivia (note, not tracked as numbered items)

- **`ListPriceSymbols` vs `Stats` disagree on empty-`symbol` series.**
  `ListPriceSymbols` (`marketdata.go:231-237`) skips any `type=price` series whose
  `symbol` label is empty, while `Stats` (`stats.go:31-38`) counts every `type=price`
  series — so a label regression would yield two different instrument counts on two
  screens. Currently **unreachable** (price series always get both labels together).
  Etna-side: drop the `sym != ""` guard or apply the same guard in `Stats`. One-line
  alignment.

- **`DefineSeries`/`definitionUnchanged` docs omit labels from the "unchanged"
  contract.** The code *does* compare labels (`series.go:120-124` via `sameLabelSet`)
  — that's what makes the `backfillSeriesLabels` escalation work — but the doc comments
  (`series.go:37-41`, `:95-99`) describe the match as precision/retention/fields only.
  Code is correct; add "and labels" to both comments.

---

## Carried over from #11 (still open)

- `Stats` is N+1 (one `Count` per series; `stats.go:21-47`). An upstream batch
  `CountAll(ctx, opts ...ListOption) (map[string]int, error)` (one `GROUP BY
  series_id`) would collapse it. Cold path; pairs with #15's `Delete` signature and
  the #16 single-query idea as a "tidy up the read/aggregate API" batch.
- `DefineSeries` fast-path two-phase lock — benign; doc note or re-check under the
  exclusive lock.
- `ensureInstrumentSeries` O(N) startup loop — an upstream `DefineSeriesMany([]Series)`
  would batch it. Not urgent.
- `WipeData` two non-atomic ops — inherent; only runs during restore.
- `Move` ranges a map for inserts (nondeterministic order) — **explicitly skipped**
  (harmless; correctness unaffected by composite-PK upsert).

---

## Verified clean (no finding)

- **fx scope** — every `timeseries` call site uses the post-#5/#9 APIs cleanly
  (`Move`→404, `LatestField`, `FieldRange`, `MatchLabel`); restore registers the pair
  before ingest; vendor dir is byte-identical to the library source on `optimize-ts`.
- `PriceAt` partial-candle rejection (#7) is sound — `Maintain`'s reducer keeps all
  OHLCV legs co-timestamped, so the rejection branch is dead defensive code, not a
  reach-around.
- `DropSeries`/`Wipe` cascade order (records → fields → labels → series) is correct.
- `DeleteInstrument` is a *soft* delete and `restoreSoftDeletedInstrument` re-defines
  by the same symbol, so leaving the series in place is defensible by design (unlike
  the rename case in #13).
