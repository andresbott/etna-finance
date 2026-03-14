# Financial Simulator - Design Spec

## Overview

Merge the two existing tools (Portfolio Simulator and Real Estate Simulator) into a single "Financial Simulator" section. The page shows a comparison chart and a list of saved simulations. Each simulation has a type and opens the corresponding type-specific editor.

## Page Layout

Single page at `/tools` with two stacked sections:

1. **Comparison Chart** (top) — ECharts line chart projecting net worth over a fixed 20-year horizon for all saved simulations. Each simulation is a separate line labeled by name. Empty state message when no simulations exist.

2. **Simulations List** (bottom) — Table similar to the transactions list.

### List Columns

| Column | Description |
|--------|-------------|
| Name | User-given name |
| Type | Icon or badge (Portfolio / Real Estate) |
| Expected Annual Return | Percentage |
| Attachment | Paperclip icon if file attached |
| Actions | Edit, Delete, Duplicate icons |

## Add / Edit Flow

### Adding a Simulation

1. Click "Add Simulation" button above the list
2. Dialog opens with three fields:
   - Name (text input)
   - Description (text input)
   - Type (dropdown: Portfolio Simulator / Real Estate Simulator)
3. On confirm, simulation is created via API and user navigates to `/tools/:toolType/:id` — the type-specific editor

### Editing

Click edit icon or row navigates to `/tools/:toolType/:id`. The editor page loads the appropriate tool view based on the URL's `toolType` param. A back button returns to `/tools`.

### Duplicating

Clones the simulation with all parameters. Appends "(copy)" to the name. If a "(copy)" suffix already exists, increments: "(copy 2)", "(copy 3)", etc. New entry appears in the list immediately.

### Deleting

Confirmation dialog, then removes simulation from list and chart.

## Routing

| Route | View |
|-------|------|
| `/tools` | Financial Simulator — list + comparison chart |
| `/tools/:toolType/:id` | Type-specific editor (Portfolio or Real Estate) with back navigation |

Legacy routes `/tools/portfolio-simulator` and `/tools/real-estate-simulator` are removed without redirects (single-user app, no backward compatibility needed).

## Sidebar Navigation

Single "Financial Simulator" menu entry replacing the current two tool entries. Points to `/tools`.

## Comparison Chart Details

- ECharts line chart, x-axis: years 0-20
- All simulations are projected over 20 years regardless of their individual duration settings (the individual editor still respects the saved duration)
- Each simulation is a line computed from its stored parameters
- **Portfolio type**: reuses existing `computeProjection()` logic — compound growth with contributions, adjusted for capital gains tax and inflation. The chart plots the inflation-adjusted net worth series.
- **Real Estate type**: reuses existing amortization/projection logic — plots total equity = (market value with appreciation) - (remaining mortgage balance) + (cumulative net rental cash flow). Multiple mortgages are aggregated by summing their individual schedules.
- Legend shows simulation names with distinct colors per line
- Chart recomputes when simulations are added, edited, or deleted

### Expected Annual Return by Type

This metric is stored in `ExpectedAnnualReturn` and shown in the list:
- **Portfolio**: `growthRate - capitalGainTax - inflation` (existing computation)
- **Real Estate**: gross annual return = `NOI / marketValue * 100` where NOI = annual rent - property tax - insurance - maintenance - other costs

## Backend & Persistence

No schema changes. Reuses existing `db_tools_data` table:

- `ToolType` — "portfolio-simulator" or "real-estate-simulator"
- `Name`, `Description` — user-provided
- `ExpectedAnnualReturn` — computed metric
- `Params` — JSON blob with tool-specific parameters
- `AttachmentID` — optional file reference

### API

Existing endpoints per tool type are sufficient. The frontend fetches both types in parallel:
- `GET /api/tools/portfolio-simulator/cases`
- `GET /api/tools/real-estate-simulator/cases`

Results are merged client-side for the unified list and chart.

Duplicate uses `POST /api/tools/{toolType}/cases` with cloned params and modified name.

## Computation

All calculations remain frontend-only. The existing computation logic in each simulator view is extracted into shared utility functions so both the editor views and the comparison chart can reuse them. For the comparison chart, a shared function takes a simulation's `toolType` and `params` and returns a 20-year array of yearly net worth values.

## Notes

- Description field is captured in the Add dialog but only displayed/edited in the editor view, not in the list table.
- The Type dropdown in the Add dialog is hardcoded to the two current types. New types require code changes.
