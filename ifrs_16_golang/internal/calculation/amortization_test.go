package calculation

import (
	"fmt"
	"ifrs16_calculator/internal/lease"
	"math"
	"testing"
	"time"
)

// --- Reusing helpers from present_value_test.go ---
// If these tests were in the same package, they wouldn't need redefining.
// For separation, we include them here.

var testDateLayoutAmort = "2006-01-02"

func mustParseDateAmort(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse date '%s' with layout '%s': %v", value, layout, err))
	}
	return t
}

// --- End Helpers ---

func TestGenerateLiabilitySchedule(t *testing.T) {
	const tolerance = 0.01 // Using a cent tolerance for schedule values

	sampleLeaseMonthly := lease.Lease{
		ID:               "L001-Sched",
		StartDate:        mustParseDateAmort(testDateLayoutAmort, "2024-01-01"),
		EndDate:          mustParseDateAmort(testDateLayoutAmort, "2024-12-31"), // 1 Year
		PaymentAmount:    1000,
		PaymentFrequency: lease.Monthly,
		DiscountRate:     0.05,
	}
	// Pre-calculated liability for the sample lease
	initialLiabilityMonthly := 11686.231696

	tests := []struct {
		name              string
		lease             lease.Lease
		initialLiability  float64
		expectedPeriods   int
		checkFirstPeriod  *AmortizationEntry // Check specific values in the first period
		checkLastPeriodCB float64            // Check final closing balance
		expectError       bool
	}{
		{
			name:             "Monthly 1 Year Schedule",
			lease:            sampleLeaseMonthly,
			initialLiability: initialLiabilityMonthly,
			expectedPeriods:  12,
			checkFirstPeriod: &AmortizationEntry{
				Period:             1,
				Date:               mustParseDateAmort(testDateLayoutAmort, "2024-02-01"),
				OpeningBalance:     roundFloat(initialLiabilityMonthly, 2),
				Payment:            1000.00,
				InterestExpense:    roundFloat(initialLiabilityMonthly*(0.05/12), 2),                                  // 48.69
				PrincipalRepayment: roundFloat(1000-(initialLiabilityMonthly*(0.05/12)), 2),                           // 951.31
				ClosingBalance:     roundFloat(initialLiabilityMonthly-(1000-(initialLiabilityMonthly*(0.05/12))), 2), // 10734.92
			},
			checkLastPeriodCB: 0.00,
			expectError:       false,
		},
		{
			name: "Zero Periods Lease",
			lease: lease.Lease{
				StartDate:        mustParseDateAmort(testDateLayoutAmort, "2024-01-01"),
				EndDate:          mustParseDateAmort(testDateLayoutAmort, "2024-01-20"),
				PaymentAmount:    1000,
				PaymentFrequency: lease.Monthly,
				DiscountRate:     0.05,
			},
			initialLiability:  0, // Liability would be 0
			expectedPeriods:   0,
			checkFirstPeriod:  nil,
			checkLastPeriodCB: 0.00,
			expectError:       false,
		},
		{
			name: "Error getting periods (invalid freq)",
			lease: lease.Lease{
				StartDate:        mustParseDateAmort(testDateLayoutAmort, "2024-01-01"),
				EndDate:          mustParseDateAmort(testDateLayoutAmort, "2024-12-31"),
				PaymentFrequency: "invalid",
				DiscountRate:     0.05,
			},
			initialLiability:  10000, // Doesn't matter, should fail earlier
			expectedPeriods:   0,
			checkFirstPeriod:  nil,
			checkLastPeriodCB: 0.00,
			expectError:       true,
		},
		// TODO: Add tests for Quarterly, Annually
		// TODO: Add tests with different start/end dates
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := GenerateLiabilitySchedule(tt.lease, tt.initialLiability)

			if (err != nil) != tt.expectError {
				t.Errorf("GenerateLiabilitySchedule() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if len(schedule) != tt.expectedPeriods {
					t.Errorf("GenerateLiabilitySchedule() schedule length = %d, want %d", len(schedule), tt.expectedPeriods)
				}

				if tt.expectedPeriods > 0 {
					// Check first period details if provided
					if tt.checkFirstPeriod != nil {
						firstEntry := schedule[0]
						expected := tt.checkFirstPeriod
						if firstEntry.Period != expected.Period {
							t.Errorf("First Entry Period = %d, want %d", firstEntry.Period, expected.Period)
						}
						if !firstEntry.Date.Equal(expected.Date) {
							t.Errorf("First Entry Date = %v, want %v", firstEntry.Date, expected.Date)
						}
						if math.Abs(firstEntry.OpeningBalance-expected.OpeningBalance) > tolerance {
							t.Errorf("First Entry OpeningBalance = %.2f, want %.2f", firstEntry.OpeningBalance, expected.OpeningBalance)
						}
						if math.Abs(firstEntry.Payment-expected.Payment) > tolerance {
							t.Errorf("First Entry Payment = %.2f, want %.2f", firstEntry.Payment, expected.Payment)
						}
						if math.Abs(firstEntry.InterestExpense-expected.InterestExpense) > tolerance {
							t.Errorf("First Entry InterestExpense = %.2f, want %.2f", firstEntry.InterestExpense, expected.InterestExpense)
						}
						if math.Abs(firstEntry.PrincipalRepayment-expected.PrincipalRepayment) > tolerance {
							t.Errorf("First Entry PrincipalRepayment = %.2f, want %.2f", firstEntry.PrincipalRepayment, expected.PrincipalRepayment)
						}
						if math.Abs(firstEntry.ClosingBalance-expected.ClosingBalance) > tolerance {
							t.Errorf("First Entry ClosingBalance = %.2f, want %.2f", firstEntry.ClosingBalance, expected.ClosingBalance)
						}
					}

					// Check last period closing balance
					lastEntry := schedule[len(schedule)-1]
					if math.Abs(lastEntry.ClosingBalance-tt.checkLastPeriodCB) > tolerance {
						t.Errorf("Last Entry ClosingBalance = %.2f, want %.2f", lastEntry.ClosingBalance, tt.checkLastPeriodCB)
					}
				}
			}
		})
	}
}

