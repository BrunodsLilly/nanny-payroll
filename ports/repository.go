package ports

import (
	"nannypayroll/domain/payroll"
)

type PayrollRepository interface {
	Save(paycheck payroll.Paycheck) error
	FindByID(id payroll.PaycheckID) (payroll.Paycheck, error)
	List(limit int, offset int) ([]payroll.Paycheck, error)
}

// RateRepository is append-only (ADR-0010) and storage-only (ADR-0012):
// selecting the rate in force for a date is domain logic
// (payroll.RateHistory.RateAsOf), not a query.
type RateRepository interface {
	Save(rate payroll.HourlyRate) error
	History() (payroll.RateHistory, error)
}
