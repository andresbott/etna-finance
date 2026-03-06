package csvimport

import (
	"context"
	"errors"
	"testing"
)

func validCategoryRule() CategoryRule {
	return CategoryRule{
		Pattern:    "AMAZON",
		IsRegex:    false,
		CategoryID: 1,
		Position:   10,
	}
}

func TestCreateCategoryRule(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		r := validCategoryRule()
		id, err := store.CreateCategoryRule(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero id")
		}
	})

	t.Run("success with valid regex", func(t *testing.T) {
		r := validCategoryRule()
		r.IsRegex = true
		r.Pattern = `^AMAZON\s+\d+`
		id, err := store.CreateCategoryRule(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero id")
		}
	})

	t.Run("invalid regex", func(t *testing.T) {
		r := validCategoryRule()
		r.IsRegex = true
		r.Pattern = `[invalid`
		_, err := store.CreateCategoryRule(ctx, r)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		tests := []struct {
			name    string
			rule    CategoryRule
			wantErr string
		}{
			{
				name:    "empty pattern",
				rule:    func() CategoryRule { r := validCategoryRule(); r.Pattern = ""; return r }(),
				wantErr: "pattern cannot be empty",
			},
			{
				name:    "zero category_id",
				rule:    func() CategoryRule { r := validCategoryRule(); r.CategoryID = 0; return r }(),
				wantErr: "category_id cannot be zero",
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, err := store.CreateCategoryRule(ctx, tc.rule)
				if err == nil {
					t.Fatal("expected validation error, got nil")
				}
				var valErr ErrValidation
				if !errors.As(err, &valErr) {
					t.Fatalf("expected ErrValidation, got %T: %v", err, err)
				}
				if err.Error() != tc.wantErr {
					t.Errorf("expected error %q, got %q", tc.wantErr, err.Error())
				}
			})
		}
	})
}

func TestGetCategoryRule(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("existing rule", func(t *testing.T) {
		r := validCategoryRule()
		id, err := store.CreateCategoryRule(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := store.GetCategoryRule(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != id {
			t.Errorf("expected id %d, got %d", id, got.ID)
		}
		if got.Pattern != r.Pattern {
			t.Errorf("expected pattern %q, got %q", r.Pattern, got.Pattern)
		}
		if got.IsRegex != r.IsRegex {
			t.Errorf("expected is_regex %v, got %v", r.IsRegex, got.IsRegex)
		}
		if got.CategoryID != r.CategoryID {
			t.Errorf("expected category_id %d, got %d", r.CategoryID, got.CategoryID)
		}
		if got.Position != r.Position {
			t.Errorf("expected position %d, got %d", r.Position, got.Position)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.GetCategoryRule(ctx, 99999)
		if !errors.Is(err, ErrCategoryRuleNotFound) {
			t.Fatalf("expected ErrCategoryRuleNotFound, got %v", err)
		}
	})
}

func TestListCategoryRules(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("empty list", func(t *testing.T) {
		rules, err := store.ListCategoryRules(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 0 {
			t.Errorf("expected 0 rules, got %d", len(rules))
		}
	})

	t.Run("ordered by position then id", func(t *testing.T) {
		r1 := validCategoryRule()
		r1.Pattern = "RULE_C"
		r1.Position = 20
		id1, err := store.CreateCategoryRule(ctx, r1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		r2 := validCategoryRule()
		r2.Pattern = "RULE_A"
		r2.Position = 5
		id2, err := store.CreateCategoryRule(ctx, r2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		r3 := validCategoryRule()
		r3.Pattern = "RULE_B"
		r3.Position = 5
		id3, err := store.CreateCategoryRule(ctx, r3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		rules, err := store.ListCategoryRules(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rules) != 3 {
			t.Fatalf("expected 3 rules, got %d", len(rules))
		}
		// position 5 (id2) first, then position 5 (id3), then position 20 (id1)
		if rules[0].ID != id2 {
			t.Errorf("expected first rule id %d, got %d", id2, rules[0].ID)
		}
		if rules[1].ID != id3 {
			t.Errorf("expected second rule id %d, got %d", id3, rules[1].ID)
		}
		if rules[2].ID != id1 {
			t.Errorf("expected third rule id %d, got %d", id1, rules[2].ID)
		}
	})
}

func TestUpdateCategoryRule(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		r := validCategoryRule()
		id, err := store.CreateCategoryRule(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated := validCategoryRule()
		updated.Pattern = "NETFLIX"
		updated.CategoryID = 2
		updated.Position = 99

		err = store.UpdateCategoryRule(ctx, id, updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRule(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Pattern != "NETFLIX" {
			t.Errorf("expected pattern %q, got %q", "NETFLIX", got.Pattern)
		}
		if got.CategoryID != 2 {
			t.Errorf("expected category_id 2, got %d", got.CategoryID)
		}
		if got.Position != 99 {
			t.Errorf("expected position 99, got %d", got.Position)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := store.UpdateCategoryRule(ctx, 99999, validCategoryRule())
		if !errors.Is(err, ErrCategoryRuleNotFound) {
			t.Fatalf("expected ErrCategoryRuleNotFound, got %v", err)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		r := validCategoryRule()
		id, err := store.CreateCategoryRule(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Run("empty pattern", func(t *testing.T) {
			u := validCategoryRule()
			u.Pattern = ""
			err := store.UpdateCategoryRule(ctx, id, u)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			var valErr ErrValidation
			if !errors.As(err, &valErr) {
				t.Fatalf("expected ErrValidation, got %T: %v", err, err)
			}
		})

		t.Run("zero category_id", func(t *testing.T) {
			u := validCategoryRule()
			u.CategoryID = 0
			err := store.UpdateCategoryRule(ctx, id, u)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			var valErr ErrValidation
			if !errors.As(err, &valErr) {
				t.Fatalf("expected ErrValidation, got %T: %v", err, err)
			}
		})
	})

	t.Run("invalid regex on update", func(t *testing.T) {
		r := validCategoryRule()
		id, err := store.CreateCategoryRule(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u := validCategoryRule()
		u.IsRegex = true
		u.Pattern = `[invalid`
		err = store.UpdateCategoryRule(ctx, id, u)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("update IsRegex back to false", func(t *testing.T) {
		r := validCategoryRule()
		r.IsRegex = true
		r.Pattern = `^AMAZON\d+`
		id, err := store.CreateCategoryRule(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u := validCategoryRule()
		u.Pattern = "AMAZON"
		u.IsRegex = false
		err = store.UpdateCategoryRule(ctx, id, u)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRule(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.IsRegex != false {
			t.Errorf("expected is_regex false, got %v", got.IsRegex)
		}
	})

	t.Run("update Position to zero", func(t *testing.T) {
		r := validCategoryRule()
		r.Position = 50
		id, err := store.CreateCategoryRule(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u := validCategoryRule()
		u.Position = 0
		err = store.UpdateCategoryRule(ctx, id, u)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetCategoryRule(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Position != 0 {
			t.Errorf("expected position 0, got %d", got.Position)
		}
	})
}

func TestDeleteCategoryRule(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		r := validCategoryRule()
		id, err := store.CreateCategoryRule(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = store.DeleteCategoryRule(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = store.GetCategoryRule(ctx, id)
		if !errors.Is(err, ErrCategoryRuleNotFound) {
			t.Fatalf("expected ErrCategoryRuleNotFound after delete, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := store.DeleteCategoryRule(ctx, 99999)
		if !errors.Is(err, ErrCategoryRuleNotFound) {
			t.Fatalf("expected ErrCategoryRuleNotFound, got %v", err)
		}
	})
}
