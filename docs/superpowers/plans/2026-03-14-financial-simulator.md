# Financial Simulator Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge Portfolio Simulator and Real Estate Simulator into a unified Financial Simulator page with a comparison chart and simulation list.

**Architecture:** Extract computation logic from both simulator views into shared utility functions. Create a new list+chart page at `/tools`. Refactor existing simulator views into case editors loaded by route param `:id`. No backend changes.

**Tech Stack:** Vue 3, TypeScript, PrimeVue (DataTable, Dialog, Select), ECharts, TanStack Query

---

## File Structure

### New Files
| File | Responsibility |
|------|---------------|
| `webui/src/lib/simulators/portfolio.ts` | Portfolio projection computation (extracted from PortfolioSimulatorView) |
| `webui/src/lib/simulators/realEstate.ts` | Real estate projection computation (extracted from RealEstateSimulatorView) |
| `webui/src/lib/simulators/projection.ts` | Unified 20-year net worth projection dispatcher (calls portfolio or real estate based on toolType) |
| `webui/src/views/tools/FinancialSimulatorView.vue` | New list + comparison chart page |
| `webui/src/views/tools/SimulationEditorView.vue` | Thin wrapper that routes to correct simulator by toolType |

### Modified Files
| File | Changes |
|------|---------|
| `webui/src/views/tools/PortfolioSimulatorView.vue` | Import extracted computation; load case from route param; add back button; remove standalone case listing UI |
| `webui/src/views/tools/RealEstateSimulatorView.vue` | Same as above |
| `webui/src/router/index.ts` | Replace tool routes with `/tools` and `/tools/:toolType/:id` |
| `webui/src/components/SidebarMenu.vue` | Replace two tool entries with single "Financial Simulator" entry |
| `webui/src/lib/api/ToolsData.ts` | No changes expected (existing API is sufficient) |

### Deleted Files
| File | Reason |
|------|--------|
| `webui/src/views/tools/ToolsView.vue` | Replaced by FinancialSimulatorView |

---

## Chunk 1: Extract Computation Utilities

### Task 1: Extract portfolio projection logic

**Files:**
- Create: `webui/src/lib/simulators/portfolio.ts`
- Modify: `webui/src/views/tools/PortfolioSimulatorView.vue`
- Reference: `webui/src/lib/api/ToolsData.ts` (for `PortfolioSimulatorParams` type)

The portfolio computation is currently an inline `computed` property at lines 146-256 of PortfolioSimulatorView.vue. Extract it into a pure function.

- [ ] **Step 1: Create `portfolio.ts` with the extracted computation**

Create `webui/src/lib/simulators/portfolio.ts`. The function takes `PortfolioSimulatorParams` and returns the projection data (years array, series, summary metrics). Copy the exact logic from the `projection` computed in PortfolioSimulatorView.vue (lines 146-256). The function signature:

```typescript
import type { PortfolioSimulatorParams } from '@/lib/api/ToolsData'

export interface PortfolioProjection {
  years: number[]
  totalContributions: number
  finalValue: number
  finalValueAfterTax: number
  realFinalValue: number
  totalGain: number
  taxPaid: number
  inflationImpact: number
  inflationAdjustedGains: number
  series: {
    invested: number[]
    netWorth: number[]
    inflationAdjustedNetWorth: number[]
    totalGains: number[]
    inflationAdjustedGains: number[]
    taxImpact: number[]
  }
}

export function computePortfolioProjection(params: PortfolioSimulatorParams): PortfolioProjection
```

- [ ] **Step 2: Add a convenience function for 20-year net worth series**

In the same file, add a function that always projects 20 years regardless of the params' `durationYears`:

```typescript
export function computePortfolioNetWorth20Y(params: PortfolioSimulatorParams): number[] {
  const projection = computePortfolioProjection({ ...params, durationYears: 20 })
  return projection.series.inflationAdjustedNetWorth
}
```

- [ ] **Step 3: Add expected annual return computation**

```typescript
export function computePortfolioExpectedReturn(params: PortfolioSimulatorParams): number {
  return params.growthRatePct - params.capitalGainTaxPct - params.inflationPct
}
```

- [ ] **Step 4: Update PortfolioSimulatorView to import the extracted function**

