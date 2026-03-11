# Category Rule Groups Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the flat category rule list with grouped rules, where each group has a name, target category, position, and N patterns.

**Architecture:** Two new DB tables (GORM default names: `db_category_rule_groups` + `db_category_rule_patterns`) replace the single `db_category_rules` table. A startup migration converts existing flat rules into single-pattern groups. Groups are ordered by position; patterns within a group are unordered. The matching logic iterates groups in order; within each group, any matching pattern triggers the group's category. The API becomes nested (groups contain patterns). The frontend uses expandable DataTable rows.

**Note:** The `validationErr` variable used in handler error checks is declared in `app/router/handlers/csvimport/profile.go` (line 30) and shared across the package.

**Tech Stack:** Go/GORM (backend), Vue 3/PrimeVue (frontend), SQLite, gorilla/mux

**Spec:** `docs/superpowers/specs/2026-03-11-category-rule-groups-design.md`

---

## Chunk 1: Backend Store Layer

### Task 1: Replace store models and CRUD in `internal/csvimport/category_rule.go`

**Files:**
- Rewrite: `internal/csvimport/category_rule.go`

- [ ] **Step 1: Rewrite `category_rule.go` with new models and CRUD**

Replace the entire file. The new file defines two DB models (`dbCategoryRuleGroup`, `dbCategoryRulePattern`), two public types (`CategoryRuleGroup`, `CategoryRulePattern`), and the following store methods:

- `CreateCategoryRuleGroup(ctx, group) (uint, error)` -- validates name + categoryID, creates group
- `GetCategoryRuleGroup(ctx, id) (CategoryRuleGroup, error)` -- returns group with patterns preloaded
- `ListCategoryRuleGroups(ctx) ([]CategoryRuleGroup, error)` -- returns all groups ordered by position ASC, ID ASC, each with patterns preloaded
- `UpdateCategoryRuleGroup(ctx, id, group) error` -- validates name + categoryID, updates group fields only (not patterns)
- `DeleteCategoryRuleGroup(ctx, id) error` -- deletes all patterns for group, then deletes group
- `CreateCategoryRulePattern(ctx, groupID, pattern) (uint, error)` -- validates pattern not empty, validates regex if isRegex, validates group exists
- `UpdateCategoryRulePattern(ctx, groupID, patternID, pattern) error` -- validates pattern not empty, validates regex if isRegex, validates pattern belongs to group
- `DeleteCategoryRulePattern(ctx, groupID, patternID) error` -- validates pattern belongs to group, deletes

