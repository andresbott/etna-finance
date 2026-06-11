# TODO — Future items to check / implement

Track items to verify, refactor, or implement later.

---

## Deferred items

- [ ] **Timeseries series orphaned on instrument rename/delete** — The relational store (GORM) and the `go-bumbu/timeseries` store have no transactional relationship. Series are keyed by symbol (`price:SYMBOL`, and `eps:SYMBOL` once EPS lands), so deleting or renaming an instrument leaves orphaned timeseries series with no migration path. Need a strategy for: (a) timeseries data lifecycle when instrument metadata changes, (b) whether symbol is the right join key or a stable instrument ID should be used in the series name, (c) garbage collection / `Wipe` of orphaned series. (Inherited from market-data TODO #13; applies to the existing price series today and to EPS when migrated.)

---

## Missing features


- [ ] **Scheduled operations** — Recurring transactions, e.g. rent payment, salary income
- [ ] Mortgage tracking
- [ ] Adjust the dashboard to inflation

http://localhost:5173/settings/about move runtime -> loglevel into http://localhost:5173/settings/configuration
check again timeseries use-case