In PortfolioSimulatorView.vue, replace the inline `projection` computed body with a call to `computePortfolioProjection()`. The computed property stays but delegates:

```typescript
import { computePortfolioProjection, computePortfolioExpectedReturn } from '@/lib/simulators/portfolio'

const projection = computed(() => computePortfolioProjection({
  durationYears: durationYears.value,
  initialContribution: initialContribution.value,
  monthlyContribution: monthlyContribution.value,
  growthRatePct: growthRatePct.value,
  inflationPct: inflationPct.value,
  capitalGainTaxPct: capitalGainTaxPct.value,
}))
```

Also update `expectedAnnualReturn` computation to use `computePortfolioExpectedReturn()`.

- [ ] **Step 5: Verify the portfolio simulator still works**

Run: `cd webui && npm run build`
Expected: No build errors. Open the portfolio simulator page and verify the chart and metrics render correctly.

- [ ] **Step 6: Commit**

```bash
git add webui/src/lib/simulators/portfolio.ts webui/src/views/tools/PortfolioSimulatorView.vue
git commit -m "refactor: extract portfolio projection into shared utility"
```

---

### Task 2: Extract real estate projection logic

**Files:**
- Create: `webui/src/lib/simulators/realEstate.ts`
- Modify: `webui/src/views/tools/RealEstateSimulatorView.vue`
- Reference: `webui/src/lib/api/ToolsData.ts` (for `RealEstateSimulatorParams` type)

The real estate computation spans multiple computed properties in RealEstateSimulatorView.vue. Extract the key computations: mortgage payment calculation, rentability metrics, amortization schedule, and chart projection.

- [ ] **Step 1: Create `realEstate.ts` with extracted computations**

Create `webui/src/lib/simulators/realEstate.ts`. Extract these functions from the view:

```typescript
import type { RealEstateSimulatorParams } from '@/lib/api/ToolsData'

// From lines 138-159
export function calcMonthlyPayment(principal: number, annualRate: number, termYears: number, amortize: boolean): number

export function calcTotalInterest(principal: number, annualRate: number, termYears: number, amortize: boolean): number

// Amortization schedule from lines 268-343
export interface AmortizationYear {
  year: number
  beginningBalance: number
  endingBalance: number
  interestPaid: number
  principalPaid: number
  // per-mortgage breakdown
  mortgages: Array<{
    beginningBalance: number
    endingBalance: number
    interestPaid: number
    principalPaid: number
  }>
}
export function computeAmortizationSchedule(params: RealEstateSimulatorParams): AmortizationYear[]

// Chart projection from lines 346-435 — returns year-by-year series
export interface RealEstateProjection {
  years: number[]
  propertyEquity: number[]
  remainingMortgage: number[]
  cumulativeInterest: number[]
  cumulativeCashFlow: number[]
}
export function computeRealEstateProjection(params: RealEstateSimulatorParams): RealEstateProjection
```

Copy the exact logic from the corresponding computed properties. **Important:** These are pure functions that must derive all intermediate values (totalEquity, totalMortgageNeeded, mortgagePrincipal per mortgage, maxTerm, incidentalCost, totalRecurringCosts, annualRent) internally from the raw params — do not rely on Vue refs. Keep the same field names used in the existing code (e.g., `totalBeginning`, `totalInterest`, `totalPrincipal`, `totalEnding` for amortization; `yearLabels` for years) to avoid breaking template references.

- [ ] **Step 2: Add a convenience function for 20-year net worth series**

Per the spec, net worth = property equity + cumulative net rental cash flow. The projection must extend to 20 years even if mortgages are shorter — continue projecting appreciation and cash flow beyond the mortgage term.

```typescript
export function computeRealEstateNetWorth20Y(params: RealEstateSimulatorParams): number[] {
  const projection = computeRealEstateProjection(params)
  // Net worth = equity + cumulative cash flow (per spec)
  const netWorth = projection.propertyEquity.map((eq, i) => eq + projection.cumulativeCashFlow[i])
  // Pad to exactly 21 entries (years 0-20), continuing appreciation if projection is shorter
  const result = netWorth.slice(0, 21)
  while (result.length < 21) {
    // Continue projecting: appreciation on market value, no more mortgage payments
    const lastYear = result.length - 1
    const growthRate = (params.housingPriceIncreasePct ?? 0) / 100
    const annualCashFlow = params.monthlyRent * 12 - params.propertyTax - params.insurance - params.maintenance - params.otherCosts
    result.push(result[lastYear] + result[lastYear] * growthRate / (lastYear || 1) + annualCashFlow)
  }
  return result
}
```

