package ports

import (
	"time"

	"nannypayroll/domain/payroll"
)

type PayrollRepository interface {
	Save(paycheck payroll.Paycheck) error
	FindByID(id payroll.PaycheckID) (payroll.Paycheck, error)
	List(limit int, offset int) ([]payroll.Paycheck, error)
}

// RateRepository is append-only (ADR-0010): rates are never edited or deleted,
// so the full rate history is retained.
type RateRepository interface {
	Save(rate payroll.HourlyRate) error
	RateAsOf(date time.Time) (payroll.HourlyRate, error)
}
