# Frontend CSS simplification strategy

This document summarizes the CSS audit of the webui (Vue 3 + PrimeVue 4 + PrimeFlex) and a strategy to reduce duplication, rely on PrimeVue/PrimeFlex where possible, and centralize shared styles in common SCSS for long-term maintainability.

## 1. What counts as “valid” component CSS (keep in components)

Per project convention, **valid** in-component CSS is:

- **Unscoped** style blocks that support **structure/layout** of the component (e.g. flex/grid containers, overflow, positioning), and do **not** define look-and-feel (colors, typography, borders, shadows).
- **Scoped** rules that are truly component-specific structure (e.g. a unique layout that does not repeat elsewhere).

Examples of valid component CSS:

- `App.vue`: `.content { display: flex; height: 100% }` — layout only, unscoped.
- Unscoped structural helpers for a single component’s DOM shape.

Everything that is **cosmetic** (colors, spacing that repeats, typography) or **duplicated** across components should move to common SCSS or be replaced by PrimeFlex/PrimeVue.

---

## 2. Current state: where styles live

| Location | Role |
|----------|------|
| `src/assets/style.scss` | Entry: gutter for PrimeFlex, imports `_main`, `_topbar` |
| `src/assets/scss/_main.scss` | Base (body, html, form labels), `.main-app-content`, `.entry-dialog`, PrimeVue input/dropdown focus overrides |
| `src/assets/scss/_variables.scss` | Scale, borderRadius, transitionDuration, mainContentWidth |
| `src/assets/scss/_mixins.scss` | `focused()`, `focused-inset()` |
| `src/assets/scss/_menu.scss` | Layout sidebar/menu (layout-sidebar, layout-menu, submenu transitions) |
| `src/assets/scss/_topbar.scss` | Layout topbar (fixed header, logo, buttons, mobile) |
| Component `<style scoped>` | Many repeated patterns + some component-specific layout/cosmetic |
| Component `<style>` (unscoped) | Footer, AppSelector, IconSelect, loadingScreen — mix of structure and look-and-feel |

PrimeFlex is already imported in `main.js` (`primeflex/primeflex.css`), so utility classes (flex, gap, padding, margin, etc.) are available everywhere.

---

## 3. Duplicated patterns and recommended treatment

### 3.1 Page/view layout (padding, max-width, title area)

Repeated in: `accounts.vue`, `InstrumentsView.vue`, `BackupRestoreView.vue`, `CsvImportProfileView.vue`, etc.

- **Pattern:**  
  - View wrapper: `padding: 2rem`, sometimes `max-width: 1400px; margin: 0 auto`.  
  - Header row: `display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem`.  
  - Title: `margin: 0` on `h1`.

- **Recommendation:**  
  - Use **PrimeFlex** on the template: e.g. `class="p-4"` (or a shared class for 2rem), `class="flex justify-content-between align-items-center mb-4"` for the header row, and `class="m-0"` for the title.  
  - Optionally add a single **common SCSS** class for the “view container” (e.g. `.view-container { padding: 2rem; max-width: 1400px; margin: 0 auto; }`) and use it on the root of each view to avoid repeating max-width logic.

### 3.2 Action buttons row (`.actions`, `.action-button`)

Repeated in: `accounts.vue`, `EntriesTable.vue`, `AccountEntriesTable.vue`, `CategoriesView.vue`, `InstrumentsView.vue`, `CurrencyExchangeView.vue`, `StockMarketView.vue`, `CsvImportProfileView.vue`, `CsvHeaderEditor.vue` (as `.header-actions` / `.action-buttons`).

- **Pattern:**  
  - Container: `display: flex; gap: 0.5rem` (sometimes `justify-content: flex-end` or `flex-start`).  
  - Button wrapper: `padding: 0.25rem`.

- **Recommendation:**  
  - **Remove** custom `.actions` / `.action-button` and use **PrimeFlex**: e.g. `class="flex gap-2 justify-content-end"` (or `justify-content-start`) and `class="p-1"` where a tighter button pad is desired.  
  - If you want one semantic name, add **one** utility in common SCSS, e.g. `.actions-row { display: flex; gap: 0.5rem; }` and combine with PrimeFlex for alignment; then delete all per-component `.actions` / `.action-button` definitions.

### 3.3 DataTable overrides (`:deep(.p-datatable...)`)