Note: The padding logic above is approximate. The implementer should review and align with the actual projection formula for consistency.

- [ ] **Step 3: Add expected annual return computation**

```typescript
export function computeRealEstateExpectedReturn(params: RealEstateSimulatorParams): number {
  const annualRent = params.monthlyRent * 12
  const totalRecurringCosts = params.propertyTax + params.insurance + params.maintenance + params.otherCosts
  const noi = annualRent - totalRecurringCosts
  return params.marketValue > 0 ? (noi / params.marketValue) * 100 : 0
}
```

- [ ] **Step 4: Update RealEstateSimulatorView to import the extracted functions**

In RealEstateSimulatorView.vue, replace the inline computations with calls to the extracted functions. The computed properties stay but delegate to the utility functions, passing the current form values as a params object.

- [ ] **Step 5: Verify the real estate simulator still works**

Run: `cd webui && npm run build`
Expected: No build errors. Open the real estate simulator page and verify the chart, metrics, and amortization schedule render correctly.

- [ ] **Step 6: Commit**

```bash
git add webui/src/lib/simulators/realEstate.ts webui/src/views/tools/RealEstateSimulatorView.vue
git commit -m "refactor: extract real estate projection into shared utility"
```

---

### Task 3: Create unified projection dispatcher

**Files:**
- Create: `webui/src/lib/simulators/projection.ts`

- [ ] **Step 1: Create `projection.ts`**

```typescript
import type { CaseStudy, PortfolioSimulatorParams, RealEstateSimulatorParams } from '@/lib/api/ToolsData'
import { computePortfolioNetWorth20Y } from './portfolio'
import { computeRealEstateNetWorth20Y } from './realEstate'

export function computeNetWorth20Y(caseStudy: CaseStudy): number[] {
  switch (caseStudy.toolType) {
    case 'portfolio-simulator':
      return computePortfolioNetWorth20Y(caseStudy.params as PortfolioSimulatorParams)
    case 'real-estate-simulator':
      return computeRealEstateNetWorth20Y(caseStudy.params as RealEstateSimulatorParams)
    default:
      return Array(21).fill(0)
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add webui/src/lib/simulators/projection.ts
git commit -m "feat: add unified 20-year projection dispatcher"
```

---

## Chunk 2: Financial Simulator List Page

### Task 4: Create FinancialSimulatorView with simulations list

**Files:**
- Create: `webui/src/views/tools/FinancialSimulatorView.vue`
- Delete: `webui/src/views/tools/ToolsView.vue`

- [ ] **Step 1: Create the view with the simulations DataTable**

Create `webui/src/views/tools/FinancialSimulatorView.vue`. This is the main page at `/tools`. It fetches cases from both tool types in parallel, merges them, and displays a PrimeVue DataTable.

