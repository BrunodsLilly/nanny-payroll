package app

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"nannypayroll/domain/payroll"
	"nannypayroll/ports"
)

type PayrollService struct {
	rates     ports.RateRepository
	paychecks ports.PayrollRepository
}

func NewPayrollService(rates ports.RateRepository, paychecks ports.PayrollRepository) PayrollService {
	return PayrollService{
		rates:     rates,
		paychecks: paychecks,
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
	paycheck := period.Calculate()
	paycheck.ID = newPaycheckID()
	if err := s.paychecks.Save(paycheck); err != nil {
		return payroll.Paycheck{}, err
	}
	return paycheck, nil
}

func (s PayrollService) Paychecks(limit int, offset int) ([]payroll.Paycheck, error) {
	return s.paychecks.List(limit, offset)
}

func newPaycheckID() payroll.PaycheckID {
	b := make([]byte, 8)
	rand.Read(b)
	return payroll.PaycheckID(hex.EncodeToString(b))
}
