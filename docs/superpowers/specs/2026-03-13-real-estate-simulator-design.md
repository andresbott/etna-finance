# Real Estate Simulator — Design Spec

## Overview

A real estate investment simulator for etna-finance, replicating the functionality of the standalone invest-calc project. Generic (not Swiss-specific) but flexible enough to model Swiss scenarios through dynamic mortgage and equity source management.

Follows the same UI pattern as the existing Portfolio Simulator: left panel for inputs, right panel for results with tabbed reports and an ECharts chart.

## Input Parameters (Left Panel — Accordion Sections)

### Property
- Purchase price (0-10M, default 500k)
- Market value (0-10M, default 500k)
- Square meters (0-1000, default 80)

### Rental Income
- Monthly rent (0-20k, default 1500)

### Recurring Costs
- Property tax / year (0-50k, default 1000)
- Insurance / year (0-20k, default 500)
- Maintenance / year (0-20k, default 1000)
- Other costs / year (0-20k, default 0)

### Personal Contribution
- Cash equity (0-5M, default 100k)
- Additional equity sources — dynamic list with name + amount (add/remove). Supports modeling Swiss 2nd/3rd pillar or any other equity source.

### Mortgages (Dynamic — Add/Remove)
Each mortgage has:
- Name (text, e.g. "1st Mortgage")
- Principal (auto-calculated as remaining balance, or manual override)
- Interest rate % (0-15, default 1.5)
- Term in years (1-50, default 25)
- Amortization toggle (yes/no — if no, interest-only payments)

Default: one mortgage covering `purchase price - total equity`. User can add more or remove any.

**Mortgage principal auto-calculation:**
- When there is a single mortgage: principal = `purchasePrice - totalEquity`
- When a new mortgage is added: principal defaults to 0 (user fills it in manually)
- The first mortgage auto-adjusts to cover the gap: `purchasePrice - totalEquity - sum(other mortgage principals)`
- If the user manually edits the first mortgage principal, auto-calculation stops for that mortgage
- Validation: total of all mortgage principals + total equity should equal purchase price. Show a warning (not a blocker) if they don't match.
- Minimum 0 mortgages allowed (100% cash purchase). No hard maximum.

### Affordability
- Gross monthly income (0-100k, default 8000)

## Report Tabs (Right Panel)

### Tab 1: Overview Chart
ECharts line chart over the longest mortgage term. Series:
- Total property equity (equity contributions + mortgage principal paid down)
- Remaining mortgage balance (sum of all mortgages)
- Cumulative net cash flow (rent income - costs - mortgage payments)
- Cumulative interest paid

Below the chart: summary cards with key metrics — total invested, net worth, leveraged ROI.

### Tab 2: Affordability
- Monthly mortgage costs breakdown (interest + amortization per mortgage)
- Total monthly housing cost (mortgages + recurring costs / 12)
- Affordability ratio: (total monthly cost / gross income) x 100
  - Color-coded: green < 25%, orange 25-33%, red > 33%
- Equity contribution %: (total equity / purchase price) x 100
  - Color-coded: red < 20%, orange 20-33%, green > 33%

### Tab 3: Rentability
- Monthly & yearly income vs expenses breakdown
- Gross annual return: (annual rent / property value) x 100
- Net Operating Income (NOI): annual rent - total recurring costs
- Cap rate: (NOI / property value) x 100
- Leveraged cap rate: (NOI - annual mortgage payments) / property value x 100
- Leveraged cash flow: NOI - annual mortgage payments
- Levered yield (ROI): (leveraged cash flow / total equity) x 100

### Tab 4: Amortization
- Per-mortgage breakdown: principal, rate, term, monthly payment, total interest, interest-to-principal ratio
- Combined yearly amortization schedule (DataTable): year, beginning balance, interest paid, principal paid, ending balance — per mortgage

## Calculation Logic

### Mortgage Monthly Payment (Amortizing)
```
monthlyRate = annualRate / 100 / 12
months = termYears * 12
payment = principal * (monthlyRate * (1 + monthlyRate)^months) / ((1 + monthlyRate)^months - 1)
```

### Mortgage Monthly Payment (Interest-Only)
```
payment = principal * annualRate / 100 / 12
```

### Key Metrics
- Annual rent = monthlyRent * 12
- Total recurring costs = propertyTax + insurance + maintenance + otherCosts
- NOI = annual rent - total recurring costs
- Total equity = cashEquity + sum(additionalEquity amounts)
- Total annual mortgage payments = sum of (monthly payment * 12) for each mortgage
- Gross annual return = (annual rent / marketValue) * 100
- Cap rate = (NOI / marketValue) * 100
- Leveraged cap rate = (NOI - total annual mortgage payments) / marketValue * 100
- Leveraged cash flow = NOI - total annual mortgage payments
- Levered yield = (leveraged cash flow / total equity) * 100
- Affordability ratio = ((total annual mortgage payments / 12 + total recurring costs / 12) / grossMonthlyIncome) * 100

### Chart Projection (Year-by-Year)
For each year from 0 to max mortgage term:
- Calculate remaining balance per mortgage (amortization schedule)
- Total remaining mortgage = sum of remaining balances
- Total equity = marketValue - total remaining mortgage
- Cumulative interest = sum of interest paid across all mortgages
- Cumulative net cash flow = cumulative (annual rent - recurring costs - mortgage payments)

## Case Study Integration

- ToolType: `real-estate-simulator`
- `expectedAnnualReturn` stores the levered yield (ROI)
- `params` stores all input values as `RealEstateSimulatorParams`
- Save/Load/Delete buttons in header, active case display — same UX as portfolio simulator

## TypeScript Interface

```typescript
interface RealEstateSimulatorParams {
  purchasePrice: number
  marketValue: number
  squareMeters: number
  monthlyRent: number
  propertyTax: number
  insurance: number
  maintenance: number
  otherCosts: number
  cashEquity: number
  additionalEquity: Array<{ name: string; amount: number }>
  mortgages: Array<{
    name: string
    principal: number
    interestRate: number
    termYears: number
    amortize: boolean
  }>
  grossMonthlyIncome: number
}
```

## Technical Approach

- Single `RealEstateSimulatorView.vue` file with all logic inline (computed properties)
- No Pinia stores — local ref() state, matching portfolio simulator pattern
- Reuse existing `ToolsData.ts` API client — no backend changes needed
- PrimeVue components: TabView (reports), InputNumber + Slider, DataTable (amortization), Card, Button, Dialog
- Input grouping via styled section headers within a single Card (no Accordion — not used in codebase)
- ECharts for the overview chart (LineChart, same setup as portfolio simulator)
- Dynamic mortgages and equity sources via ref<Array> with add/remove buttons

## Files to Create/Modify

- **Modify:** `webui/src/views/tools/RealEstateSimulatorView.vue` (replace stub)
- **Modify:** `webui/src/lib/api/ToolsData.ts` (add `RealEstateSimulatorParams` interface)
- No backend changes required
- No router changes required (route already exists)
