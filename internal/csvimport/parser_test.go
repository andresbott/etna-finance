package csvimport

import (
	"fmt"
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

func TestParsePreview_ReturnsHeadersAndRows(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Salary,1500.00
02/03/2026,Grocery,-45.30
`
	profile := defaultProfile()
	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
	if result.Headers[0] != "Date" || result.Headers[1] != "Description" || result.Headers[2] != "Amount" {
		t.Errorf("unexpected headers: %v", result.Headers)
	}
	if len(result.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(result.Rows))
	}
	if result.TotalRows != 2 {
		t.Errorf("expected TotalRows=2, got %d", result.TotalRows)
	}
}

func TestParsePreview_CapsAt10Rows(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("Date,Description,Amount\n")
	for i := 0; i < 20; i++ {
		sb.WriteString(fmt.Sprintf("01/03/2026,Item %d,%.2f\n", i, float64(i+1)))
	}
	profile := defaultProfile()
	result, err := ParsePreview(strings.NewReader(sb.String()), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 10 {
		t.Errorf("expected 10 preview rows, got %d", len(result.Rows))
	}
	if result.TotalRows != 20 {
		t.Errorf("expected TotalRows=20, got %d", result.TotalRows)
	}
}

func TestParsePreview_HeadersOnlyWhenNoMapping(t *testing.T) {
	csv := `Date,Description,Amount
01/03/2026,Salary,1500.00
`
	profile := ImportProfile{
		CsvSeparator: ",",
		SkipRows:     0,
	}
	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
	if len(result.Rows) != 0 {
		t.Errorf("expected 0 rows when no mapping, got %d", len(result.Rows))
	}
	if result.TotalRows != 1 {
		t.Errorf("expected TotalRows=1, got %d", result.TotalRows)
	}
}

func TestParsePreview_SplitColumns(t *testing.T) {
	csv := `Date,Description,Credit,Debit
01/03/2026,Salary,1500.00,
02/03/2026,Grocery,,45.30
`
	profile := ImportProfile{
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}
	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result.Rows))
	}
	if result.Rows[0].Type != "income" {
		t.Errorf("row 0: expected type=income, got %s", result.Rows[0].Type)
	}
	if result.Rows[0].Amount != 1500.00 {
		t.Errorf("row 0: expected amount=1500.00, got %f", result.Rows[0].Amount)
	}
	if result.Rows[1].Type != "expense" {
		t.Errorf("row 1: expected type=expense, got %s", result.Rows[1].Type)
	}
	if result.Rows[1].Amount != -45.30 {
		t.Errorf("row 1: expected amount=-45.30, got %f", result.Rows[1].Amount)
	}
}

func TestParsePreview_SplitBothPopulated(t *testing.T) {
	csv := `Date,Description,Credit,Debit
01/03/2026,Weird,100.00,50.00
`
	profile := ImportProfile{
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}
	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rows[0].Error == "" {
		t.Error("expected error when both credit and debit are populated")
	}
}

func TestParsePreview_SplitBothEmpty(t *testing.T) {
	csv := `Date,Description,Credit,Debit
01/03/2026,Empty,,
`
	profile := ImportProfile{
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}
	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rows[0].Error == "" {
		t.Error("expected error when both credit and debit are empty")
	}
}

func TestParsePreview_SkipRows(t *testing.T) {
	csv := `Preamble line
Date,Description,Amount
01/03/2026,Test,50.00
`
	profile := defaultProfile()
	profile.SkipRows = 1
	result, err := ParsePreview(strings.NewReader(csv), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
	if result.Rows[0].Amount != 50.00 {
		t.Errorf("expected amount=50.00, got %f", result.Rows[0].Amount)
	}
}

func TestDetectCSVSettings_CommaSeparated(t *testing.T) {
	data := []byte("Date,Description,Amount\n01/03/2026,Salary,1500.00\n02/03/2026,Grocery,-45.30\n")
	sep, skip := DetectCSVSettings(data)
	if sep != "," {
		t.Errorf("expected separator=',', got %q", sep)
	}
	if skip != 0 {
		t.Errorf("expected skipRows=0, got %d", skip)
	}
}

func TestDetectCSVSettings_SemicolonSeparated(t *testing.T) {
	data := []byte("Date;Description;Amount\n01/03/2026;Salary;1500.00\n02/03/2026;Grocery;-45.30\n")
	sep, _ := DetectCSVSettings(data)
	if sep != ";" {
		t.Errorf("expected separator=';', got %q", sep)
	}
}

func TestDetectCSVSettings_TabSeparated(t *testing.T) {
	data := []byte("Date\tDescription\tAmount\n01/03/2026\tSalary\t1500.00\n02/03/2026\tGrocery\t-45.30\n")
	sep, _ := DetectCSVSettings(data)
	if sep != "\t" {
		t.Errorf("expected separator='\\t', got %q", sep)
	}
}

func TestDetectCSVSettings_WithPreamble(t *testing.T) {
	data := []byte("Account: 12345\nPeriod: March 2026\nDate,Description,Amount\n01/03/2026,Salary,1500.00\n02/03/2026,Grocery,-45.30\n")
	sep, skip := DetectCSVSettings(data)
	if sep != "," {
		t.Errorf("expected separator=',', got %q", sep)
	}
	if skip != 2 {
		t.Errorf("expected skipRows=2, got %d", skip)
	}
}

func TestDetectCSVSettings_SemicolonWithPreamble(t *testing.T) {
	data := []byte("Bank Export\nDate;Description;Amount\n01/03/2026;Salary;1500.00\n")
	sep, skip := DetectCSVSettings(data)
	if sep != ";" {
		t.Errorf("expected separator=';', got %q", sep)
	}
	if skip != 1 {
		t.Errorf("expected skipRows=1, got %d", skip)
	}
}

func TestParsePreviewWithAutoDetect_AutoDetects(t *testing.T) {
	data := []byte("Date,Description,Amount\n01/03/2026,Salary,1500.00\n02/03/2026,Grocery,-45.30\n")
	profile := ImportProfile{} // empty separator triggers auto-detect
	result, err := ParsePreviewWithAutoDetect(data, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DetectedSeparator != "," {
		t.Errorf("expected DetectedSeparator=',', got %q", result.DetectedSeparator)
	}
	if result.DetectedSkipRows != 0 {
		t.Errorf("expected DetectedSkipRows=0, got %d", result.DetectedSkipRows)
	}
	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
}

func TestParsePreviewWithAutoDetect_RespectsExplicitSeparator(t *testing.T) {
	data := []byte("Date;Description;Amount\n01/03/2026;Salary;1500.00\n")
	profile := ImportProfile{CsvSeparator: ";"} // explicit separator
	result, err := ParsePreviewWithAutoDetect(data, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should NOT set DetectedSeparator when explicit
	if result.DetectedSeparator != "" {
		t.Errorf("expected empty DetectedSeparator, got %q", result.DetectedSeparator)
	}
	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
}

func TestDetectDateFormat_ISO(t *testing.T) {
	samples := []string{"2026-03-05", "2026-02-20", "2026-01-15"}
	got := DetectDateFormat(samples)
	if got != "2006-01-02" {
		t.Errorf("expected 2006-01-02, got %q", got)
	}
}

func TestDetectDateFormat_EuropeanDots(t *testing.T) {
	samples := []string{"05.03.2026", "20.02.2026", "15.01.2026"}
	got := DetectDateFormat(samples)
	if got != "02.01.2006" {
		t.Errorf("expected 02.01.2006, got %q", got)
	}
}

func TestDetectDateFormat_EuropeanSlashes(t *testing.T) {
	// Day > 12 disambiguates from MM/DD/YYYY
	samples := []string{"05/03/2026", "20/02/2026", "15/01/2026"}
	got := DetectDateFormat(samples)
	if got != "02/01/2006" {
		t.Errorf("expected 02/01/2006, got %q", got)
	}
}

func TestDetectDateFormat_USSlashes(t *testing.T) {
	// Month position > 12 disambiguates from DD/MM/YYYY
	samples := []string{"03/25/2026", "02/14/2026", "01/31/2026"}
	got := DetectDateFormat(samples)
	if got != "01/02/2006" {
		t.Errorf("expected 01/02/2006, got %q", got)
	}
}

func TestDetectDateFormat_Ambiguous(t *testing.T) {
	// All values have both components <= 12, can't distinguish DD/MM from MM/DD.
	// Should return the first matching candidate (DD/MM/YYYY preferred over MM/DD/YYYY).
	samples := []string{"01/02/2026", "03/04/2026", "05/06/2026"}
	got := DetectDateFormat(samples)
	// Both formats match; DD/MM comes first in the candidate list
	if got != "02/01/2006" {
		t.Errorf("expected 02/01/2006 (DD/MM preferred when ambiguous), got %q", got)
	}
}

func TestDetectDateFormat_Empty(t *testing.T) {
	got := DetectDateFormat([]string{})
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestDetectDateFormat_NoMatch(t *testing.T) {
	samples := []string{"not-a-date", "foobar", "123"}
	got := DetectDateFormat(samples)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestParsePreviewWithAutoDetect_DetectsDateFormat(t *testing.T) {
	data := []byte("Date;Description;Amount\n05.03.2026;Coffee;-8\n20.02.2026;Grocery;-52\n")
	profile := ImportProfile{
		CsvSeparator: ";",
		DateColumn:   "Date",
		// DateFormat empty — should be auto-detected
		DescriptionColumn: "Description",
		AmountColumn:      "Amount",
	}
	result, err := ParsePreviewWithAutoDetect(data, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DetectedDateFormat != "02.01.2006" {
		t.Errorf("expected DetectedDateFormat=02.01.2006, got %q", result.DetectedDateFormat)
	}
	// Rows should be parsed correctly with the detected format
	if len(result.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result.Rows))
	}
	if result.Rows[0].Date != "2026-03-05" {
		t.Errorf("expected first row date=2026-03-05, got %s", result.Rows[0].Date)
	}
}

// bankExportCSV is a realistic bank export CSV with:
// - 5 metadata preamble lines + blank line before header (skip 6)
// - Semicolon separator
// - Split credit/debit columns (debit values are already negative)
// - Blank line between header and first data row
// - Multiple entries on same date with same description and amount
// - Footer disclaimer lines
// All data is anonymized/fictional.
const bankExportCSV = `Start date:;="01.01.2026"
End date:;="06.03.2026"
Category:;="All"
Account:;="CH0000000000000000000"
Currency:;="CHF"

Date;Type;Description;Credit in CHF;Debit in CHF;Label;Category

05.03.2026;Booking;"Purchase 05.03.2026, Coffee House";;-8;;Dining
05.03.2026;Booking;"Purchase 05.03.2026, Grocery Store Main St";;-52.30;;Groceries
04.03.2026;Booking;"Online Purchase Electronics Shop";;-199.90;;Electronics
03.03.2026;Booking;"Purchase 03.03.2026, Grocery Store Central";;-3.50;;Groceries
03.03.2026;Booking;"Direct Debit Postal Services";;-18.75;;Shipping
02.03.2026;Booking;"Purchase 02.03.2026, Pizza Place";;-22;;Dining
02.03.2026;Booking;"Mobile Payment Received from +41700000001";850;;;Income
02.03.2026;Booking;"Direct Debit Insurance Co";;-310.55;;Insurance
20.02.2026;Booking;"Purchase 20.02.2026, Pub Downtown";;-7;;Dining
20.02.2026;Booking;"Purchase 20.02.2026, Pub Downtown";;-7;;Dining
20.02.2026;Booking;"Purchase 20.02.2026, Pub Downtown";;-7;;Dining
20.02.2026;Booking;"Purchase 20.02.2026, Pub Downtown";;-15;;Dining
20.02.2026;Booking;"Purchase 20.02.2026, Pub Downtown";;-7;;Dining
20.02.2026;Booking;"Purchase 20.02.2026, Grocery Store Main St";;-41.20;;Groceries
15.02.2026;Booking;"Salary from Employer AG";5500;;;Salary
10.02.2026;Booking;"Refund Online Store";35;;;Refund

Disclaimer:
Content generated by filter settings. Bank is not responsible for completeness.`

// bankExportProfile returns the profile matching bankExportCSV.
func bankExportProfile() ImportProfile {
	return ImportProfile{
		CsvSeparator:      ";",
		SkipRows:          5,
		DateColumn:        "Date",
		DateFormat:        "02.01.2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit in CHF",
		DebitColumn:       "Debit in CHF",
	}
}

func TestDetectCSVSettings_BankExport(t *testing.T) {
	sep, skip := DetectCSVSettings([]byte(bankExportCSV))
	if sep != ";" {
		t.Errorf("expected separator=';', got %q", sep)
	}
	if skip != 5 {
		t.Errorf("expected skipRows=5, got %d", skip)
	}
}

func TestParsePreviewWithAutoDetect_BankExport(t *testing.T) {
	profile := ImportProfile{
		// Empty separator triggers auto-detection
		DateColumn:        "Date",
		DateFormat:        "02.01.2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit in CHF",
		DebitColumn:       "Debit in CHF",
	}
	result, err := ParsePreviewWithAutoDetect([]byte(bankExportCSV), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DetectedSeparator != ";" {
		t.Errorf("expected DetectedSeparator=';', got %q", result.DetectedSeparator)
	}
	if result.DetectedSkipRows != 5 {
		t.Errorf("expected DetectedSkipRows=5, got %d", result.DetectedSkipRows)
	}
	if len(result.Headers) != 7 {
		t.Errorf("expected 7 headers, got %d: %v", len(result.Headers), result.Headers)
	}
	if result.Headers[0] != "Date" {
		t.Errorf("expected first header='Date', got %q", result.Headers[0])
	}

	// Should have up to 10 preview rows (out of the valid data rows)
	if len(result.Rows) == 0 {
		t.Fatal("expected preview rows, got 0")
	}
	if len(result.Rows) > 10 {
		t.Errorf("expected at most 10 preview rows, got %d", len(result.Rows))
	}

	// First valid row should be the Coffee House expense
	first := result.Rows[0]
	if first.Date != "2026-03-05" {
		t.Errorf("first row: expected date=2026-03-05, got %s", first.Date)
	}
	if first.Type != "expense" {
		t.Errorf("first row: expected type=expense, got %s", first.Type)
	}
	if first.Amount != -8 {
		t.Errorf("first row: expected amount=-8, got %f", first.Amount)
	}
}

func TestParse_BankExport_FullParse(t *testing.T) {
	profile := bankExportProfile()
	rows, err := Parse(strings.NewReader(bankExportCSV), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count valid rows (non-error) — the blank line after header and footer lines
	// will produce error rows since they have fewer columns.
	var validRows []ParsedRow
	for _, r := range rows {
		if r.Error == "" {
			validRows = append(validRows, r)
		}
	}

	// We have 16 valid data lines + 2 footer lines = 18 total data rows
	// (blank lines are swallowed by the CSV reader)
	if len(validRows) != 16 {
		t.Errorf("expected 16 valid rows, got %d", len(validRows))
	}

	// Verify expenses
	expenses := 0
	incomes := 0
	for _, r := range validRows {
		switch r.Type {
		case "expense":
			expenses++
			if r.Amount >= 0 {
				t.Errorf("expense row %d has non-negative amount: %f", r.RowNumber, r.Amount)
			}
		case "income":
			incomes++
			if r.Amount <= 0 {
				t.Errorf("income row %d has non-positive amount: %f", r.RowNumber, r.Amount)
			}
		}
	}

	// 13 expenses, 3 incomes (850, 5500, 35)
	if expenses != 13 {
		t.Errorf("expected 13 expenses, got %d", expenses)
	}
	if incomes != 3 {
		t.Errorf("expected 3 incomes, got %d", incomes)
	}

	// Verify the salary row
	var salaryRow *ParsedRow
	for i := range validRows {
		if strings.Contains(validRows[i].Description, "Salary from Employer") {
			salaryRow = &validRows[i]
			break
		}
	}
	if salaryRow == nil {
		t.Fatal("salary row not found")
	}
	if salaryRow.Amount != 5500 {
		t.Errorf("salary: expected amount=5500, got %f", salaryRow.Amount)
	}
	if salaryRow.Date != "2026-02-15" {
		t.Errorf("salary: expected date=2026-02-15, got %s", salaryRow.Date)
	}
}

func TestParse_BankExport_MultipleSameDateSameAmount(t *testing.T) {
	profile := bankExportProfile()
	rows, err := Parse(strings.NewReader(bankExportCSV), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// On 20.02.2026 there are 4 entries for "Pub Downtown" at -7 and 1 at -15.
	// All should be parsed as separate rows — none should be lost.
	pubRows := make([]ParsedRow, 0)
	for _, r := range rows {
		if r.Error == "" && r.Date == "2026-02-20" && strings.Contains(r.Description, "Pub Downtown") {
			pubRows = append(pubRows, r)
		}
	}

	if len(pubRows) != 5 {
		t.Fatalf("expected 5 Pub Downtown rows on 20.02.2026, got %d", len(pubRows))
	}

	// Count: 4 at -7, 1 at -15
	countMinus7 := 0
	countMinus15 := 0
	for _, r := range pubRows {
		if math.Abs(r.Amount-(-7)) < 0.01 {
			countMinus7++
		} else if math.Abs(r.Amount-(-15)) < 0.01 {
			countMinus15++
		}
	}
	if countMinus7 != 4 {
		t.Errorf("expected 4 rows at -7, got %d", countMinus7)
	}
	if countMinus15 != 1 {
		t.Errorf("expected 1 row at -15, got %d", countMinus15)
	}
}

func TestParse_BankExport_DuplicateDetectionWithRepeats(t *testing.T) {
	profile := bankExportProfile()

	// Simulate: DB already has ONE "Pub Downtown" entry at -7 on 2026-02-20
	existing := []ExistingTx{
		{Date: "2026-02-20", Amount: -7},
	}

	rows, err := Parse(strings.NewReader(bankExportCSV), profile, nil, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 4 of the -7 Pub Downtown rows should be flagged as duplicate
	// (they all match date|amount = "2026-02-20|-7.00")
	// The -15 one should NOT be flagged.
	pubDups := 0
	pubNonDups := 0
	for _, r := range rows {
		if r.Error == "" && r.Date == "2026-02-20" && strings.Contains(r.Description, "Pub Downtown") {
			if r.IsDuplicate {
				pubDups++
			} else {
				pubNonDups++
			}
		}
	}

	// All 4 at -7 match the existing tx, so all 4 are marked duplicate
	if pubDups != 4 {
		t.Errorf("expected 4 duplicate Pub Downtown rows, got %d", pubDups)
	}
	// The -15 row is NOT a duplicate
	if pubNonDups != 1 {
		t.Errorf("expected 1 non-duplicate Pub Downtown row, got %d", pubNonDups)
	}

	// Verify other rows are NOT flagged as duplicates
	nonPubDups := 0
	for _, r := range rows {
		if r.Error == "" && r.IsDuplicate && (r.Date != "2026-02-20" || !strings.Contains(r.Description, "Pub Downtown")) {
			nonPubDups++
		}
	}
	if nonPubDups != 0 {
		t.Errorf("expected 0 non-pub duplicates, got %d", nonPubDups)
	}
}

func TestParse_BankExport_BlankAndFooterRowsAreErrors(t *testing.T) {
	profile := bankExportProfile()
	rows, err := Parse(strings.NewReader(bankExportCSV), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The 2 footer lines ("Disclaimer:" + second line) should produce error rows.
	// (blank lines are swallowed by the CSV reader)
	errorCount := 0
	for _, r := range rows {
		if r.Error != "" {
			errorCount++
		}
	}

	if errorCount < 2 {
		t.Errorf("expected at least 2 error rows (blank line + footer), got %d", errorCount)
	}
}

func TestParse_BankExport_CategoryMatching(t *testing.T) {
	profile := bankExportProfile()
	rules := []CategoryRule{
		{Pattern: "grocery", IsRegex: false, CategoryID: 10, Position: 1},
		{Pattern: "salary", IsRegex: false, CategoryID: 20, Position: 2},
		{Pattern: `Insurance`, IsRegex: false, CategoryID: 30, Position: 3},
	}

	rows, err := Parse(strings.NewReader(bankExportCSV), profile, rules, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that grocery rows get CategoryID 10
	for _, r := range rows {
		if r.Error != "" {
			continue
		}
		if strings.Contains(strings.ToLower(r.Description), "grocery") && r.CategoryID != 10 {
			t.Errorf("row %d (%s): expected CategoryID=10, got %d", r.RowNumber, r.Description, r.CategoryID)
		}
		if strings.Contains(strings.ToLower(r.Description), "salary") && r.CategoryID != 20 {
			t.Errorf("row %d (%s): expected CategoryID=20, got %d", r.RowNumber, r.Description, r.CategoryID)
		}
		if strings.Contains(r.Description, "Insurance") && r.CategoryID != 30 {
			t.Errorf("row %d (%s): expected CategoryID=30, got %d", r.RowNumber, r.Description, r.CategoryID)
		}
	}
}

func TestDetectColumns_SimpleCSV(t *testing.T) {
	headers := []string{"Date", "Description", "Amount"}
	dataRows := [][]string{
		{"01/03/2026", "Salary payment from employer", "1500.00"},
		{"02/03/2026", "Grocery store purchase", "-45.30"},
		{"03/03/2026", "Coffee shop", "-3.50"},
	}
	cols := DetectColumns(headers, dataRows)
	if cols == nil {
		t.Fatal("expected detected columns, got nil")
	}
	if cols.DateColumn != "Date" {
		t.Errorf("expected DateColumn=Date, got %q", cols.DateColumn)
	}
	if cols.DescriptionColumn != "Description" {
		t.Errorf("expected DescriptionColumn=Description, got %q", cols.DescriptionColumn)
	}
	if cols.AmountMode != "single" {
		t.Errorf("expected AmountMode=single, got %q", cols.AmountMode)
	}
	if cols.AmountColumn != "Amount" {
		t.Errorf("expected AmountColumn=Amount, got %q", cols.AmountColumn)
	}
}

func TestDetectColumns_SplitCreditDebit(t *testing.T) {
	headers := []string{"Fecha", "Concepto", "Haber", "Debe", "Categoria"}
	dataRows := [][]string{
		{"05.03.2026", "Coffee", "", "-8", "Dining"},
		{"04.03.2026", "Electronics", "", "-199.90", "Shopping"},
		{"03.03.2026", "Direct Debit", "", "-18.75", "Shipping"},
		{"02.03.2026", "Pizza", "", "-22", "Dining"},
		{"02.03.2026", "Mobile Payment", "850", "", "Income"},
		{"15.02.2026", "Salary from Employer", "5500", "", "Income"},
	}
	cols := DetectColumns(headers, dataRows)
	if cols == nil {
		t.Fatal("expected detected columns, got nil")
	}
	if cols.DateColumn != "Fecha" {
		t.Errorf("expected DateColumn=Fecha, got %q", cols.DateColumn)
	}
	if cols.AmountMode != "split" {
		t.Errorf("expected AmountMode=split, got %q", cols.AmountMode)
	}
	if cols.CreditColumn != "Haber" {
		t.Errorf("expected CreditColumn=Haber, got %q", cols.CreditColumn)
	}
	if cols.DebitColumn != "Debe" {
		t.Errorf("expected DebitColumn=Debe, got %q", cols.DebitColumn)
	}
	// Description should be the longest text column
	if cols.DescriptionColumn != "Concepto" {
		t.Errorf("expected DescriptionColumn=Concepto, got %q", cols.DescriptionColumn)
	}
}

func TestDetectColumns_NoData(t *testing.T) {
	cols := DetectColumns([]string{}, nil)
	if cols != nil {
		t.Errorf("expected nil for empty headers, got %+v", cols)
	}

	cols = DetectColumns([]string{"A", "B"}, [][]string{})
	if cols != nil {
		t.Errorf("expected nil for empty data rows, got %+v", cols)
	}
}

func TestDetectColumns_AllTextColumns(t *testing.T) {
	headers := []string{"Name", "City", "Country"}
	dataRows := [][]string{
		{"Alice", "Zurich", "Switzerland"},
		{"Bob", "Berlin", "Germany"},
	}
	cols := DetectColumns(headers, dataRows)
	// No dates, no numbers — should detect description at best
	if cols == nil {
		t.Skip("no detectable columns is acceptable for pure text data")
	}
	if cols.DateColumn != "" {
		t.Errorf("expected no date column, got %q", cols.DateColumn)
	}
	if cols.AmountColumn != "" {
		t.Errorf("expected no amount column, got %q", cols.AmountColumn)
	}
}

func TestDetectColumns_NoDatesInData(t *testing.T) {
	headers := []string{"ID", "Label", "Value"}
	dataRows := [][]string{
		{"not-a-date", "Salary", "1500.00"},
		{"also-not", "Grocery", "-45.30"},
	}
	cols := DetectColumns(headers, dataRows)
	if cols == nil {
		t.Fatal("expected detected columns (label+value), got nil")
	}
	if cols.DateColumn != "" {
		t.Errorf("expected empty DateColumn (no dates in data), got %q", cols.DateColumn)
	}
	if cols.AmountColumn != "Value" {
		t.Errorf("expected AmountColumn=Value, got %q", cols.AmountColumn)
	}
}

func TestDetectColumns_ArbitraryHeaderNames(t *testing.T) {
	// Headers have no semantic meaning — detection must rely purely on content
	headers := []string{"Col_A", "Col_B", "Col_C"}
	dataRows := [][]string{
		{"2026-03-01", "Salary payment from employer", "1500.00"},
		{"2026-03-02", "Grocery store downtown", "-45.30"},
		{"2026-03-03", "Coffee house visit", "-3.50"},
		{"2026-03-04", "Online electronics purchase", "-199.90"},
	}
	cols := DetectColumns(headers, dataRows)
	if cols == nil {
		t.Fatal("expected detected columns, got nil")
	}
	if cols.DateColumn != "Col_A" {
		t.Errorf("expected DateColumn=Col_A, got %q", cols.DateColumn)
	}
	if cols.DescriptionColumn != "Col_B" {
		t.Errorf("expected DescriptionColumn=Col_B, got %q", cols.DescriptionColumn)
	}
	if cols.AmountMode != "single" {
		t.Errorf("expected AmountMode=single, got %q", cols.AmountMode)
	}
	if cols.AmountColumn != "Col_C" {
		t.Errorf("expected AmountColumn=Col_C, got %q", cols.AmountColumn)
	}
}

func TestDetectColumns_BankExportContent(t *testing.T) {
	// Realistic bank export with split credit/debit and extra columns
	headers := []string{"Date", "Type", "Description", "Credit in CHF", "Debit in CHF", "Label", "Category"}
	dataRows := [][]string{
		{"05.03.2026", "Booking", "Purchase Coffee House", "", "-8", "", "Dining"},
		{"04.03.2026", "Booking", "Online Purchase Electronics", "", "-199.90", "", "Electronics"},
		{"03.03.2026", "Booking", "Direct Debit Postal Services", "", "-18.75", "", "Shipping"},
		{"02.03.2026", "Booking", "Mobile Payment Received", "850", "", "", "Income"},
		{"15.02.2026", "Booking", "Salary from Employer AG", "5500", "", "", "Salary"},
	}
	cols := DetectColumns(headers, dataRows)
	if cols == nil {
		t.Fatal("expected detected columns, got nil")
	}
	if cols.DateColumn != "Date" {
		t.Errorf("expected DateColumn=Date, got %q", cols.DateColumn)
	}
	if cols.DescriptionColumn != "Description" {
		t.Errorf("expected DescriptionColumn=Description, got %q", cols.DescriptionColumn)
	}
	if cols.AmountMode != "split" {
		t.Errorf("expected AmountMode=split, got %q", cols.AmountMode)
	}
}

func TestParsePreviewWithAutoDetect_DetectsColumns(t *testing.T) {
	data := []byte("Date,Description,Amount\n01/03/2026,Salary from employer,1500.00\n02/03/2026,Grocery store,-45.30\n")
	profile := ImportProfile{} // empty — triggers full auto-detect
	result, err := ParsePreviewWithAutoDetect(data, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DetectedColumns == nil {
		t.Fatal("expected DetectedColumns, got nil")
	}
	if result.DetectedColumns.DateColumn != "Date" {
		t.Errorf("expected DateColumn=Date, got %q", result.DetectedColumns.DateColumn)
	}
	if result.DetectedColumns.DescriptionColumn != "Description" {
		t.Errorf("expected DescriptionColumn=Description, got %q", result.DetectedColumns.DescriptionColumn)
	}
	if result.DetectedColumns.AmountColumn != "Amount" {
		t.Errorf("expected AmountColumn=Amount, got %q", result.DetectedColumns.AmountColumn)
	}
	// Since columns were auto-detected and applied, rows should be parsed
	if len(result.Rows) != 2 {
		t.Errorf("expected 2 preview rows (auto-detected columns applied), got %d", len(result.Rows))
	}
}

func TestParsePreviewWithAutoDetect_BankExportDetectsColumns(t *testing.T) {
	profile := ImportProfile{} // fully empty — auto-detect everything
	result, err := ParsePreviewWithAutoDetect([]byte(bankExportCSV), profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DetectedColumns == nil {
		t.Fatal("expected DetectedColumns, got nil")
	}
	if result.DetectedColumns.DateColumn != "Date" {
		t.Errorf("expected DateColumn=Date, got %q", result.DetectedColumns.DateColumn)
	}
	if result.DetectedColumns.AmountMode != "split" {
		t.Errorf("expected AmountMode=split, got %q", result.DetectedColumns.AmountMode)
	}
	// Preview rows should be parsed since columns were auto-detected
	if len(result.Rows) == 0 {
		t.Error("expected preview rows from auto-detected columns, got 0")
	}
}

func TestParse_SplitColumns(t *testing.T) {
	csv := `Date,Description,Credit,Debit
01/03/2026,Salary,1500.00,
02/03/2026,Grocery,,45.30
`
	profile := ImportProfile{
		CsvSeparator:      ",",
		DateColumn:        "Date",
		DateFormat:        "02/01/2006",
		DescriptionColumn: "Description",
		AmountMode:        "split",
		CreditColumn:      "Credit",
		DebitColumn:       "Debit",
	}
	rows, err := Parse(strings.NewReader(csv), profile, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Type != "income" || rows[0].Amount != 1500.00 {
		t.Errorf("row 0: expected income/1500.00, got %s/%f", rows[0].Type, rows[0].Amount)
	}
	if rows[1].Type != "expense" || rows[1].Amount != -45.30 {
		t.Errorf("row 1: expected expense/-45.30, got %s/%f", rows[1].Type, rows[1].Amount)
	}
}
