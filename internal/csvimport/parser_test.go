package csvimport

import (
	"math"
	"strings"
	"testing"
)

func defaultProfile() ImportProfile {
	return ImportProfile{
		CsvSeparator:      ",",
		SkipRows:          0,
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountColumn:      "Amount",
	}
}

func TestParse_ValidCSV(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Salary,1500.00
02/03/2026,Grocery Store,-45.30
`
	profile := defaultProfile()

	rows, err := Parse(strings.NewReader(csv), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	r := rows[0]
	if r.RowNumber != 2 {
		t.Errorf("row 0: expected RowNumber=2, got %d", r.RowNumber)
	}
	if r.Date != "2026-03-01" {
		t.Errorf("row 0: expected Date=2026-03-01, got %s", r.Date)
	}
	if r.Description != "Salary" {
		t.Errorf("row 0: expected Description=Salary, got %s", r.Description)
	}
	if r.Amount != 1500.00 {
		t.Errorf("row 0: expected Amount=1500.00, got %f", r.Amount)
	}
	if r.Type != "income" {
		t.Errorf("row 0: expected Type=income, got %s", r.Type)
	}
	if r.Error != "" {
		t.Errorf("row 0: unexpected error: %s", r.Error)
	}

	r = rows[1]
	if r.Date != "2026-03-02" {
		t.Errorf("row 1: expected Date=2026-03-02, got %s", r.Date)
	}
	if r.Amount != -45.30 {
		t.Errorf("row 1: expected Amount=-45.30, got %f", r.Amount)
	}
	if r.Type != "expense" {
		t.Errorf("row 1: expected Type=expense, got %s", r.Type)
	}
}

func TestParse_SemicolonSeparator(t *testing.T) {
	csv := `Date;Description;Amount
01/03/2026;Coffee Shop;-3.50
`
	profile := defaultProfile()
	profile.CsvSeparator = ";"

	rows, err := Parse(strings.NewReader(csv), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Description != "Coffee Shop" {
		t.Errorf("expected Description='Coffee Shop', got %q", rows[0].Description)
	}
	if rows[0].Amount != -3.50 {
		t.Errorf("expected Amount=-3.50, got %f", rows[0].Amount)
	}
}

func TestParse_AmountSignType(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Income,100.00
01/03/2026,Zero,0.00
01/03/2026,Expense,-50.00
`
	profile := defaultProfile()

	rows, err := Parse(strings.NewReader(csv), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].Type != "income" {
		t.Errorf("positive amount: expected type=income, got %s", rows[0].Type)
	}
	if rows[1].Type != "income" {
		t.Errorf("zero amount: expected type=income, got %s", rows[1].Type)
	}
	if rows[2].Type != "expense" {
		t.Errorf("negative amount: expected type=expense, got %s", rows[2].Type)
	}
}

func TestParse_CategoryMatchSubstring(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,GROCERY store purchase,-25.00
`
	profile := defaultProfile()
	rules := []CategoryRule{
		{Pattern: "grocery", IsRegex: false, CategoryID: 10, Position: 1},
	}

	rows, err := Parse(strings.NewReader(csv), profile, rules, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].CategoryID != 10 {
		t.Errorf("expected CategoryID=10, got %d", rows[0].CategoryID)
	}
}

func TestParse_CategoryMatchRegex(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Payment REF-12345,-100.00
`
	profile := defaultProfile()
	rules := []CategoryRule{
		{Pattern: `REF-\d+`, IsRegex: true, CategoryID: 20, Position: 1},
	}

	rows, err := Parse(strings.NewReader(csv), profile, rules, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].CategoryID != 20 {
		t.Errorf("expected CategoryID=20, got %d", rows[0].CategoryID)
	}
}

