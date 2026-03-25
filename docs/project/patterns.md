# Patterns

## Adding a New Account Type

Files to modify:
1. `internal/accounting/account.go` — add to enum iota, update `String()` and `parseAccountType()`
2. `app/router/handlers/finance/account.go` — add to handler's type validation
3. `internal/backup/import.go` — add to backup's `parseAccountType()` (uses `AccountType.String()` values)
4. `internal/backup/dataV1.go` — no change needed (type stored as string)
5. `webui/src/types/account.ts` — add to `ACCOUNT_TYPES` and `ACCOUNT_TYPE_LABELS`

Naming conventions:
- Backend enum: PascalCase (`PrepaidExpense`)
- `String()` output: lowercase spaced (`prepaid expense`)
- Handler type string: lowercase no separator (`prepaidexpense`)
- Frontend const: UPPER_SNAKE (`PREPAID_EXPENSE`)
- Frontend label: title case (`Prepaid expense`)

## Adding a New Account Field (Full-Stack)

Example: the `favorite` boolean field.

### Backend
1. `internal/accounting/account.go`:
   - Add field to `Account` struct
   - Add field to `dbAccount` struct (GORM model)
   - Map in `dbToAccount()`
   - Add to `CreateAccount` (pass through to dbAccount)
   - Add pointer field to `AccountUpdatePayload` (e.g. `Favorite *bool`)
   - Handle in `UpdateAccount` using the pointer pattern:
     ```go
     if item.Favorite != nil {
         updateStruct.Favorite = *item.Favorite
         selectedFields = append(selectedFields, "Favorite")
     }
     ```
2. `app/router/handlers/finance/account.go`:
   - Add to `accountPayload` (create) with json tag
   - Add pointer to `accountUpdatePayload` (update) with json tag
   - Map in create handler and update handler

### Backup
3. `internal/backup/dataV1.go` — add field to `accountV1` struct with `json:"field,omitempty"`
4. `internal/backup/export.go` — map field in `writeAccounts()`
5. `internal/backup/import.go` — map field in account import logic

### Frontend
6. `webui/src/types/account.ts` — add optional field to `Account` interface
7. `webui/src/lib/api/Account.ts` — add to `AccountItem` interface
8. `webui/src/composables/useAccounts.ts` — add to `updateAccount` mutation params and API call
9. Component (e.g. `accounts.vue`) — add UI element, wire to composable

### Key pattern: pointer-based partial updates
The Go update payload uses `*bool` / `*string` pointers so the handler can distinguish "field not sent" (nil) from "field set to zero value" (false/""). Only non-nil fields get added to `selectedFields` for GORM's `Select()` update.

## Tabler Icons — Outline and Filled Coexistence

The project uses `@tabler/icons-webfont` for outline icons (class-based: `ti ti-{name}`). Filled variants require a separate setup because the outline and filled fonts use **different unicode codepoints** for the same icon class name.

### How it works
- `webui/src/assets/scss/_tabler-filled.scss` defines:
  1. A `@font-face` loading the filled font (`tabler-icons-filled.woff2`)
  2. `.ti.ti-filled` rule switching `font-family` to the filled font
  3. Per-icon `content` overrides mapping to the filled font's codepoints
- Usage: `class="ti ti-filled ti-star"` renders a filled star

### Adding a new filled icon
1. Find the filled codepoint: `grep -A1 '.ti-{name}:before' node_modules/@tabler/icons-webfont/dist/tabler-icons-filled.css`
2. Add to `_tabler-filled.scss`: `.ti-filled.ti-{name}:before { content: "\XXXX"; }`

### ⚠ Caveats
- **Updating `@tabler/icons-webfont`**: Codepoints may change between versions. After updating, regenerate `_tabler-filled.scss` (see below).
- **No generation script yet**: Currently each filled icon codepoint is added manually. To regenerate or add all icons at once, extract from the filled CSS:
  ```bash
  # Find a specific icon's filled codepoint:
  grep -A1 '.ti-{name}:before' node_modules/@tabler/icons-webfont/dist/tabler-icons-filled.css

  # Generate all overrides (pipe to file if full set needed):
  grep -oP '\.ti-[\w-]+(?=:before)' node_modules/@tabler/icons-webfont/dist/tabler-icons-filled.css | while read cls; do
    code=$(grep -A1 "${cls}:before" node_modules/@tabler/icons-webfont/dist/tabler-icons-filled.css | grep content | grep -oP '\\\\[0-9a-f]+')
    echo ".ti-filled${cls}:before { content: \"${code}\"; }"
  done
  ```
  If more filled icons are needed, consider turning this into a proper script in `webui/scripts/`.
- **Why not import `tabler-icons-filled.min.css`**: That CSS globally sets `.ti { font-family: "tabler-icons-filled" }` which overrides ALL icons to filled. No way to scope it without stripping that rule.

## Frontend View Pattern

Views follow a consistent structure:
- `<script setup>` with local PrimeVue imports (not globally registered)
- Data fetching via TanStack Query composables (e.g. `useAccounts()`)
- Template uses PrimeFlex utility classes for layout

### PrimeVue 4 Tabs
Use the new API: `Tabs` > `TabList` > `Tab` + `TabPanels` > `TabPanel`. Not the old `TabView`/`TabPanel` API.

### SCSS imports
Global styles in `webui/src/assets/style.scss` use `@use` rules. All `@use` statements must come before any other rules (CSS/SCSS requirement). Custom rules go after.