Template structure:
```
<div class="financial-simulator">
  <!-- Comparison chart placeholder (Task 7) -->
  <Card class="mb-4">
    <template #content>
      <div v-if="allCases.length === 0" class="text-center p-4 text-color-secondary">
        No simulations yet. Add one to get started.
      </div>
      <div v-else ref="chartRef" style="height: 400px"></div>
    </template>
  </Card>

  <!-- Simulations list -->
  <Card>
    <template #title>
      <div class="flex justify-between items-center">
        <span>Simulations</span>
        <Button label="Add Simulation" icon="pi pi-plus" @click="showAddDialog = true" />
      </div>
    </template>
    <template #content>
      <DataTable :value="allCases" stripedRows>
        <Column field="name" header="Name" />
        <Column header="Type">
          <template #body="{ data }">
            <i :class="typeIcon(data.toolType)" class="mr-2" />
            <span>{{ typeLabel(data.toolType) }}</span>
          </template>
        </Column>
        <Column header="Expected Return">
          <template #body="{ data }">
            {{ data.expectedAnnualReturn.toFixed(2) }}%
          </template>
        </Column>
        <Column header="Attachment">
          <template #body="{ data }">
            <i v-if="data.attachmentId" class="pi pi-paperclip" />
          </template>
        </Column>
        <Column header="Actions">
          <template #body="{ data }">
            <Button icon="pi pi-pencil" text @click="editCase(data)" />
            <Button icon="pi pi-copy" text @click="duplicateCase(data)" />
            <Button icon="pi pi-trash" text severity="danger" @click="confirmDelete(data)" />
          </template>
        </Column>
      </DataTable>
    </template>
  </Card>

  <!-- Add dialog -->
  <Dialog v-model:visible="showAddDialog" header="Add Simulation" modal>
    <div class="flex flex-col gap-4">
      <div>
        <label>Name</label>
        <InputText v-model="newName" class="w-full" />
      </div>
      <div>
        <label>Description</label>
        <InputText v-model="newDescription" class="w-full" />
      </div>
      <div>
        <label>Type</label>
        <Select v-model="newToolType" :options="toolTypeOptions" optionLabel="label" optionValue="value" class="w-full" />
      </div>
    </div>
    <template #footer>
      <Button label="Cancel" text @click="showAddDialog = false" />
      <Button label="Create" @click="handleCreate" :disabled="!newName || !newToolType" />
    </template>
  </Dialog>
</div>
```

Script setup:
```typescript
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { listCases, createCase, deleteCase } from '@/lib/api/ToolsData'
import type { CaseStudy } from '@/lib/api/ToolsData'

const router = useRouter()

// Fetch both types in parallel
const portfolioCases = ref<CaseStudy[]>([])
const realEstateCases = ref<CaseStudy[]>([])

async function fetchAll() {
  const [p, r] = await Promise.all([
    listCases('portfolio-simulator'),
    listCases('real-estate-simulator'),
  ])
  portfolioCases.value = p
  realEstateCases.value = r
}
fetchAll()

const allCases = computed(() =>
  [...portfolioCases.value, ...realEstateCases.value]
    .sort((a, b) => a.id - b.id)
)

// Type helpers
const toolTypeOptions = [
  { label: 'Portfolio Simulator', value: 'portfolio-simulator' },
  { label: 'Real Estate Simulator', value: 'real-estate-simulator' },
]
function typeIcon(toolType: string) {
  return toolType === 'portfolio-simulator' ? 'pi pi-chart-pie' : 'pi pi-home'
}
function typeLabel(toolType: string) {
  return toolTypeOptions.find(o => o.value === toolType)?.label ?? toolType
}

// Add dialog
const showAddDialog = ref(false)
const newName = ref('')
const newDescription = ref('')
const newToolType = ref('portfolio-simulator')

async function handleCreate() {
  const cs = await createCase(newToolType.value, {
    name: newName.value,
    description: newDescription.value,
    expectedAnnualReturn: 0,
    params: {},
  })
  showAddDialog.value = false
  newName.value = ''
  newDescription.value = ''
  router.push(`/tools/${cs.toolType}/${cs.id}`)
}

// Edit
function editCase(cs: CaseStudy) {
  router.push(`/tools/${cs.toolType}/${cs.id}`)
}

// Duplicate
async function duplicateCase(cs: CaseStudy) {
  let copyName = cs.name + ' (copy)'
  // Check for existing copies and increment
  const existing = allCases.value.filter(c => c.name.startsWith(cs.name + ' (copy'))
  if (existing.some(c => c.name === copyName)) {
    let i = 2
    while (existing.some(c => c.name === `${cs.name} (copy ${i})`)) i++
    copyName = `${cs.name} (copy ${i})`
  }
  await createCase(cs.toolType, {
    name: copyName,
    description: cs.description,
    expectedAnnualReturn: cs.expectedAnnualReturn,
    params: cs.params,
  })
  await fetchAll()
}

// Delete
async function confirmDelete(cs: CaseStudy) {
  // Use PrimeVue ConfirmDialog or simple confirm()
  if (confirm(`Delete "${cs.name}"?`)) {
    await deleteCase(cs.toolType, cs.id)
    await fetchAll()
  }
}
```

