package parsing

import (
	"encoding/csv"
	"fmt"
	"ifrs16_calculator/internal/lease"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const dateLayout = "2006-01-02" // Define a standard date format for parsing (YYYY-MM-DD)

// Config options for parsing
type ParseConfig struct {
	SkipHeader bool
	// Add other options like required columns, custom date formats etc.
}

// Utility functions for parsing values

// parseDateValue parses a string into a time.Time using the standard date layout
func parseDateValue(value string) (time.Time, error) {
	return time.Parse(dateLayout, strings.TrimSpace(value))
}

// parseFloatValue parses a string into a float64
func parseFloatValue(value string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

// ParseLeasesFromFile reads lease data from the provided reader based on the file type.
// It requires the file content to be fully available (e.g., read into memory or a temp file)
// especially for XLSX files due to library constraints.
func ParseLeasesFromFile(reader io.Reader, fileType string, config ParseConfig) ([]lease.Lease, error) {
	// Read all data into a buffer first, as excelize might need ReaderAt and size,
	// and CSV parsing is simpler when not dealing with potential stream interruptions.
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read input data: %w", err)
	}

	// Use bytes.NewReader as it implements io.Reader, io.Seeker, io.ReaderAt
	dataReader := strings.NewReader(string(data)) // Use string for easier handling with CSV

	switch strings.ToLower(fileType) {
	case "csv":
		return ParseCSV(dataReader, config)
	case "xlsx":
		// For excelize OpenReader, we still need ReaderAt and Size.
		// Since we read all data, we can create a ReaderAt compatible reader.
		// Using strings.NewReader works as it implements ReaderAt.
		dataBytesReader := strings.NewReader(string(data))
		xlsxFile, err := excelize.OpenReader(dataBytesReader)
		if err != nil {
			return nil, fmt.Errorf("failed to open xlsx reader: %w", err)
		}
		defer func() {
			if err := xlsxFile.Close(); err != nil {
				log.Printf("Error closing XLSX file: %v", err) // Log error on close
			}
		}()
		return ParseXLSX(xlsxFile, config)
	default:
		return nil, fmt.Errorf("unsupported file type: %s (supported: csv, xlsx)", fileType)
	}
}

// ParseCSV parses lease data from a CSV reader.
func ParseCSV(reader io.Reader, config ParseConfig) ([]lease.Lease, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	// Skip header row if configured
	if config.SkipHeader {
		_, err := csvReader.Read()
		if err == io.EOF {
			return []lease.Lease{}, nil // Empty file is valid if we expect a header
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read csv header: %w", err)
		}
	}
	// TODO: Optionally read and validate header against expected columns

	leases := []lease.Lease{}
	lineNum := 0
	if config.SkipHeader {
		lineNum = 1
	}

	for {
		lineNum++
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Ignore empty lines silently?
			if parseErr, ok := err.(*csv.ParseError); ok && parseErr.Err == csv.ErrFieldCount {
				// Potentially log this? Or treat as ignorable error?
				log.Printf("Warning: Skipping line %d due to incorrect field count.", lineNum)
				continue
			}
			return nil, fmt.Errorf("error reading csv line %d: %w", lineNum, err)
		}

		// Basic check for empty row
		isEmpty := true
		for _, field := range record {
			if strings.TrimSpace(field) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			log.Printf("Warning: Skipping empty line %d.", lineNum)
			continue
		}

		l, err := parseRecordToLease(record, lineNum)
		if err != nil {
			// Option: Collect errors and continue? For now, fail fast.
			return nil, fmt.Errorf("error parsing line %d: %w", lineNum, err)
		}
		leases = append(leases, l)
	}

	return leases, nil
}

// ParseXLSX parses lease data from an opened excelize File object.
func ParseXLSX(f *excelize.File, config ParseConfig) ([]lease.Lease, error) {
	sheetName := f.GetSheetName(0) // Attempts to get the first sheet by index
	if sheetName == "" {
		sheetList := f.GetSheetList()
		if len(sheetList) == 0 {
			return nil, fmt.Errorf("excel file contains no sheets")
		}
		sheetName = sheetList[0] // Fallback to the first listed sheet name
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from sheet '%s': %w", sheetName, err)
	}

	startRowIndex := 0
	if config.SkipHeader {
		if len(rows) > 0 {
			// TODO: Optionally validate header row (rows[0])
			startRowIndex = 1
		} else {
			return []lease.Lease{}, nil // Only header existed, or empty file
		}
	}

	if len(rows) <= startRowIndex {
		return []lease.Lease{}, nil // No data rows
	}

	leases := []lease.Lease{}
	for i, row := range rows[startRowIndex:] {
		lineNum := i + startRowIndex + 1 // Excel rows are 1-indexed

		// Basic check for empty row (all cells are empty string)
		isEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			log.Printf("Warning: Skipping empty excel row %d.", lineNum)
			continue
		}

		// Pad row with empty strings if shorter than expected to avoid index out of bounds during parsing
		const expectedCols = 6 // ID, Start, End, Payment, Freq, Rate
		if len(row) < expectedCols {
			row = append(row, make([]string, expectedCols-len(row))...)
		}

		l, err := parseRecordToLease(row, lineNum)
		if err != nil {
			// Option: Collect errors and continue? For now, fail fast.
			return nil, fmt.Errorf("error parsing excel row %d: %w", lineNum, err)
		}
		leases = append(leases, l)
	}

	return leases, nil
}

