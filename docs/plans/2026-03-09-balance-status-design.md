# Balance Status Design

## Problem

Balance calculation is regenerated on every request. Changing an entry can shift all subsequent balances with no way to detect the error. There is no mechanism to capture the real balance as stated on a bank statement, making it hard to go back and correct values.

## Solution

Introduce a new movement type "Balance Status" for cash accounts. It is a no-op that records the real balance from a bank statement at a given date. It does not affect the calculated balance. The system computes and displays the discrepancy between the stated and calculated balance, giving a reference point to find and fix incorrect entries.

## Data Model

- New `TxType`: `BalanceStatusTransaction` (value 9)
- New `entryType`: `balanceStatusEntry`
- One `dbTransaction` + one `dbEntry` per balance status record
- Entry stores: `AccountID` (cash account), `Amount` (stated balance), `Date`
- `balanceStatusEntry` is NOT added to `balanceEntryTypes` -- it does not affect running balance
- Discrepancy is computed on read (stated balance minus calculated balance at that date), not stored

## Backend API

- `CreateBalanceStatus` in accounting store: creates one transaction + one entry
- Validation: account must exist and must be a cash account
- Update/delete through existing `UpdateTransaction`/`DeleteTransaction` paths
- Balance status entries appear in `ListTx()` results like any other type
- When returning a balance status entry, the API computes `calculatedBalance` at that date using `AccountBalanceSingle` and includes it in the response
- One extra balance query per balance status entry in the list; acceptable given these entries are infrequent (roughly once per month per account)

## Frontend

- New display type `balance-status` in `entryDisplay.ts` (replaces unused `opening-balance`)
- Entries table: balance status rows show a distinct icon/color, the stated balance in the Amount column, and a discrepancy indicator (green checkmark if match, red/orange "Diff: +X" if mismatch)
- Create form: new "Balance Status" type option with fields: Account (cash only), Date, Amount, Description (optional)
- No category or target account fields

## Constraints

- Cash accounts only -- investment/stock accounts are not eligible
- One account per balance status entry
- Informational only -- does not adjust or override the calculated balance
