# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Etna is an opinionated personal finance app (early stage, not production-ready) for tracking
expenses and investments. It is a Go backend serving a JSON API plus an embedded Vue 3 SPA.
It builds on top of the `go-bumbu` libraries (config, http, userauth, timeseries, closure-tree,
tempo). The module path is `github.com/andresbott/etna`.

## Commands

All targets are in the `Makefile` (`make help` lists them). Common ones:

| Task | Command |
| --- | --- |
| Run backend (debug logging, built-in defaults) | `make run` |
| Build UI + run backend | `make run-ui` |
| Frontend dev server | `cd webui && npm run dev` |
| Fast Go tests | `make test` (`go test ./... -cover`) |
| Frontend unit tests | `make ui-test` (`cd webui && npm test`) |
| DB integration tests (all DB backends) | `make it-test` (uses `--alldbs` flag) |
| e2e tests (starts app + browser) | `make e2e-test` (`E2E=true HEADLESS=false go test -v ./zarf/e2e`) |
| Go lint | `make lint` (`golangci-lint run`) |
| Coverage gate (70% on internal/ + libs/) | `make coverage` |
| Full verification suite | `make verify` (test, ui-test, license-check, lint, benchmark, coverage) |
| Seed sample data into a running instance | `make sampledata` |

Run a single Go test: `go test ./internal/accounting -run TestName`.
Run a single frontend test: `cd webui && npx vitest run path/to/file.test.ts`.
The frontend `npm test` runs `vue-tsc --noEmit` first, so type errors fail the test run.

There is also a `/verify` skill that runs the full suite and fixes issues.

## Architecture

### Backend layering

The backend has a strict three-layer structure:

1. **`internal/*`** — domain logic and persistence ("stores"). Each domain package
   (`accounting`, `marketdata`, `csvimport`, `toolsdata`, `filestore`, `backup`, `taskrunner`)
   exposes a `Store` struct created with `NewStore(db, ...)`. Stores own their GORM models
   (`dbXxx` structs, unexported) and run `db.AutoMigrate(...)` plus inline data migrations in
   their constructor. Public domain types (e.g. `Account`) are mapped to/from `dbAccount`.
2. **`app/router/handlers/*`** — HTTP handlers grouped by domain (`finance`, `marketdata`,
   `stats`, `backup`, `tasks`, `csvimport`, `toolsdata`). A `Handler` struct holds the stores it
   needs as fields; methods return `http.Handler`. Handlers decode JSON payloads, call store
   methods with `r.Context()`, and encode responses.
3. **`app/router`** — wiring. `main.go` defines `MainAppHandler` and `Cfg`; `api_v0.go` mounts
   all API routes under `/api/v0` behind auth middleware. `app/cmd/server.go` (`runServer`)
   constructs the DB, all stores, the task runner/scheduler, and the router.

`app/cmd` is the Cobra CLI entrypoint (`main.go` → `cmd.Execute()`). The single SQLite DB file is
`carbon.db`. The SPA is embedded via `go:embed` in `app/spa` and served from `files/ui`.

### Auth

Auth uses `go-bumbu/userauth` with session cookies. When `AuthDisabled` is true (the default for
local dev), `NewNoAuthHandler(defaultUser)` injects a default user and no login is required. Demo
credentials when auth is on: `demo:demo` / `admin:admin`. Handlers read user data via
`sessionauth.CtxGetUserData(r)`.

### Stores pass `context.Context`

Store methods take `ctx` first and use `db.WithContext(ctx)`. Handlers pass `r.Context()`.
Categories use `go-bumbu/closure-tree` (a tenant-scoped tree). Market data (prices, FX, EPS) is
stored as time series via `go-bumbu/timeseries`, keyed by name prefixes (`price:`, `eps:`); see
`internal/marketdata`.

### Tasks and scheduling

`app/tasks` defines `TaskDef`s (backup, financial import/backfill, FX import/backfill, EPS import,
plus dev-only debug tasks) run by `internal/taskrunner`. Dev-only tasks are hidden in production
via `DevOnlyTaskIDs`. External market data comes through `internal/marketdata/importer` (the
`massive` client, with rate limiting and worker pools).

### Frontend

Vue 3 + TypeScript + Vite, PrimeVue 4 + PrimeFlex, TanStack Query (vue-query), Pinia, ECharts
(`vue-echarts`), Zod for validation, `@tabler/icons-webfont` for icons. `@` aliases `webui/src`.
Layout structure under `webui/src`: `views/`, `components/`, `composables/`, `lib/api/`, `store/`
(Pinia), `types/` (shared DTOs), `utils/`.

## Conventions

These are enforced by Cursor rules (`.cursor/rules/`) and project docs (`docs/project/`):

- **API layer** (`webui/src/lib/api/**`): always use `apiClient` from `@/lib/api/client` (never raw
  axios). Exported API functions are **camelCase** (`getEntries`, `createEntry`). Shared DTOs live
  in `webui/src/types/`. Use `with404Null(apiCall)` from `@/lib/api/helpers` for optional resources
  that may 404.
- **Dates**: all user-visible dates must use `useDateFormat()` from `@/composables/useDateFormat`
  (`formatDate(date)` for display, `pickerDateFormat` for PrimeVue pickers). Never
  `toLocaleDateString()` or hardcoded formats.
- **Composables**: prefer `useQuery`/`useMutation` with a single query key; invalidate on mutation
  success via `invalidateAndRefetch` from `@/composables/queryUtils`.
- **PrimeVue 4 Tabs**: use the new `Tabs` > `TabList` > `Tab` + `TabPanels` > `TabPanel` API, not
  the old `TabView`. Import PrimeVue components locally in `<script setup>` (not globally).
- **Partial updates (backend)**: update payloads use pointer fields (`*bool`, `*string`) so the
  handler distinguishes "field not sent" (nil) from "set to zero value". Only non-nil fields are
  added to GORM's `Select()`. See `docs/project/patterns.md` for the full full-stack recipe for
  adding an account type/field.
- **Linting**: `golangci-lint` v2; `gocyclo`/`gocognit`/`nestif` thresholds are tightened (20/20/5).
  `nolint` directives require a specific linter and an explanation.

`docs/project/patterns.md` and `docs/project/decisions.md` document multi-step full-stack patterns
(adding account types/fields across store → handler → backup → frontend) and notable decisions
(e.g. the Tabler filled-icon `@font-face` approach). Read them before cross-cutting changes.

## Notes

- Per global user instructions: leave changes uncommitted (the user commits); use single-line
  commit messages with no Co-Authored-By line; run `git add` separately from `git commit`.
- `docs/superpowers/{plans,specs}/` contain dated design specs and implementation plans for
  features — useful context when working on a feature area.
