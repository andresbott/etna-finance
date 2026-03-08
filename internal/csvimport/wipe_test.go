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

	// Create a category rule
	r := validCategoryRule()
	_, err = store.CreateCategoryRule(ctx, r)
	if err != nil {
		t.Fatalf("unexpected error creating category rule: %v", err)
	}

	// Verify data exists
	profiles, err := store.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing profiles: %v", err)
	}
	if len(profiles) == 0 {
		t.Fatal("expected at least one profile before wipe")
	}

	rules, err := store.ListCategoryRules(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing category rules: %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("expected at least one category rule before wipe")
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

	rules, err = store.ListCategoryRules(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing category rules after wipe: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 category rules after wipe, got %d", len(rules))
	}
}
