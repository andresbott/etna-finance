package csvimport

import (
	"bytes"
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

// PreviewResult holds the result of a CSV preview parse.
type PreviewResult struct {
	Headers            []string         `json:"headers"`
	Rows               []ParsedRow      `json:"rows"`
	TotalRows          int              `json:"totalRows"`
	DetectedSeparator  string           `json:"detectedSeparator,omitempty"`
	DetectedSkipRows   int              `json:"detectedSkipRows"`
	DetectedDateFormat string           `json:"detectedDateFormat,omitempty"`
	DetectedColumns    *DetectedColumns `json:"detectedColumns,omitempty"`
}

// DetectedColumns holds auto-detected column mappings.
type DetectedColumns struct {
	DateColumn        string `json:"dateColumn,omitempty"`
	DescriptionColumn string `json:"descriptionColumn,omitempty"`
	AmountColumn      string `json:"amountColumn,omitempty"`
	AmountMode        string `json:"amountMode,omitempty"` // "single" or "split"
	CreditColumn      string `json:"creditColumn,omitempty"`
	DebitColumn       string `json:"debitColumn,omitempty"`
}

// DetectCSVSettings analyzes raw CSV bytes to auto-detect the separator and skip rows.
// It tries each candidate separator and scores by how many rows share the most common
// column count (must be > 1). For skip rows, it finds the first row matching the
// dominant column count.
func DetectCSVSettings(data []byte) (separator string, skipRows int) {
	candidates := []string{",", ";", "\t"}

	type result struct {
		sep       string
		score     int
		modeCount int
		rows      [][]string
	}

	var best result

	for _, sep := range candidates {
		reader := csv.NewReader(bytes.NewReader(data))
		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1
		reader.Comma = rune(sep[0])

		allRows, err := reader.ReadAll()
		if err != nil {
			continue
		}

		// Count column counts per row
		countFreq := make(map[int]int)
		for _, row := range allRows {
			countFreq[len(row)]++
		}

		// Find mode column count (excluding count=1)
		modeCount := 0
		modeFreq := 0
		for colCount, freq := range countFreq {
			if colCount <= 1 {
				continue
			}
			if freq > modeFreq || (freq == modeFreq && colCount > modeCount) {
				modeCount = colCount
				modeFreq = freq
			}
		}

		score := modeFreq

		// Tie-break: prefer earlier candidate (,  >  ;  >  \t)
		if score > best.score || (score == best.score && score > 0) {
			// Only replace if strictly better
			if score > best.score {
				best = result{sep: sep, score: score, modeCount: modeCount, rows: allRows}
			}
		} else if best.sep == "" {
			best = result{sep: sep, score: score, modeCount: modeCount, rows: allRows}
		}
	}

	if best.sep == "" {
		return ",", 0
	}

	// Find skipRows: first row matching the dominant column count
	skipRows = 0
	for i, row := range best.rows {
		if len(row) == best.modeCount {
			skipRows = i
			break
		}
	}

	return best.sep, skipRows
}

// columnProfile describes the detected type of a single CSV column based on content analysis.
type columnProfile struct {
	index       int
	header      string
	dateRatio   float64 // fraction of non-empty values that parse as dates
	numRatio    float64 // fraction of non-empty values that parse as amounts
	emptyRatio  float64 // fraction of values that are empty
	avgTextLen  float64 // average length of non-empty string values
	nonEmpty    int     // count of non-empty values in the sample
	hasNegative bool    // whether any numeric value is negative
	hasPositive bool    // whether any numeric value is positive
}

// DetectColumns examines CSV column content to infer column mappings.
// It classifies each column as date, numeric, or text by analyzing sample data,
// then picks the best candidate for each role.
func DetectColumns(headers []string, dataRows [][]string) *DetectedColumns {
	if len(headers) == 0 || len(dataRows) == 0 {
		return nil
	}

	sampleSize := min(len(dataRows), 20)
	sample := dataRows[:sampleSize]

	profiles := make([]columnProfile, len(headers))
	for i, h := range headers {
		profiles[i] = profileColumn(i, strings.TrimSpace(h), sample)
	}

	result := &DetectedColumns{}

	bestDateIdx := detectDateColumn(profiles, result)
	numericCols := collectNumericCols(profiles, bestDateIdx)
	detectAmountColumns(numericCols, sample, result)
	detectDescriptionColumn(profiles, numericCols, bestDateIdx, result)

	if result.DateColumn == "" && result.DescriptionColumn == "" && result.AmountColumn == "" && result.CreditColumn == "" {
		return nil
	}

	return result
}

// detectDateColumn picks the column with the highest date ratio (>50%) and sets it on result.
func detectDateColumn(profiles []columnProfile, result *DetectedColumns) int {
	bestDateIdx := -1
	bestDateRatio := 0.5
	for i, p := range profiles {
		if p.dateRatio > bestDateRatio {
			bestDateRatio = p.dateRatio
			bestDateIdx = i
		}
	}
	if bestDateIdx >= 0 {
		result.DateColumn = profiles[bestDateIdx].header
	}
	return bestDateIdx
}

// collectNumericCols returns columns with numRatio > 50%, excluding the date column.
func collectNumericCols(profiles []columnProfile, excludeIdx int) []columnProfile {
	var numericCols []columnProfile
	for i, p := range profiles {
		if i == excludeIdx {
			continue
		}
		if p.numRatio > 0.5 {
			numericCols = append(numericCols, p)
		}
	}
	return numericCols
}

// detectAmountColumns detects split or single amount columns from numeric columns.
func detectAmountColumns(numericCols []columnProfile, sample [][]string, result *DetectedColumns) {
	if len(numericCols) >= 2 {
		splitA, splitB, isSplit := detectSplitPair(numericCols, sample)
		if isSplit {
			result.AmountMode = "split"
			result.CreditColumn = splitA
			result.DebitColumn = splitB
			return
		}
	}
	if len(numericCols) > 0 {
		result.AmountMode = "single"
		best := numericCols[0]
		for _, p := range numericCols[1:] {
			if p.numRatio > best.numRatio || (p.numRatio == best.numRatio && p.emptyRatio < best.emptyRatio) {
				best = p
			}
		}
		result.AmountColumn = best.header
	}
}

// detectDescriptionColumn picks the text column with the longest average text length.
func detectDescriptionColumn(profiles []columnProfile, numericCols []columnProfile, dateIdx int, result *DetectedColumns) {
	usedCols := map[int]bool{}
	if dateIdx >= 0 {
		usedCols[dateIdx] = true
	}
	for _, p := range numericCols {
		usedCols[p.index] = true
	}

	bestDescIdx := -1
	var bestDescLen float64
	for i, p := range profiles {
		if usedCols[i] || p.emptyRatio > 0.5 {
			continue
		}
		if p.avgTextLen > bestDescLen {
			bestDescLen = p.avgTextLen
			bestDescIdx = i
		}
	}
	if bestDescIdx >= 0 {
		result.DescriptionColumn = profiles[bestDescIdx].header
	}
}

// profileColumn analyzes sample data for a single column and returns its profile.
func profileColumn(index int, header string, sample [][]string) columnProfile {
	p := columnProfile{index: index, header: header}

	var dateCount, numCount, emptyCount int
	var totalTextLen int
	var nonEmptyCount int

	for _, row := range sample {
		if index >= len(row) {
			emptyCount++
			continue
		}
		v := strings.TrimSpace(row[index])
		if v == "" {
			emptyCount++
			continue
		}

		nonEmptyCount++
		totalTextLen += len(v)

		// Check if it's a date
		if DetectDateFormat([]string{v}) != "" {
			dateCount++
		}

		// Check if it's a numeric amount
		if amt, err := parseAmount(v); err == nil {
			numCount++
			if amt < 0 {
				p.hasNegative = true
			}
			if amt > 0 {
				p.hasPositive = true
			}
		}
	}

	total := len(sample)
	if total > 0 {
		p.emptyRatio = float64(emptyCount) / float64(total)
	}
	if nonEmptyCount > 0 {
		p.dateRatio = float64(dateCount) / float64(nonEmptyCount)
		p.numRatio = float64(numCount) / float64(nonEmptyCount)
		p.avgTextLen = float64(totalTextLen) / float64(nonEmptyCount)
	}
	p.nonEmpty = nonEmptyCount

	return p
}

// detectSplitPair looks for two numeric columns that form a credit/debit split:
// their non-empty values should be mutually exclusive (one is empty when the other
// has a value). Returns the credit header (positive values), debit header (negative
// values), and whether a split was found.
func detectSplitPair(numericCols []columnProfile, sample [][]string) (credit, debit string, found bool) {
	for i := 0; i < len(numericCols); i++ {
		for j := i + 1; j < len(numericCols); j++ {
			a := numericCols[i]
			b := numericCols[j]

			// Both columns must have a significant empty ratio (since they alternate)
			if a.emptyRatio < 0.2 && b.emptyRatio < 0.2 {
				continue
			}

			// Check mutual exclusivity in the sample
			mutuallyExclusive := 0
			bothPresent := 0
			for _, row := range sample {
				aEmpty := a.index >= len(row) || strings.TrimSpace(row[a.index]) == ""
				bEmpty := b.index >= len(row) || strings.TrimSpace(row[b.index]) == ""
				if aEmpty != bEmpty {
					mutuallyExclusive++
				} else if !aEmpty && !bEmpty {
					bothPresent++
				}
			}

			total := mutuallyExclusive + bothPresent
			if total == 0 {
				continue
			}

			// At least 80% of rows with any value should be mutually exclusive
			if float64(mutuallyExclusive)/float64(total) < 0.8 {
				continue
			}

			// Determine which is credit (positive) and which is debit (negative/expense)
			creditCol, debitCol := a.header, b.header
			if a.hasNegative && !b.hasNegative {
				creditCol, debitCol = b.header, a.header
			}

			return creditCol, debitCol, true
		}
	}
	return "", "", false
}

// ParsePreviewWithAutoDetect reads raw CSV bytes and returns a preview result.
// If the profile's CsvSeparator is empty, it auto-detects the separator and skip rows.
// If the profile's DateFormat is empty and a DateColumn is set, it auto-detects the date format.
func ParsePreviewWithAutoDetect(data []byte, profile ImportProfile) (PreviewResult, error) {
	var detectedSep string
	var detectedSkip int

	if profile.CsvSeparator == "" {
		detectedSep, detectedSkip = DetectCSVSettings(data)
		profile.CsvSeparator = detectedSep
		profile.SkipRows = detectedSkip
	}

	var detectedCols *DetectedColumns
	if !hasMappings(profile) {
		detectedCols, profile = autoDetectColumns(data, profile)
	}

	detectedDateFormat := autoDetectDateFormat(data, profile)
	if detectedDateFormat != "" {
		profile.DateFormat = detectedDateFormat
	}

	result, err := ParsePreview(bytes.NewReader(data), profile)
	if err != nil {
		return PreviewResult{}, err
	}

	if detectedSep != "" {
		result.DetectedSeparator = detectedSep
		result.DetectedSkipRows = detectedSkip
	}
	if detectedDateFormat != "" {
		result.DetectedDateFormat = detectedDateFormat
	}
	if detectedCols != nil {
		result.DetectedColumns = detectedCols
	}

	return result, nil
}

// autoDetectColumns detects column mappings from CSV data and applies them to the profile.
func autoDetectColumns(data []byte, profile ImportProfile) (*DetectedColumns, ImportProfile) {
	header, dataRows, _, _, err := readCSV(bytes.NewReader(data), profile)
	if err != nil {
		return nil, profile
	}
	detectedCols := DetectColumns(header, dataRows)
	if detectedCols == nil {
		return nil, profile
	}
	profile = applyDetectedColumns(profile, detectedCols)
	return detectedCols, profile
}

// applyDetectedColumns merges detected column mappings into the profile where fields are empty.
func applyDetectedColumns(profile ImportProfile, cols *DetectedColumns) ImportProfile {
	if profile.DateColumn == "" && cols.DateColumn != "" {
		profile.DateColumn = cols.DateColumn
	}
	if profile.DescriptionColumn == "" && cols.DescriptionColumn != "" {
		profile.DescriptionColumn = cols.DescriptionColumn
	}
	if cols.AmountMode == "split" && (profile.AmountMode == "" || profile.AmountMode == "single") {
		if profile.CreditColumn == "" && cols.CreditColumn != "" {
			profile.AmountMode = "split"
			profile.CreditColumn = cols.CreditColumn
		}
		if profile.DebitColumn == "" && cols.DebitColumn != "" {
			profile.DebitColumn = cols.DebitColumn
		}
	} else if profile.AmountColumn == "" && cols.AmountColumn != "" {
		profile.AmountColumn = cols.AmountColumn
	}
	return profile
}

// autoDetectDateFormat detects the date format from CSV data if not already set.
// Returns the detected format string, or "" if detection was not needed or failed.
func autoDetectDateFormat(data []byte, profile ImportProfile) string {
	if profile.DateFormat != "" || profile.DateColumn == "" {
		return ""
	}
	_, dataRows, _, colIndex, err := readCSV(bytes.NewReader(data), profile)
	if err != nil {
		return ""
	}
	dateIdx, ok := colIndex[profile.DateColumn]
	if !ok {
		return ""
	}
	samples := make([]string, 0, min(len(dataRows), 20))
	for i := 0; i < len(dataRows) && i < 20; i++ {
		if dateIdx < len(dataRows[i]) {
			samples = append(samples, dataRows[i][dateIdx])
		}
	}
	return DetectDateFormat(samples)
}

// resolveAmount reads amount information from a CSV row, supporting both single
// and split (credit/debit) column modes.
// Returns (amount, type, errorString). errorString is "" on success.
func resolveAmount(row []string, colIndex map[string]int, mode string, profile ImportProfile) (float64, string, string) {
	switch mode {
	case "split":
		creditIdx, creditOK := colIndex[profile.CreditColumn]
		debitIdx, debitOK := colIndex[profile.DebitColumn]

		var creditStr, debitStr string
		if creditOK && creditIdx < len(row) {
			creditStr = strings.TrimSpace(row[creditIdx])
		}
		if debitOK && debitIdx < len(row) {
			debitStr = strings.TrimSpace(row[debitIdx])
		}

		hasCredit := creditStr != ""
		hasDebit := debitStr != ""

		if hasCredit && hasDebit {
			return 0, "", "both credit and debit have values"
		}
		if !hasCredit && !hasDebit {
			return 0, "", "no amount found"
		}

		if hasCredit {
			amt, err := parseAmount(creditStr)
			if err != nil {
				return 0, "", fmt.Sprintf("invalid credit amount %q: %v", creditStr, err)
			}
			return math.Abs(amt), "income", ""
		}

		// hasDebit
		amt, err := parseAmount(debitStr)
		if err != nil {
			return 0, "", fmt.Sprintf("invalid debit amount %q: %v", debitStr, err)
		}
		return -math.Abs(amt), "expense", ""

	default: // "single"
		amtIdx, ok := colIndex[profile.AmountColumn]
		if !ok || amtIdx >= len(row) {
			return 0, "", fmt.Sprintf("amount column %q not accessible", profile.AmountColumn)
		}
		rawAmount := strings.TrimSpace(row[amtIdx])
		amount, err := parseAmount(rawAmount)
		if err != nil {
			return 0, "", fmt.Sprintf("invalid amount %q: %v", rawAmount, err)
		}
		txType := "income"
		if amount < 0 {
			txType = "expense"
		}
		return amount, txType, ""
	}
}

// readCSV reads and returns all CSV rows from r using the profile's separator and skip settings.
// Returns the header row, data rows, skip count, colIndex, and any error.
func readCSV(r io.Reader, profile ImportProfile) (header []string, dataRows [][]string, skip int, colIndex map[string]int, err error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	sep := profile.CsvSeparator
	if sep == "" {
		sep = ","
	}
	if len(sep) > 0 {
		reader.Comma = rune(sep[0])
	}

	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, 0, nil, fmt.Errorf("error reading CSV: %w", err)
	}

	skip = profile.SkipRows
	if skip >= len(allRows) {
		return nil, nil, 0, nil, fmt.Errorf("skip_rows (%d) >= total rows (%d)", skip, len(allRows))
	}
	allRows = allRows[skip:]

	if len(allRows) == 0 {
		return nil, nil, 0, nil, fmt.Errorf("no rows after skipping preamble")
	}

	header = allRows[0]
	colIndex = make(map[string]int, len(header))
	for i, h := range header {
		colIndex[strings.TrimSpace(h)] = i
	}

	dataRows = allRows[1:]
	return header, dataRows, skip, colIndex, nil
}

