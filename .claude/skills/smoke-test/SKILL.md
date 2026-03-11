---
name: smoke-test
description: Run a full smoke test - start backend & frontend, then use Playwright MCP to navigate every page and check for console errors
---

# Smoke Test

This skill starts the full application stack and uses Playwright MCP tools to navigate all pages, checking for console errors and rendering issues.

## Prerequisites

- Backend builds and runs (`make run`)
- Frontend builds and runs (`cd webui && make run`)
- Playwright MCP server is available (provides `browser_navigate`, `browser_console_messages`, `browser_snapshot`, `browser_click`, etc.)

## Procedure

### Step 1: Kill any leftover processes from previous runs

Before starting, ensure ports 8085 and 5173 are free. Check and kill any existing processes:

```bash
# Check if anything is already on the backend port
lsof -ti :8085 | xargs kill 2>/dev/null || true
# Check if anything is already on the frontend port
lsof -ti :5173 | xargs kill 2>/dev/null || true
```

Wait 2 seconds after killing to let ports release, then verify both ports are free:

```bash
sleep 2 && lsof -i :8085 -i :5173 2>/dev/null || echo "Ports are free"
```

If either port is still occupied, do NOT proceed — report the issue.

### Step 2: Start the backend

Run the backend in the background. It listens on port 8085 by default.

**IMPORTANT:** Use the absolute project root path, regardless of your current working directory:

```bash
# Run with run_in_background: true
cd /home/bott/.datos/edit/programacion-privado/etna-finance && APP_LOG_LEVEL="debug" go run main.go start
```

Wait for it to be ready by polling (up to 15 seconds):

```bash
for i in $(seq 1 15); do
  code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/api/v1/settings 2>/dev/null)
  if [ "$code" = "200" ] || [ "$code" = "401" ]; then echo "Backend ready (HTTP $code)"; exit 0; fi
  sleep 1
done
echo "ERROR: Backend failed to start after 15s"
exit 1
```

If the backend does not respond after 15 seconds, stop and report the failure.

### Step 3: Start the frontend dev server

Run the frontend in the background. Vite dev server listens on port 5173 and proxies `/api/` to the backend.

**IMPORTANT:** You MUST use the `webui/` subdirectory path. Running `make run` from the project root starts the backend again, not the frontend:

```bash
# Run with run_in_background: true
cd /home/bott/.datos/edit/programacion-privado/etna-finance/webui && npm run dev
```

Wait for it to be ready by polling (up to 30 seconds — Vite may need to prebundle dependencies):

```bash
for i in $(seq 1 30); do
  code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:5173 2>/dev/null)
  if [ "$code" = "200" ]; then echo "Frontend ready"; exit 0; fi
  sleep 1
done
echo "ERROR: Frontend failed to start after 30s"
exit 1
```

If the frontend does not respond after 30 seconds, stop and report the failure.

### Step 4: Navigate with Playwright MCP

Use the Playwright MCP tools to open the app and visit each page. **After each navigation, always:**

1. Wait for the page to settle (use `browser_snapshot` to check the page rendered)
2. Check `browser_console_messages` for errors or warnings
3. Take note of any issues found

#### Pages to visit (in order)

Navigate to `http://localhost:5173` first (redirects to `/reports/overview`).

Then visit each of these routes:

| Route | Page | What to look for |
|-------|------|-------------------|
| `/reports/overview` | Dashboard | Charts render, no JS errors |
| `/accounts` | Accounts | Account list loads |
| `/entries` | Entries | Entry table renders |
| `/financial-transactions` | Financial Transactions | Filtered entries view |
| `/categories` | Categories | Category tree renders |
| `/reports/income-expense` | Income/Expense Report | Report chart/table loads |
| `/settings` | Settings | Settings form renders |
| `/market-data/currency-exchange` | Currency Exchange | Currency list loads |
| `/setup/csv-profiles` | CSV Profiles | Profile list renders |
| `/tasks` | Tasks | Task list renders |
| `/tools/portfolio-simulator` | Portfolio Simulator | Simulator form renders |
| `/tools/real-estate-simulator` | Real Estate Simulator | Simulator form renders |
| `/backup-restore` | Backup & Restore | Backup UI renders |

For each page:
```
1. browser_navigate to http://localhost:5173{route}
2. browser_snapshot — verify the page rendered meaningful content (not blank, not error page)
3. browser_console_messages — collect any errors/warnings
```

### Step 5: Report results

After visiting all pages, compile a summary report:

```
## Smoke Test Results

### Pages Visited: X/Y passed

| Page | Status | Issues |
|------|--------|--------|
| Dashboard | OK | — |
| Accounts | ERROR | Console error: "..." |
| ... | ... | ... |

### Console Errors
- List any JS errors found, grouped by page

### Console Warnings
- List any notable warnings (ignore common noise like deprecation warnings from dependencies)

### Rendering Issues
- List any pages that appeared blank, showed error boundaries, or had broken layouts
```

### Step 6: Cleanup

**IMPORTANT:** You MUST clean up ALL processes after the smoke test, even if it failed partway through. Always run cleanup, never skip it.

1. Close the Playwright browser:
```
browser_close
```

2. Kill processes by port (most reliable method — catches the compiled Go binary, not just `go run`):
```bash
# Kill whatever is listening on the backend port
lsof -ti :8085 | xargs kill 2>/dev/null || true
# Kill whatever is listening on the frontend port
lsof -ti :5173 | xargs kill 2>/dev/null || true
# Also kill any orphaned node/esbuild processes from Vite
pkill -f "node.*vite" 2>/dev/null || true
pkill -f "esbuild.*service" 2>/dev/null || true
```

3. **Verify** everything is actually stopped:
```bash
sleep 1 && lsof -i :8085 -i :5173 2>/dev/null || echo "All clean — ports 8085 and 5173 are free"
```

If any process remains, kill it by PID explicitly and verify again. Do NOT report the smoke test as complete until ports are confirmed free.

## Important Notes

- **Auth is disabled by default** in dev mode (`ETNA_AUTH_ENABLED=false`), so pages should load without login.
- If a page redirects to `/login`, the backend may have auth enabled. Check `ETNA_AUTH_ENABLED` env var.
- Some pages require the `instruments` feature flag (Investment Report, Instruments, Stock Market). If these redirect away, that's expected behavior, not an error.
- The frontend Vite dev server proxies API calls to `http://localhost:8085`, so both must be running.
- Console messages from dependencies (e.g., PrimeVue deprecation warnings) can be noted but aren't smoke test failures.
- Focus on: uncaught exceptions, failed API calls (4xx/5xx in console), blank pages, and render errors.
- **Common pitfall:** `make run` in the project root starts the backend. `make run` in `webui/` starts the frontend. Running the wrong one from the wrong directory will start duplicate backends.
- **Common pitfall:** `pkill -f "go run main.go"` does NOT kill the compiled Go binary. The backend compiles to a temporary binary named `main` (or similar). Always use `lsof -ti :8085 | xargs kill` to reliably kill it.
