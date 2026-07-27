package ports

import (
	"time"

	"nannypayroll/domain/payroll"
)

type PayrollRepository interface {
	Save(paycheck payroll.Paycheck) error
	FindByID(id payroll.PaycheckID) (payroll.Paycheck, error)
	List(limit int, offset int) ([]payroll.Paycheck, error)
	// SumGrossForYear totals the gross of paychecks whose period ends in the
	// given calendar year; used to apply annual employer wage bases (ADR-0016).
	SumGrossForYear(year int) (float64, error)
	// ExistsForPeriodEnd reports whether a paycheck is already recorded for this
	// period end, so a period is never double-run or double-backfilled (ADR-0017).
	ExistsForPeriodEnd(periodEnd time.Time) (bool, error)
}

// RateRepository is append-only (ADR-0010) and storage-only (ADR-0012):
// selecting the rate in force for a date is domain logic
// (payroll.RateHistory.RateAsOf), not a query.
type RateRepository interface {
	Save(rate payroll.HourlyRate) error
	History() (payroll.RateHistory, error)
}

// PaymentRepository is append-only (ADR-0018) and storage-only (ADR-0012):
// deciding whether an amount matches what's owed is domain logic
// (payroll.NewCorrectionPayment), not a query.
type PaymentRepository interface {
	Save(payment payroll.CorrectionPayment) error
	List(limit int, offset int) ([]payroll.CorrectionPayment, error)
	// SettledPaycheckIDs reports which paychecks already have a recorded
	// correction payment, so a paycheck is never settled twice.
	SettledPaycheckIDs() (map[payroll.PaycheckID]bool, error)
}
