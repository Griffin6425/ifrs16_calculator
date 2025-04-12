package parsing

import (
	"ifrs16_calculator/internal/lease"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

// Helper for creating a date in tests
func parseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseCSV(t *testing.T) {
	tests := []struct {
		name    string
		csv     string
		config  ParseConfig
		want    []lease.Lease
		wantErr bool
	}{
		{
			name: "Valid CSV with header",
			csv: `ID,StartDate,EndDate,PaymentAmount,PaymentFrequency,DiscountRate
L001,2023-01-01,2027-12-31,5000,Monthly,0.05
L002,2023-02-01,2026-01-31,10000,Quarterly,0.045`,
			config: ParseConfig{
				SkipHeader: true,
			},
			want: []lease.Lease{
				{
					ID:               "L001",
					StartDate:        parseDate("2023-01-01"),
					EndDate:          parseDate("2027-12-31"),
					PaymentAmount:    5000,
					PaymentFrequency: lease.Monthly,
					DiscountRate:     0.05,
				},
				{
					ID:               "L002",
					StartDate:        parseDate("2023-02-01"),
					EndDate:          parseDate("2026-01-31"),
					PaymentAmount:    10000,
					PaymentFrequency: lease.Quarterly,
					DiscountRate:     0.045,
				},
			},
			wantErr: false,
		},
		{
			name: "Valid CSV without header",
			csv: `L001,2023-01-01,2027-12-31,5000,Monthly,0.05
L002,2023-02-01,2026-01-31,10000,Quarterly,0.045`,
			config: ParseConfig{
				SkipHeader: false,
			},
			want: []lease.Lease{
				{
					ID:               "L001",
					StartDate:        parseDate("2023-01-01"),
					EndDate:          parseDate("2027-12-31"),
					PaymentAmount:    5000,
					PaymentFrequency: lease.Monthly,
					DiscountRate:     0.05,
				},
				{
					ID:               "L002",
					StartDate:        parseDate("2023-02-01"),
					EndDate:          parseDate("2026-01-31"),
					PaymentAmount:    10000,
					PaymentFrequency: lease.Quarterly,
					DiscountRate:     0.045,
				},
			},
			wantErr: false,
		},
		{
			name: "Empty CSV",
			csv:  "",
			config: ParseConfig{
				SkipHeader: false,
			},
			want:    []lease.Lease{},
			wantErr: false,
		},
		{
			name: "Empty CSV with header only",
			csv:  "ID,StartDate,EndDate,PaymentAmount,PaymentFrequency,DiscountRate",
			config: ParseConfig{
				SkipHeader: true,
			},
			want:    []lease.Lease{},
			wantErr: false,
		},
		{
			name: "Case insensitive frequency",
			csv:  "L001,2023-01-01,2027-12-31,5000,monthly,0.05",
			config: ParseConfig{
				SkipHeader: false,
			},
			want: []lease.Lease{
				{
					ID:               "L001",
					StartDate:        parseDate("2023-01-01"),
					EndDate:          parseDate("2027-12-31"),
					PaymentAmount:    5000,
					PaymentFrequency: lease.Monthly,
					DiscountRate:     0.05,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid date format",
			csv:  "L001,01/01/2023,2027-12-31,5000,Monthly,0.05",
			config: ParseConfig{
				SkipHeader: false,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Missing required field",
			csv:  "L001,2023-01-01,2027-12-31,,Monthly,0.05",
			config: ParseConfig{
				SkipHeader: false,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "End date before start date",
			csv:  "L001,2023-01-01,2022-12-31,5000,Monthly,0.05",
			config: ParseConfig{
				SkipHeader: false,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Invalid payment frequency",
			csv:  "L001,2023-01-01,2027-12-31,5000,Weekly,0.05",
			config: ParseConfig{
				SkipHeader: false,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.csv)
			got, err := ParseCSV(reader, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRecordToLease(t *testing.T) {
	tests := []struct {
		name    string
		record  []string
		want    lease.Lease
		wantErr bool
	}{
		{
			name:   "Valid record",
			record: []string{"L001", "2023-01-01", "2027-12-31", "5000", "Monthly", "0.05"},
			want: lease.Lease{
				ID:               "L001",
				StartDate:        parseDate("2023-01-01"),
				EndDate:          parseDate("2027-12-31"),
				PaymentAmount:    5000,
				PaymentFrequency: lease.Monthly,
				DiscountRate:     0.05,
			},
			wantErr: false,
		},
		{
			name:    "Insufficient columns",
			record:  []string{"L001", "2023-01-01", "2027-12-31", "5000", "Monthly"},
			want:    lease.Lease{},
			wantErr: true,
		},
		{
			name:    "Invalid payment amount",
			record:  []string{"L001", "2023-01-01", "2027-12-31", "abc", "Monthly", "0.05"},
			want:    lease.Lease{},
			wantErr: true,
		},
		{
			name:    "Negative payment amount",
			record:  []string{"L001", "2023-01-01", "2027-12-31", "-5000", "Monthly", "0.05"},
			want:    lease.Lease{},
			wantErr: true,
		},
		{
			name:    "Invalid discount rate",
			record:  []string{"L001", "2023-01-01", "2027-12-31", "5000", "Monthly", "xyz"},
			want:    lease.Lease{},
			wantErr: true,
		},
		{
			name:    "Zero discount rate",
			record:  []string{"L001", "2023-01-01", "2027-12-31", "5000", "Monthly", "0"},
			want:    lease.Lease{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRecordToLease(tt.record, 1)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Since XLSX testing would require working with actual files or mocks,
// here we'll focus on the functionality that works with in-memory data.
func TestParseXLSXHelper(t *testing.T) {
	// Create a new Excel file in memory
	f := excelize.NewFile()
	sheetName := "Sheet1"

	// Add header row
	f.SetCellValue(sheetName, "A1", "ID")
	f.SetCellValue(sheetName, "B1", "StartDate")
	f.SetCellValue(sheetName, "C1", "EndDate")
	f.SetCellValue(sheetName, "D1", "PaymentAmount")
	f.SetCellValue(sheetName, "E1", "PaymentFrequency")
	f.SetCellValue(sheetName, "F1", "DiscountRate")

	// Add data rows
	f.SetCellValue(sheetName, "A2", "L001")
	f.SetCellValue(sheetName, "B2", "2023-01-01")
	f.SetCellValue(sheetName, "C2", "2027-12-31")
	f.SetCellValue(sheetName, "D2", 5000)
	f.SetCellValue(sheetName, "E2", "Monthly")
	f.SetCellValue(sheetName, "F2", 0.05)

	f.SetCellValue(sheetName, "A3", "L002")
	f.SetCellValue(sheetName, "B3", "2023-02-01")
	f.SetCellValue(sheetName, "C3", "2026-01-31")
	f.SetCellValue(sheetName, "D3", 10000)
	f.SetCellValue(sheetName, "E3", "Quarterly")
	f.SetCellValue(sheetName, "F3", 0.045)

	// Add an empty row that should be skipped
	f.SetCellValue(sheetName, "A4", "")
	f.SetCellValue(sheetName, "B4", "")
	f.SetCellValue(sheetName, "C4", "")
	f.SetCellValue(sheetName, "D4", "")
	f.SetCellValue(sheetName, "E4", "")
	f.SetCellValue(sheetName, "F4", "")

	// Add a row with an invalid date
	f.SetCellValue(sheetName, "A5", "L003")
	f.SetCellValue(sheetName, "B5", "01/01/2023") // Invalid format
	f.SetCellValue(sheetName, "C5", "2027-12-31")
	f.SetCellValue(sheetName, "D5", 7500)
	f.SetCellValue(sheetName, "E5", "Annually")
	f.SetCellValue(sheetName, "F5", 0.06)

	// Test parsing with SkipHeader = true (standard case)
	t.Run("Parse with header", func(t *testing.T) {
		leases, err := ParseXLSX(f, ParseConfig{SkipHeader: true})
		if assert.NoError(t, err) {
			assert.Len(t, leases, 2) // Only two valid rows
			assert.Equal(t, "L001", leases[0].ID)
			assert.Equal(t, "L002", leases[1].ID)
		}
	})

	// Test parsing with SkipHeader = false (if no header is expected)
	t.Run("Parse without header", func(t *testing.T) {
		// Create a new sheet without header
		f.NewSheet("NoHeader")
		f.SetCellValue("NoHeader", "A1", "L001")
		f.SetCellValue("NoHeader", "B1", "2023-01-01")
		f.SetCellValue("NoHeader", "C1", "2027-12-31")
		f.SetCellValue("NoHeader", "D1", 5000)
		f.SetCellValue("NoHeader", "E1", "Monthly")
		f.SetCellValue("NoHeader", "F1", 0.05)

		_, err := ParseXLSX(f, ParseConfig{SkipHeader: false})
		if assert.Error(t, err) {
			// We expect an error because the first sheet (Sheet1) will be parsed,
			// and its first row is a header, not valid data
			assert.Contains(t, err.Error(), "invalid")
		}

		// Change the active sheet to NoHeader and try again
		index, err := f.GetSheetIndex("NoHeader")
		if assert.NoError(t, err) {
			f.SetActiveSheet(index)
			leases, err := ParseXLSX(f, ParseConfig{SkipHeader: false})
			if assert.NoError(t, err) {
				assert.Len(t, leases, 1)
				assert.Equal(t, "L001", leases[0].ID)
			}
		}
	})

	// Test error case with invalid data
	t.Run("Parse with invalid data", func(t *testing.T) {
		// Create a new sheet with only Sheet1 visible
		f.SetActiveSheet(0) // Set Sheet1 active
		f.DeleteSheet("NoHeader")

		// Expected behavior: when we try to parse data including row 5 (with invalid date),
		// we should get an error
		_, err := ParseXLSX(f, ParseConfig{SkipHeader: true})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}
