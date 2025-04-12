package calculation

import (
	"ifrs16_calculator/internal/lease"
	"testing"
	"time"
)

// Reusing the mustParseDate helper if needed, though not strictly required for this simple test
// func mustParseDate(layout, value string) time.Time { ... } - Assume available or copy if needed

func TestCalculateInitialRoUAsset(t *testing.T) {
	// Basic lease definition (details don't affect the current simple calculation)
	sampleLease := lease.Lease{
		ID:               "TestLease",
		StartDate:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:          time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		PaymentAmount:    1000,
		PaymentFrequency: lease.Monthly,
		DiscountRate:     0.05,
	}

	tests := []struct {
		name             string
		leaseLiability   float64
		lease            lease.Lease
		expectedRoUAsset float64
		expectError      bool
	}{
		{
			name:             "Positive Liability",
			leaseLiability:   11686.23,
			lease:            sampleLease,
			expectedRoUAsset: 11686.23,
			expectError:      false,
		},
		{
			name:             "Zero Liability",
			leaseLiability:   0.0,
			lease:            sampleLease,
			expectedRoUAsset: 0.0,
			expectError:      false,
		},
		// Although CalculateLeaseLiability should prevent negative inputs,
		// test the function's behavior directly if possible.
		// Current implementation doesn't explicitly prevent negative liability input.
		{
			name:             "Negative Liability Input (hypothetical)",
			leaseLiability:   -1000.0,
			lease:            sampleLease,
			expectedRoUAsset: -1000.0, // Currently returns the input directly
			expectError:      false,   // No error thrown for negative input currently
			// TODO: Consider adding validation if negative RoU asset is impossible
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rouAsset, err := CalculateInitialRoUAsset(tt.leaseLiability, tt.lease)

			if (err != nil) != tt.expectError {
				t.Errorf("CalculateInitialRoUAsset() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if rouAsset != tt.expectedRoUAsset { // Direct comparison ok for now
				t.Errorf("CalculateInitialRoUAsset() = %v, want %v", rouAsset, tt.expectedRoUAsset)
			}
		})
	}
}
