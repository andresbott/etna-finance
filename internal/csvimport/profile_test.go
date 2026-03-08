package csvimport

import (
	"context"
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

func validProfile() ImportProfile {
	return ImportProfile{
		Name:              "My Bank",
		CsvSeparator:      ";",
		SkipRows:          1,
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountColumn:      "Amount",
	}
}

func TestNewStore_NilDB(t *testing.T) {
	_, err := NewStore(nil)
	if err == nil {
		t.Fatal("expected error for nil db, got nil")
	}
}

func TestCreateProfile(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		p := validProfile()
		id, err := store.CreateProfile(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == 0 {
			t.Fatal("expected non-zero id")
		}
	})

	t.Run("default csv separator", func(t *testing.T) {
		p := validProfile()
		p.CsvSeparator = ""
		id, err := store.CreateProfile(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := store.GetProfile(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.CsvSeparator != "," {
			t.Errorf("expected default separator ',', got %q", got.CsvSeparator)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		tests := []struct {
			name    string
			profile ImportProfile
			wantErr string
		}{
			{
				name:    "empty name",
				profile: func() ImportProfile { p := validProfile(); p.Name = ""; return p }(),
				wantErr: "name cannot be empty",
			},
			{
				name:    "empty date_column",
				profile: func() ImportProfile { p := validProfile(); p.DateColumn = ""; return p }(),
				wantErr: "date_column cannot be empty",
			},
			{
				name:    "empty date_format",
				profile: func() ImportProfile { p := validProfile(); p.DateFormat = ""; return p }(),
				wantErr: "date_format cannot be empty",
			},
			{
				name:    "empty description_column",
				profile: func() ImportProfile { p := validProfile(); p.DescriptionColumn = ""; return p }(),
				wantErr: "description_column cannot be empty",
			},
			{
				name:    "empty amount_column",
				profile: func() ImportProfile { p := validProfile(); p.AmountColumn = ""; return p }(),
				wantErr: "amount_column cannot be empty",
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, err := store.CreateProfile(ctx, tc.profile)
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

func TestGetProfile(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("existing profile", func(t *testing.T) {
		p := validProfile()
		id, err := store.CreateProfile(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := store.GetProfile(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != id {
			t.Errorf("expected id %d, got %d", id, got.ID)
		}
		if got.Name != p.Name {
			t.Errorf("expected name %q, got %q", p.Name, got.Name)
		}
		if got.CsvSeparator != p.CsvSeparator {
			t.Errorf("expected separator %q, got %q", p.CsvSeparator, got.CsvSeparator)
		}
		if got.SkipRows != p.SkipRows {
			t.Errorf("expected skip_rows %d, got %d", p.SkipRows, got.SkipRows)
		}
		if got.DateColumn != p.DateColumn {
			t.Errorf("expected date_column %q, got %q", p.DateColumn, got.DateColumn)
		}
		if got.DateFormat != p.DateFormat {
			t.Errorf("expected date_format %q, got %q", p.DateFormat, got.DateFormat)
		}
		if got.DescriptionColumn != p.DescriptionColumn {
			t.Errorf("expected description_column %q, got %q", p.DescriptionColumn, got.DescriptionColumn)
		}
		if got.AmountColumn != p.AmountColumn {
			t.Errorf("expected amount_column %q, got %q", p.AmountColumn, got.AmountColumn)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.GetProfile(ctx, 99999)
		if !errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("expected ErrProfileNotFound, got %v", err)
		}
	})
}

func TestListProfiles(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("empty list", func(t *testing.T) {
		profiles, err := store.ListProfiles(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 0 {
			t.Errorf("expected 0 profiles, got %d", len(profiles))
		}
	})

	t.Run("multiple profiles ordered by id", func(t *testing.T) {
		p1 := validProfile()
		p1.Name = "Profile B"
		id1, err := store.CreateProfile(ctx, p1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		p2 := validProfile()
		p2.Name = "Profile A"
		id2, err := store.CreateProfile(ctx, p2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		profiles, err := store.ListProfiles(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 2 {
			t.Fatalf("expected 2 profiles, got %d", len(profiles))
		}
		if profiles[0].ID != id1 {
			t.Errorf("expected first profile id %d, got %d", id1, profiles[0].ID)
		}
		if profiles[1].ID != id2 {
			t.Errorf("expected second profile id %d, got %d", id2, profiles[1].ID)
		}
	})
}

func TestUpdateProfile(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		p := validProfile()
		id, err := store.CreateProfile(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated := validProfile()
		updated.Name = "Updated Name"
		updated.CsvSeparator = "\t"
		updated.SkipRows = 3
		updated.DateColumn = "Fecha"
		updated.DateFormat = "2006-01-02"
		updated.DescriptionColumn = "Concepto"
		updated.AmountColumn = "Importe"

		err = store.UpdateProfile(ctx, id, updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetProfile(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "Updated Name" {
			t.Errorf("expected name %q, got %q", "Updated Name", got.Name)
		}
		if got.CsvSeparator != "\t" {
			t.Errorf("expected separator %q, got %q", "\t", got.CsvSeparator)
		}
		if got.SkipRows != 3 {
			t.Errorf("expected skip_rows 3, got %d", got.SkipRows)
		}
		if got.DateColumn != "Fecha" {
			t.Errorf("expected date_column %q, got %q", "Fecha", got.DateColumn)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := store.UpdateProfile(ctx, 99999, validProfile())
		if !errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("expected ErrProfileNotFound, got %v", err)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		p := validProfile()
		id, err := store.CreateProfile(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		empty := ImportProfile{}
		err = store.UpdateProfile(ctx, id, empty)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		var valErr ErrValidation
		if !errors.As(err, &valErr) {
			t.Fatalf("expected ErrValidation, got %T: %v", err, err)
		}
	})

	t.Run("update skip_rows to zero", func(t *testing.T) {
		p := validProfile()
		p.SkipRows = 5
		id, err := store.CreateProfile(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated := validProfile()
		updated.SkipRows = 0
		err = store.UpdateProfile(ctx, id, updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetProfile(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.SkipRows != 0 {
			t.Errorf("expected skip_rows 0, got %d", got.SkipRows)
		}
	})

	t.Run("default csv separator on update", func(t *testing.T) {
		p := validProfile()
		id, err := store.CreateProfile(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated := validProfile()
		updated.CsvSeparator = ""
		err = store.UpdateProfile(ctx, id, updated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := store.GetProfile(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.CsvSeparator != "," {
			t.Errorf("expected default separator ',', got %q", got.CsvSeparator)
		}
	})
}

func TestCreateProfile_SplitMode(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	p := ImportProfile{
		Name:              "Split Bank",
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "2006-01-02",
		DescriptionColumn: "Desc",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}
	id, err := store.CreateProfile(ctx, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := store.GetProfile(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AmountMode != "split" {
		t.Errorf("expected AmountMode=split, got %s", got.AmountMode)
	}
	if got.CreditColumn != "Credit" {
		t.Errorf("expected CreditColumn=Credit, got %s", got.CreditColumn)
	}
	if got.DebitColumn != "Debit" {
		t.Errorf("expected DebitColumn=Debit, got %s", got.DebitColumn)
	}
}

func TestCreateProfile_SplitMode_MissingColumns(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	p := ImportProfile{
		Name:              "Split Bank",
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "2006-01-02",
		DescriptionColumn: "Desc",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		// DebitColumn missing
	}
	_, err := store.CreateProfile(ctx, p)
	if err == nil {
		t.Fatal("expected validation error for missing debit column")
	}
}

func TestCreateProfile_SingleMode_BackwardCompat(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	p := validProfile()
	p.AmountMode = ""
	id, err := store.CreateProfile(ctx, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := store.GetProfile(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AmountMode != "single" {
		t.Errorf("expected AmountMode=single, got %s", got.AmountMode)
	}
}

func TestDeleteProfile(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		p := validProfile()
		id, err := store.CreateProfile(ctx, p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = store.DeleteProfile(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = store.GetProfile(ctx, id)
		if !errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("expected ErrProfileNotFound after delete, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := store.DeleteProfile(ctx, 99999)
		if !errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("expected ErrProfileNotFound, got %v", err)
		}
	})
}