// knownDateFormats lists the Go reference-time formats we try for auto-detection,
// ordered from least ambiguous to most ambiguous.
var knownDateFormats = []string{
	// Date-time formats (most specific first)
	"2006-01-02 15:04:05",  // YYYY-MM-DD HH:MM:SS (ISO 8601)
	"02/01/2006 15:04:05",  // DD/MM/YYYY HH:MM:SS (European)
	"01/02/2006 15:04:05",  // MM/DD/YYYY HH:MM:SS (US)
	"02.01.2006 15:04:05",  // DD.MM.YYYY HH:MM:SS (European dots)
	"2006/01/02 15:04:05",  // YYYY/MM/DD HH:MM:SS
	"02-01-2006 15:04:05",  // DD-MM-YYYY HH:MM:SS
	"01-02-2006 15:04:05",  // MM-DD-YYYY HH:MM:SS
	"2006-01-02T15:04:05",  // ISO 8601 with T separator
	"2006-01-02T15:04:05Z", // ISO 8601 UTC

	// Date-only formats
	"2006-01-02",  // YYYY-MM-DD (ISO 8601)
	"02.01.2006",  // DD.MM.YYYY (European with dots)
	"02/01/2006",  // DD/MM/YYYY (European with slashes)
	"01/02/2006",  // MM/DD/YYYY (US with slashes)
	"02-01-2006",  // DD-MM-YYYY
	"01-02-2006",  // MM-DD-YYYY
	"2006/01/02",  // YYYY/MM/DD
	"Jan 2, 2006", // Mon D, YYYY
	"2 Jan 2006",  // D Mon YYYY
}

