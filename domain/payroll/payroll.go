package payroll

import (
	"errors"
	"fmt"
	"math"
	"time"
)

const oasdiTax = 0.062
const medicareTax = 0.0145
const federalIncomeTax = 0.12
const sdi = 0.009
const overtimeRate = 1.5

// simplified flat rate for a single CA filer (~41,600/year, weekly pay assumed)
// TODO: replace with annualization method when pay frequency is added to PayPeriod
const stateIncomeTax = 0.06

// maxHoursPerPeriod caps hours at a full week of hours; weekly pay assumed (ADR-0005).
const maxHoursPerPeriod = 168

var (
	ErrNonPositiveRate = errors.New("hourly rate must be a positive amount")
	ErrMissingDate     = errors.New("date is required")
	ErrInvalidHours    = fmt.Errorf("hours must be positive and at most %d", maxHoursPerPeriod)
	ErrNoRateInForce   = errors.New("no hourly rate in force")
)

type PaycheckID string

// HourlyRate is effective-dated (ADR-0010): rates are appended, never edited,
// and the rate in force for a pay period is the latest one effective on or
// before the period's end.
type HourlyRate struct {
	Amount        float64
	EffectiveFrom time.Time
}

// NewHourlyRate enforces the rate invariants (ADR-0012): adapters cannot
// construct an invalid rate.
func NewHourlyRate(amount float64, effectiveFrom time.Time) (HourlyRate, error) {
	if amount <= 0 {
		return HourlyRate{}, ErrNonPositiveRate
	}
	if effectiveFrom.IsZero() {
		return HourlyRate{}, fmt.Errorf("effective date: %w", ErrMissingDate)
	}
	return HourlyRate{Amount: amount, EffectiveFrom: effectiveFrom}, nil
}

// RateHistory is the append-only record of every rate ever set.
type RateHistory []HourlyRate

// RateAsOf returns the rate in force on date: the latest rate whose
// EffectiveFrom is on or before date (ADR-0010).
func (h RateHistory) RateAsOf(date time.Time) (HourlyRate, error) {
	var inForce HourlyRate
	found := false
	for _, rate := range h {
		if rate.EffectiveFrom.After(date) {
			continue
		}
		if !found || rate.EffectiveFrom.After(inForce.EffectiveFrom) {
			inForce = rate
			found = true
		}
	}
	if !found {
		return HourlyRate{}, fmt.Errorf("%w on or before %s", ErrNoRateInForce, date.Format("2006-01-02"))
	}
	return inForce, nil
}

type PayPeriod struct {
	End   time.Time
	Hours float64
	Rate  HourlyRate
}

// NewPayPeriod enforces the period invariants (ADR-0012).
func NewPayPeriod(end time.Time, hours float64, rate HourlyRate) (PayPeriod, error) {
	if end.IsZero() {
		return PayPeriod{}, fmt.Errorf("period end: %w", ErrMissingDate)
	}
	if hours <= 0 || hours > maxHoursPerPeriod {
		return PayPeriod{}, ErrInvalidHours
	}
	return PayPeriod{End: end, Hours: hours, Rate: rate}, nil
}

// Paycheck snapshots its inputs (hours, rate) so the stored record stays
// self-explanatory after later raises (ADR-0006, ADR-0010).
type Paycheck struct {
	ID                       PaycheckID
	PeriodEnd                time.Time
	Hours                    float64
	HourlyRate               float64
	Gross                    float64
	OASDI                    float64
	Medicare                 float64
	FederalIncomeTax         float64
	StateDisabilityInsurance float64
	StateIncomeTax           float64
	NetPay                   float64
}

// Rounds to cents
func round(num float64) float64 {
	return math.Round(num*100) / 100
}

func (p PayPeriod) Calculate() Paycheck {
	overtimeHours := 0.0
	if p.Hours > 40 {
		overtimeHours = p.Hours - 40
	}
	total := round(p.Rate.Amount * (p.Hours - overtimeHours + overtimeRate*overtimeHours))
	oasdi := round(total * oasdiTax)
	medicare := round(total * medicareTax)
	federalIncomeTax := round(total * federalIncomeTax)
	stateDisabilityInsurance := round(total * sdi)
	stateIncomeTax := round(total * stateIncomeTax)
	netPay := round(total - oasdi - medicare - federalIncomeTax - stateDisabilityInsurance - stateIncomeTax)
	return Paycheck{
		PeriodEnd:                p.End,
		Hours:                    p.Hours,
		HourlyRate:               p.Rate.Amount,
		Gross:                    total,
		OASDI:                    oasdi,
		Medicare:                 medicare,
		FederalIncomeTax:         federalIncomeTax,
		StateDisabilityInsurance: stateDisabilityInsurance,
		StateIncomeTax:           stateIncomeTax,
		NetPay:                   netPay,
	}
}