// parseRecordToLease converts a string slice (from CSV/Excel row) into a Lease struct.
func parseRecordToLease(record []string, lineNum int) (lease.Lease, error) {
	var l lease.Lease
	var err error

	// Ensure record has enough columns (already padded in XLSX, check for CSV)
	const expectedCols = 6
	if len(record) < expectedCols {
		return l, fmt.Errorf("insufficient columns: expected %d, got %d", expectedCols, len(record))
	}

	// Trim whitespace from all fields
	for i := range record {
		record[i] = strings.TrimSpace(record[i])
	}

	// Assuming fixed column order: ID, StartDate, EndDate, PaymentAmount, PaymentFrequency, DiscountRate
	l.ID = record[0]
	if l.ID == "" {
		// Allow generating an ID later if needed, but flag it? Or require it?
		// For now, let's require a non-empty ID.
		return l, fmt.Errorf("missing required field: ID")
	}

	// Parse StartDate
	if record[1] == "" {
		return l, fmt.Errorf("missing required field: StartDate")
	}
	l.StartDate, err = time.Parse(dateLayout, record[1])
	if err != nil {
		return l, fmt.Errorf("invalid StartDate format '%s' (expected %s): %w", record[1], dateLayout, err)
	}

	// Parse EndDate
	if record[2] == "" {
		return l, fmt.Errorf("missing required field: EndDate")
	}
	l.EndDate, err = time.Parse(dateLayout, record[2])
	if err != nil {
		return l, fmt.Errorf("invalid EndDate format '%s' (expected %s): %w", record[2], dateLayout, err)
	}

	// Parse PaymentAmount
	if record[3] == "" {
		return l, fmt.Errorf("missing required field: PaymentAmount")
	}
	l.PaymentAmount, err = strconv.ParseFloat(record[3], 64)
	if err != nil {
		return l, fmt.Errorf("invalid PaymentAmount '%s': %w", record[3], err)
	}

	// Parse and Validate PaymentFrequency
	if record[4] == "" {
		return l, fmt.Errorf("missing required field: PaymentFrequency")
	}
	freq := lease.PaymentFrequency(record[4])
	switch freq {
	case lease.Monthly, lease.Quarterly, lease.Annually:
		l.PaymentFrequency = freq
	default:
		// Case-insensitive check might be good here
		lowerFreq := strings.ToLower(record[4])
		switch lease.PaymentFrequency(strings.Title(lowerFreq)) {
		case lease.Monthly, lease.Quarterly, lease.Annually:
			l.PaymentFrequency = lease.PaymentFrequency(strings.Title(lowerFreq))
		default:
			return l, fmt.Errorf("invalid PaymentFrequency '%s' (expected Monthly, Quarterly, or Annually)", record[4])
		}
	}

	// Parse DiscountRate
	if record[5] == "" {
		return l, fmt.Errorf("missing required field: DiscountRate")
	}
	l.DiscountRate, err = strconv.ParseFloat(record[5], 64)
	if err != nil {
		return l, fmt.Errorf("invalid DiscountRate '%s': %w", record[5], err)
	}
	// Handle percentage input? Assume decimal for now.
	// if l.DiscountRate > 1 { /* maybe user entered 5 for 5% */ log warning or error? }

	// Basic validation (already performed partially by parsing)
	if l.EndDate.Before(l.StartDate) {
		return l, fmt.Errorf("EndDate (%s) cannot be before StartDate (%s)", l.EndDate.Format(dateLayout), l.StartDate.Format(dateLayout))
	}
	if l.PaymentAmount <= 0 {
		return l, fmt.Errorf("PaymentAmount must be positive (got %.2f)", l.PaymentAmount)
	}
	if l.DiscountRate <= 0 {
		return l, fmt.Errorf("DiscountRate must be positive (got %.4f)", l.DiscountRate)
	}

	return l, nil
}

// Add function to parse extra payments

