# Decisions

## Tabler Icons Filled Variant Approach
**Date:** 2026-03-25
**Context:** Needed both outline and filled star icons for the account favorite feature. Tabler Icons provides separate font files for outline (`tabler-icons-300`) and filled (`tabler-icons-filled`) variants.
**Decision:** Custom `@font-face` in `_tabler-filled.scss` with per-icon codepoint overrides, applied via `.ti-filled` CSS class modifier.
**Rationale:** Three approaches were considered:
1. ~~Import `tabler-icons-filled.min.css` globally~~ — rejected because it sets `.ti { font-family: "tabler-icons-filled" }` which overrides ALL icons to filled
2. ~~Switch to SVG icons (`@tabler/icons-vue`)~~ — rejected because it requires changing icon usage throughout the codebase; PrimeVue Button's `icon` prop expects CSS classes
3. **Custom `@font-face` + scoped codepoint overrides** — chosen. Only loads the font file and maps specific icons we need. Minimal footprint, no global side effects. Trade-off: manual codepoint lookup per icon.