```go
package csvimport

import (
	"context"
	"errors"
	"regexp"
	"time"

	"gorm.io/gorm"
)

var (
	ErrCategoryRuleGroupNotFound   = errors.New("category rule group not found")
	ErrCategoryRulePatternNotFound = errors.New("category rule pattern not found")
)

type dbCategoryRuleGroup struct {
	ID         uint                   `gorm:"primarykey"`
	Name       string                 `gorm:"not null"`
	CategoryID uint                   `gorm:"not null;index"`
	Position   int                    `gorm:"not null;index"`
	Patterns   []dbCategoryRulePattern `gorm:"foreignKey:GroupID"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type dbCategoryRulePattern struct {
	ID        uint   `gorm:"primarykey"`
	GroupID   uint   `gorm:"not null;index"`
	Pattern   string `gorm:"not null"`
	IsRegex   bool   `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CategoryRuleGroup struct {
	ID         uint
	Name       string
	CategoryID uint
	Position   int
	Patterns   []CategoryRulePattern
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CategoryRulePattern struct {
	ID        uint
	GroupID   uint
	Pattern   string
	IsRegex   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func dbToGroup(in dbCategoryRuleGroup) CategoryRuleGroup {
	g := CategoryRuleGroup{
		ID:         in.ID,
		Name:       in.Name,
		CategoryID: in.CategoryID,
		Position:   in.Position,
		CreatedAt:  in.CreatedAt,
		UpdatedAt:  in.UpdatedAt,
	}
	for _, p := range in.Patterns {
		g.Patterns = append(g.Patterns, CategoryRulePattern{
			ID:        p.ID,
			GroupID:   p.GroupID,
			Pattern:   p.Pattern,
			IsRegex:   p.IsRegex,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return g
}

func (s *Store) CreateCategoryRuleGroup(ctx context.Context, g CategoryRuleGroup) (uint, error) {
	if g.Name == "" {
		return 0, ErrValidation("name cannot be empty")
	}
	if g.CategoryID == 0 {
		return 0, ErrValidation("category_id cannot be zero")
	}

	row := dbCategoryRuleGroup{
		Name:       g.Name,
		CategoryID: g.CategoryID,
		Position:   g.Position,
	}
	for _, p := range g.Patterns {
		if p.Pattern == "" {
			return 0, ErrValidation("pattern cannot be empty")
		}
		if p.IsRegex {
			if _, err := regexp.Compile(p.Pattern); err != nil {
				return 0, ErrValidation("invalid regex pattern: " + err.Error())
			}
		}
		row.Patterns = append(row.Patterns, dbCategoryRulePattern{
			Pattern: p.Pattern,
			IsRegex: p.IsRegex,
		})
	}

	d := s.db.WithContext(ctx).Create(&row)
	if d.Error != nil {
		return 0, d.Error
	}
	return row.ID, nil
}

func (s *Store) GetCategoryRuleGroup(ctx context.Context, id uint) (CategoryRuleGroup, error) {
	var row dbCategoryRuleGroup
	d := s.db.WithContext(ctx).Preload("Patterns").Where("id = ?", id).First(&row)
	if d.Error != nil {
		if errors.Is(d.Error, gorm.ErrRecordNotFound) {
			return CategoryRuleGroup{}, ErrCategoryRuleGroupNotFound
		}
		return CategoryRuleGroup{}, d.Error
	}
	return dbToGroup(row), nil
}

func (s *Store) ListCategoryRuleGroups(ctx context.Context) ([]CategoryRuleGroup, error) {
	var rows []dbCategoryRuleGroup
	d := s.db.WithContext(ctx).Preload("Patterns").Order("position ASC, id ASC").Find(&rows)
	if d.Error != nil {
		return nil, d.Error
	}
	groups := make([]CategoryRuleGroup, 0, len(rows))
	for _, row := range rows {
		groups = append(groups, dbToGroup(row))
	}
	return groups, nil
}

func (s *Store) UpdateCategoryRuleGroup(ctx context.Context, id uint, g CategoryRuleGroup) error {
	if g.Name == "" {
		return ErrValidation("name cannot be empty")
	}
	if g.CategoryID == 0 {
		return ErrValidation("category_id cannot be zero")
	}

	d := s.db.WithContext(ctx).Model(&dbCategoryRuleGroup{}).Where("id = ?", id).
		Select("Name", "CategoryID", "Position").
		Updates(dbCategoryRuleGroup{
			Name:       g.Name,
			CategoryID: g.CategoryID,
			Position:   g.Position,
		})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRuleGroupNotFound
	}
	return nil
}

func (s *Store) DeleteCategoryRuleGroup(ctx context.Context, id uint) error {
	// Delete patterns first
	if err := s.db.WithContext(ctx).Where("group_id = ?", id).Delete(&dbCategoryRulePattern{}).Error; err != nil {
		return err
	}
	d := s.db.WithContext(ctx).Where("id = ?", id).Delete(&dbCategoryRuleGroup{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRuleGroupNotFound
	}
	return nil
}

func (s *Store) CreateCategoryRulePattern(ctx context.Context, groupID uint, p CategoryRulePattern) (uint, error) {
	if p.Pattern == "" {
		return 0, ErrValidation("pattern cannot be empty")
	}
	if p.IsRegex {
		if _, err := regexp.Compile(p.Pattern); err != nil {
			return 0, ErrValidation("invalid regex pattern: " + err.Error())
		}
	}
	// Verify group exists
	var count int64
	s.db.WithContext(ctx).Model(&dbCategoryRuleGroup{}).Where("id = ?", groupID).Count(&count)
	if count == 0 {
		return 0, ErrCategoryRuleGroupNotFound
	}

	row := dbCategoryRulePattern{
		GroupID: groupID,
		Pattern: p.Pattern,
		IsRegex: p.IsRegex,
	}
	d := s.db.WithContext(ctx).Create(&row)
	if d.Error != nil {
		return 0, d.Error
	}
	return row.ID, nil
}

func (s *Store) UpdateCategoryRulePattern(ctx context.Context, groupID, patternID uint, p CategoryRulePattern) error {
	if p.Pattern == "" {
		return ErrValidation("pattern cannot be empty")
	}
	if p.IsRegex {
		if _, err := regexp.Compile(p.Pattern); err != nil {
			return ErrValidation("invalid regex pattern: " + err.Error())
		}
	}

	d := s.db.WithContext(ctx).Model(&dbCategoryRulePattern{}).
		Where("id = ? AND group_id = ?", patternID, groupID).
		Select("Pattern", "IsRegex").
		Updates(dbCategoryRulePattern{
			Pattern: p.Pattern,
			IsRegex: p.IsRegex,
		})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRulePatternNotFound
	}
	return nil
}

func (s *Store) DeleteCategoryRulePattern(ctx context.Context, groupID, patternID uint) error {
	d := s.db.WithContext(ctx).Where("id = ? AND group_id = ?", patternID, groupID).Delete(&dbCategoryRulePattern{})
	if d.Error != nil {
		return d.Error
	}
	if d.RowsAffected == 0 {
		return ErrCategoryRulePatternNotFound
	}
	return nil
}
```

- [ ] **Step 2: Update `csvimport.go` AutoMigrate and WipeData**

Modify: `internal/csvimport/csvimport.go`

Change AutoMigrate to use new models and WipeData to reference new table names:

```go
// In NewStore:
err := db.AutoMigrate(&dbImportProfile{}, &dbCategoryRuleGroup{}, &dbCategoryRulePattern{})

// In WipeData:
tables := []string{"db_category_rule_patterns", "db_category_rule_groups", "db_import_profiles"}
```

Note: patterns table must be wiped before groups table due to FK.

- [ ] **Step 3: Add data migration function**

Add a `migrateOldCategoryRules` function to `csvimport.go` that runs in `NewStore` after AutoMigrate. It converts existing flat `db_category_rules` rows (if the table exists) into the new group+pattern structure, then drops the old table.

```go
func migrateOldCategoryRules(db *gorm.DB) error {
	// Check if old table exists
	if !db.Migrator().HasTable("db_category_rules") {
		return nil
	}

	type oldRule struct {
		ID         uint
		Pattern    string
		IsRegex    bool
		CategoryID uint
		Position   int
	}

	var oldRules []oldRule
	if err := db.Table("db_category_rules").Find(&oldRules).Error; err != nil {
		return fmt.Errorf("failed to read old category rules: %w", err)
	}

	for _, old := range oldRules {
		group := dbCategoryRuleGroup{
			Name:       old.Pattern,
			CategoryID: old.CategoryID,
			Position:   old.Position,
			Patterns: []dbCategoryRulePattern{
				{Pattern: old.Pattern, IsRegex: old.IsRegex},
			},
		}
		if err := db.Create(&group).Error; err != nil {
			return fmt.Errorf("failed to migrate rule %d: %w", old.ID, err)
		}
	}

	if err := db.Migrator().DropTable("db_category_rules"); err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}
	return nil
}
```

Call it in `NewStore` after AutoMigrate:

```go
err = migrateOldCategoryRules(db)
if err != nil {
    return nil, fmt.Errorf("error migrating old category rules: %w", err)
}
```

- [ ] **Step 4: Run compilation check**

Run: `cd internal/csvimport && go build ./...`
Expected: Compilation errors in test files (they reference old types). The main code should compile.

- [ ] **Step 5: Commit**

```
feat: replace category rule models with group+pattern structure
```

---

### Task 2: Rewrite store tests in `internal/csvimport/category_rule_test.go`

**Files:**
- Rewrite: `internal/csvimport/category_rule_test.go`
- Modify: `internal/csvimport/wipe_test.go`

- [ ] **Step 1: Rewrite `category_rule_test.go`**

Replace the entire file with tests covering:

1. `TestCreateCategoryRuleGroup` -- success, success with initial patterns, validation errors (empty name, zero categoryID), invalid regex in initial pattern
2. `TestGetCategoryRuleGroup` -- existing group with patterns, not found
3. `TestListCategoryRuleGroups` -- empty list, ordered by position then id, patterns included
4. `TestUpdateCategoryRuleGroup` -- success, not found, validation errors, position to zero
5. `TestDeleteCategoryRuleGroup` -- success (also deletes patterns), not found
6. `TestCreateCategoryRulePattern` -- success, success regex, invalid regex, empty pattern, group not found
7. `TestUpdateCategoryRulePattern` -- success, not found, wrong group, validation
8. `TestDeleteCategoryRulePattern` -- success, not found, wrong group

Use the existing `newTestStore(t)` helper and follow the same test patterns as the current file.

Helper function at top:

```go
func validCategoryRuleGroup() CategoryRuleGroup {
	return CategoryRuleGroup{
		Name:       "Test Group",
		CategoryID: 1,
		Position:   10,
	}
}
```

- [ ] **Step 2: Update `wipe_test.go`**

Replace the category rule section to use the new group/pattern methods:

```go
// Create a category rule group with a pattern
g := validCategoryRuleGroup()
g.Patterns = []CategoryRulePattern{{Pattern: "TEST", IsRegex: false}}
_, err = store.CreateCategoryRuleGroup(ctx, g)
if err != nil {
    t.Fatalf("unexpected error creating category rule group: %v", err)
}

// Verify data exists
groups, err := store.ListCategoryRuleGroups(ctx)
if err != nil {
    t.Fatalf("unexpected error listing category rule groups: %v", err)
}
if len(groups) == 0 {
    t.Fatal("expected at least one category rule group before wipe")
}
```

And after wipe:

```go
groups, err = store.ListCategoryRuleGroups(ctx)
if err != nil {
    t.Fatalf("unexpected error listing category rule groups after wipe: %v", err)
}
if len(groups) != 0 {
    t.Fatalf("expected 0 category rule groups after wipe, got %d", len(groups))
}
```

- [ ] **Step 3: Run tests**

Run: `cd internal/csvimport && go test ./... -count=1`
Expected: All tests pass. There may be compilation errors in `parser_test.go` if it references old types -- those are fixed in Task 3.

- [ ] **Step 4: Commit**

```
test: rewrite category rule store tests for group+pattern model
```

---

### Task 3: Update parser matching logic

**Files:**
- Modify: `internal/csvimport/parser.go` (lines 811, 892, 914-929)
- Modify: `internal/csvimport/parser_test.go` (update any calls to `Parse` or `MatchCategory` that reference old types)

- [ ] **Step 1: Update `MatchCategory` function signature and logic**

Replace the `MatchCategory` function (line 914-929):

```go
// MatchCategory iterates groups in order and returns the categoryID of the first
// group where any pattern matches, or 0 if none match.
func MatchCategory(description string, groups []CategoryRuleGroup) uint {
	descLower := strings.ToLower(description)
	for _, group := range groups {
		for _, pattern := range group.Patterns {
			if pattern.IsRegex {
				matched, err := regexp.MatchString(pattern.Pattern, description)
				if err == nil && matched {
					return group.CategoryID
				}
			} else {
				if strings.Contains(descLower, strings.ToLower(pattern.Pattern)) {
					return group.CategoryID
				}
			}
		}
	}
	return 0
}
```

- [ ] **Step 2: Update `Parse` function signature**

Change line 811 from:

```go
func Parse(r io.Reader, profile ImportProfile, rules []CategoryRule, existing []ExistingTx) ([]ParsedRow, error) {
```

to:

```go
func Parse(r io.Reader, profile ImportProfile, groups []CategoryRuleGroup, existing []ExistingTx) ([]ParsedRow, error) {
```

And update line 892 from `MatchCategory(parsed.Description, rules)` to `MatchCategory(parsed.Description, groups)`.

- [ ] **Step 3: Update `parser_test.go`**

Update any calls to `Parse()` that pass `[]CategoryRule` to now pass `[]CategoryRuleGroup`. If tests pass `nil` for rules, they can pass `nil` for groups (the type change is compatible). If any tests call `MatchCategory` directly, update the signature to pass groups instead of flat rules.

- [ ] **Step 4: Run tests**

Run: `cd internal/csvimport && go test ./... -count=1`
Expected: All pass.

- [ ] **Step 5: Commit**

```
feat: update MatchCategory and Parse to use rule groups
```

---

## Chunk 2: Backend Handlers and Router

### Task 4: Rewrite HTTP handler in `app/router/handlers/csvimport/category_rule.go`

**Files:**
- Rewrite: `app/router/handlers/csvimport/category_rule.go`

- [ ] **Step 1: Rewrite `category_rule.go` with group + pattern handlers**

Replace the entire file. The new handler struct is `CategoryRuleGroupHandler` with store reference. Methods:

- `ListCategoryRuleGroups() http.Handler` -- GET, returns JSON array of groups with nested patterns
- `CreateCategoryRuleGroup() http.Handler` -- POST, accepts group JSON (with optional patterns array)
- `UpdateCategoryRuleGroup(id uint) http.Handler` -- PUT, accepts group JSON (name, categoryId, position)
- `DeleteCategoryRuleGroup(id uint) http.Handler` -- DELETE
- `CreateCategoryRulePattern(groupID uint) http.Handler` -- POST, accepts pattern JSON
- `UpdateCategoryRulePattern(groupID, patternID uint) http.Handler` -- PUT, accepts pattern JSON
- `DeleteCategoryRulePattern(groupID, patternID uint) http.Handler` -- DELETE

```go
package csvimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andresbott/etna/internal/csvimport"
)

type CategoryRuleGroupHandler struct {
	Store *csvimport.Store
}

type ruleGroupPayload struct {
	ID         uint                 `json:"id"`
	Name       string               `json:"name"`
	CategoryID uint                 `json:"categoryId"`
	Position   int                  `json:"position"`
	Patterns   []rulePatternPayload `json:"patterns"`
}

type rulePatternPayload struct {
	ID      uint   `json:"id"`
	Pattern string `json:"pattern"`
	IsRegex bool   `json:"isRegex"`
}

func (h *CategoryRuleGroupHandler) ListCategoryRuleGroups() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groups, err := h.Store.ListCategoryRuleGroups(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to list category rule groups: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		items := make([]ruleGroupPayload, len(groups))
		for i, g := range groups {
			patterns := make([]rulePatternPayload, len(g.Patterns))
			for j, p := range g.Patterns {
				patterns[j] = rulePatternPayload{
					ID:      p.ID,
					Pattern: p.Pattern,
					IsRegex: p.IsRegex,
				}
			}
			items[i] = ruleGroupPayload{
				ID:         g.ID,
				Name:       g.Name,
				CategoryID: g.CategoryID,
				Position:   g.Position,
				Patterns:   patterns,
			}
		}

		respJSON, err := json.Marshal(items)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *CategoryRuleGroupHandler) CreateCategoryRuleGroup() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload ruleGroupPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		group := csvimport.CategoryRuleGroup{
			Name:       payload.Name,
			CategoryID: payload.CategoryID,
			Position:   payload.Position,
		}
		for _, p := range payload.Patterns {
			group.Patterns = append(group.Patterns, csvimport.CategoryRulePattern{
				Pattern: p.Pattern,
				IsRegex: p.IsRegex,
			})
		}

		id, err := h.Store.CreateCategoryRuleGroup(r.Context(), group)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create category rule group: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		// Re-fetch to get generated pattern IDs
		created, err := h.Store.GetCategoryRuleGroup(r.Context(), id)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to fetch created group: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		respPayload := ruleGroupPayload{
			ID:         created.ID,
			Name:       created.Name,
			CategoryID: created.CategoryID,
			Position:   created.Position,
		}
		for _, p := range created.Patterns {
			respPayload.Patterns = append(respPayload.Patterns, rulePatternPayload{
				ID:      p.ID,
				Pattern: p.Pattern,
				IsRegex: p.IsRegex,
			})
		}

		respJSON, err := json.Marshal(respPayload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *CategoryRuleGroupHandler) UpdateCategoryRuleGroup(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload ruleGroupPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		group := csvimport.CategoryRuleGroup{
			Name:       payload.Name,
			CategoryID: payload.CategoryID,
			Position:   payload.Position,
		}

		err := h.Store.UpdateCategoryRuleGroup(r.Context(), id, group)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, csvimport.ErrCategoryRuleGroupNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update category rule group: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *CategoryRuleGroupHandler) DeleteCategoryRuleGroup(id uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.DeleteCategoryRuleGroup(r.Context(), id)
		if err != nil {
			if errors.Is(err, csvimport.ErrCategoryRuleGroupNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete category rule group: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *CategoryRuleGroupHandler) CreateCategoryRulePattern(groupID uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload rulePatternPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		pattern := csvimport.CategoryRulePattern{
			Pattern: payload.Pattern,
			IsRegex: payload.IsRegex,
		}

		id, err := h.Store.CreateCategoryRulePattern(r.Context(), groupID, pattern)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, csvimport.ErrCategoryRuleGroupNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to create pattern: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		payload.ID = id
		respJSON, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}

func (h *CategoryRuleGroupHandler) UpdateCategoryRulePattern(groupID, patternID uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "request had empty body", http.StatusBadRequest)
			return
		}

		var payload rulePatternPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("unable to decode json: %s", err.Error()), http.StatusBadRequest)
			return
		}

		pattern := csvimport.CategoryRulePattern{
			Pattern: payload.Pattern,
			IsRegex: payload.IsRegex,
		}

		err := h.Store.UpdateCategoryRulePattern(r.Context(), groupID, patternID, pattern)
		if err != nil {
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, csvimport.ErrCategoryRulePatternNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to update pattern: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *CategoryRuleGroupHandler) DeleteCategoryRulePattern(groupID, patternID uint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h.Store.DeleteCategoryRulePattern(r.Context(), groupID, patternID)
		if err != nil {
			if errors.Is(err, csvimport.ErrCategoryRulePatternNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("unable to delete pattern: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd app/router/handlers/csvimport && go build ./...`
Expected: May fail due to router not updated yet. The handler package itself should compile.

- [ ] **Step 3: Commit**

```
feat: add HTTP handlers for category rule groups and patterns
```

---

### Task 5: Update import handler and reapply handler

**Files:**
- Modify: `app/router/handlers/csvimport/import.go` (lines 66, 80)
- Modify: `app/router/handlers/csvimport/reapply.go` (lines 37, 99, 118)
- Modify: `app/router/handlers/csvimport/reapply_test.go` (update any references to old types/methods)

- [ ] **Step 1: Update `import.go`**

Line 66: change `h.CsvStore.ListCategoryRules(r.Context())` to `h.CsvStore.ListCategoryRuleGroups(r.Context())`

Line 68: change error message from `"unable to list category rules"` to `"unable to list category rule groups"`

Line 80: the variable name changes from `rules` to `groups`. Update the `Parse` call:
```go
rows, err := csvimport.Parse(file, profile, groups, existing)
```

- [ ] **Step 2: Update `reapply.go`**

Line 37: change `h.CsvStore.ListCategoryRules(ctx)` to `h.CsvStore.ListCategoryRuleGroups(ctx)`
Line 38: update error message to `"unable to list category rule groups"`
Line 43: change `len(rules)` to `len(groups)`, variable name `rules` to `groups`

Line 99: change `csvimport.MatchCategory(item.Description, rules)` to `csvimport.MatchCategory(item.Description, groups)`
Line 118: same change for the expense case.

- [ ] **Step 3: Update `reapply_test.go`**

Update any references to `ListCategoryRules`, `CategoryRule`, or `MatchCategory` with old signatures. If the test creates mock rules, convert them to groups.

- [ ] **Step 4: Verify compilation and tests**

Run: `go build ./app/router/handlers/csvimport/... && go test ./app/router/handlers/csvimport/... -count=1`
Expected: Should compile and tests pass.

- [ ] **Step 5: Commit**

```
refactor: update import and reapply handlers to use rule groups
```

---

### Task 6: Update router registration in `app/router/api_v0.go`

**Files:**
- Modify: `app/router/api_v0.go` (lines 660, 669, 718-762)

- [ ] **Step 1: Add `getVarId` helper for extracting named path vars**

Add near the existing `getId` function (after line 898):

```go
func getVarId(r *http.Request, name string) (uint, *httpError) {
	vars := mux.Vars(r)
	val, ok := vars[name]
	if !ok {
		return 0, &httpError{
			Error: fmt.Sprintf("could not extract %s from request context", name),
			Code:  http.StatusInternalServerError,
		}
	}
	if val == "" {
		return 0, &httpError{
			Error: fmt.Sprintf("no %s provided", name),
			Code:  http.StatusBadRequest,
		}
	}
	u64, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, &httpError{
			Error: fmt.Sprintf("unable to convert %s to number", name),
			Code:  http.StatusBadRequest,
		}
	}
	return uint(u64), nil
}
```

- [ ] **Step 2: Replace category rules route section**

Change the path constant (line 660):
```go
const importCategoryRuleGroupPath = "/import/category-rule-groups"
```

Change the handler initialization (line 669):
```go
ruleGroupHndlr := csvimportHandler.CategoryRuleGroupHandler{Store: h.csvImportStore}
```

Replace the entire Category Rules section (lines 718-762) with:

```go
	// ==========================================================================
	// Category Rule Groups
	// ==========================================================================

	r.Path(importCategoryRuleGroupPath).Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		ruleGroupHndlr.ListCategoryRuleGroups().ServeHTTP(w, r)
	})

	r.Path(importCategoryRuleGroupPath).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		ruleGroupHndlr.CreateCategoryRuleGroup().ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", importCategoryRuleGroupPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		ruleGroupHndlr.UpdateCategoryRuleGroup(itemId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{id}", importCategoryRuleGroupPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		itemId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		ruleGroupHndlr.DeleteCategoryRuleGroup(itemId).ServeHTTP(w, r)
	})

	// Category Rule Patterns (nested under groups)

	r.Path(fmt.Sprintf("%s/{groupId}/patterns", importCategoryRuleGroupPath)).Methods(http.MethodPost).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		groupId, httpErr := getVarId(r, "groupId")
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		ruleGroupHndlr.CreateCategoryRulePattern(groupId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{groupId}/patterns/{id}", importCategoryRuleGroupPath)).Methods(http.MethodPut).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		groupId, httpErr := getVarId(r, "groupId")
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		patternId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		ruleGroupHndlr.UpdateCategoryRulePattern(groupId, patternId).ServeHTTP(w, r)
	})

	r.Path(fmt.Sprintf("%s/{groupId}/patterns/{id}", importCategoryRuleGroupPath)).Methods(http.MethodDelete).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := sessionauth.CtxGetUserData(r); err != nil {
			http.Error(w, fmt.Sprintf("unable to read user data: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		groupId, httpErr := getVarId(r, "groupId")
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		patternId, httpErr := getId(r)
		if httpErr != nil {
			http.Error(w, httpErr.Error, httpErr.Code)
			return
		}
		ruleGroupHndlr.DeleteCategoryRulePattern(groupId, patternId).ServeHTTP(w, r)
	})
```

- [ ] **Step 3: Build the entire backend**

Run: `go build ./...`
Expected: Should compile. Fix any remaining references to old types.

- [ ] **Step 4: Run all Go tests**

Run: `go test ./... -count=1`
Expected: All pass. This catches any remaining compilation or test failures across the whole project.

- [ ] **Step 5: Commit**

```
feat: register category rule group routes in router
```

---

## Chunk 3: Backup System

### Task 7: Update backup data model, export, and import

**Files:**
- Modify: `internal/backup/dataV1.go` (lines 96, 133-139)
- Modify: `internal/backup/export.go` (lines 107, 425-441)
- Modify: `internal/backup/import.go` (lines 89, 306, 467-489)

- [ ] **Step 1: Update `dataV1.go`**

Replace `categoryRuleV1` struct (lines 133-139) with:

```go
type categoryRuleGroupV1 struct {
	ID         uint                    `json:"id"`
	Name       string                  `json:"name"`
	CategoryID uint                    `json:"categoryId"`
	Position   int                     `json:"position"`
	Patterns   []categoryRulePatternV1 `json:"patterns"`
}

type categoryRulePatternV1 struct {
	ID      uint   `json:"id"`
	Pattern string `json:"pattern"`
	IsRegex bool   `json:"isRegex"`
}
```

Keep the `categoryRulesFile` constant unchanged (line 96).

- [ ] **Step 2: Update `export.go`**

Replace `writeCategoryRules` function (lines 425-441):

```go
func writeCategoryRules(ctx context.Context, zw *zipWriter, csvStore *csvimport.Store) error {
	groups, err := csvStore.ListCategoryRuleGroups(ctx)
	if err != nil {
		return err
	}
	jsonData := make([]categoryRuleGroupV1, len(groups))
	for i, g := range groups {
		patterns := make([]categoryRulePatternV1, len(g.Patterns))
		for j, p := range g.Patterns {
			patterns[j] = categoryRulePatternV1{
				ID:      p.ID,
				Pattern: p.Pattern,
				IsRegex: p.IsRegex,
			}
		}
		jsonData[i] = categoryRuleGroupV1{
			ID:         g.ID,
			Name:       g.Name,
			CategoryID: g.CategoryID,
			Position:   g.Position,
			Patterns:   patterns,
		}
	}
	return zw.writeJsonFile(categoryRulesFile, jsonData)
}
```

- [ ] **Step 3: Update `import.go`**

Update the `loadV1Json` type constraint (line 306): replace `[]categoryRuleV1` with `[]categoryRuleGroupV1`.

Replace `importCategoryRules` function (lines 467-489):

```go
func importCategoryRules(ctx context.Context, csvStore *csvimport.Store, r *zip.ReadCloser, incomeMap, expenseMap map[uint]uint) error {
	groups, err := loadV1Json[[]categoryRuleGroupV1](r, categoryRulesFile)
	if err != nil {
		return err
	}
	for _, g := range groups {
		catID := incomeMap[g.CategoryID]
		if catID == 0 {
			catID = expenseMap[g.CategoryID]
		}
		item := csvimport.CategoryRuleGroup{
			Name:       g.Name,
			CategoryID: catID,
			Position:   g.Position,
		}
		for _, p := range g.Patterns {
			item.Patterns = append(item.Patterns, csvimport.CategoryRulePattern{
				Pattern: p.Pattern,
				IsRegex: p.IsRegex,
			})
		}
		_, err := csvStore.CreateCategoryRuleGroup(ctx, item)
		if err != nil {
			return fmt.Errorf("failed to create category rule group: %w", err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Update backup tests**

Update `internal/backup/export_test.go` and `internal/backup/import_test.go` to use the new types:

In `export_test.go`:
- Replace `categoryRuleV1` references with `categoryRuleGroupV1`
- Update test data to use grouped format with nested patterns
- Update `CategoryRules` field in the test struct to use `[]categoryRuleGroupV1`
- Update sample data creation to use `csvStore.CreateCategoryRuleGroup` instead of `csvStore.CreateCategoryRule`
- Update `cmpopts` references from `categoryRuleV1` to `categoryRuleGroupV1`

In `import_test.go`:
- Update `assertImportedCategoryRules` to call `ListCategoryRuleGroups` and check group+pattern structure

- [ ] **Step 5: Run backup tests**

Run: `go test ./internal/backup/... -count=1`
Expected: All pass.

- [ ] **Step 6: Commit**

```
feat: update backup export/import for category rule groups
```

---

## Chunk 4: Frontend

### Task 8: Update TypeScript types and API client

**Files:**
- Modify: `webui/src/types/csvimport.ts` (lines 15-21)
- Modify: `webui/src/lib/api/CsvImport.ts` (lines 11-14)

- [ ] **Step 1: Update `csvimport.ts` types**

Replace the `CategoryRule` interface (lines 15-21) with:

```ts
export interface CategoryRuleGroup {
  id: number
  name: string
  categoryId: number
  position: number
  patterns: CategoryRulePattern[]
}

export interface CategoryRulePattern {
  id: number
  pattern: string
  isRegex: boolean
}
```

- [ ] **Step 2: Update `CsvImport.ts` API functions**

Replace the category rules section (lines 11-14) with:

```ts
// Category Rule Groups
export const getCategoryRuleGroups = () => apiClient.get<CategoryRuleGroup[]>('/import/category-rule-groups').then(r => r.data)
export const createCategoryRuleGroup = (g: Omit<CategoryRuleGroup, 'id'>) => apiClient.post<CategoryRuleGroup>('/import/category-rule-groups', g).then(r => r.data)
export const updateCategoryRuleGroup = (id: number, g: Partial<CategoryRuleGroup>) => apiClient.put(`/import/category-rule-groups/${id}`, g).then(r => r.data)
export const deleteCategoryRuleGroup = (id: number) => apiClient.delete(`/import/category-rule-groups/${id}`).then(r => r.data)

// Category Rule Patterns
export const createCategoryRulePattern = (groupId: number, p: Omit<CategoryRulePattern, 'id'>) => apiClient.post<CategoryRulePattern>(`/import/category-rule-groups/${groupId}/patterns`, p).then(r => r.data)
export const updateCategoryRulePattern = (groupId: number, patternId: number, p: Partial<CategoryRulePattern>) => apiClient.put(`/import/category-rule-groups/${groupId}/patterns/${patternId}`, p).then(r => r.data)
export const deleteCategoryRulePattern = (groupId: number, patternId: number) => apiClient.delete(`/import/category-rule-groups/${groupId}/patterns/${patternId}`).then(r => r.data)
```

Update the import statement at the top to use the new type names:
```ts
import type { ImportProfile, CategoryRuleGroup, CategoryRulePattern, ParsedRow, PreviewResult, ReapplyRow, ReapplySubmitItem } from '@/types/csvimport'
```

- [ ] **Step 3: Update API test file**

Modify `webui/src/lib/api/CsvImport.test.ts` to use the new function names and types. Replace references to `getCategoryRules`, `createCategoryRule`, `updateCategoryRule`, `deleteCategoryRule` with the new group/pattern equivalents.

- [ ] **Step 4: Commit**

```
feat: update frontend types and API client for rule groups
```

---

### Task 9: Rewrite `CategoryRulesView.vue`

**Files:**
- Rewrite: `webui/src/views/csvimport/CategoryRulesView.vue`

- [ ] **Step 1: Rewrite the view**

The new view uses:
- PrimeVue DataTable with row expansion for groups
- Group columns: Position, Name, Category, # Patterns, Actions (edit/delete)
- Expanded row: list of patterns with columns: Pattern, Type (Regex/Substring), Actions (edit/delete) + "Add Pattern" button
- Group create/edit dialog with fields: Name, Category (Select with filter), Position
- Pattern create/edit dialog with fields: Pattern (InputText), Is Regex (Checkbox)

Key changes from current view:
- Import `getCategoryRuleGroups, createCategoryRuleGroup, updateCategoryRuleGroup, deleteCategoryRuleGroup, createCategoryRulePattern, updateCategoryRulePattern, deleteCategoryRulePattern` instead of old functions
- `rules` ref becomes `groups` ref
- `loadRules` becomes `loadGroups`, calls `getCategoryRuleGroups()`
- Two dialogs: one for groups, one for patterns
- `expandedRows` ref for DataTable row expansion
- The DataTable uses `v-model:expandedRows` and a `<template #expansion>` slot for nested patterns

Replace the entire file with a component that:

1. **State:** `groups`, `isLoading`, `expandedRows`, `showGroupDialog`, `editingGroup`, `showPatternDialog`, `editingPattern`, `editingPatternGroupId`, `isSaving`
2. **Group form state:** `formName`, `formCategoryId`, `formPosition`
3. **Pattern form state:** `formPattern`, `formIsRegex`
4. **Methods:**
   - `loadGroups()` -- fetches and sorts by position
   - `openCreateGroupDialog()` / `openEditGroupDialog(group)`
   - `handleSaveGroup()` -- create or update group
   - `handleDeleteGroup(group)` -- confirm and delete
   - `openCreatePatternDialog(groupId)` / `openEditPatternDialog(groupId, pattern)`
   - `handleSavePattern()` -- create or update pattern
   - `handleDeletePattern(groupId, pattern)` -- confirm and delete

5. **Template structure:**
   - Header with title + "New Group" button + "Re-apply Rules" button
   - Card containing DataTable with:
     - Column: expander
     - Column: Position
     - Column: Name
     - Column: Category (resolved name)
     - Column: Patterns count (`data.patterns.length`)
     - Column: Actions (edit/delete buttons)
     - Expansion template: inner table of patterns + "Add Pattern" button
   - Group Dialog (create/edit)
   - Pattern Dialog (create/edit)

- [ ] **Step 2: Verify frontend builds**

Run: `cd webui && npm run build`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```
feat: rewrite CategoryRulesView for grouped rules with expandable rows
```

---

## Chunk 5: Integration Testing and Cleanup

### Task 10: Full backend test pass

**Files:** None new

- [ ] **Step 1: Run all Go tests**

Run: `go test ./... -count=1`
Expected: All pass. Fix any remaining compilation errors from old `CategoryRule` references.

- [ ] **Step 2: Commit any fixes**

```
fix: resolve remaining references to old CategoryRule type
```

---

### Task 11: Run smoke test

- [ ] **Step 1: Run the smoke test skill**

Use the `smoke-test` skill to start backend & frontend, navigate to the category rules page, and verify:
- The groups list loads
- Creating a group works
- Adding patterns to a group works
- Expanding a group shows its patterns
- No console errors

- [ ] **Step 2: Fix any issues found**

- [ ] **Step 3: Final commit**

```
feat: category rule groups - complete implementation
```
