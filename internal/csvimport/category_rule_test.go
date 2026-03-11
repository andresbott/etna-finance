package csvimport

import (
	"context"
	"errors"
	"testing"
)

func validCategoryRuleGroup() CategoryRuleGroup {
	return CategoryRuleGroup{
		Name:       "Test Group",
		CategoryID: 1,
		Priority:   10,
	}
}

func TestCreateCategoryRuleGroup(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		g := validCategoryRuleGroup()
		id, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero id")
		}
	})

	t.Run("success with initial patterns", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Patterns = []CategoryRulePattern{
			{Pattern: "AMAZON", IsRegex: false},
			{Pattern: "NETFLIX", IsRegex: false},
		}
		id, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero id")
		}

		got, err := store.GetCategoryRuleGroup(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Patterns) != 2 {
			t.Fatalf("expected 2 patterns, got %d", len(got.Patterns))
		}
	})

	t.Run("validation error empty name", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Name = ""
		_, err := store.CreateCategoryRuleGroup(ctx, g)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
		if err.Error() != "name cannot be empty" {
			t.Errorf("expected error %q, got %q", "name cannot be empty", err.Error())
		}
	})

	t.Run("validation error zero categoryID", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.CategoryID = 0
		_, err := store.CreateCategoryRuleGroup(ctx, g)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
		if err.Error() != "category_id cannot be zero" {
			t.Errorf("expected error %q, got %q", "category_id cannot be zero", err.Error())
		}
	})

	t.Run("invalid regex in initial pattern", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Patterns = []CategoryRulePattern{
			{Pattern: `[invalid`, IsRegex: true},
		}
		_, err := store.CreateCategoryRuleGroup(ctx, g)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})
}

