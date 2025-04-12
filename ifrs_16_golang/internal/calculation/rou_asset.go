package calculation

import (
	"ifrs16_calculator/internal/lease"
)

// CalculateInitialRoUAsset calculates the initial value of the Right-of-Use asset.
//
// According to IFRS 16, the RoU asset initially comprises:
// 1. The amount of the initial measurement of the lease liability.
// 2. Lease payments made at or before the commencement date, less lease incentives received.
// 3. Initial direct costs incurred by the lessee.
// 4. Estimated costs of dismantling/removing the asset (Asset Retirement Obligation).
//
// This initial implementation only includes component 1.
// TODO: Enhance this function to include components 2, 3, and 4 by adding relevant fields to the lease.Lease struct.
func CalculateInitialRoUAsset(leaseLiability float64, l lease.Lease) (float64, error) {
	// Basic calculation: RoU Asset = Initial Lease Liability
	rouAsset := leaseLiability

	// TODO: Add adjustments here:
	// + Payments made at/before commencement
	// - Lease incentives received
	// + Initial direct costs
	// + Estimated dismantling costs

	return rouAsset, nil
}