func TestParse_CategoryFirstMatchWins(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Grocery Store payment,-30.00
`
	profile := defaultProfile()
	rules := []CategoryRule{
		{Pattern: "grocery", IsRegex: false, CategoryID: 10, Position: 1},
		{Pattern: "store", IsRegex: false, CategoryID: 20, Position: 2},
	}

	rows, err := Parse(strings.NewReader(csv), profile, rules, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].CategoryID != 10 {
		t.Errorf("expected first match CategoryID=10, got %d", rows[0].CategoryID)
	}
}

func TestParse_CategoryNoMatch(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Random purchase,-15.00
`
	profile := defaultProfile()
	rules := []CategoryRule{
		{Pattern: "grocery", IsRegex: false, CategoryID: 10, Position: 1},
	}

	rows, err := Parse(strings.NewReader(csv), profile, rules, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].CategoryID != 0 {
		t.Errorf("expected CategoryID=0, got %d", rows[0].CategoryID)
	}
}

func TestParse_DuplicateDetection(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Salary,1500.00
02/03/2026,Grocery,-45.30
`
	profile := defaultProfile()
	existing := []ExistingTx{
		{Date: "2026-03-01", Amount: 1500.00},
	}

	rows, err := Parse(strings.NewReader(csv), profile, nil, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rows[0].IsDuplicate {
		t.Error("row 0: expected IsDuplicate=true")
	}
	if rows[1].IsDuplicate {
		t.Error("row 1: expected IsDuplicate=false")
	}
}

func TestParse_ErrorRows(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Good Row,100.00
bad-date,Bad Date Row,50.00
01/03/2026,Bad Amount,notanumber
`
	profile := defaultProfile()

	rows, err := Parse(strings.NewReader(csv), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// First row should be fine
	if rows[0].Error != "" {
		t.Errorf("row 0: unexpected error: %s", rows[0].Error)
	}
	if rows[0].Amount != 100.00 {
		t.Errorf("row 0: expected Amount=100.00, got %f", rows[0].Amount)
	}

	// Second row: bad date
	if rows[1].Error == "" {
		t.Error("row 1: expected error for bad date")
	}

	// Third row: bad amount
	if rows[2].Error == "" {
		t.Error("row 2: expected error for bad amount")
	}
}

func TestParse_SkipRows(t *testing.T) {
	csv := `This is a preamble line
Another preamble line
Date,Description,Amount
01/03/2026,Test,50.00
`
	profile := defaultProfile()
	profile.SkipRows = 2

	rows, err := Parse(strings.NewReader(csv), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Description != "Test" {
		t.Errorf("expected Description='Test', got %q", rows[0].Description)
	}
	if rows[0].Amount != 50.00 {
		t.Errorf("expected Amount=50.00, got %f", rows[0].Amount)
	}
	// Row number should account for skipped rows + header
	if rows[0].RowNumber != 4 {
		t.Errorf("expected RowNumber=4, got %d", rows[0].RowNumber)
	}
}

func TestParse_MissingColumn(t *testing.T) {
	csv := `Date,Description,Total
01/03/2026,Test,50.00
`
	profile := defaultProfile() // expects "Amount" column

	_, err := Parse(strings.NewReader(csv), profile, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing column, got nil")
	}
	if !strings.Contains(err.Error(), "Amount") {
		t.Errorf("error should mention missing column 'Amount', got: %s", err.Error())
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{name: "standard", input: "1234.56", want: 1234.56},
		{name: "comma decimal", input: "1234,56", want: 1234.56},
		{name: "european mixed", input: "1.234,56", want: 1234.56},
		{name: "us mixed", input: "1,234.56", want: 1234.56},
		{name: "negative", input: "-45.60", want: -45.60},
		{name: "positive integer", input: "100", want: 100.00},
		{name: "negative comma decimal", input: "-1234,56", want: -1234.56},
		{name: "empty", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "large european", input: "1.234.567,89", want: 1234567.89},
		{name: "large us", input: "1,234,567.89", want: 1234567.89},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAmount(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (value=%f)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("expected %f, got %f", tt.want, got)
			}
		})
	}
}