Repeated in: `EntriesTable.vue`, `AccountEntriesTable.vue`, `InstrumentsView.vue`, `CurrencyExchangeView.vue`, `StockMarketView.vue`, `AccountTypesList.vue`, `CsvHeaderEditor.vue`, `TasksView.vue`.

- **Patterns:**  
  - Tighter body cells: `:deep(.p-datatable-tbody > tr > td) { padding-top: 0; padding-bottom: 0; }`  
  - Row hover: `:deep(.p-datatable .p-datatable-tbody > tr:hover) { background-color: rgba(0,0,0,0.1) !important; }`  
  - Right-align column header: `:deep(.amount-column .p-datatable-column-title), :deep(.balance-column .p-datatable-column-title) { margin-left: auto; }`

- **Recommendation:**  
  - Move **global** DataTable tweaks to **common SCSS** (e.g. a `_datatable.scss` partial) so all tables share the same compact padding and hover.  
  - Keep **only** table-specific overrides in the component (e.g. TasksView clickable row styling) as scoped or unscoped structural rules.

### 3.4 Amount / balance semantic colors

Repeated in: `EntriesTable.vue`, `AccountEntriesTable.vue` (and conceptually similar in reports).

- **Pattern:**  
  - `.amount.expense`, `.amount.income`, `.amount.transfer`, `.amount.stock-trade .stock-trade-total.buy/sell`, `.amount.amount-positive`, `.amount.amount-negative`, `.balance-negative` using `var(--c-red-600)`, `var(--c-green-600)`, `var(--c-blue-600)`.

- **Recommendation:**  
  - Move to **common SCSS** (e.g. `_amounts.scss` or inside `_main.scss`) as global semantic classes so any table or view can use `.amount.expense`, `.amount.income`, etc. without redefining.  
  - Ensures consistent finance semantics app-wide.

### 3.5 Sidebar menu (SidebarMenu.vue)

Current: large block of **scoped** styles (colors, hover, active state, typography, spacers, animations).

- **Recommendation:**  
  - Treat as **layout/shell** styling. Move to a new partial, e.g. `_sidebar.scss`, and import in `style.scss`.  
  - Use a single **wrapper class** on the sidebar root (e.g. `.app-sidebar`) and scope all rules under it (no Vue scoped).  
  - Keeps the component focused on structure/template; look-and-feel lives in one place and is easier to theme.

### 3.6 Topbar (topbar.vue)

Current: scoped overrides for background, Menubar, hamburger icon color.

- **Recommendation:**  
  - Prefer **theme/topbar** in common SCSS: e.g. extend `_topbar.scss` (or a small `_app-topbar.scss`) with app-specific topbar background and Menubar border/padding.  
  - Use a single class on the topbar root (e.g. `.app-topbar`) and keep only minimal, truly component-specific structure in the Vue file (or remove scoped styles entirely if everything moves to SCSS).

### 3.7 Entry dialogs and “entry-dialog” class

- **Already centralized:** `.entry-dialog` in `_main.scss` (width, max-width variants, full-width inputs).  
- **Recommendation:** Use this class consistently on all entry-type dialogs; avoid per-dialog width/spacing overrides unless necessary.

### 3.8 Unscoped component styles (Footer, AppSelector, IconSelect, loadingScreen)

- **Footer.vue:** Layout + look-and-feel (background, border, font-size, link color).  
  - Move to common SCSS under a class like `.app-footer` and keep only structural wrapper in the component if needed.
- **AppSelector.vue:** Flex + gap + colors.  
  - Prefer PrimeFlex for layout; move colors/typography to common SCSS (e.g. `.app-icon-text`, `.app-icon`) so they can be reused.
- **IconSelect.vue:** Popover structure + grid + icon item/hover/selected.  
  - Structural (grid, overflow) can stay as unscoped if it’s specific to this component; shared “icon picker” look (border-radius, hover/selected colors) could move to a common `.icon-picker-content` / `.icon-item` in SCSS.
- **loadingScreen.vue:** Overlay (fixed, full size, flex center, z-index).  
  - Structure is valid; consider moving to common SCSS as `.c-loading` so it’s clearly a global overlay pattern and can be reused (e.g. for other full-screen overlays).

---

## 4. PrimeFlex usage (already available)

Prefer these instead of custom flex/gap/padding/margin in components:

