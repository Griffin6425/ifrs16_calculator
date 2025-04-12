package export

import (
	"ifrs16_calculator/internal/calculation"
	"testing"
	"time"
)

func TestExportToExcel(t *testing.T) {
	// Create a simple test result
	results := []LeaseResultExport{
		{
			LeaseID:          "TEST001",
			StartDate:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:          time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			PaymentAmount:    1000.00,
			PaymentFrequency: "Monthly",
			DiscountRate:     0.05,
			InitialLiability: 11681.22,
			InitialRoUAsset:  11681.22,
			LiabilitySchedule: []calculation.AmortizationEntry{
				{
					Period:             1,
					Date:               time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
					OpeningBalance:     11681.22,
					Payment:            1000.00,
					InterestExpense:    48.67,
					PrincipalRepayment: 951.33,
					ClosingBalance:     10729.89,
				},
				{
					Period:             2,
					Date:               time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
					OpeningBalance:     10729.89,
					Payment:            1000.00,
					InterestExpense:    44.71,
					PrincipalRepayment: 955.29,
					ClosingBalance:     9774.60,
				},
				// Add more entries as needed
			},
			RoUAssetSchedule: []calculation.AmortizationEntry{
				{
					Period:         1,
					Date:           time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
					OpeningBalance: 11681.22,
					Depreciation:   973.44,
					ClosingBalance: 10707.78,
				},
				{
					Period:         2,
					Date:           time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
					OpeningBalance: 10707.78,
					Depreciation:   973.44,
					ClosingBalance: 9734.34,
				},
				// Add more entries as needed
			},
		},
	}

	// Test with no results
	_, err := ExportToExcel([]LeaseResultExport{})
	if err == nil {
		t.Error("Expected error when exporting empty results, got nil")
	}

	// Test with valid results
	excelBytes, err := ExportToExcel(results)
	if err != nil {
		t.Errorf("Error exporting results: %v", err)
	}
	if len(excelBytes) == 0 {
		t.Error("Expected non-empty Excel file bytes")
	}

	// We could further validate the Excel content by reading it back with excelize,
	// but that's more complex and might be overkill for a basic test.
	// The main validation here is that we get bytes back without errors.
}
