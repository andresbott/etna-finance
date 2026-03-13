# Tools Data Storage

## Problem

The app has tools (portfolio simulator, real estate simulator) that run client-side calculations. Users want to save input parameter sets as named "case studies" and later compare them on common metrics like expected annual return. Adding a dedicated GORM table per tool doesn't scale — each new tool would require migrations, structs, and CRUD boilerplate.

## Decision

A single generic GORM table with typed common fields and a JSON blob for tool-specific parameters. No external library needed.

## Backend Package: `internal/toolsdata`

### DB Model

```go
type dbToolsData struct {
    ID                   uint           `gorm:"primarykey"`
    ToolType             string         `gorm:"uniqueIndex:idx_tool_name;not null"` // e.g. "portfolio-simulator"
    Name                 string         `gorm:"uniqueIndex:idx_tool_name;not null"`
    Description          string
    ExpectedAnnualReturn float64
    Params               datatypes.JSON // tool-specific input blob
    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

### Public Types

```go
type CaseStudy struct {
    ID                   uint
    ToolType             string
    Name                 string
    Description          string
    ExpectedAnnualReturn float64
    Params               json.RawMessage
    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

### Store

```go
type Store struct {
    db *gorm.DB
}

func NewStore(db *gorm.DB) (*Store, error)
```

`NewStore` calls `AutoMigrate(&dbToolsData{})`, consistent with other stores.

### CRUD Methods

- `List(ctx context.Context, toolType string) ([]CaseStudy, error)`
- `Get(ctx context.Context, id uint) (CaseStudy, error)`
- `Create(ctx context.Context, cs CaseStudy) (CaseStudy, error)`
- `Update(ctx context.Context, id uint, cs CaseStudy) (CaseStudy, error)`
- `Delete(ctx context.Context, id uint) error`

`List` filters by `toolType`. `Create`/`Update` enforce that `ToolType` and `Name` are non-empty.

## API Routes

```
GET    /api/v0/tools/{toolType}/cases       → list case studies
GET    /api/v0/tools/{toolType}/cases/{id}  → get single case study
POST   /api/v0/tools/{toolType}/cases       → create case study
PUT    /api/v0/tools/{toolType}/cases/{id}   → update case study
DELETE /api/v0/tools/{toolType}/cases/{id}   → delete case study
```

The `{toolType}` path segment is constrained via mux regex: `{toolType:[a-z0-9-]+}`.

### Handler

`app/router/handlers/toolsdata/handler.go`:

```go
type Handler struct {
    Store *toolsdata.Store
}
```

Methods: `ListCases(toolType)`, `GetCase(toolType, id)`, `CreateCase(toolType)`, `UpdateCase(toolType, id)`, `DeleteCase(toolType, id)` — each returns `http.Handler`.

`toolType` is extracted from `mux.Vars(r)["toolType"]` inside the route registration closure and passed to the handler method, matching the `marketDataAPI` pattern for `{symbol}`. The request body for `POST`/`PUT` does not include `toolType` — it is derived solely from the URL.

### Router Wiring

- Add `ToolsDataStore *toolsdata.Store` to `router.Cfg`
- Add `toolsDataStore *toolsdata.Store` to `MainAppHandler`
- Call `h.toolsDataAPI(r)` in `attachApiV0`
- `toolsDataAPI` registers the CRUD routes under `/tools/{toolType}/cases`

## Frontend

### API Client: `webui/src/lib/api/ToolsData.ts`

```typescript
interface CaseStudy<T = Record<string, unknown>> {
    id: number
    toolType: string
    name: string
    description: string
    expectedAnnualReturn: number
    params: T
    createdAt: string
    updatedAt: string
}

function listCases<T>(toolType: string): Promise<CaseStudy<T>[]>
function getCase<T>(toolType: string, id: number): Promise<CaseStudy<T>>
function createCase<T>(toolType: string, data: Omit<CaseStudy<T>, 'id' | 'toolType' | 'createdAt' | 'updatedAt'>): Promise<CaseStudy<T>>
function updateCase<T>(toolType: string, id: number, data: Partial<Omit<CaseStudy<T>, 'id' | 'toolType' | 'createdAt' | 'updatedAt'>>): Promise<CaseStudy<T>>
function deleteCase(toolType: string, id: number): Promise<void>
```

### Portfolio Simulator Params Type

```typescript
interface PortfolioSimulatorParams {
    durationYears: number
    initialContribution: number
    monthlyContribution: number
    growthRatePct: number
    inflationPct: number
    capitalGainTaxPct: number
}
```

### UI Changes to PortfolioSimulatorView

Add a save/load panel to the existing view:
- "Save" button captures current form inputs + computed `expectedAnnualReturn`
- Dialog/inline form for name + description
- Dropdown or list to load a saved case study (populates the form inputs)
- Delete button per saved case study

No changes to the calculation logic or chart.

## Testing

- Backend: store-level tests (CRUD round-trip, unique constraint, validation) and handler-level tests, following the pattern of existing test files (e.g. `import_test.go`, `Account.test.ts`)
- Frontend: API client tests following existing patterns (e.g. `Account.test.ts`)

## Follow-up Items

- **Backup/restore integration**: add `toolsdata.Store` to the backup handler so case studies survive backup/restore cycles (separate task)

## Out of Scope

- Cross-tool comparison view (future: query by `expectedAnnualReturn` across tool types)
- Real estate simulator case study integration (same pattern, separate task)
- Pagination on the list endpoint (not needed for expected data volumes)
