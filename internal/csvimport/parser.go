package csvimport

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"time"
)

// ParsedRow represents a single parsed CSV row ready for preview.
type ParsedRow struct {
	RowNumber   int     `json:"rowNumber"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	CategoryID  uint    `json:"categoryId"`
	IsDuplicate bool    `json:"isDuplicate"`
	Error       string  `json:"error,omitempty"`
}

// ExistingTx holds minimal info for duplicate detection.
type ExistingTx struct {
	Date   string // YYYY-MM-DD
	Amount float64
}

// Parse reads a CSV from r using the given profile's column mappings, applies
// category matching rules, and detects duplicates against existing transactions.
// It is a pure function with no DB access.
func Parse(r io.Reader, profile ImportProfile, rules []CategoryRule, existing []ExistingTx) ([]ParsedRow, error) {
	// Set up CSV reader
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // allow variable field counts

	sep := profile.CsvSeparator
	if sep == "" {
		sep = ","
	}
	if len(sep) > 0 {
		reader.Comma = rune(sep[0])
	}

	// Read all rows
	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	// Skip preamble rows
	skip := profile.SkipRows
	if skip >= len(allRows) {
		return nil, fmt.Errorf("skip_rows (%d) >= total rows (%d)", skip, len(allRows))
	}
	allRows = allRows[skip:]

	if len(allRows) == 0 {
		return nil, fmt.Errorf("no rows after skipping preamble")
	}

	// First remaining row is the header
	header := allRows[0]
	colIndex := make(map[string]int, len(header))
	for i, h := range header {
		colIndex[strings.TrimSpace(h)] = i
	}

	// Validate required columns
	dateIdx, ok := colIndex[profile.DateColumn]
	if !ok {
		return nil, fmt.Errorf("required column %q not found in headers", profile.DateColumn)
	}
	descIdx, ok := colIndex[profile.DescriptionColumn]
	if !ok {
		return nil, fmt.Errorf("required column %q not found in headers", profile.DescriptionColumn)
	}
	amtIdx, ok := colIndex[profile.AmountColumn]
	if !ok {
		return nil, fmt.Errorf("required column %q not found in headers", profile.AmountColumn)
	}

	// Build duplicate detection set from existing transactions
	dupSet := make(map[string]struct{}, len(existing))
	for _, ex := range existing {
		key := dupKey(ex.Date, ex.Amount)
		dupSet[key] = struct{}{}
	}

	// Parse data rows (everything after header)
	dataRows := allRows[1:]
	result := make([]ParsedRow, 0, len(dataRows))

	for i, row := range dataRows {
		rowNum := skip + 1 + i + 1 // 1-based, accounting for skipped rows and header
		parsed := ParsedRow{RowNumber: rowNum}

		// Safely extract fields
		if dateIdx >= len(row) || descIdx >= len(row) || amtIdx >= len(row) {
			parsed.Error = "row has fewer columns than expected"
			result = append(result, parsed)
			continue
		}

		// Parse date
		rawDate := strings.TrimSpace(row[dateIdx])
		t, err := time.Parse(profile.DateFormat, rawDate)
		if err != nil {
			parsed.Error = fmt.Sprintf("invalid date %q: %v", rawDate, err)
			result = append(result, parsed)
			continue
		}
		parsed.Date = t.Format("2006-01-02")

		// Parse description
		parsed.Description = strings.TrimSpace(row[descIdx])

		// Parse amount
		rawAmount := strings.TrimSpace(row[amtIdx])
		amount, err := parseAmount(rawAmount)
		if err != nil {
			parsed.Error = fmt.Sprintf("invalid amount %q: %v", rawAmount, err)
			result = append(result, parsed)
			continue
		}
		parsed.Amount = amount

		// Determine type
		if amount >= 0 {
			parsed.Type = "income"
		} else {
			parsed.Type = "expense"
		}

		// Match category
		parsed.CategoryID = matchCategory(parsed.Description, rules)

		// Check duplicate
		key := dupKey(parsed.Date, parsed.Amount)
		if _, found := dupSet[key]; found {
			parsed.IsDuplicate = true
		}

		result = append(result, parsed)
	}

	return result, nil
}

// dupKey returns a string key for duplicate detection: "date|amount" with amount
// formatted to 2 decimal places.
func dupKey(date string, amount float64) string {
	return fmt.Sprintf("%s|%.2f", date, amount)
}

// matchCategory iterates rules in order and returns the categoryID of the first
// matching rule, or 0 if none match.
func matchCategory(description string, rules []CategoryRule) uint {
	descLower := strings.ToLower(description)
	for _, rule := range rules {
		if rule.IsRegex {
			matched, err := regexp.MatchString(rule.Pattern, description)
			if err == nil && matched {
				return rule.CategoryID
			}
		} else {
			if strings.Contains(descLower, strings.ToLower(rule.Pattern)) {
				return rule.CategoryID
			}
		}
	}
	return 0
}

// parseAmount parses a string amount handling various formats:
//   - Standard:        "1234.56"
//   - Comma decimal:   "1234,56"
//   - European mixed:  "1.234,56" (dot = thousands, comma = decimal)
//   - US mixed:        "1,234.56" (comma = thousands, dot = decimal)
//   - Negative:        "-45.60"
//   - Empty → error
func parseAmount(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}

	// Detect sign and strip it for processing
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}

	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")

	var cleaned string

	switch {
	case hasDot && hasComma:
		// Both present: determine which is the decimal separator
		lastDot := strings.LastIndex(s, ".")
		lastComma := strings.LastIndex(s, ",")
		if lastComma > lastDot {
			// European: 1.234,56 — dot is thousands, comma is decimal
			cleaned = strings.ReplaceAll(s, ".", "")
			cleaned = strings.Replace(cleaned, ",", ".", 1)
		} else {
			// US: 1,234.56 — comma is thousands, dot is decimal
			cleaned = strings.ReplaceAll(s, ",", "")
		}
	case hasComma && !hasDot:
		// Could be "1234,56" (comma as decimal) or "1,234" (comma as thousands)
		// Heuristic: if exactly 2 digits after the last comma, treat as decimal
		lastComma := strings.LastIndex(s, ",")
		afterComma := s[lastComma+1:]
		if len(afterComma) == 2 {
			cleaned = strings.Replace(s, ",", ".", 1)
		} else {
			// Thousands separator only
			cleaned = strings.ReplaceAll(s, ",", "")
		}
	default:
		// Only dots or no separators
		cleaned = s
	}

	// Parse the cleaned string
	var val float64
	_, err := fmt.Sscanf(cleaned, "%f", &val)
	if err != nil {
		return 0, fmt.Errorf("cannot parse amount %q: %w", s, err)
	}

	if negative {
		val = -val
	}

	// Round to 2 decimal places to avoid floating point artifacts
	val = math.Round(val*100) / 100

	return val, nil
}
