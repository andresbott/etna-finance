# Tools Data Storage Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add backend storage and API for tool case studies (named parameter sets with a common `expectedAnnualReturn` field), and integrate save/load into the portfolio simulator frontend.

**Architecture:** A new `internal/toolsdata` package with a single GORM table stores case studies as typed common fields + JSON blob. HTTP handlers in `app/router/handlers/toolsdata/` expose CRUD scoped by `toolType`. The frontend API client in `webui/src/lib/api/ToolsData.ts` provides typed access, and `PortfolioSimulatorView.vue` gets a save/load panel.

**Tech Stack:** Go, GORM, SQLite, Gorilla Mux, Vue 3, TypeScript, PrimeVue, Axios, Vitest

**Spec:** `docs/superpowers/specs/2026-03-12-tools-data-storage-design.md`

---

## File Structure

### Backend (new files)
- `internal/toolsdata/toolsdata.go` — Store struct, NewStore, DB model, public CaseStudy type, error types
- `internal/toolsdata/toolsdata_test.go` — Store CRUD tests
- `app/router/handlers/toolsdata/handler.go` — HTTP handler with CRUD methods
- `app/router/handlers/toolsdata/handler_test.go` — Handler HTTP tests

### Backend (modified files)
- `app/router/main.go` — Add `ToolsDataStore` to `Cfg` and `MainAppHandler`
- `app/router/api_v0.go` — Add `toolsDataAPI` method and call it from `attachApiV0`
- `app/cmd/server.go` — Create `toolsdata.Store` in `initStores` and pass to router

### Frontend (new files)
- `webui/src/lib/api/ToolsData.ts` — API client functions
- `webui/src/lib/api/ToolsData.test.ts` — API client tests

### Frontend (modified files)
- `webui/src/views/tools/PortfolioSimulatorView.vue` — Add save/load case study panel

---

## Chunk 1: Backend Store

### Task 1: Create toolsdata package with types and store

**Files:**
- Create: `internal/toolsdata/toolsdata.go`

- [ ] **Step 1: Create the package with types, errors, and store skeleton**

Create `internal/toolsdata/toolsdata.go`:

```go
package toolsdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrValidation represents a validation error for case study data.
type ErrValidation string

func (v ErrValidation) Error() string {
	return string(v)
}

var ErrCaseStudyNotFound = errors.New("case study not found")

// dbToolsData is the DB-internal representation of a case study.
type dbToolsData struct {
	ID                   uint   `gorm:"primarykey"`
	ToolType             string `gorm:"uniqueIndex:idx_tool_name;not null"`
	Name                 string `gorm:"uniqueIndex:idx_tool_name;not null"`
	Description          string
	ExpectedAnnualReturn float64
	Params               string // JSON string stored as TEXT in SQLite
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// CaseStudy is the public-facing representation of a tool case study.
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

func dbToCaseStudy(in dbToolsData) CaseStudy {
	return CaseStudy{
		ID:                   in.ID,
		ToolType:             in.ToolType,
		Name:                 in.Name,
		Description:          in.Description,
		ExpectedAnnualReturn: in.ExpectedAnnualReturn,
		Params:               json.RawMessage(in.Params),
		CreatedAt:            in.CreatedAt,
		UpdatedAt:            in.UpdatedAt,
	}
}

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	err := db.AutoMigrate(&dbToolsData{})
	if err != nil {
		return nil, fmt.Errorf("error running auto migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, cs CaseStudy) (CaseStudy, error) {
	if cs.ToolType == "" {
		return CaseStudy{}, ErrValidation("tool_type cannot be empty")
	}
	if cs.Name == "" {
		return CaseStudy{}, ErrValidation("name cannot be empty")
	}

	row := dbToolsData{
		ToolType:             cs.ToolType,
		Name:                 cs.Name,
		Description:          cs.Description,
		ExpectedAnnualReturn: cs.ExpectedAnnualReturn,
		Params:               string(cs.Params),
	}
	d := s.db.WithContext(ctx).Create(&row)
	if d.Error != nil {
		return CaseStudy{}, d.Error
	}
	return dbToCaseStudy(row), nil
}

func (s *Store) Get(ctx context.Context, toolType string, id uint) (CaseStudy, error) {
	var row dbToolsData
	d := s.db.WithContext(ctx).Where("id = ? AND tool_type = ?", id, toolType).First(&row)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return CaseStudy{}, ErrCaseStudyNotFound
		}
		return CaseStudy{}, d.Error
	}
	return dbToCaseStudy(row), nil
}

func (s *Store) List(ctx context.Context, toolType string) ([]CaseStudy, error) {
	var rows []dbToolsData
	d := s.db.WithContext(ctx).Where("tool_type = ?", toolType).Order("id ASC").Find(&rows)
	if d.Error != nil {
		return nil, d.Error
	}
	result := make([]CaseStudy, 0, len(rows))
	for _, row := range rows {
		result = append(result, dbToCaseStudy(row))
	}
	return result, nil
}

func (s *Store) Update(ctx context.Context, toolType string, id uint, cs CaseStudy) (CaseStudy, error) {
	if cs.Name == "" {
		return CaseStudy{}, ErrValidation("name cannot be empty")
	}

	d := s.db.WithContext(ctx).Model(&dbToolsData{}).Where("id = ? AND tool_type = ?", id, toolType).
		Select("Name", "Description", "ExpectedAnnualReturn", "Params").
		Updates(dbToolsData{
			Name:                 cs.Name,
			Description:          cs.Description,
			ExpectedAnnualReturn: cs.ExpectedAnnualReturn,
			Params:               string(cs.Params),
		})
	if d.Error != nil {
		return CaseStudy{}, d.Error
	}
	if d.RowsAffected == 0 {
		return CaseStudy{}, ErrCaseStudyNotFound
	}
	return s.Get(ctx, toolType, id)
}

func (s *Store) Delete(ctx context.Context, toolType string, id uint) error {
	d := s.db.WithContext(ctx).Where("id = ? AND tool_type = ?", id, toolType).Delete(&dbToolsData{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCaseStudyNotFound
	}
	return nil
}
```

