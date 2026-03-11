package csvimport

import (
	"context"
	"testing"
)

func TestWipeData(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Create a profile
	p := validProfile()
	_, err := store.CreateProfile(ctx, p)
	if err != nil {
		t.Fatalf("unexpected error creating profile: %v", err)
	}

	// Create a category rule group with a pattern
	g := validCategoryRuleGroup()
	g.Patterns = []CategoryRulePattern{{Pattern: "TEST", IsRegex: false}}
	_, err = store.CreateCategoryRuleGroup(ctx, g)
	if err != nil {
		t.Fatalf("unexpected error creating category rule group: %v", err)
	}

	// Verify data exists
	profiles, err := store.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing profiles: %v", err)
	}
	if len(profiles) == 0 {
		t.Fatal("expected at least one profile before wipe")
	}

	groups, err := store.ListCategoryRuleGroups(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing category rule groups: %v", err)
	}
	if len(groups) == 0 {
		t.Fatal("expected at least one category rule group before wipe")
	}

	// Wipe data
	err = store.WipeData(ctx)
	if err != nil {
		t.Fatalf("unexpected error wiping data: %v", err)
	}

	// Assert both lists are empty
	profiles, err = store.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing profiles after wipe: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected 0 profiles after wipe, got %d", len(profiles))
	}

	groups, err = store.ListCategoryRuleGroups(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing category rule groups after wipe: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected 0 category rule groups after wipe, got %d", len(groups))
	}
}
