package calculation

import (
	"errors"
	"fmt"
	"ifrs16_calculator/internal/lease"
	"math"
)

// CalculateLeaseLiability calculates the initial lease liability based on IFRS 16.
// It computes the present value of the lease payments.
// Assumes payments are made at the end of each period.
func CalculateLeaseLiability(l lease.Lease) (float64, error) {
	if l.PaymentAmount <= 0 {
		return 0, errors.New("payment amount must be positive")
	}
	if l.DiscountRate <= 0 {
		return 0, errors.New("discount rate must be positive")
	}
	if l.StartDate.IsZero() || l.EndDate.IsZero() || l.EndDate.Before(l.StartDate) {
		return 0, errors.New("invalid start or end date")
	}

	periods, periodicRate, err := getPeriodsAndRate(l)
	if err != nil {
		return 0, fmt.Errorf("failed to get periods and rate: %w", err)
	}

	if periods == 0 {
		// A lease with zero calculated periods should have zero liability.
		// Return 0.0 and no error.
		return 0.0, nil
	}

	// Enhanced PV calculation for more precision
	// This uses a standard formula for calculating present value of regular payments
	presentValue := 0.0

	// Calculate the annuity formula (for payments at the end of each period)
	// PV = PMT * [1 - (1+r)^-n] / r
	if periodicRate > 0 {
		// Standard formula for payments at the end of the period
		presentValue = l.PaymentAmount * ((1 - math.Pow(1+periodicRate, float64(-periods))) / periodicRate)
	} else {
		// Fallback to simple sum if rate is effectively zero (though we check for this earlier)
		presentValue = l.PaymentAmount * float64(periods)
	}

	// Round to minimize floating point imprecision (to 6 decimal places)
	presentValue = roundToDecimalPlaces(presentValue, 6)

	// Change to 2 decimal places as financial calculations typically use currency precision
	presentValue = roundToDecimalPlaces(presentValue, 2)

	return presentValue, nil
}

// roundToDecimalPlaces rounds a float64 to the specified number of decimal places
func roundToDecimalPlaces(value float64, places int) float64 {
	multiplier := math.Pow10(places)
	return math.Round(value*multiplier) / multiplier
}

// getPeriodsAndRate calculates the number of payment periods and the periodic discount rate.
func getPeriodsAndRate(l lease.Lease) (int, float64, error) {
	var periodsPerYear int
	var monthsPerPeriod int // Added for date stepping

	switch l.PaymentFrequency {
	case lease.Monthly:
		periodsPerYear = 12
		monthsPerPeriod = 1
	case lease.Quarterly:
		periodsPerYear = 4
		monthsPerPeriod = 3
	case lease.Annually:
		periodsPerYear = 1
		monthsPerPeriod = 12
	default:
		return 0, 0, fmt.Errorf("unsupported payment frequency: %s", l.PaymentFrequency)
	}

	if l.DiscountRate <= 0 {
		return 0, 0, fmt.Errorf("discount rate must be positive")
	}
	periodicRate := l.DiscountRate / float64(periodsPerYear)

	// --- Accurate Period Calculation (Attempt 13) ---

	// Explicit check for zero duration or leases shorter than one full period.
	// Calculate the end date of the very first potential period.
	firstPeriodEndDate := l.StartDate.AddDate(0, monthsPerPeriod, 0)

	// If the lease ends strictly *before* the first period would have ended,
	// then zero payment periods occur.
	if l.EndDate.Before(firstPeriodEndDate) {
		// Also explicitly handle zero-duration lease (start==end)
		if l.EndDate.Equal(l.StartDate) {
			return 0, periodicRate, nil
		}
		// Otherwise it's just shorter than one period.
		return 0, periodicRate, nil
	}

	// If the lease term is at least one period long, count the intervals.
	periodCount := 0
	current := l.StartDate
	// Loop while the start of the period we are considering is strictly before the EndDate.
	// This counts the number of intervals starting before the EndDate.
	for current.Before(l.EndDate) {
		periodCount++
		// Move to the start of the next interval.
		current = current.AddDate(0, monthsPerPeriod, 0)

		// Safety break
		if periodCount > 12000 {
			return 0, 0, fmt.Errorf("period calculation safety limit exceeded (12000)")
		}
	}
	// --- End Accurate Period Calculation ---

	return periodCount, periodicRate, nil
}

// addPeriods is a helper placeholder - needs proper implementation
// --> This specific helper might not be needed anymore if date stepping is done within schedule funcs.
// func addPeriods(startDate time.Time, freq lease.PaymentFrequency, numPeriods int) time.Time {
//     // Placeholder - requires careful implementation based on frequency
//     // e.g., for monthly, add numPeriods months; for quarterly, add 3*numPeriods months, etc.
//     // Need to handle date arithmetic correctly (e.g., month ends).
//     return startDate // Replace with actual calculation
// }
