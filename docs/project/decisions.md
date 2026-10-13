# Decisions

## Tabler Icons Filled Variant Approach
**Date:** 2026-03-25
**Context:** Needed both outline and filled star icons for the account favorite feature. Tabler Icons provides separate font files for outline (`tabler-icons-300`) and filled (`tabler-icons-filled`) variants.
**Decision:** Custom `@font-face` in `_tabler-filled.scss` with per-icon codepoint overrides, applied via `.ti-filled` CSS class modifier.
**Rationale:** Three approaches were considered:
1. ~~Import `tabler-icons-filled.min.css` globally~~ — rejected because it sets `.ti { font-family: "tabler-icons-filled" }` which overrides ALL icons to filled
2. ~~Switch to SVG icons (`@tabler/icons-vue`)~~ — rejected because it requires changing icon usage throughout the codebase; PrimeVue Button's `icon` prop expects CSS classes
3. **Custom `@font-face` + scoped codepoint overrides** — chosen. Only loads the font file and maps specific icons we need. Minimal footprint, no global side effects. Trade-off: manual codepoint lookup per icon.

## Balance Report Returns Both Native and Main-Currency Balances
**Date:** 2026-10-13
**Context:** `GET /fin/report/balance` returned a single `sum`. #68 changed it from the account's own currency to the main currency, but the account entries view, the accounts page and the revaluation dialog kept treating it as the account's currency. A EUR account then showed its opening balance converted to CHF (25,689.66 instead of 27,309.96).
**Decision:** Every balance point carries `nativeBalance` (account currency) and `mainCurrencyBalance` (main currency), plus `hasUnconvertedAmounts` and `entryCount`. `sum` was removed so no caller keeps reading the old meaning.
**Rationale:** Three approaches were considered:
1. ~~Always return the account currency~~ — rejected because the overview chart and the account-type totals add up accounts in different currencies and need each step converted at its historical rate, which belongs in the backend
2. ~~A `native=true` query flag~~ — rejected because the same field would hold different currencies depending on the request, which is the ambiguity that caused the bug
3. **Both values, named by currency** — chosen. Both are already computed for every step, so the cost is negligible, and each caller picks the field it needs. Investment accounts convert positions from the instrument currency to the account currency through the main-currency rates.