| Custom CSS | PrimeFlex / approach |
|------------|----------------------|
| `display: flex; justify-content: space-between; align-items: center` | `class="flex justify-content-between align-items-center"` |
| `display: flex; gap: 0.5rem; justify-content: flex-end` | `class="flex gap-2 justify-content-end"` |
| `padding: 0.25rem` | `class="p-1"` |
| `padding: 2rem` | `class="p-4"` (or check PrimeFlex scale for 2rem) |
| `margin-bottom: 2rem` | `class="mb-4"` |
| `margin: 0` | `class="m-0"` |
| `display: inline-flex; align-items: center; gap: 0.5rem` | `class="inline-flex align-items-center gap-2"` |

Use PrimeFlex for **layout and spacing** first; add common SCSS only for **semantic** or **repeated design** patterns (e.g. view container, amount colors, DataTable defaults).

---

## 5. Proposed common SCSS structure

- **`_main.scss`** (existing): Keep base, `.main-app-content`, `.entry-dialog`, PrimeVue overrides.
- **`_view-layout.scss`** (new):  
  - `.view-container` — padding, max-width, margin (optional).  
  - Optional `.page-header` for the standard “title + actions” row (if you prefer a class over PrimeFlex only).
- **`_datatable.scss`** (new):  
  - Compact body cell padding and row hover **opt-in** via class `datatable-compact` on the DataTable (or wrapper).  
  - Right-aligned column header for `.amount-column`, `.balance-column`, `.actions-column` (global).
- **`_amounts.scss`** (new):  
  - Semantic amount/balance colors (`.amount.expense`, `.amount.income`, `.amount.transfer`, `.amount.amount-positive`, `.amount.amount-negative`, `.balance-negative`, stock-trade variants).
- **`_sidebar.scss`** (new):  
  - All current SidebarMenu.vue look-and-feel under `.app-sidebar` (or similar).
- **`_topbar.scss`** (existing):  
  - Add app-specific topbar/Menubar styling under a clear class if not already.
- **`_footer.scss`** (new, optional):  
  - Footer layout and look-and-feel (or merge into `_main.scss`).
- **`_loading.scss`** (new, optional):  
  - `.c-loading` overlay so it’s a global pattern.

Import new partials from `style.scss` after `_main` / `_topbar` as needed.

---

## 6. Phased migration (for long-term maintainability)

1. **Phase 1 – Low risk**  
   - Add `_amounts.scss` and `_datatable.scss`; move amount/balance colors and shared DataTable overrides there.  
   - Remove the same rules from `EntriesTable.vue` and `AccountEntriesTable.vue` (and any other table that uses them).  
   - Optionally add `.view-container` and use it in 2–3 views; replace local padding/max-width with the class.

2. **Phase 2 – Replace duplicated layout with PrimeFlex**  
   - In one view (e.g. `accounts.vue` or `InstrumentsView.vue`), replace `.header`, `.actions`, `.action-button` with PrimeFlex classes.  
   - If the result is good, roll out to all views that use the same pattern; delete the corresponding scoped CSS.

3. **Phase 3 – Shell components**  
   - Move SidebarMenu styles to `_sidebar.scss`.  
   - Move topbar-specific styles to `_topbar.scss` (or `_app-topbar.scss`).  
   - Move Footer (and optionally loadingScreen, AppSelector, IconSelect) to common SCSS; keep only structural markup in components.

4. **Phase 4 – Cleanup**  
   - Remove empty or redundant `<style>` blocks.  
   - Standardize on: **scoped** only for component-specific structure that doesn’t belong in global SCSS; **unscoped** only when intentionally supporting structure; everything else in common SCSS or PrimeFlex.

---

## 7. Summary

- **Valid component CSS:** Unscoped (or minimal scoped) **structural** rules that don’t duplicate layout or look-and-feel.
- **Remove from components:** Repeated layout (header, actions, view padding), repeated DataTable overrides, and semantic amount/balance colors.
- **Use PrimeFlex for:** Flex, gap, padding, margin, alignment in templates.
- **Use common SCSS for:** View container, DataTable defaults, amount/balance semantics, sidebar, topbar, footer, loading overlay.
- **Result:** Less duplicated CSS, single place for theming and table/amount behavior, and a clear rule for what stays in components — improving long-term maintainability.