func TestGetCategoryRuleGroup(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("existing group with patterns", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Patterns = []CategoryRulePattern{
			{Pattern: "AMAZON", IsRegex: false},
		}
		id, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRuleGroup(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != id {
			t.Errorf("expected id %d, got %d", id, got.ID)
		}
		if got.Name != g.Name {
			t.Errorf("expected name %q, got %q", g.Name, got.Name)
		}
		if got.CategoryID != g.CategoryID {
			t.Errorf("expected category_id %d, got %d", g.CategoryID, got.CategoryID)
		}
		if got.Priority != g.Priority {
			t.Errorf("expected priority %d, got %d", g.Priority, got.Priority)
		}
		if len(got.Patterns) != 1 {
			t.Fatalf("expected 1 pattern, got %d", len(got.Patterns))
		}
		if got.Patterns[0].Pattern != "AMAZON" {
			t.Errorf("expected pattern %q, got %q", "AMAZON", got.Patterns[0].Pattern)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.GetCategoryRuleGroup(ctx, 99999)
		if !errors.Is(err, ErrCategoryRuleGroupNotFound) {
			t.Fatalf("expected ErrCategoryRuleGroupNotFound, got %v", err)
		}
	})
}

func TestListCategoryRuleGroups(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("empty list", func(t *testing.T) {
		groups, err := store.ListCategoryRuleGroups(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(groups) != 0 {
			t.Errorf("expected 0 groups, got %d", len(groups))
		}
	})

	t.Run("ordered by priority then id", func(t *testing.T) {
		g1 := validCategoryRuleGroup()
		g1.Name = "Group C"
		g1.Priority = 20
		id1, err := store.CreateCategoryRuleGroup(ctx, g1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		g2 := validCategoryRuleGroup()
		g2.Name = "Group A"
		g2.Priority = 5
		id2, err := store.CreateCategoryRuleGroup(ctx, g2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		g3 := validCategoryRuleGroup()
		g3.Name = "Group B"
		g3.Priority = 5
		id3, err := store.CreateCategoryRuleGroup(ctx, g3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		groups, err := store.ListCategoryRuleGroups(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(groups) != 3 {
			t.Fatalf("expected 3 groups, got %d", len(groups))
		}
		if groups[0].ID != id2 {
			t.Errorf("expected first group id %d, got %d", id2, groups[0].ID)
		}
		if groups[1].ID != id3 {
			t.Errorf("expected second group id %d, got %d", id3, groups[1].ID)
		}
		if groups[2].ID != id1 {
			t.Errorf("expected third group id %d, got %d", id1, groups[2].ID)
		}
	})

	t.Run("patterns included", func(t *testing.T) {
		store2 := newTestStore(t)
		g := validCategoryRuleGroup()
		g.Patterns = []CategoryRulePattern{
			{Pattern: "PAT1", IsRegex: false},
			{Pattern: "PAT2", IsRegex: true},
		}
		_, err := store2.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		groups, err := store2.ListCategoryRuleGroups(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(groups))
		}
		if len(groups[0].Patterns) != 2 {
			t.Errorf("expected 2 patterns, got %d", len(groups[0].Patterns))
		}
	})
}

func TestUpdateCategoryRuleGroup(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		g := validCategoryRuleGroup()
		id, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated := validCategoryRuleGroup()
		updated.Name = "Updated Group"
		updated.CategoryID = 2
		updated.Priority = 99

		err = store.UpdateCategoryRuleGroup(ctx, id, updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRuleGroup(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "Updated Group" {
			t.Errorf("expected name %q, got %q", "Updated Group", got.Name)
		}
		if got.CategoryID != 2 {
			t.Errorf("expected category_id 2, got %d", got.CategoryID)
		}
		if got.Priority != 99 {
			t.Errorf("expected priority 99, got %d", got.Priority)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := store.UpdateCategoryRuleGroup(ctx, 99999, validCategoryRuleGroup())
		if !errors.Is(err, ErrCategoryRuleGroupNotFound) {
			t.Fatalf("expected ErrCategoryRuleGroupNotFound, got %v", err)
		}
	})

	t.Run("validation error empty name", func(t *testing.T) {
		g := validCategoryRuleGroup()
		id, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u := validCategoryRuleGroup()
		u.Name = ""
		err = store.UpdateCategoryRuleGroup(ctx, id, u)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("validation error zero categoryID", func(t *testing.T) {
		g := validCategoryRuleGroup()
		id, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u := validCategoryRuleGroup()
		u.CategoryID = 0
		err = store.UpdateCategoryRuleGroup(ctx, id, u)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("priority to zero", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Priority = 50
		id, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u := validCategoryRuleGroup()
		u.Priority = 0
		err = store.UpdateCategoryRuleGroup(ctx, id, u)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRuleGroup(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Priority != 0 {
			t.Errorf("expected priority 0, got %d", got.Priority)
		}
	})
}

func TestDeleteCategoryRuleGroup(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success with patterns deleted", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Patterns = []CategoryRulePattern{
			{Pattern: "AMAZON", IsRegex: false},
			{Pattern: "NETFLIX", IsRegex: false},
		}
		id, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = store.DeleteCategoryRuleGroup(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = store.GetCategoryRuleGroup(ctx, id)
		if !errors.Is(err, ErrCategoryRuleGroupNotFound) {
			t.Fatalf("expected ErrCategoryRuleGroupNotFound after delete, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := store.DeleteCategoryRuleGroup(ctx, 99999)
		if !errors.Is(err, ErrCategoryRuleGroupNotFound) {
			t.Fatalf("expected ErrCategoryRuleGroupNotFound, got %v", err)
		}
	})
}

func TestCreateCategoryRulePattern(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		g := validCategoryRuleGroup()
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		p := CategoryRulePattern{Pattern: "AMAZON", IsRegex: false}
		pid, err := store.CreateCategoryRulePattern(ctx, gid, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pid == 0 {
			t.Fatal("expected non-zero pattern id")
		}

		got, err := store.GetCategoryRuleGroup(ctx, gid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Patterns) != 1 {
			t.Fatalf("expected 1 pattern, got %d", len(got.Patterns))
		}
		if got.Patterns[0].Pattern != "AMAZON" {
			t.Errorf("expected pattern %q, got %q", "AMAZON", got.Patterns[0].Pattern)
		}
	})

	t.Run("success regex", func(t *testing.T) {
		g := validCategoryRuleGroup()
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		p := CategoryRulePattern{Pattern: `^AMAZON\s+\d+`, IsRegex: true}
		pid, err := store.CreateCategoryRulePattern(ctx, gid, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pid == 0 {
			t.Fatal("expected non-zero pattern id")
		}
	})

	t.Run("invalid regex", func(t *testing.T) {
		g := validCategoryRuleGroup()
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		p := CategoryRulePattern{Pattern: `[invalid`, IsRegex: true}
		_, err = store.CreateCategoryRulePattern(ctx, gid, p)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("empty pattern", func(t *testing.T) {
		g := validCategoryRuleGroup()
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		p := CategoryRulePattern{Pattern: "", IsRegex: false}
		_, err = store.CreateCategoryRulePattern(ctx, gid, p)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("group not found", func(t *testing.T) {
		p := CategoryRulePattern{Pattern: "TEST", IsRegex: false}
		_, err := store.CreateCategoryRulePattern(ctx, 99999, p)
		if !errors.Is(err, ErrCategoryRuleGroupNotFound) {
			t.Fatalf("expected ErrCategoryRuleGroupNotFound, got %v", err)
		}
	})
}

func TestUpdateCategoryRulePattern(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Patterns = []CategoryRulePattern{{Pattern: "OLD", IsRegex: false}}
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRuleGroup(ctx, gid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pid := got.Patterns[0].ID

		err = store.UpdateCategoryRulePattern(ctx, gid, pid, CategoryRulePattern{Pattern: "NEW", IsRegex: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err = store.GetCategoryRuleGroup(ctx, gid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Patterns[0].Pattern != "NEW" {
			t.Errorf("expected pattern %q, got %q", "NEW", got.Patterns[0].Pattern)
		}
	})

	t.Run("not found", func(t *testing.T) {
		g := validCategoryRuleGroup()
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = store.UpdateCategoryRulePattern(ctx, gid, 99999, CategoryRulePattern{Pattern: "X", IsRegex: false})
		if !errors.Is(err, ErrCategoryRulePatternNotFound) {
			t.Fatalf("expected ErrCategoryRulePatternNotFound, got %v", err)
		}
	})

	t.Run("wrong group", func(t *testing.T) {
		g1 := validCategoryRuleGroup()
		g1.Patterns = []CategoryRulePattern{{Pattern: "PAT", IsRegex: false}}
		gid1, err := store.CreateCategoryRuleGroup(ctx, g1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRuleGroup(ctx, gid1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pid := got.Patterns[0].ID

		g2 := validCategoryRuleGroup()
		gid2, err := store.CreateCategoryRuleGroup(ctx, g2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = store.UpdateCategoryRulePattern(ctx, gid2, pid, CategoryRulePattern{Pattern: "X", IsRegex: false})
		if !errors.Is(err, ErrCategoryRulePatternNotFound) {
			t.Fatalf("expected ErrCategoryRulePatternNotFound, got %v", err)
		}
	})

	t.Run("validation error empty pattern", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Patterns = []CategoryRulePattern{{Pattern: "PAT", IsRegex: false}}
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRuleGroup(ctx, gid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pid := got.Patterns[0].ID

		err = store.UpdateCategoryRulePattern(ctx, gid, pid, CategoryRulePattern{Pattern: "", IsRegex: false})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("validation error invalid regex", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Patterns = []CategoryRulePattern{{Pattern: "PAT", IsRegex: false}}
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRuleGroup(ctx, gid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pid := got.Patterns[0].ID

		err = store.UpdateCategoryRulePattern(ctx, gid, pid, CategoryRulePattern{Pattern: `[invalid`, IsRegex: true})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})
}

func TestDeleteCategoryRulePattern(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		g := validCategoryRuleGroup()
		g.Patterns = []CategoryRulePattern{{Pattern: "DEL_ME", IsRegex: false}}
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRuleGroup(ctx, gid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pid := got.Patterns[0].ID

		err = store.DeleteCategoryRulePattern(ctx, gid, pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err = store.GetCategoryRuleGroup(ctx, gid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Patterns) != 0 {
			t.Errorf("expected 0 patterns after delete, got %d", len(got.Patterns))
		}
	})

	t.Run("not found", func(t *testing.T) {
		g := validCategoryRuleGroup()
		gid, err := store.CreateCategoryRuleGroup(ctx, g)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = store.DeleteCategoryRulePattern(ctx, gid, 99999)
		if !errors.Is(err, ErrCategoryRulePatternNotFound) {
			t.Fatalf("expected ErrCategoryRulePatternNotFound, got %v", err)
		}
	})

	t.Run("wrong group", func(t *testing.T) {
		g1 := validCategoryRuleGroup()
		g1.Patterns = []CategoryRulePattern{{Pattern: "PAT", IsRegex: false}}
		gid1, err := store.CreateCategoryRuleGroup(ctx, g1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRuleGroup(ctx, gid1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pid := got.Patterns[0].ID

		g2 := validCategoryRuleGroup()
		gid2, err := store.CreateCategoryRuleGroup(ctx, g2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = store.DeleteCategoryRulePattern(ctx, gid2, pid)
		if !errors.Is(err, ErrCategoryRulePatternNotFound) {
			t.Fatalf("expected ErrCategoryRulePatternNotFound, got %v", err)
		}
	})
}
