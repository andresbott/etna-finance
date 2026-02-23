# TODO — Future items to check / implement

Track items to verify, refactor, or implement later.

---

## To implement

- [ ] Log a warning in the task when the client is not configured (e.g. missing API key)
- [ ] Update the dashboard with currency-corrected data
- [ ] Add a dashboard view: current balance (quantity of stocks) + value of every owned financial instrument

---

## To check / verify



---

## Refactor / improve

- [ ] **Reporting backend** — Use historical values as of report date: value financial instruments at market price at reported time; use historical currency rates for both cash and investment items
- [ ] Simplify the stock and forex tasks, implementation wise
- [ ] Handler should rely on list tasks from the runner instead of listing them separately
- [ ] See how we can unify the scheduler and the runner in an easier way

---

## Missing features

- [ ] **Tax budget** — Track money that must be paid but is still in the account (e.g. taxes): reserve/liability so “available” balance reflects what’s left after setting aside tax money
- [ ] **Scheduled operations** — Recurring transactions, e.g. rent payment, salary income
- [ ] CSV import
- [ ] Mortgage tracking
- [ ] ROI of real estate calculator
- [ ] Adjust the dashboard to inflation
- [ ] Store JSONs in the DB and be able to retrieve them
- [ ] Add a Sankey graph to income/expense
- [ ] **Backend API card** — Generate graph data for stock price adjusted instruments
- [ ] **User auth** — Improve login mechanism with better user authentication features

---

## Notes

_Add dated notes or context here._
