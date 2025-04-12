package lease

import "time"

// PaymentFrequency defines the possible frequencies for lease payments.
type PaymentFrequency string

const (
	Monthly   PaymentFrequency = "Monthly"
	Quarterly PaymentFrequency = "Quarterly"
	Annually  PaymentFrequency = "Annually"
	// Add other frequencies as needed (e.g., SemiAnnually)
)

// ExtraPayment represents a one-time payment for a lease
type ExtraPayment struct {
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"`
}

// Lease represents the core data for an IFRS 16 lease agreement.
type Lease struct {
	ID                string           `json:"id" csv:"ID"`                             // Unique identifier for the lease
	Description       string           `json:"description" csv:"Description"`           // Description of the lease
	Lessor            string           `json:"lessor" csv:"Lessor"`                     // Name of the lessor
	StartDate         time.Time        `json:"startDate" csv:"StartDate"`               // Commencement date of the lease
	EndDate           time.Time        `json:"endDate" csv:"EndDate"`                   // End date of the lease term
	PaymentAmount     float64          `json:"paymentAmount" csv:"PaymentAmount"`       // Amount of each regular lease payment
	PaymentFrequency  PaymentFrequency `json:"paymentFrequency" csv:"PaymentFrequency"` // How often payments are made
	DiscountRate      float64          `json:"discountRate" csv:"DiscountRate"`         // Annual discount rate (e.g., IBR), expressed as a decimal (e.g., 0.05 for 5%)
	InitialDirectCost float64          `json:"initialDirectCost" csv:"InitialDirectCost"`
	ResidualValue     float64          `json:"residualValue" csv:"ResidualValue"`
	ExtraPayments     []ExtraPayment   `json:"extraPayments" csv:"ExtraPayments"`
	// TODO: Add fields for Lease Incentives, Residual Value Guarantees, Purchase Options, etc.
}