- [ ] **Step 2: Verify it builds**

Run: `cd webui && npm run build`
Expected: No build errors.

- [ ] **Step 3: Commit**

```bash
git add webui/src/views/tools/FinancialSimulatorView.vue
git rm webui/src/views/tools/ToolsView.vue
git commit -m "feat: add Financial Simulator list page"
```

---

### Task 5: Update routing

**Files:**
- Modify: `webui/src/router/index.ts` (lines 111-129)

- [ ] **Step 1: Replace tool routes**

Replace ALL existing tools routes (lines 111-129), including the `/tools` redirect and both simulator routes, with:

```typescript
{
  path: '/tools',
  name: 'financial-simulator',
  meta: { requiresAuth: true },
  component: () => import('@/views/tools/FinancialSimulatorView.vue'),
},
{
  path: '/tools/:toolType/:id',
  name: 'simulation-editor',
  meta: { requiresAuth: true },
  component: () => import('@/views/tools/SimulationEditorView.vue'),
},
```

Note: `SimulationEditorView.vue` is a thin wrapper that loads the correct simulator based on `:toolType` — see Task 6.

- [ ] **Step 2: Commit**

```bash
git add webui/src/router/index.ts
git commit -m "feat: update tools routing for financial simulator"
```

---

### Task 6: Create SimulationEditorView wrapper

**Files:**
- Create: `webui/src/views/tools/SimulationEditorView.vue`
- Modify: `webui/src/views/tools/PortfolioSimulatorView.vue`
- Modify: `webui/src/views/tools/RealEstateSimulatorView.vue`

The editor view is a thin wrapper that reads `:toolType` and `:id` from the route and renders the correct simulator component, passing the case ID as a prop.

- [ ] **Step 1: Create SimulationEditorView.vue**

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import PortfolioSimulatorView from './PortfolioSimulatorView.vue'
import RealEstateSimulatorView from './RealEstateSimulatorView.vue'

const route = useRoute()
const toolType = computed(() => route.params.toolType as string)
const caseId = computed(() => Number(route.params.id))
</script>

<template>
  <PortfolioSimulatorView v-if="toolType === 'portfolio-simulator'" :caseId="caseId" />
  <RealEstateSimulatorView v-else-if="toolType === 'real-estate-simulator'" :caseId="caseId" />
  <div v-else>Unknown simulation type: {{ toolType }}</div>
</template>
```

- [ ] **Step 2: Modify PortfolioSimulatorView to accept `caseId` prop and auto-load**

Add prop definition and auto-load logic. Remove the standalone case listing dialog (case listing is now on the parent page). Add a back button at the top.

Changes needed:
- **Note:** PortfolioSimulatorView.vue uses `<script setup>` without `lang="ts"`. Either add `lang="ts"` to the script tag, or use runtime props syntax: `defineProps({ caseId: { type: Number, required: true } })`
- Add `getCase` to imports from `@/lib/api/ToolsData` (currently not imported)
- On mount, fetch the case via `getCase('portfolio-simulator', props.caseId)` and populate form fields. If params is empty (new case created from list page), keep default values.
- Add a back button: `<Button icon="pi pi-arrow-left" label="Back" text @click="router.push('/tools')" />`
- Remove these items (case management moved to list page):
  - `showCasesDialog` ref and the "Scenarios" DataTable dialog
  - `showSaveDialog` ref and the "Save as New Scenario" dialog
  - `loadCases()` function and `cases` ref
  - `clearActiveCase()` and `removeScenario()` functions
  - The "Scenarios" button in the title bar
- Reuse `loadCase()` logic for auto-loading from the `getCase` response
- Keep the "Save" button which now always calls `updateCase()` with the `caseId` prop (never `createCase` — creation happens from the list page)
- Update `expectedAnnualReturn` on save using `computePortfolioExpectedReturn()`

- [ ] **Step 3: Modify RealEstateSimulatorView similarly**

Same changes as Step 2 but for the real estate view (this one already has `lang="ts"`):
- Add `caseId` prop via `defineProps<{ caseId: number }>()`
- Add `getCase` to imports from `@/lib/api/ToolsData`
- Auto-load case on mount, handle empty params gracefully
- Add back button
- Remove standalone case listing dialog and related refs/functions
- Keep save (as `updateCase` only) and attachment functionality

- [ ] **Step 4: Verify both editors work**

Run: `cd webui && npm run build`
Expected: No build errors.

- [ ] **Step 5: Commit**

```bash
git add webui/src/views/tools/SimulationEditorView.vue webui/src/views/tools/PortfolioSimulatorView.vue webui/src/views/tools/RealEstateSimulatorView.vue
git commit -m "feat: refactor simulator views as case editors with route-based loading"
```

---

### Task 7: Update sidebar menu

**Files:**
- Modify: `webui/src/components/SidebarMenu.vue` (lines 137-152)

- [ ] **Step 1: Replace tools menu entries**

Replace lines 137-152 with a single entry:

```html
<!-- TOOLS SECTION -->
<li class="menu-section">
  <div class="menu-section-label">Tools</div>