// DetectDateFormat tries each known date format against the provided sample values
// and returns the best match. It handles DD/MM vs MM/DD ambiguity by checking
// whether any value has a day-position > 12 (which disambiguates the format).
// Returns empty string if no format matches.
type dateCandidate struct {
	format string
	count  int
}

func DetectDateFormat(samples []string) string {
	if len(samples) == 0 {
		return ""
	}

	var candidates []dateCandidate
	for _, fmt := range knownDateFormats {
		matches := 0
		for _, s := range samples {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if _, err := time.Parse(fmt, s); err == nil {
				matches++
			}
		}
		if matches > 0 {
			candidates = append(candidates, dateCandidate{format: fmt, count: matches})
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	// If there's only one candidate with the highest match count, return it
	maxCount := 0
	for _, c := range candidates {
		if c.count > maxCount {
			maxCount = c.count
		}
	}

	var best []dateCandidate
	for _, c := range candidates {
		if c.count == maxCount {
			best = append(best, c)
		}
	}

	if len(best) == 1 {
		return best[0].format
	}

	// Ambiguity: multiple formats match all samples (e.g., DD/MM vs MM/DD).
	// Try to disambiguate by checking if any sample has a first or second
	// numeric component > 12, which rules out that position being a month.
	return disambiguateDateFormats(best, samples)
}

// disambiguateDateFormats resolves ambiguous date format candidates by examining
// the numeric components of sample values. If any value has a component > 12
// in the day position of a format, that format is preferred.
func disambiguateDateFormats(candidates []dateCandidate, samples []string) string {
	// Extract the separator used in the samples
	digitSplitter := regexp.MustCompile(`\d+`)

	for _, s := range samples {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		parts := digitSplitter.FindAllString(s, -1)
		if len(parts) < 3 {
			continue
		}

		// Check if any numeric part > 12 — that part can't be a month
		for i, p := range parts[:3] {
			val := 0
			for _, ch := range p {
				val = val*10 + int(ch-'0')
			}
			if val > 12 {
				// This position is NOT a month. Find the candidate format where
				// position i is a day (not a month).
				for _, c := range candidates {
					refParts := digitSplitter.FindAllString(c.format, -1)
					if len(refParts) < 3 {
						continue
					}
					// In Go's reference time: month=01, day=02, year=2006
					// If the value > 12 is at position i, the format at position i
					// must be "02" (day) or "2" (day), not "01" (month).
					if refParts[i] == "02" || refParts[i] == "2" {
						return c.format
					}
				}
			}
		}
	}

	// Could not disambiguate — return the first candidate (least ambiguous by order)
	return candidates[0].format
}

// amountMode returns the resolved amount mode for a profile, defaulting to "single".
func amountMode(profile ImportProfile) string {
	if profile.AmountMode == "split" {
		return "split"
	}
	return "single"
}

// hasMappings returns true if the profile has enough column mappings to parse rows.
func hasMappings(profile ImportProfile) bool {
	if profile.DateColumn == "" || profile.DescriptionColumn == "" {
		return false
	}
	mode := amountMode(profile)
	if mode == "split" {
		return profile.CreditColumn != "" && profile.DebitColumn != ""
	}
	return profile.AmountColumn != ""
}

// ParsePreview reads a CSV and returns headers plus up to 10 parsed rows.
// If the profile has no column mappings, only headers and totalRows are returned.
// It does NOT do category matching or duplicate detection.
func ParsePreview(r io.Reader, profile ImportProfile) (PreviewResult, error) {
	header, dataRows, _, colIndex, err := readCSV(r, profile)
	if err != nil {
		return PreviewResult{}, err
	}

	// Trim header names
	headers := make([]string, len(header))
	for i, h := range header {
		headers[i] = strings.TrimSpace(h)
	}

	result := PreviewResult{
		Headers:   headers,
		TotalRows: len(dataRows),
	}

	if !hasMappings(profile) {
		result.Rows = []ParsedRow{}
		return result, nil
	}

	// Validate required columns exist
	dateIdx, ok := colIndex[profile.DateColumn]
	if !ok {
		return PreviewResult{}, fmt.Errorf("required column %q not found in headers", profile.DateColumn)
	}
	descIdx, ok := colIndex[profile.DescriptionColumn]
	if !ok {
		return PreviewResult{}, fmt.Errorf("required column %q not found in headers", profile.DescriptionColumn)
	}

	mode := amountMode(profile)
	if mode == "split" {
		if _, ok := colIndex[profile.CreditColumn]; !ok {
			return PreviewResult{}, fmt.Errorf("required column %q not found in headers", profile.CreditColumn)
		}
		if _, ok := colIndex[profile.DebitColumn]; !ok {
			return PreviewResult{}, fmt.Errorf("required column %q not found in headers", profile.DebitColumn)
		}
	} else {
		if _, ok := colIndex[profile.AmountColumn]; !ok {
			return PreviewResult{}, fmt.Errorf("required column %q not found in headers", profile.AmountColumn)
		}
	}

	// Parse up to 10 rows
	maxRows := 10
	if len(dataRows) < maxRows {
		maxRows = len(dataRows)
	}

	rows := make([]ParsedRow, 0, maxRows)
	for i := 0; i < maxRows; i++ {
		row := dataRows[i]
		parsed := ParsedRow{RowNumber: i + 1}

		// Safely check date/desc columns
		if dateIdx >= len(row) || descIdx >= len(row) {
			parsed.Error = "row has fewer columns than expected"
			rows = append(rows, parsed)
			continue
		}

		// Parse date
		rawDate := strings.TrimSpace(row[dateIdx])
		if rawDate == "" {
			parsed.Error = "empty date"
			rows = append(rows, parsed)
			continue
		}
		t, err := time.Parse(profile.DateFormat, rawDate)
		if err != nil {
			parsed.Error = fmt.Sprintf("invalid date %q: %v", rawDate, err)
			rows = append(rows, parsed)
			continue
		}
		parsed.Date = t.Format("2006-01-02")

		// Parse description
		parsed.Description = strings.TrimSpace(row[descIdx])

		// Parse amount via resolveAmount
		amount, txType, errStr := resolveAmount(row, colIndex, mode, profile)
		if errStr != "" {
			parsed.Error = errStr
			rows = append(rows, parsed)
			continue
		}
		parsed.Amount = amount
		parsed.Type = txType

		rows = append(rows, parsed)
	}

	result.Rows = rows
	return result, nil
}

// Parse reads a CSV from r using the given profile's column mappings, applies
// category matching rules, and detects duplicates against existing transactions.
// It is a pure function with no DB access.
func Parse(r io.Reader, profile ImportProfile, groups []CategoryRuleGroup, existing []ExistingTx) ([]ParsedRow, error) {
	header, dataRows, skip, colIndex, err := readCSV(r, profile)
	if err != nil {
		return nil, err
	}
	_ = header

	// Validate required columns
	dateIdx, ok := colIndex[profile.DateColumn]
	if !ok {
		return nil, fmt.Errorf("required column %q not found in headers", profile.DateColumn)
	}
	descIdx, ok := colIndex[profile.DescriptionColumn]
	if !ok {
		return nil, fmt.Errorf("required column %q not found in headers", profile.DescriptionColumn)
	}

	mode := amountMode(profile)
	if mode == "split" {
		if _, ok := colIndex[profile.CreditColumn]; !ok {
			return nil, fmt.Errorf("required column %q not found in headers", profile.CreditColumn)
		}
		if _, ok := colIndex[profile.DebitColumn]; !ok {
			return nil, fmt.Errorf("required column %q not found in headers", profile.DebitColumn)
		}
	} else {
		if _, ok := colIndex[profile.AmountColumn]; !ok {
			return nil, fmt.Errorf("required column %q not found in headers", profile.AmountColumn)
		}
	}

	// Build duplicate detection set from existing transactions
	dupSet := make(map[string]struct{}, len(existing))
	for _, ex := range existing {
		key := dupKey(ex.Date, ex.Amount)
		dupSet[key] = struct{}{}
	}

	// Parse data rows
	result := make([]ParsedRow, 0, len(dataRows))

	for i, row := range dataRows {
		rowNum := skip + 1 + i + 1 // 1-based, accounting for skipped rows and header
		parsed := ParsedRow{RowNumber: rowNum}

		// Safely extract fields
		if dateIdx >= len(row) || descIdx >= len(row) {
			parsed.Error = "row has fewer columns than expected"
			result = append(result, parsed)
			continue
		}

		// Parse date
		rawDate := strings.TrimSpace(row[dateIdx])
		if rawDate == "" {
			parsed.Error = "empty date"
			result = append(result, parsed)
			continue
		}
		t, err := time.Parse(profile.DateFormat, rawDate)
		if err != nil {
			parsed.Error = fmt.Sprintf("invalid date %q: %v", rawDate, err)
			result = append(result, parsed)
			continue
		}
		parsed.Date = t.Format("2006-01-02")

		// Parse description
		parsed.Description = strings.TrimSpace(row[descIdx])

		// Parse amount via resolveAmount
		amount, txType, errStr := resolveAmount(row, colIndex, mode, profile)
		if errStr != "" {
			parsed.Error = errStr
			result = append(result, parsed)
			continue
		}
		parsed.Amount = amount
		parsed.Type = txType

		// Match category
		parsed.CategoryID = MatchCategory(parsed.Description, groups)

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