Note: We use `string` for the `Params` DB field instead of `datatypes.JSON` since that dependency isn't in the project. SQLite stores it as TEXT, and we convert to/from `json.RawMessage` in the public type.

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/toolsdata/`
Expected: no errors

---

### Task 2: Write store tests

**Files:**
- Create: `internal/toolsdata/toolsdata_test.go`

- [ ] **Step 1: Write CRUD tests**

Create `internal/toolsdata/toolsdata_test.go`:

```go
package toolsdata

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to open sqlite: %v", err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("unable to create store: %v", err)
	}
	return store
}

func validCaseStudy() CaseStudy {
	return CaseStudy{
		ToolType:             "portfolio-simulator",
		Name:                 "Conservative",
		Description:          "Low risk scenario",
		ExpectedAnnualReturn: 4.5,
		Params:               json.RawMessage(`{"durationYears":20,"growthRatePct":6}`),
	}
}

func TestNewStore_NilDB(t *testing.T) {
	_, err := NewStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db, got nil")
	}
}

func TestCreate(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		cs := validCaseStudy()
		got, err := store.Create(ctx, cs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID == 0 {
			t.Fatal("expected non-zero id")
		}
		if got.Name != cs.Name {
			t.Errorf("expected name %q, got %q", cs.Name, got.Name)
		}
		if got.ToolType != cs.ToolType {
			t.Errorf("expected tool_type %q, got %q", cs.ToolType, got.ToolType)
		}
		if got.ExpectedAnnualReturn != cs.ExpectedAnnualReturn {
			t.Errorf("expected annual return %f, got %f", cs.ExpectedAnnualReturn, got.ExpectedAnnualReturn)
		}
		if string(got.Params) != string(cs.Params) {
			t.Errorf("expected params %s, got %s", cs.Params, got.Params)
		}
	})

	t.Run("validation: empty tool_type", func(t *testing.T) {
		cs := validCaseStudy()
		cs.ToolType = ""
		_, err := store.Create(ctx, cs)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("validation: empty name", func(t *testing.T) {
		cs := validCaseStudy()
		cs.Name = ""
		_, err := store.Create(ctx, cs)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("unique constraint: duplicate tool_type+name", func(t *testing.T) {
		s := newTestStore(t)
		cs := validCaseStudy()
		_, err := s.Create(ctx, cs)
		if err != nil {
			t.Fatalf("unexpected error on first create: %v", err)
		}
		_, err = s.Create(ctx, cs)
		if err == nil {
			t.Fatal("expected error on duplicate tool_type+name, got nil")
		}
	})
}

func TestGet(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("existing", func(t *testing.T) {
		cs := validCaseStudy()
		created, err := store.Create(ctx, cs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := store.Get(ctx, "portfolio-simulator", created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != created.ID {
			t.Errorf("expected id %d, got %d", created.ID, got.ID)
		}
		if got.Name != cs.Name {
			t.Errorf("expected name %q, got %q", cs.Name, got.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.Get(ctx, "portfolio-simulator", 99999)
		if !errors.Is(err, ErrCaseStudyNotFound) {
			t.Fatalf("expected ErrCaseStudyNotFound, got %v", err)
		}
	})
}

func TestList(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("empty list", func(t *testing.T) {
		items, err := store.List(ctx, "portfolio-simulator")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})

	t.Run("filters by tool_type", func(t *testing.T) {
		cs1 := validCaseStudy()
		cs1.Name = "Case A"
		_, err := store.Create(ctx, cs1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cs2 := validCaseStudy()
		cs2.ToolType = "real-estate-simulator"
		cs2.Name = "Case B"
		_, err = store.Create(ctx, cs2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		items, err := store.List(ctx, "portfolio-simulator")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}
		if items[0].Name != "Case A" {
			t.Errorf("expected name %q, got %q", "Case A", items[0].Name)
		}
	})
}

func TestUpdate(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		cs := validCaseStudy()
		created, err := store.Create(ctx, cs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated, err := store.Update(ctx, "portfolio-simulator", created.ID, CaseStudy{
			Name:                 "Aggressive",
			Description:          "High risk scenario",
			ExpectedAnnualReturn: 8.0,
			Params:               json.RawMessage(`{"durationYears":30,"growthRatePct":10}`),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != "Aggressive" {
			t.Errorf("expected name %q, got %q", "Aggressive", updated.Name)
		}
		if updated.ExpectedAnnualReturn != 8.0 {
			t.Errorf("expected annual return 8.0, got %f", updated.ExpectedAnnualReturn)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.Update(ctx, "portfolio-simulator", 99999, CaseStudy{Name: "X"})
		if !errors.Is(err, ErrCaseStudyNotFound) {
			t.Fatalf("expected ErrCaseStudyNotFound, got %v", err)
		}
	})

	t.Run("validation: empty name", func(t *testing.T) {
		cs := validCaseStudy()
		created, err := store.Create(ctx, cs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = store.Update(ctx, "portfolio-simulator", created.ID, CaseStudy{Name: ""})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})
}

func TestDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		cs := validCaseStudy()
		created, err := store.Create(ctx, cs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = store.Delete(ctx, "portfolio-simulator", created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = store.Get(ctx, "portfolio-simulator", created.ID)
		if !errors.Is(err, ErrCaseStudyNotFound) {
			t.Fatalf("expected ErrCaseStudyNotFound after delete, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := store.Delete(ctx, "portfolio-simulator", 99999)
		if !errors.Is(err, ErrCaseStudyNotFound) {
			t.Fatalf("expected ErrCaseStudyNotFound, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/toolsdata/ -v`
Expected: all tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/toolsdata/
git commit -m "feat: add toolsdata store with CRUD and tests"
```

---

## Chunk 2: Backend HTTP Handler and Router Wiring

### Task 3: Create HTTP handler

**Files:**
- Create: `app/router/handlers/toolsdata/handler.go`

- [ ] **Step 1: Write the handler**

Create `app/router/handlers/toolsdata/handler.go`:

```go
package toolsdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/toolsdata"
)

type Handler struct {
	Store *toolsdata.Store
}

type casePayload struct {
	ID                   uint            `json:"id"`
	ToolType             string          `json:"toolType"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	ExpectedAnnualReturn float64         `json:"expectedAnnualReturn"`
	Params               json.RawMessage `json:"params"`
	CreatedAt            string          `json:"createdAt"`
	UpdatedAt            string          `json:"updatedAt"`
}

func toPayload(cs toolsdata.CaseStudy) casePayload {
	return casePayload{
		ID:                   cs.ID,
		ToolType:             cs.ToolType,
		Name:                 cs.Name,
		Description:          cs.Description,
		ExpectedAnnualReturn: cs.ExpectedAnnualReturn,
		Params:               cs.Params,
		CreatedAt:            cs.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:            cs.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (h *Handler) ListCases(toolType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items, err := h.Store.List(r.Context(), toolType)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list case studies: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		payloads := make([]casePayload, len(items))
		for i, cs := range items {
			payloads[i] = toPayload(cs)
		}
		respJSON, err := json.Marshal(payloads)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) GetCase(toolType string, id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cs, err := h.Store.Get(r.Context(), toolType, id)
		if err != nil {
			if errors.Is(err, toolsdata.ErrCaseStudyNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to get case study: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		respJSON, err := json.Marshal(toPayload(cs))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) CreateCase(toolType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		var payload casePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		cs := toolsdata.CaseStudy{
			ToolType:             toolType,
			Name:                 payload.Name,
			Description:          payload.Description,
			ExpectedAnnualReturn: payload.ExpectedAnnualReturn,
			Params:               payload.Params,
		}

		created, err := h.Store.Create(r.Context(), cs)
		if err != nil {
			var target toolsdata.ErrValidation
			if errors.As(err, &target) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create case study: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		respJSON, err := json.Marshal(toPayload(created))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) UpdateCase(toolType string, id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}
		var payload casePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		cs := toolsdata.CaseStudy{
			Name:                 payload.Name,
			Description:          payload.Description,
			ExpectedAnnualReturn: payload.ExpectedAnnualReturn,
			Params:               payload.Params,
		}

		updated, err := h.Store.Update(r.Context(), toolType, id, cs)
		if err != nil {
			var target toolsdata.ErrValidation
			if errors.As(err, &target) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, toolsdata.ErrCaseStudyNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update case study: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		respJSON, err := json.Marshal(toPayload(updated))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *Handler) DeleteCase(toolType string, id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.Delete(r.Context(), toolType, id)
		if err != nil {
			if errors.Is(err, toolsdata.ErrCaseStudyNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete case study: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./app/router/handlers/toolsdata/`
Expected: no errors

---

### Task 4: Wire store and handler into the router

**Files:**
- Modify: `app/router/main.go:23-66` (add to Cfg and MainAppHandler)
- Modify: `app/router/api_v0.go:19-46` (add toolsDataAPI call and method)
- Modify: `app/cmd/server.go:105-109,130-155` (create store, pass to router)

- [ ] **Step 1: Add ToolsDataStore to router.Cfg and MainAppHandler**

In `app/router/main.go`, add to the `Cfg` struct (after `AttachmentStore`):

```go
ToolsDataStore *toolsdata.Store
```

Add to `MainAppHandler` struct (after `attachmentStore`):

```go
toolsDataStore *toolsdata.Store
```

Add the import: `"github.com/andresbott/etna/internal/toolsdata"`

In `New()`, assign `toolsDataStore: cfg.ToolsDataStore,` in the `MainAppHandler` initialization.

- [ ] **Step 2: Add toolsDataAPI route registration**

In `app/router/api_v0.go`, add a call in `attachApiV0` (before the catch-all 400 handler):

```go
h.toolsDataAPI(r)
```

Add the method and import at the end of the file:

```go
const toolsDataPath = "/tools/{toolType:[a-z0-9-]+}/cases"

func (h *MainAppHandler) toolsDataAPI(r *mux.Router) {
	if h.toolsDataStore == nil {
		return
	}
	tdHandler := toolsDataHandler.Handler{Store: h.toolsDataStore}

	r.Path(toolsDataPath).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		tdHandler.ListCases(toolType).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", toolsDataPath)).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		tdHandler.GetCase(toolType, itemId).ServeHTTP(w, r)
	})

	r.Path(toolsDataPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		tdHandler.CreateCase(toolType).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", toolsDataPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		tdHandler.UpdateCase(toolType, itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", toolsDataPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		toolType := mux.Vars(r)["toolType"]
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		tdHandler.DeleteCase(toolType, itemId).ServeHTTP(w, r)
	})
}
```

Add import: `toolsDataHandler "github.com/andresbott/etna/app/router/handlers/toolsdata"`

- [ ] **Step 3: Create the store in server.go and pass to router**

In `app/cmd/server.go`:

1. Add import: `"github.com/andresbott/etna/internal/toolsdata"`

2. In `initStores`, after `attachmentStore` creation (before the return), add:

```go
toolsDataStore, err := toolsdata.NewStore(db)
if err != nil {
	return nil, nil, nil, nil, nil, fmt.Errorf("tools data store: %w", err)
}
```

3. Update `initStores` signature and return to include `*toolsdata.Store`:

```go
func initStores(db *gorm.DB, cfg AppCfg) (*marketdata.Store, *accounting.Store, *csvimport.Store, *filestore.Store, *toolsdata.Store, error) {
```

Final return: `return marketStore, finStore, csvImportStore, attachmentStore, toolsDataStore, nil`

**Important:** Also update ALL existing early-return error statements in `initStores` — each currently returns 4 `nil` values + error. They all need to return 5 `nil` values + error now (e.g. `return nil, nil, nil, nil, nil, fmt.Errorf(...)`).

4. Update the caller in `runServer` (~line 106):

```go
marketStore, finStore, csvImportStore, attachmentStore, toolsDataStore, err := initStores(db, cfg)
```

5. Add `ToolsDataStore: toolsDataStore,` to the `routerCfg` struct literal (~line 154).

- [ ] **Step 4: Verify full build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 5: Run all existing tests to verify no regressions**

Run: `go test ./...`
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add app/router/handlers/toolsdata/ app/router/main.go app/router/api_v0.go app/cmd/server.go
git commit -m "feat: add tools data HTTP handler and router wiring"
```

---

### Task 5: Write handler tests

**Files:**
- Create: `app/router/handlers/toolsdata/handler_test.go`

- [ ] **Step 1: Write handler HTTP tests**

Create `app/router/handlers/toolsdata/handler_test.go`:

```go
package toolsdata

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andresbott/etna/internal/toolsdata"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testHandler(t *testing.T) *Handler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to open sqlite: %v", err)
	}
	store, err := toolsdata.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create store: %v", err)
	}
	return &Handler{Store: store}
}

func TestListCases_Empty(t *testing.T) {
	h := testHandler(t)
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v0/tools/portfolio-simulator/cases", nil)
	h.ListCases("portfolio-simulator").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var items []casePayload
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("unable to decode response: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestCreateCase_Success(t *testing.T) {
	h := testHandler(t)
	body := `{"name":"Test Case","description":"desc","expectedAnnualReturn":5.0,"params":{"x":1}}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v0/tools/portfolio-simulator/cases", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var p casePayload
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("unable to decode response: %v", err)
	}
	if p.ID == 0 {
		t.Error("expected non-zero id")
	}
	if p.Name != "Test Case" {
		t.Errorf("expected name %q, got %q", "Test Case", p.Name)
	}
	if p.ToolType != "portfolio-simulator" {
		t.Errorf("expected toolType %q, got %q", "portfolio-simulator", p.ToolType)
	}
}

func TestCreateCase_ValidationError(t *testing.T) {
	h := testHandler(t)
	body := `{"name":"","description":"desc"}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v0/tools/portfolio-simulator/cases", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetCase_NotFound(t *testing.T) {
	h := testHandler(t)
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v0/tools/portfolio-simulator/cases/99999", nil)
	h.GetCase("portfolio-simulator", 99999).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateCase_Success(t *testing.T) {
	h := testHandler(t)

	// Create first
	body := `{"name":"Original","description":"desc","expectedAnnualReturn":5.0,"params":{"x":1}}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	var created casePayload
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// Update
	updateBody := `{"name":"Updated","description":"new desc","expectedAnnualReturn":7.0,"params":{"x":2}}`
	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("PUT", "/", bytes.NewBufferString(updateBody))
	h.UpdateCase("portfolio-simulator", created.ID).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var updated casePayload
	_ = json.Unmarshal(rec2.Body.Bytes(), &updated)
	if updated.Name != "Updated" {
		t.Errorf("expected name %q, got %q", "Updated", updated.Name)
	}
}

func TestUpdateCase_NotFound(t *testing.T) {
	h := testHandler(t)
	body := `{"name":"X","description":""}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/", bytes.NewBufferString(body))
	h.UpdateCase("portfolio-simulator", 99999).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestListCases_AfterCreate(t *testing.T) {
	h := testHandler(t)

	// Create a case
	body := `{"name":"Listed Case","description":"desc","expectedAnnualReturn":4.0,"params":{}}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	// List
	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/", nil)
	h.ListCases("portfolio-simulator").ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec2.Code)
	}
	var items []casePayload
	_ = json.Unmarshal(rec2.Body.Bytes(), &items)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "Listed Case" {
		t.Errorf("expected name %q, got %q", "Listed Case", items[0].Name)
	}
}

func TestDeleteCase_Success(t *testing.T) {
	h := testHandler(t)

	// Create first
	body := `{"name":"To Delete","description":"","expectedAnnualReturn":3.0,"params":{}}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	var created casePayload
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// Delete
	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("DELETE", "/", nil)
	h.DeleteCase("portfolio-simulator", created.ID).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec2.Code)
	}
}
```

- [ ] **Step 2: Run handler tests**

Run: `go test ./app/router/handlers/toolsdata/ -v`
Expected: all tests PASS

- [ ] **Step 3: Commit**

```bash
git add app/router/handlers/toolsdata/handler_test.go
git commit -m "test: add handler tests for tools data API"
```

---

## Chunk 3: Frontend API Client and Portfolio Simulator Integration

### Task 6: Create frontend API client and tests

**Files:**
- Create: `webui/src/lib/api/ToolsData.ts`
- Create: `webui/src/lib/api/ToolsData.test.ts`

- [ ] **Step 1: Write the API client**

Create `webui/src/lib/api/ToolsData.ts`:

```typescript
import { apiClient } from '@/lib/api/client'

export interface CaseStudy<T = Record<string, unknown>> {
    id: number
    toolType: string
    name: string
    description: string
    expectedAnnualReturn: number
    params: T
    createdAt: string
    updatedAt: string
}

export interface PortfolioSimulatorParams {
    durationYears: number
    initialContribution: number
    monthlyContribution: number
    growthRatePct: number
    inflationPct: number
    capitalGainTaxPct: number
}

function toolPath(toolType: string): string {
    return `/tools/${toolType}/cases`
}

export async function listCases<T = Record<string, unknown>>(toolType: string): Promise<CaseStudy<T>[]> {
    const { data } = await apiClient.get<CaseStudy<T>[]>(toolPath(toolType))
    return data ?? []
}

export async function getCase<T = Record<string, unknown>>(toolType: string, id: number): Promise<CaseStudy<T>> {
    const { data } = await apiClient.get<CaseStudy<T>>(`${toolPath(toolType)}/${id}`)
    return data
}

export async function createCase<T = Record<string, unknown>>(
    toolType: string,
    payload: { name: string; description: string; expectedAnnualReturn: number; params: T }
): Promise<CaseStudy<T>> {
    const { data } = await apiClient.post<CaseStudy<T>>(toolPath(toolType), payload)
    return data
}

export async function updateCase<T = Record<string, unknown>>(
    toolType: string,
    id: number,
    payload: { name?: string; description?: string; expectedAnnualReturn?: number; params?: T }
): Promise<CaseStudy<T>> {
    const { data } = await apiClient.put<CaseStudy<T>>(`${toolPath(toolType)}/${id}`, payload)
    return data
}

export async function deleteCase(toolType: string, id: number): Promise<void> {
    await apiClient.delete(`${toolPath(toolType)}/${id}`)
}
```

- [ ] **Step 2: Write the API client tests**

Create `webui/src/lib/api/ToolsData.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { apiClient } from './client'
import { listCases, getCase, createCase, updateCase, deleteCase, type CaseStudy } from './ToolsData'

vi.mock('./client', () => ({
    apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

beforeEach(() => vi.clearAllMocks())

const mockCase: CaseStudy = {
    id: 1,
    toolType: 'portfolio-simulator',
    name: 'Conservative',
    description: 'Low risk',
    expectedAnnualReturn: 4.5,
    params: { durationYears: 20, growthRatePct: 6 },
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
}

describe('listCases', () => {
    it('calls GET /tools/{toolType}/cases and returns items', async () => {
        const items = [mockCase];
        (apiClient.get as Mock).mockResolvedValue({ data: items })

        const result = await listCases('portfolio-simulator')

        expect(apiClient.get).toHaveBeenCalledWith('/tools/portfolio-simulator/cases')
        expect(result).toEqual(items)
    })

    it('returns empty array when data is null', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: null })

        const result = await listCases('portfolio-simulator')

        expect(result).toEqual([])
    })
})

describe('getCase', () => {
    it('calls GET /tools/{toolType}/cases/{id}', async () => {
        (apiClient.get as Mock).mockResolvedValue({ data: mockCase })

        const result = await getCase('portfolio-simulator', 1)

        expect(apiClient.get).toHaveBeenCalledWith('/tools/portfolio-simulator/cases/1')
        expect(result).toEqual(mockCase)
    })
})

describe('createCase', () => {
    it('calls POST /tools/{toolType}/cases with payload', async () => {
        const payload = { name: 'New', description: 'desc', expectedAnnualReturn: 5.0, params: { x: 1 } };
        (apiClient.post as Mock).mockResolvedValue({ data: mockCase })

        const result = await createCase('portfolio-simulator', payload)

        expect(apiClient.post).toHaveBeenCalledWith('/tools/portfolio-simulator/cases', payload)
        expect(result).toEqual(mockCase)
    })
})

describe('updateCase', () => {
    it('calls PUT /tools/{toolType}/cases/{id} with payload', async () => {
        const payload = { name: 'Updated' };
        (apiClient.put as Mock).mockResolvedValue({ data: { ...mockCase, name: 'Updated' } })

        const result = await updateCase('portfolio-simulator', 1, payload)

        expect(apiClient.put).toHaveBeenCalledWith('/tools/portfolio-simulator/cases/1', payload)
        expect(result.name).toBe('Updated')
    })
})

describe('deleteCase', () => {
    it('calls DELETE /tools/{toolType}/cases/{id}', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        await deleteCase('portfolio-simulator', 1)

        expect(apiClient.delete).toHaveBeenCalledWith('/tools/portfolio-simulator/cases/1')
    })

    it('returns void', async () => {
        (apiClient.delete as Mock).mockResolvedValue({})

        const result = await deleteCase('portfolio-simulator', 1)

        expect(result).toBeUndefined()
    })
})
```

- [ ] **Step 3: Run frontend tests**

Run: `cd webui && npx vitest run src/lib/api/ToolsData.test.ts`
Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add webui/src/lib/api/ToolsData.ts webui/src/lib/api/ToolsData.test.ts
git commit -m "feat: add frontend API client for tools data"
```

---

### Task 7: Add save/load panel to PortfolioSimulatorView

**Files:**
- Modify: `webui/src/views/tools/PortfolioSimulatorView.vue`

- [ ] **Step 1: Add case study save/load functionality**

In `webui/src/views/tools/PortfolioSimulatorView.vue`, add the following changes:

1. Add imports at the top of the `<script setup>` block (after existing imports):

```typescript
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Dialog from 'primevue/dialog'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import { listCases, createCase, updateCase, deleteCase, type CaseStudy, type PortfolioSimulatorParams } from '@/lib/api/ToolsData'
```

2. Add state variables (after existing refs):

```typescript
const TOOL_TYPE = 'portfolio-simulator'

const cases = ref<CaseStudy<PortfolioSimulatorParams>[]>([])
const showSaveDialog = ref(false)
const saveName = ref('')
const saveDescription = ref('')
const editingCaseId = ref<number | null>(null)

async function loadCases() {
    try {
        cases.value = await listCases<PortfolioSimulatorParams>(TOOL_TYPE)
    } catch (e) {
        console.error('Failed to load case studies:', e)
    }
}

function getCurrentParams(): PortfolioSimulatorParams {
    return {
        durationYears: durationYears.value,
        initialContribution: initialContribution.value,
        monthlyContribution: monthlyContribution.value,
        growthRatePct: growthRatePct.value,
        inflationPct: inflationPct.value,
        capitalGainTaxPct: capitalGainTaxPct.value,
    }
}

function computeExpectedAnnualReturn(): number {
    const growth = growthRatePct.value ?? 0
    const tax = capitalGainTaxPct.value ?? 0
    const inflation = inflationPct.value ?? 0
    return growth - tax - inflation
}

function openSaveDialog(cs?: CaseStudy<PortfolioSimulatorParams>) {
    if (cs) {
        editingCaseId.value = cs.id
        saveName.value = cs.name
        saveDescription.value = cs.description
    } else {
        editingCaseId.value = null
        saveName.value = ''
        saveDescription.value = ''
    }
    showSaveDialog.value = true
}

async function saveCase() {
    const payload = {
        name: saveName.value,
        description: saveDescription.value,
        expectedAnnualReturn: computeExpectedAnnualReturn(),
        params: getCurrentParams(),
    }
    try {
        if (editingCaseId.value) {
            await updateCase<PortfolioSimulatorParams>(TOOL_TYPE, editingCaseId.value, payload)
        } else {
            await createCase<PortfolioSimulatorParams>(TOOL_TYPE, payload)
        }
        showSaveDialog.value = false
        await loadCases()
    } catch (e) {
        console.error('Failed to save case study:', e)
    }
}

function loadCase(cs: CaseStudy<PortfolioSimulatorParams>) {
    const p = cs.params
    durationYears.value = p.durationYears
    initialContribution.value = p.initialContribution
    monthlyContribution.value = p.monthlyContribution
    growthRatePct.value = p.growthRatePct
    inflationPct.value = p.inflationPct
    capitalGainTaxPct.value = p.capitalGainTaxPct
}

async function removeCaseStudy(id: number) {
    try {
        await deleteCase(TOOL_TYPE, id)
        await loadCases()
    } catch (e) {
        console.error('Failed to delete case study:', e)
    }
}

// Load cases on mount
loadCases()
```

3. Add the UI panel in the template, inside the existing grid, after the Projection card's `</div>` (the `col-12 md:col-8` div). Add a new full-width row:

```html
<div class="col-12">
    <Card>
        <template #title>
            <div class="flex align-items-center justify-content-between">
                <span>Case Studies</span>
                <Button label="Save Current" icon="pi pi-save" size="small" @click="openSaveDialog()" />
            </div>
        </template>
        <template #content>
            <DataTable :value="cases" size="small" v-if="cases.length > 0">
                <Column field="name" header="Name" />
                <Column field="description" header="Description" />
                <Column field="expectedAnnualReturn" header="Expected Annual Return">
                    <template #body="{ data }">{{ data.expectedAnnualReturn.toFixed(2) }}%</template>
                </Column>
                <Column header="Actions" style="width: 10rem">
                    <template #body="{ data }">
                        <div class="flex gap-1">
                            <Button icon="pi pi-upload" size="small" text @click="loadCase(data)" title="Load" />
                            <Button icon="pi pi-pencil" size="small" text @click="openSaveDialog(data)" title="Update with current values" />
                            <Button icon="pi pi-trash" size="small" text severity="danger" @click="removeCaseStudy(data.id)" title="Delete" />
                        </div>
                    </template>
                </Column>
            </DataTable>
            <p v-else class="text-color-secondary">No saved case studies yet. Use "Save Current" to store your parameters.</p>
        </template>
    </Card>
</div>

<Dialog v-model:visible="showSaveDialog" header="Save Case Study" :modal="true" :style="{ width: '30rem' }">
    <div class="flex flex-column gap-3">
        <div class="field">
            <label for="caseName">Name</label>
            <InputText id="caseName" v-model="saveName" class="w-full" />
        </div>
        <div class="field">
            <label for="caseDesc">Description</label>
            <Textarea id="caseDesc" v-model="saveDescription" rows="3" class="w-full" />
        </div>
    </div>
    <template #footer>
        <Button label="Cancel" text @click="showSaveDialog = false" />
        <Button label="Save" @click="saveCase" :disabled="!saveName" />
    </template>
</Dialog>
```

- [ ] **Step 2: Verify the frontend builds**

Run: `cd webui && npm run build`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add webui/src/views/tools/PortfolioSimulatorView.vue
git commit -m "feat: add save/load case studies to portfolio simulator"
```

---

### Task 8: Smoke test

- [ ] **Step 1: Run full backend test suite**

Run: `go test ./...`
Expected: all tests PASS

- [ ] **Step 2: Run full frontend test suite**

Run: `cd webui && npx vitest run`
Expected: all tests PASS

- [ ] **Step 3: Manual smoke test**

Start the app and verify:
1. Navigate to portfolio simulator
2. Set parameters, click "Save Current", enter a name → case appears in table
3. Change parameters, click load on saved case → form repopulates
4. Delete a case → removed from table

- [ ] **Step 4: Final commit if any cleanup needed**
