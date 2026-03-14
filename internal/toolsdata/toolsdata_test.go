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
