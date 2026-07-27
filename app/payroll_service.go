package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"nannypayroll/domain/payroll"
	"nannypayroll/ports"
)

type PayrollService struct {
	rates     ports.RateRepository
	paychecks ports.PayrollRepository
	payments  ports.PaymentRepository
}

func NewPayrollService(rates ports.RateRepository, paychecks ports.PayrollRepository, payments ports.PaymentRepository) PayrollService {
	return PayrollService{
		rates:     rates,
		paychecks: paychecks,
		payments:  payments,
	}
}

func (s PayrollService) SetRate(amount float64, effectiveFrom time.Time) error {
	rate, err := payroll.NewHourlyRate(amount, effectiveFrom)
	if err != nil {
		return err
	}
	return s.rates.Save(rate)
}

// RunPayroll computes the paycheck for a period from the stored rate effective
// as of the period end, persists it immediately, and returns the stored record
// (ADR-0006: computed once, never recalculated).
func (s PayrollService) RunPayroll(hours float64, periodEnd time.Time) (payroll.Paycheck, error) {
	exists, err := s.paychecks.ExistsForPeriodEnd(periodEnd)
	if err != nil {
		return payroll.Paycheck{}, err
	}
	if exists {
		return payroll.Paycheck{}, fmt.Errorf("a paycheck for period ending %s already exists", periodEnd.Format("2006-01-02"))
	}
	history, err := s.rates.History()
	if err != nil {
		return payroll.Paycheck{}, err
	}
	rate, err := history.RateAsOf(periodEnd)
	if err != nil {
		return payroll.Paycheck{}, err
	}
	period, err := payroll.NewPayPeriod(periodEnd, hours, rate)
	if err != nil {
		return payroll.Paycheck{}, err
	}
	priorYearWages, err := s.paychecks.SumGrossForYear(periodEnd.Year())
	if err != nil {
		return payroll.Paycheck{}, err
	}
	paycheck := period.Calculate(priorYearWages)
	paycheck.ID = newPaycheckID()
	if err := s.paychecks.Save(paycheck); err != nil {
		return payroll.Paycheck{}, err
	}
	return paycheck, nil
}

// BackfillPaycheck records a historical pay period that actually happened before
// this tool was used: it computes the correct paycheck the same way RunPayroll
// does (same wage-base tapering via SumGrossForYear), but takes the hourly rate
// directly rather than consulting RateRepository, and overrides ActualNetPaid
// with what was really paid, so the difference can be reported (ADR-0017).
func (s PayrollService) BackfillPaycheck(hours float64, rateAmount float64, periodEnd time.Time, actualNetPaid float64) (payroll.Paycheck, error) {
	exists, err := s.paychecks.ExistsForPeriodEnd(periodEnd)
	if err != nil {
		return payroll.Paycheck{}, err
	}
	if exists {
		return payroll.Paycheck{}, fmt.Errorf("a paycheck for period ending %s already exists", periodEnd.Format("2006-01-02"))
	}
	rate, err := payroll.NewHourlyRate(rateAmount, periodEnd)
	if err != nil {
		return payroll.Paycheck{}, err
	}
	period, err := payroll.NewPayPeriod(periodEnd, hours, rate)
	if err != nil {
		return payroll.Paycheck{}, err
	}
	priorYearWages, err := s.paychecks.SumGrossForYear(periodEnd.Year())
	if err != nil {
		return payroll.Paycheck{}, err
	}
	paycheck := period.Calculate(priorYearWages)
	paycheck.ActualNetPaid = actualNetPaid
	paycheck.ID = newPaycheckID()
	if err := s.paychecks.Save(paycheck); err != nil {
		return payroll.Paycheck{}, err
	}
	return paycheck, nil
}

func (s PayrollService) Paychecks(limit int, offset int) ([]payroll.Paycheck, error) {
	return s.paychecks.List(limit, offset)
}

// RecordCorrectionPayment records a real transaction settling the corrections
// on the given paychecks (ADR-0018). It refuses to double-settle a paycheck
// already covered by an earlier payment; the amount-matches-what's-owed check
// itself lives in payroll.NewCorrectionPayment.
func (s PayrollService) RecordCorrectionPayment(paidOn time.Time, amount float64, paycheckIDs []payroll.PaycheckID, note string) (payroll.CorrectionPayment, error) {
	settled, err := s.payments.SettledPaycheckIDs()
	if err != nil {
		return payroll.CorrectionPayment{}, err
	}
	paychecks := make([]payroll.Paycheck, len(paycheckIDs))
	for i, id := range paycheckIDs {
		if settled[id] {
			return payroll.CorrectionPayment{}, fmt.Errorf("paycheck %s is already settled by an earlier payment", id)
		}
		p, err := s.paychecks.FindByID(id)
		if err != nil {
			return payroll.CorrectionPayment{}, fmt.Errorf("paycheck %s: %w", id, err)
		}
		paychecks[i] = p
	}
	payment, err := payroll.NewCorrectionPayment(paidOn, amount, paychecks, note)
	if err != nil {
		return payroll.CorrectionPayment{}, err
	}
	payment.ID = payroll.PaymentID(newID())
	if err := s.payments.Save(payment); err != nil {
		return payroll.CorrectionPayment{}, err
	}
	return payment, nil
}

func (s PayrollService) CorrectionPayments(limit int, offset int) ([]payroll.CorrectionPayment, error) {
	return s.payments.List(limit, offset)
}

// SettledPaycheckIDs reports which paychecks already have a recorded correction
// payment, for callers (e.g. the CLI's list command) that need to show which
// outstanding corrections remain.
func (s PayrollService) SettledPaycheckIDs() (map[payroll.PaycheckID]bool, error) {
	return s.payments.SettledPaycheckIDs()
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func newPaycheckID() payroll.PaycheckID {
	return payroll.PaycheckID(newID())
}
