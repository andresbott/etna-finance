# Portfolio Simulator ROI Redesign — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Simplify the portfolio simulator to compare annual ROI across investment methods using only an initial lump-sum contribution, with correct tax models and expense ratio.

**Architecture:** Update the `PortfolioSimulatorParams` type, rewrite the pure computation functions in `portfolio.ts` with TDD, then update the Vue view to match the new params. No backend changes needed — params are stored as JSON blob.

**Tech Stack:** TypeScript, Vitest, Vue 3, PrimeVue

**Spec:** `docs/superpowers/specs/2026-03-14-portfolio-simulator-roi-design.md`

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `webui/src/lib/api/ToolsData.ts` | Modify | Update `PortfolioSimulatorParams` type |
| `webui/src/lib/simulators/portfolio.ts` | Rewrite | All computation logic |
| `webui/src/lib/simulators/portfolio.test.ts` | Create | Tests for computation logic |
| `webui/src/views/tools/PortfolioSimulatorView.vue` | Modify | Update form inputs and chart binding |
| `webui/src/lib/api/ToolsData.test.ts` | Modify | Update mock data to match new params shape |

**No changes needed:** `webui/src/lib/simulators/projection.ts` — it calls `computePortfolioNetWorth20Y` which still exists with the same signature. Duration is hardcoded inside `portfolio.ts` now, so `projection.ts` works as-is.

---

## Chunk 1: Type and Computation

### Task 1: Update `PortfolioSimulatorParams` type

**Files:**
- Modify: `webui/src/lib/api/ToolsData.ts:15-22`

- [ ] **Step 1: Update the interface**

Replace the current `PortfolioSimulatorParams` with:

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

- [ ] **Step 2: Verify no type errors**

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | head -30`
Expected: Type errors in `portfolio.ts` and `PortfolioSimulatorView.vue` (expected — we'll fix those next).

- [ ] **Step 3: Commit**

```bash
git add webui/src/lib/api/ToolsData.ts
git commit -m "refactor: update PortfolioSimulatorParams type for ROI redesign"
```

---

### Task 2: Write tests for `computePortfolioProjection` — exit tax model

**Files:**
- Create: `webui/src/lib/simulators/portfolio.test.ts`

- [ ] **Step 1: Write tests for exit tax projection**

Create `webui/src/lib/simulators/portfolio.test.ts`:

```ts
import { describe, it, expect } from 'vitest'
import { computePortfolioProjection, computePortfolioExpectedReturn } from './portfolio'
import type { PortfolioSimulatorParams } from '@/lib/api/ToolsData'

const BASE_PARAMS: PortfolioSimulatorParams = {
    initialContribution: 10000,
    growthRatePct: 7,
    expenseRatioPct: 0.2,
    inflationPct: 2,
    capitalGainTaxPct: 19,
    taxModel: 'exit',
}

