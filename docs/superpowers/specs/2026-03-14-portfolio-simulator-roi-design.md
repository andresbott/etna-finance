# Portfolio Simulator — ROI Comparison Redesign

## Goal

Simplify the portfolio simulator to focus on comparing annual ROI across different investment methods using only an initial lump-sum contribution. Remove monthly contributions and fix the broken tax model.

## Parameters

| Parameter | Type | Default | UI Range | Notes |
|---|---|---|---|---|
| `initialContribution` | number | 10,000 | 0–1,000,000 | Lump sum invested at year 0 |
| `growthRatePct` | number | 7.0 | 0–30 | Gross annual return |
| `expenseRatioPct` | number | 0.20 | 0–5 | Annual TER / management fee, drag on total balance |
| `inflationPct` | number | 2.0 | 0–15 | Annual inflation rate |
| `capitalGainTaxPct` | number | 19.0 | 0–50 | Tax rate on capital gains |
| `taxModel` | `"exit"` \| `"annual"` | `"exit"` | toggle | When tax is applied |

**Removed:** `monthlyContribution`, `durationYears` (hardcoded to 20 years).

## Computation

### Effective annual return

```
effectiveReturn = growthRatePct - expenseRatioPct
```

TER is a direct annual drag on gross returns.

### Monthly compounding

```
monthlyRate = (1 + effectiveReturn / 100) ^ (1/12) - 1
```

Compound `initialContribution` for 240 months (20 years).

### Tax model — exit

Tax is applied only at year 20 on total gains:

```
gains = finalBalance - initialContribution
tax = gains * (capitalGainTaxPct / 100)
netWorth = finalBalance - tax
```

Per-year chart series: tax impact is 0 for years 0–19, jumps to full tax at year 20.

### Tax model — annual

Each year, tax is applied only to that year's growth:

```
yearGain = balanceEndOfYear - balanceStartOfYear
tax = yearGain * (capitalGainTaxPct / 100)
balance = balanceEndOfYear - tax
```

Per-year chart series: cumulative tax grows each year.

### Inflation adjustment

Per year: `realValue(y) = nominalValue(y) / (1 + inflationPct / 100) ^ y`

Applied to net worth, gains, and cost basis series. Since all contributions happen at year 0, the cost basis inflation adjustment is straightforward: `realCostBasis = initialContribution / (1 + inflationPct / 100) ^ y`.

### Expected annual return (for comparison chart)

Used by `computePortfolioExpectedReturn` for the main page comparison. Duration is always 20:

```
estimatedTaxDrag = (exit) ? (capitalGainTaxPct / 100) * effectiveReturn / 20
                          : (capitalGainTaxPct / 100) * effectiveReturn
expectedReturn = effectiveReturn - estimatedTaxDrag - inflationPct
```

This is an approximation — the actual projection is the source of truth.

### Edge cases

- `growthRatePct < expenseRatioPct`: effective return goes negative, portfolio loses value through fees. The math handles this (negative compounding). No special UI warning needed.
- Zero growth: portfolio stays flat (exit tax) or loses to expense ratio drag.
- Old saved data missing new fields: `expenseRatioPct` defaults to `0`, `taxModel` defaults to `"exit"`.

## Chart & Summary

### Chart (same structure as current)

6 line series over 20 years:
- Total Invested (flat line at `initialContribution`)
- Net Worth (after tax)
- Inflation Adjusted Net Worth
- Total Gains
- Inflation Adjusted Gains
- Tax Impact (cumulative)

### Summary box

- Total Invested
- Net Worth
- Inflation Adjusted Net Worth
- Total Gains
- Inflation Adjusted Gains
- Inflation Impact
- Tax Impact

## UI Changes

### Parameter card

- Remove monthly contribution field and slider
- Remove duration field and slider
- Add expense ratio field with slider (0–5%, step 0.05)
- Add tax model toggle (two-option segmented button: "Exit Tax" / "Annual Tax")
- Keep: initial contribution, growth rate, inflation, capital gain tax

### No other UI changes

View structure, save/load, delete/duplicate, back button, comparison chart on main page — all unchanged.

## Type Changes

### `PortfolioSimulatorParams` in `ToolsData.ts`

```ts
export interface PortfolioSimulatorParams {
    initialContribution: number
    growthRatePct: number
    expenseRatioPct: number
    inflationPct: number
    capitalGainTaxPct: number
    taxModel: 'exit' | 'annual'
    // Deprecated — ignored if present in saved data
    monthlyContribution?: number
    durationYears?: number
}
```

Old saved cases load fine:
- `monthlyContribution` and `durationYears` are ignored by the new computation (always 0 contributions after initial, always 20 years).
- Missing `expenseRatioPct` defaults to `0`. Missing `taxModel` defaults to `"exit"`.
- The `PortfolioProjection` interface drops the redundant `finalValue` field (was identical to `finalValueAfterTax`). Only `finalValueBeforeTax` and `finalValueAfterTax` remain.

## Files Changed

- `webui/src/lib/api/ToolsData.ts` — update `PortfolioSimulatorParams` type
- `webui/src/lib/simulators/portfolio.ts` — rewrite computation logic
- `webui/src/views/tools/PortfolioSimulatorView.vue` — update form and chart