// parseExtraPayments parses the extra payments data from string format
func parseExtraPayments(input string) ([]lease.ExtraPayment, error) {
	if input == "" {
		return nil, nil
	}

	var extraPayments []lease.ExtraPayment

	// Format expected: "DATE1:AMOUNT1;DATE2:AMOUNT2"
	pairs := strings.Split(input, ";")

	for _, pair := range pairs {
		if pair == "" {
			continue
		}

		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid extra payment format: %s", pair)
		}

		dateStr := strings.TrimSpace(parts[0])
		amountStr := strings.TrimSpace(parts[1])

		date, err := parseDateValue(dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date in extra payment: %s", err)
		}

		amount, err := parseFloatValue(amountStr)
		if err != nil {
			return nil, fmt.Errorf("invalid amount in extra payment: %s", err)
		}

		extraPayment := lease.ExtraPayment{
			Date:   date,
			Amount: amount,
		}

		extraPayments = append(extraPayments, extraPayment)
	}

	return extraPayments, nil
}

// parseLeaseFromRow converts a row of string values into a Lease struct.
func parseLeaseFromRow(row []string, columnMap map[string]int) (lease.Lease, error) {
	l := lease.Lease{}

	// Set defaults
	l.InitialDirectCost = 0
	l.ResidualValue = 0

	if idIdx, ok := columnMap["LeaseID"]; ok && idIdx < len(row) {
		l.ID = row[idIdx]
	}

	if descIdx, ok := columnMap["Description"]; ok && descIdx < len(row) {
		l.Description = row[descIdx]
	}

	if lessorIdx, ok := columnMap["Lessor"]; ok && lessorIdx < len(row) {
		l.Lessor = row[lessorIdx]
	}

	if startDateIdx, ok := columnMap["StartDate"]; ok && startDateIdx < len(row) {
		startDate, err := parseDateValue(row[startDateIdx])
		if err != nil {
			return l, fmt.Errorf("invalid start date: %w", err)
		}
		l.StartDate = startDate
	}

	if endDateIdx, ok := columnMap["EndDate"]; ok && endDateIdx < len(row) {
		endDate, err := parseDateValue(row[endDateIdx])
		if err != nil {
			return l, fmt.Errorf("invalid end date: %w", err)
		}
		l.EndDate = endDate
	}

	if paymentAmountIdx, ok := columnMap["PaymentAmount"]; ok && paymentAmountIdx < len(row) {
		paymentAmount, err := parseFloatValue(row[paymentAmountIdx])
		if err != nil {
			return l, fmt.Errorf("invalid payment amount: %w", err)
		}
		l.PaymentAmount = paymentAmount
	}

	if paymentFreqIdx, ok := columnMap["PaymentFrequency"]; ok && paymentFreqIdx < len(row) {
		freqStr := strings.ToLower(strings.TrimSpace(row[paymentFreqIdx]))
		var freq lease.PaymentFrequency

		switch freqStr {
		case "monthly", "month", "m":
			freq = lease.Monthly
		case "quarterly", "quarter", "q":
			freq = lease.Quarterly
		case "annually", "annual", "yearly", "year", "a", "y":
			freq = lease.Annually
		default:
			return l, fmt.Errorf("invalid payment frequency: %s", freqStr)
		}

		l.PaymentFrequency = freq
	}

	if discountRateIdx, ok := columnMap["DiscountRate"]; ok && discountRateIdx < len(row) {
		rateStr := row[discountRateIdx]
		rate, err := parseFloatValue(rateStr)
		if err != nil {
			return l, fmt.Errorf("invalid discount rate: %w", err)
		}

		// Check if rate is provided as percentage and convert to decimal
		if rate > 1.0 {
			rate = rate / 100.0
		}

		l.DiscountRate = rate
	}

	// Parse initial direct cost if present
	if idcIdx, ok := columnMap["InitialDirectCost"]; ok && idcIdx < len(row) {
		if row[idcIdx] != "" {
			idc, err := parseFloatValue(row[idcIdx])
			if err != nil {
				return l, fmt.Errorf("invalid initial direct cost: %w", err)
			}
			l.InitialDirectCost = idc
		}
	}

	// Parse residual value if present
	if rvIdx, ok := columnMap["ResidualValue"]; ok && rvIdx < len(row) {
		if row[rvIdx] != "" {
			rv, err := parseFloatValue(row[rvIdx])
			if err != nil {
				return l, fmt.Errorf("invalid residual value: %w", err)
			}
			l.ResidualValue = rv
		}
	}

	// Parse extra payments if present
	if epIdx, ok := columnMap["ExtraPayments"]; ok && epIdx < len(row) {
		if row[epIdx] != "" {
			extraPayments, err := parseExtraPayments(row[epIdx])
			if err != nil {
				return l, fmt.Errorf("invalid extra payments: %w", err)
			}
			l.ExtraPayments = extraPayments
		}
	}

	return l, nil
}