describe('computePortfolioProjection — exit tax', () => {
    const result = computePortfolioProjection(BASE_PARAMS)

    it('returns 21 data points (years 0–20)', () => {
        expect(result.years).toHaveLength(21)
        expect(result.years[0]).toBe(0)
        expect(result.years[20]).toBe(20)
    })

    it('starts at initial contribution', () => {
        expect(result.series.netWorth[0]).toBe(10000)
        expect(result.series.totalInvested[0]).toBe(10000)
    })

    it('total invested is flat (no monthly contributions)', () => {
        const allSame = result.series.totalInvested.every(v => v === 10000)
        expect(allSame).toBe(true)
    })

    it('net worth grows over time (before exit tax applied at year 20)', () => {
        // Years 1-19: no tax deducted, balance grows freely
        expect(result.series.netWorth[10]).toBeGreaterThan(10000)
        // Year 20: exit tax applied, but still should be higher than initial
        expect(result.series.netWorth[20]).toBeGreaterThan(10000)
    })

    it('tax impact is 0 for years 0-19, positive at year 20', () => {
        for (let i = 0; i <= 19; i++) {
            expect(result.series.taxImpact[i]).toBe(0)
        }
        expect(result.series.taxImpact[20]).toBeGreaterThan(0)
    })

    it('exit tax is applied only to gains, not total balance', () => {
        // Pre-tax balance at year 20 = netWorth + taxPaid
        const preTaxBalance = result.finalValueAfterTax + result.taxPaid
        const gains = preTaxBalance - 10000
        expect(result.taxPaid).toBeCloseTo(gains * 0.19, 2)
    })

    it('inflation adjusted net worth is less than nominal', () => {
        expect(result.realFinalValue).toBeLessThan(result.finalValueAfterTax)
    })

    it('uses effective return (growth minus expense ratio)', () => {
        // 7% growth - 0.2% TER = 6.8% effective
        // 10000 * (1.068)^20 ≈ 37,338 (before tax)
        const preTaxBalance = result.finalValueAfterTax + result.taxPaid
        expect(preTaxBalance).toBeCloseTo(10000 * Math.pow(1.068, 20), 0)
    })

    it('finalValueBeforeTax equals balance before exit tax', () => {
        expect(result.finalValueBeforeTax).toBeCloseTo(
            result.finalValueAfterTax + result.taxPaid, 2
        )
    })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd webui && npx vitest run src/lib/simulators/portfolio.test.ts 2>&1 | tail -20`
Expected: FAIL — current implementation doesn't match new interface (no `taxModel`, wrong tax logic, etc.)

---

### Task 3: Write tests for `computePortfolioProjection` — annual tax model

**Files:**
- Modify: `webui/src/lib/simulators/portfolio.test.ts`

- [ ] **Step 1: Add annual tax tests**

Append to `portfolio.test.ts`:

```ts
describe('computePortfolioProjection — annual tax', () => {
    const params: PortfolioSimulatorParams = { ...BASE_PARAMS, taxModel: 'annual' }
    const result = computePortfolioProjection(params)

    it('returns 21 data points', () => {
        expect(result.years).toHaveLength(21)
    })

    it('tax impact grows each year', () => {
        expect(result.series.taxImpact[1]).toBeGreaterThan(0)
        expect(result.series.taxImpact[10]).toBeGreaterThan(result.series.taxImpact[5])
        expect(result.series.taxImpact[20]).toBeGreaterThan(result.series.taxImpact[10])
    })

    it('annual tax net worth is lower than exit tax net worth (tax drag)', () => {
        const exitResult = computePortfolioProjection(BASE_PARAMS)
        // Annual taxation creates drag — less compounding
        expect(result.finalValueAfterTax).toBeLessThan(exitResult.finalValueAfterTax)
    })

    it('taxes only year-over-year gains, not full balance', () => {
        // Year 1: gain = balance_after_growth - initial
        // Tax should be gain * 0.19, NOT balance * 0.19
        const effectiveReturn = 7 - 0.2
        const balanceAfterYear1 = 10000 * Math.pow(1 + effectiveReturn / 100, 1)
        const year1Gain = balanceAfterYear1 - 10000
        const year1Tax = year1Gain * 0.19
        expect(result.series.taxImpact[1]).toBeCloseTo(year1Tax, 0)
    })

    it('finalValueBeforeTax equals net worth plus cumulative tax', () => {
        expect(result.finalValueBeforeTax).toBeCloseTo(
            result.finalValueAfterTax + result.taxPaid, 2
        )
    })
})
```

---

### Task 4: Write tests for `computePortfolioExpectedReturn` and edge cases

**Files:**
- Modify: `webui/src/lib/simulators/portfolio.test.ts`

- [ ] **Step 1: Add expected return and edge case tests**

Append to `portfolio.test.ts`:

```ts
describe('computePortfolioExpectedReturn', () => {
    it('computes exit tax expected return', () => {
        const result = computePortfolioExpectedReturn(BASE_PARAMS)
        // effectiveReturn = 7 - 0.2 = 6.8
        // estimatedTaxDrag = (19/100) * 6.8 / 20 = 0.0646
        // expected = 6.8 - 0.0646 - 2 = 4.7354
        expect(result).toBeCloseTo(4.7354, 2)
    })

    it('computes annual tax expected return', () => {
        const params = { ...BASE_PARAMS, taxModel: 'annual' as const }
        const result = computePortfolioExpectedReturn(params)
        // estimatedTaxDrag = (19/100) * 6.8 = 1.292
        // expected = 6.8 - 1.292 - 2 = 3.508
        expect(result).toBeCloseTo(3.508, 2)
    })
})

describe('computePortfolioNetWorth20Y', () => {
    it('returns 21 inflation-adjusted values', () => {
        const { computePortfolioNetWorth20Y } = require('./portfolio')
        const values = computePortfolioNetWorth20Y(BASE_PARAMS)
        expect(values).toHaveLength(21)
        expect(values[0]).toBe(BASE_PARAMS.initialContribution)
        expect(values[20]).toBeGreaterThan(0)
        // Should be inflation-adjusted (less than nominal net worth)
        const projection = computePortfolioProjection(BASE_PARAMS)
        expect(values[20]).toBeLessThan(projection.series.netWorth[20])
    })
})

describe('edge cases', () => {
    it('handles zero growth', () => {
        const params = { ...BASE_PARAMS, growthRatePct: 0, expenseRatioPct: 0 }
        const result = computePortfolioProjection(params)
        expect(result.finalValueAfterTax).toBe(10000)
        expect(result.taxPaid).toBe(0)
    })

    it('handles negative effective return (expense > growth)', () => {
        const params = { ...BASE_PARAMS, growthRatePct: 0.1, expenseRatioPct: 0.5 }
        const result = computePortfolioProjection(params)
        expect(result.finalValueAfterTax).toBeLessThan(10000)
        expect(result.taxPaid).toBe(0) // no gains, no tax
    })

    it('handles zero initial contribution', () => {
        const params = { ...BASE_PARAMS, initialContribution: 0 }
        const result = computePortfolioProjection(params)
        expect(result.finalValueAfterTax).toBe(0)
    })

    it('defaults missing expenseRatioPct to 0', () => {
        const params = { ...BASE_PARAMS }
        delete (params as any).expenseRatioPct
        const result = computePortfolioProjection(params)
        // Without TER, effective = 7%, so balance > with TER
        const withTER = computePortfolioProjection(BASE_PARAMS)
        expect(result.finalValueAfterTax).toBeGreaterThan(withTER.finalValueAfterTax)
    })

    it('totalGain equals finalValueAfterTax + taxPaid - initial', () => {
        const result = computePortfolioProjection(BASE_PARAMS)
        expect(result.totalGain).toBeCloseTo(
            result.finalValueAfterTax + result.taxPaid - BASE_PARAMS.initialContribution, 2
        )
    })

    it('defaults missing taxModel to exit', () => {
        const params = { ...BASE_PARAMS }
        delete (params as any).taxModel
        const result = computePortfolioProjection(params)
        const exitResult = computePortfolioProjection({ ...BASE_PARAMS, taxModel: 'exit' })
        expect(result.finalValueAfterTax).toBeCloseTo(exitResult.finalValueAfterTax, 2)
    })
})
```

---

### Task 5: Rewrite `computePortfolioProjection` implementation

**Files:**
- Rewrite: `webui/src/lib/simulators/portfolio.ts`

- [ ] **Step 1: Rewrite the full file**

Replace `webui/src/lib/simulators/portfolio.ts` with:

```ts
import type { PortfolioSimulatorParams } from '@/lib/api/ToolsData'

const DURATION_YEARS = 20

export interface PortfolioProjectionSeries {
    totalInvested: number[]
    netWorth: number[]
    inflationAdjustedNetWorth: number[]
    totalGains: number[]
    taxImpact: number[]
    inflationAdjustedGains: number[]
}

export interface PortfolioProjection {
    years: number[]
    totalContributions: number
    finalValueBeforeTax: number
    finalValueAfterTax: number
    realFinalValue: number
    totalGain: number
    taxPaid: number
    inflationImpact: number
    inflationAdjustedGains: number
    series: PortfolioProjectionSeries
}

export function computePortfolioExpectedReturn(params: PortfolioSimulatorParams): number {
    const effectiveReturn = (params.growthRatePct ?? 0) - (params.expenseRatioPct ?? 0)
    const taxRate = (params.capitalGainTaxPct ?? 0) / 100
    const taxModel = params.taxModel ?? 'exit'
    const inflation = params.inflationPct ?? 0

    const estimatedTaxDrag = taxModel === 'exit'
        ? taxRate * effectiveReturn / DURATION_YEARS
        : taxRate * effectiveReturn

    return effectiveReturn - estimatedTaxDrag - inflation
}

export function computePortfolioProjection(params: PortfolioSimulatorParams): PortfolioProjection {
    const initial = params.initialContribution ?? 0
    const growthPct = params.growthRatePct ?? 0
    const expensePct = params.expenseRatioPct ?? 0
    const taxPct = params.capitalGainTaxPct ?? 0
    const taxModel = params.taxModel ?? 'exit'
    const inflationPct = params.inflationPct ?? 0

    const effectiveReturn = growthPct - expensePct
    const taxRate = taxPct / 100
    const inflationFactor = 1 + inflationPct / 100

    if (initial === 0) {
        const zeros = Array(DURATION_YEARS + 1).fill(0)
        return {
            years: Array.from({ length: DURATION_YEARS + 1 }, (_, i) => i),
            totalContributions: 0,
            finalValueBeforeTax: 0,
            finalValueAfterTax: 0,
            realFinalValue: 0,
            totalGain: 0,
            taxPaid: 0,
            inflationImpact: 0,
            inflationAdjustedGains: 0,
            series: {
                totalInvested: zeros,
                netWorth: zeros,
                inflationAdjustedNetWorth: zeros,
                totalGains: zeros,
                taxImpact: zeros,
                inflationAdjustedGains: zeros,
            },
        }
    }

    const monthlyRate = Math.pow(1 + effectiveReturn / 100, 1 / 12) - 1

    const yearLabels: number[] = [0]
    const balances: number[] = [initial]
    const cumulativeTaxes: number[] = [0]

    let balance = initial
    let cumulativeTax = 0

    for (let y = 1; y <= DURATION_YEARS; y++) {
        const balanceStartOfYear = balance

        // Compound monthly for 12 months
        for (let m = 0; m < 12; m++) {
            balance = balance * (1 + monthlyRate)
        }

        if (taxModel === 'annual') {
            const yearGain = balance - balanceStartOfYear
            if (yearGain > 0) {
                const tax = yearGain * taxRate
                cumulativeTax += tax
                balance -= tax
            }
        }

        yearLabels.push(y)
        balances.push(balance)
        cumulativeTaxes.push(cumulativeTax)
    }

    // For exit tax: apply at the end
    let finalValueBeforeTax = balance
    let finalValueAfterTax = balance
    if (taxModel === 'exit') {
        const totalGains = balance - initial
        if (totalGains > 0) {
            const exitTax = totalGains * taxRate
            cumulativeTax = exitTax
            finalValueAfterTax = balance - exitTax
            // Update last balance for chart
            balances[DURATION_YEARS] = finalValueAfterTax
            cumulativeTaxes[DURATION_YEARS] = exitTax
        }
    } else {
        finalValueBeforeTax = balance + cumulativeTax
        finalValueAfterTax = balance
    }

    const totalTaxPaid = cumulativeTax
    const realFinalValue = inflationPct > 0
        ? finalValueAfterTax / Math.pow(inflationFactor, DURATION_YEARS)
        : finalValueAfterTax
    const inflationImpact = finalValueAfterTax - realFinalValue
    const realCostBasis = inflationPct > 0
        ? initial / Math.pow(inflationFactor, DURATION_YEARS)
        : initial
    const inflationAdjustedGainsFinal = realFinalValue - realCostBasis

    // Build chart series
    const totalInvestedSeries = Array(DURATION_YEARS + 1).fill(initial)
    const netWorthSeries = balances
    const taxImpactSeries = cumulativeTaxes
    const totalGainsSeries = yearLabels.map((_, i) =>
        netWorthSeries[i] + cumulativeTaxes[i] - initial
    )
    const inflationAdjustedNetWorthSeries = yearLabels.map((y, i) =>
        inflationPct > 0 ? netWorthSeries[i] / Math.pow(inflationFactor, y) : netWorthSeries[i]
    )
    const realCostBasisSeries = yearLabels.map((y) =>
        inflationPct > 0 ? initial / Math.pow(inflationFactor, y) : initial
    )
    const inflationAdjustedGainsSeries = yearLabels.map((_, i) =>
        inflationAdjustedNetWorthSeries[i] - realCostBasisSeries[i]
    )

    return {
        years: yearLabels,
        totalContributions: initial,
        finalValueBeforeTax,
        finalValueAfterTax,
        realFinalValue,
        totalGain: finalValueAfterTax + totalTaxPaid - initial,
        taxPaid: totalTaxPaid,
        inflationImpact,
        inflationAdjustedGains: inflationAdjustedGainsFinal,
        series: {
            totalInvested: totalInvestedSeries,
            netWorth: netWorthSeries,
            inflationAdjustedNetWorth: inflationAdjustedNetWorthSeries,
            totalGains: totalGainsSeries,
            taxImpact: taxImpactSeries,
            inflationAdjustedGains: inflationAdjustedGainsSeries,
        },
    }
}

export function computePortfolioNetWorth20Y(params: PortfolioSimulatorParams): number[] {
    const result = computePortfolioProjection(params)
    return result.series.inflationAdjustedNetWorth
}
```

- [ ] **Step 2: Run all tests to verify they pass**

Run: `cd webui && npx vitest run src/lib/simulators/portfolio.test.ts 2>&1 | tail -30`
Expected: All tests PASS.

- [ ] **Step 3: Commit**

```bash
git add webui/src/lib/simulators/portfolio.ts webui/src/lib/simulators/portfolio.test.ts
git commit -m "feat: rewrite portfolio simulator with exit/annual tax models and expense ratio"
```

---

## Chunk 2: View Update

### Task 6: Update `PortfolioSimulatorView.vue`

**Files:**
- Modify: `webui/src/views/tools/PortfolioSimulatorView.vue`

- [ ] **Step 1: Update script — replace refs and param handling**

In `PortfolioSimulatorView.vue`, replace the form input refs (lines 27-32) with:

```js
const initialContribution = ref(10000)
const growthRatePct = ref(7)
const expenseRatioPct = ref(0.2)
const inflationPct = ref(2)
const capitalGainTaxPct = ref(19)
const taxModel = ref('exit')
```

Update `getCurrentParams()` (lines 40-49) to:

```js
function getCurrentParams() {
    return {
        initialContribution: initialContribution.value,
        growthRatePct: growthRatePct.value,
        expenseRatioPct: expenseRatioPct.value,
        inflationPct: inflationPct.value,
        capitalGainTaxPct: capitalGainTaxPct.value,
        taxModel: taxModel.value,
    }
}
```

Update `loadCaseData` (lines 71-83) to:

```js
function loadCaseData(cs) {
    const p = cs.params
    if (p) {
        initialContribution.value = p.initialContribution ?? initialContribution.value
        growthRatePct.value = p.growthRatePct ?? growthRatePct.value
        expenseRatioPct.value = p.expenseRatioPct ?? 0
        inflationPct.value = p.inflationPct ?? inflationPct.value
        capitalGainTaxPct.value = p.capitalGainTaxPct ?? capitalGainTaxPct.value
        taxModel.value = p.taxModel ?? 'exit'
    }
    activeCaseName.value = cs.name
    activeCaseDescription.value = cs.description ?? ''
}
```

- [ ] **Step 2: Update the summary results section**

In the template, replace `projection.finalValueAfterTax` reference (line 308) — it stays the same.

Replace `projection.totalContributions` (line 304) — stays the same (field name unchanged).

The result section already displays the correct fields. Just verify that `projection.finalValue` is NOT referenced (it was removed from the interface). If it is, replace with `projection.finalValueAfterTax`.

- [ ] **Step 3: Update template — replace parameter form fields**

Replace the entire `<div class="form-grid">` content (lines 199-289) with:

```html
<div class="form-grid">
    <div class="field">
        <label for="initial">Initial contribution</label>
        <div class="field-controls">
            <InputNumber
                id="initial"
                v-model="initialContribution"
                :min="0"
                :max="1000000"
                mode="decimal"
                :minFractionDigits="0"
                :maxFractionDigits="0"
                class="field-input"
            />
            <Slider v-model="initialContribution" :min="0" :max="1000000" :step="5000" class="field-slider" />
        </div>
    </div>
    <div class="field">
        <label for="growth">Annual growth rate (%)</label>
        <div class="field-controls">
            <InputNumber
                id="growth"
                v-model="growthRatePct"
                :min="0"
                :max="100"
                :minFractionDigits="1"
                :maxFractionDigits="2"
                class="field-input"
            />
            <Slider v-model="growthRatePct" :min="0" :max="30" :step="0.5" class="field-slider" />
        </div>
    </div>
    <div class="field">
        <label for="expense">Expense ratio / TER (%)</label>
        <div class="field-controls">
            <InputNumber
                id="expense"
                v-model="expenseRatioPct"
                :min="0"
                :max="5"
                :minFractionDigits="2"
                :maxFractionDigits="2"
                class="field-input"
            />
            <Slider v-model="expenseRatioPct" :min="0" :max="5" :step="0.05" class="field-slider" />
        </div>
    </div>
    <div class="field">
        <label for="inflation">Inflation (%)</label>
        <div class="field-controls">
            <InputNumber
                id="inflation"
                v-model="inflationPct"
                :min="0"
                :max="50"
                :minFractionDigits="1"
                :maxFractionDigits="2"
                class="field-input"
            />
            <Slider v-model="inflationPct" :min="0" :max="15" :step="0.5" class="field-slider" />
        </div>
    </div>
    <div class="field">
        <label for="tax">Capital gain tax (%)</label>
        <div class="field-controls">
            <InputNumber
                id="tax"
                v-model="capitalGainTaxPct"
                :min="0"
                :max="100"
                :minFractionDigits="1"
                :maxFractionDigits="2"
                class="field-input"
            />
            <Slider v-model="capitalGainTaxPct" :min="0" :max="50" :step="1" class="field-slider" />
        </div>
    </div>
    <div class="field">
        <label>Tax model</label>
        <SelectButton
            v-model="taxModel"
            :options="[
                { label: 'Exit Tax', value: 'exit' },
                { label: 'Annual Tax', value: 'annual' },
            ]"
            optionLabel="label"
            optionValue="value"
        />
    </div>
</div>
```

- [ ] **Step 4: Add SelectButton import**

Add to the imports at the top of `<script setup>`:

```js
import SelectButton from 'primevue/selectbutton'
```

- [ ] **Step 5: Verify it compiles**

Run: `cd webui && npx vue-tsc --noEmit 2>&1 | tail -20`
Expected: No errors.

- [ ] **Step 6: Run all tests**

Run: `cd webui && npx vitest run 2>&1 | tail -20`
Expected: All tests pass.

- [ ] **Step 7: Update mock data in ToolsData.test.ts**

In `webui/src/lib/api/ToolsData.test.ts`, update the `mockCase` params (line 17) from:

```ts
params: { durationYears: 20, growthRatePct: 6 },
```

to:

```ts
params: { initialContribution: 10000, growthRatePct: 7, expenseRatioPct: 0.2, inflationPct: 2, capitalGainTaxPct: 19, taxModel: 'exit' },
```

- [ ] **Step 8: Run all tests**

Run: `cd webui && npx vitest run 2>&1 | tail -20`
Expected: All tests pass.

- [ ] **Step 9: Commit**

```bash
git add webui/src/views/tools/PortfolioSimulatorView.vue webui/src/lib/api/ToolsData.test.ts
git commit -m "feat: update portfolio simulator UI for ROI comparison redesign"
```