</li>
<li>
  <router-link to="/tools" class="menu-item">
    <i class="pi pi-calculator menu-icon"></i>
    <span class="menu-label">Financial Simulator</span>
  </router-link>
</li>
```

- [ ] **Step 2: Commit**

```bash
git add webui/src/components/SidebarMenu.vue
git commit -m "feat: single Financial Simulator sidebar entry"
```

---

## Chunk 3: Comparison Chart

### Task 8: Add comparison chart to FinancialSimulatorView

**Files:**
- Modify: `webui/src/views/tools/FinancialSimulatorView.vue`
- Reference: `webui/src/lib/simulators/projection.ts`

- [ ] **Step 1: Add the ECharts comparison chart**

**Important:** The codebase uses `vue-echarts` (`VChart` component with `autoresize` prop) everywhere. Do NOT use raw `echarts.init()`. Follow the existing pattern.

Update the template's chart placeholder to use `VChart`:

```vue
<VChart v-if="allCases.length > 0" :option="chartOptions" autoresize style="height: 400px" />
```

Add the computed chart options in the script:

```typescript
import { computeNetWorth20Y } from '@/lib/simulators/projection'
import VChart from 'vue-echarts'
// Import ECharts modules matching the pattern in existing simulator views

const chartOptions = computed(() => {
  const years = Array.from({ length: 21 }, (_, i) => `Year ${i}`)
  const series = allCases.value.map(cs => ({
    name: cs.name,
    type: 'line' as const,
    data: computeNetWorth20Y(cs),
    smooth: true,
  }))

  return {
    tooltip: { trigger: 'axis' },
    legend: { data: allCases.value.map(cs => cs.name) },
    xAxis: { type: 'category', data: years },
    yAxis: {
      type: 'value',
      axisLabel: { formatter: (v: number) => formatShortCurrency(v) },
    },
    series,
  }
})
```

Define `formatShortCurrency` locally (or extract from existing views — both simulators define a similar formatter for abbreviating large numbers as k/M).

- [ ] **Step 3: Verify the chart renders**

Run: `cd webui && npm run build`
Expected: No build errors. The comparison chart should render with lines for each saved simulation.

- [ ] **Step 4: Commit**

```bash
git add webui/src/views/tools/FinancialSimulatorView.vue
git commit -m "feat: add 20-year comparison chart to Financial Simulator"
```

---

## Chunk 4: Polish and Integration Testing

### Task 9: End-to-end manual verification

- [ ] **Step 1: Start the app and verify the full flow**

Run: `cd webui && npm run dev` (and backend if needed)

Verify:
1. Sidebar shows single "Financial Simulator" entry
2. `/tools` shows the list page with chart area and empty state
3. "Add Simulation" opens dialog with name, description, type fields
4. Creating a portfolio simulation navigates to the editor with back button
5. Saving the simulation and going back shows it in the list
6. Creating a real estate simulation works the same way
7. Both simulations appear as lines in the comparison chart
8. Duplicate creates a copy with "(copy)" suffix
9. Delete removes the simulation after confirmation
10. Edit navigates to the correct editor

- [ ] **Step 2: Commit any fixes**

```bash
git add -u
git commit -m "fix: polish financial simulator integration"
```