func TestGenerateRoUAssetSchedule(t *testing.T) {
	const tolerance = 0.01 // Using a cent tolerance

	sampleLeaseMonthlyRoU := lease.Lease{
		ID:               "L001-RoU-Sched",
		StartDate:        mustParseDateAmort(testDateLayoutAmort, "2024-01-01"),
		EndDate:          mustParseDateAmort(testDateLayoutAmort, "2024-12-31"), // 1 Year
		PaymentFrequency: lease.Monthly,
		DiscountRate:     0.05,
		// Other fields don't directly impact SL depreciation calc
	}
	initialRoUAssetMonthly := 11686.23 // Using rounded value for simplicity

	tests := []struct {
		name              string
		lease             lease.Lease
		initialRoUAsset   float64
		expectedPeriods   int
		checkFirstPeriod  *AmortizationEntry // Check specific values in the first period
		checkLastPeriodCB float64            // Check final closing balance
		expectError       bool
	}{
		{
			name:            "Monthly 1 Year RoU Schedule",
			lease:           sampleLeaseMonthlyRoU,
			initialRoUAsset: initialRoUAssetMonthly,
			expectedPeriods: 12,
			checkFirstPeriod: &AmortizationEntry{
				Period:         1,
				Date:           mustParseDateAmort(testDateLayoutAmort, "2024-02-01"),
				OpeningBalance: roundFloat(initialRoUAssetMonthly, 2),
				Depreciation:   roundFloat(initialRoUAssetMonthly/12, 2),                          // 973.85
				ClosingBalance: roundFloat(initialRoUAssetMonthly-(initialRoUAssetMonthly/12), 2), // 10712.38
			},
			checkLastPeriodCB: 0.00,
			expectError:       false,
		},
		{
			name:            "Zero Initial RoU Asset",
			lease:           sampleLeaseMonthlyRoU,
			initialRoUAsset: 0.0,
			expectedPeriods: 12,
			checkFirstPeriod: &AmortizationEntry{
				Period:         1,
				Date:           mustParseDateAmort(testDateLayoutAmort, "2024-02-01"),
				OpeningBalance: 0.00,
				Depreciation:   0.00,
				ClosingBalance: 0.00,
			},
			checkLastPeriodCB: 0.00,
			expectError:       false,
		},
		{
			name:              "Negative Initial RoU Asset",
			lease:             sampleLeaseMonthlyRoU,
			initialRoUAsset:   -1000.0,
			expectedPeriods:   0,
			checkFirstPeriod:  nil,
			checkLastPeriodCB: 0.00,
			expectError:       true, // Function should error on negative initial asset
		},
		{
			name: "Error getting periods (invalid freq)",
			lease: lease.Lease{
				StartDate:        mustParseDateAmort(testDateLayoutAmort, "2024-01-01"),
				EndDate:          mustParseDateAmort(testDateLayoutAmort, "2024-12-31"),
				PaymentFrequency: "invalid",
			},
			initialRoUAsset:   10000,
			expectedPeriods:   0,
			checkFirstPeriod:  nil,
			checkLastPeriodCB: 0.00,
			expectError:       true,
		},
		// TODO: Add Quarterly/Annually tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := GenerateRoUAssetSchedule(tt.lease, tt.initialRoUAsset)

			if (err != nil) != tt.expectError {
				t.Errorf("GenerateRoUAssetSchedule() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if len(schedule) != tt.expectedPeriods {
					t.Errorf("GenerateRoUAssetSchedule() schedule length = %d, want %d", len(schedule), tt.expectedPeriods)
				}

				if tt.expectedPeriods > 0 {
					// Check first period details if provided
					if tt.checkFirstPeriod != nil {
						firstEntry := schedule[0]
						expected := tt.checkFirstPeriod
						if firstEntry.Period != expected.Period {
							t.Errorf("First Entry Period = %d, want %d", firstEntry.Period, expected.Period)
						}
						if !firstEntry.Date.Equal(expected.Date) {
							t.Errorf("First Entry Date = %v, want %v", firstEntry.Date, expected.Date)
						}
						if math.Abs(firstEntry.OpeningBalance-expected.OpeningBalance) > tolerance {
							t.Errorf("First Entry OpeningBalance = %.2f, want %.2f", firstEntry.OpeningBalance, expected.OpeningBalance)
						}
						if math.Abs(firstEntry.Depreciation-expected.Depreciation) > tolerance {
							t.Errorf("First Entry Depreciation = %.2f, want %.2f", firstEntry.Depreciation, expected.Depreciation)
						}
						if math.Abs(firstEntry.ClosingBalance-expected.ClosingBalance) > tolerance {
							t.Errorf("First Entry ClosingBalance = %.2f, want %.2f", firstEntry.ClosingBalance, expected.ClosingBalance)
						}
					}

					// Check last period closing balance
					lastEntry := schedule[len(schedule)-1]
					if math.Abs(lastEntry.ClosingBalance-tt.checkLastPeriodCB) > tolerance {
						t.Errorf("Last Entry ClosingBalance = %.2f, want %.2f", lastEntry.ClosingBalance, tt.checkLastPeriodCB)
					}
				}
			}
		})
	}
}
