package payroll

import (
	"errors"
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestCalculateNetPay_40Hours_At20PerHour_DeductsTaxes(t *testing.T) {
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 20.0}}.Calculate(0)

	if p.Gross != 800.0 {
		t.Errorf("Expected gross 800, got %v", p.Gross)
	}
	if p.OASDI != 49.60 {
		t.Errorf("expected OASDI 49.60, got %v", p.OASDI)
	}
	if p.Medicare != 11.60 {
		t.Errorf("expected Medicare 11.60, got %v", p.Medicare)
	}
	// Pub 15-T 2026 weekly single: 23.80 + 12% of (800-548).
	if p.FederalIncomeTax != 54.04 {
		t.Errorf("expected FIT 54.04, got %v", p.FederalIncomeTax)
	}
	// CA SDI 2026: 1.3% of 800.
	if p.StateDisabilityInsurance != 10.40 {
		t.Errorf("expected SDI 10.40, got %v", p.StateDisabilityInsurance)
	}
	// CA Method B 2026 weekly single, 1 allowance: 8.76 + 4.4% of (690-505) - 3.24.
	if p.StateIncomeTax != 13.66 {
		t.Errorf("expected StateIncomeTax 13.66, got %v", p.StateIncomeTax)
	}
	if p.NetPay != 660.70 {
		t.Errorf("Expected NetPay 660.70, got %v", p.NetPay)
	}
}

// The reference week from docs/research/2026-ca-nanny-payroll-withholding.md:
// single CA filer, $27/hr x 40hr = $1,080 gross, verified against IRS Pub 15-T
// and CA EDD DE 44 Method B (2026).
func TestCalculateNetPay_ReferenceWeek_40Hours_At27PerHour(t *testing.T) {
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 27.0}}.Calculate(0)

	cases := []struct {
		name string
		got  float64
		want float64
	}{
		{"gross", p.Gross, 1080.00},
		{"OASDI", p.OASDI, 66.96},
		{"Medicare", p.Medicare, 15.66},
		{"federal income tax", p.FederalIncomeTax, 87.64},
		{"SDI", p.StateDisabilityInsurance, 14.04},
		{"state income tax", p.StateIncomeTax, 29.79},
		{"net pay", p.NetPay, 865.91},
		// Employer side, first week of the year (under the $7,000 wage base).
		{"employer OASDI", p.EmployerOASDI, 66.96},
		{"employer Medicare", p.EmployerMedicare, 15.66},
		{"FUTA", p.FUTA, 6.48},
		{"state UI", p.StateUnemploymentInsurance, 36.72},
		{"ETT", p.EmploymentTrainingTax, 1.08},
		{"employer taxes", p.EmployerTaxes, 126.90},
		{"cost of employment", p.CostOfEmployment, 1206.90},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: expected %.2f, got %v", c.name, c.want, c.got)
		}
	}
}

// Once year-to-date wages exceed the $7,000 base, FUTA/UI/ETT stop; the employer
// still owes its OASDI and Medicare share (ADR-0016).
func TestCalculate_UnemploymentTaxesStopAboveWageBase(t *testing.T) {
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 27.0}}.Calculate(7000)
	if p.FUTA != 0 || p.StateUnemploymentInsurance != 0 || p.EmploymentTrainingTax != 0 {
		t.Errorf("expected no unemployment taxes above the wage base, got FUTA %v UI %v ETT %v",
			p.FUTA, p.StateUnemploymentInsurance, p.EmploymentTrainingTax)
	}
	if p.EmployerTaxes != 82.62 { // 66.96 + 15.66
		t.Errorf("expected employer taxes 82.62 (OASDI+Medicare only), got %v", p.EmployerTaxes)
	}
}

// A period that straddles the $7,000 base is taxed only on the portion still under it.
func TestCalculate_UnemploymentTaxesProratedAtWageBase(t *testing.T) {
	// $6,600 already paid this year; this $1,080 check has $400 of room left.
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 27.0}}.Calculate(6600)
	if p.FUTA != 2.40 { // 400 * 0.006
		t.Errorf("expected FUTA 2.40 on $400 of remaining base, got %v", p.FUTA)
	}
	if p.StateUnemploymentInsurance != 13.60 { // 400 * 0.034
		t.Errorf("expected UI 13.60 on $400 of remaining base, got %v", p.StateUnemploymentInsurance)
	}
	if p.EmploymentTrainingTax != 0.40 { // 400 * 0.001
		t.Errorf("expected ETT 0.40 on $400 of remaining base, got %v", p.EmploymentTrainingTax)
	}
}

// Below the CA low-income exemption ($363/week single), CA withholds no income tax.
func TestCalculateNetPay_BelowCaLowIncomeExemption_NoStateIncomeTax(t *testing.T) {
	p := PayPeriod{Hours: 10.0, Rate: HourlyRate{Amount: 15.0}}.Calculate(0) // $150 gross
	if p.StateIncomeTax != 0 {
		t.Errorf("expected no CA income tax below the low-income exemption, got %v", p.StateIncomeTax)
	}
}

func TestCalculateNetPay_OvertimeHours_IncreasesGross(t *testing.T) {
	p := PayPeriod{Hours: 45.0, Rate: HourlyRate{Amount: 20.0}}.Calculate(0)
	if p.Gross != 950.00 {
		t.Errorf("expected total 950.00, got %v with overtime", p.Gross)
	}
}

func TestCalculate_SnapshotsHoursAndRate(t *testing.T) {
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 20.0}}.Calculate(0)
	if p.Hours != 40.0 {
		t.Errorf("expected hours 40 snapshotted, got %v", p.Hours)
	}
	if p.HourlyRate != 20.0 {
		t.Errorf("expected rate 20 snapshotted, got %v", p.HourlyRate)
	}
}

func TestRateAsOf_PicksLatestRateOnOrBeforeDate(t *testing.T) {
	history := RateHistory{
		{Amount: 20.0, EffectiveFrom: date(2026, time.July, 1)},
		{Amount: 25.0, EffectiveFrom: date(2026, time.August, 1)},
	}

	rate, err := history.RateAsOf(date(2026, time.July, 15))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate.Amount != 20.0 {
		t.Errorf("expected July rate 20 in force on July 15, got %v", rate.Amount)
	}

	rate, err = history.RateAsOf(date(2026, time.August, 1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate.Amount != 25.0 {
		t.Errorf("expected raise 25 in force on its effective date, got %v", rate.Amount)
	}
}

func TestRateAsOf_NoRateInForce_Fails(t *testing.T) {
	history := RateHistory{{Amount: 20.0, EffectiveFrom: date(2026, time.July, 1)}}
	if _, err := history.RateAsOf(date(2026, time.June, 30)); !errors.Is(err, ErrNoRateInForce) {
		t.Errorf("expected ErrNoRateInForce before first rate, got %v", err)
	}
	if _, err := (RateHistory{}).RateAsOf(date(2026, time.July, 1)); !errors.Is(err, ErrNoRateInForce) {
		t.Errorf("expected ErrNoRateInForce for empty history, got %v", err)
	}
}

func TestNewHourlyRate_RejectsInvalidInput(t *testing.T) {
	if _, err := NewHourlyRate(0, date(2026, time.July, 1)); !errors.Is(err, ErrNonPositiveRate) {
		t.Errorf("expected ErrNonPositiveRate for zero amount, got %v", err)
	}
	if _, err := NewHourlyRate(-5, date(2026, time.July, 1)); !errors.Is(err, ErrNonPositiveRate) {
		t.Errorf("expected ErrNonPositiveRate for negative amount, got %v", err)
	}
	if _, err := NewHourlyRate(20, time.Time{}); !errors.Is(err, ErrMissingDate) {
		t.Errorf("expected ErrMissingDate for zero effective date, got %v", err)
	}
}

func TestNewPayPeriod_RejectsInvalidHours(t *testing.T) {
	rate := HourlyRate{Amount: 20.0, EffectiveFrom: date(2026, time.July, 1)}
	if _, err := NewPayPeriod(date(2026, time.July, 5), 0, rate); !errors.Is(err, ErrInvalidHours) {
		t.Errorf("expected ErrInvalidHours for zero hours, got %v", err)
	}
	if _, err := NewPayPeriod(date(2026, time.July, 5), 169, rate); !errors.Is(err, ErrInvalidHours) {
		t.Errorf("expected ErrInvalidHours for more hours than a week holds, got %v", err)
	}
	if _, err := NewPayPeriod(time.Time{}, 40, rate); !errors.Is(err, ErrMissingDate) {
		t.Errorf("expected ErrMissingDate for zero period end, got %v", err)
	}
}

// A normal Calculate has no correction: ActualNetPaid defaults to the computed
// NetPay (ADR-0017).
func TestCalculate_DefaultsActualNetPaidToNetPay(t *testing.T) {
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 27.0}}.Calculate(0)
	if p.ActualNetPaid != p.NetPay {
		t.Errorf("expected ActualNetPaid to default to NetPay %v, got %v", p.NetPay, p.ActualNetPaid)
	}
	if p.Correction() != 0 {
		t.Errorf("expected no correction by default, got %v", p.Correction())
	}
}

// Correction reports what's owed when the actual paid amount differs from the
// correct one: positive means the employee is owed more (ADR-0017).
func TestPaycheck_Correction_ReportsUnderAndOverpayment(t *testing.T) {
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 27.0}}.Calculate(0)
	p.ActualNetPaid = 862.81 // what the old spreadsheet actually paid for this week
	if got, want := p.Correction(), 3.10; got != want {
		t.Errorf("expected correction %.2f (underpaid), got %v", want, got)
	}

	p.ActualNetPaid = 900.00 // overpaid relative to correct net
	if got, want := p.Correction(), -34.09; got != want {
		t.Errorf("expected correction %.2f (overpaid), got %v", want, got)
	}
}

func backfilledPaycheck(id PaycheckID, actualNetPaid float64) Paycheck {
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 27.0}}.Calculate(0)
	p.ID = id
	p.ActualNetPaid = actualNetPaid
	return p
}

// The amount of a correction payment must equal exactly what's owed across
// the paychecks it settles (ADR-0018) — no bare "paid" flag, no fudging.
func TestNewCorrectionPayment_RequiresExactAmountMatch(t *testing.T) {
	week1 := backfilledPaycheck("w1", 862.81) // owes 3.10
	week2 := backfilledPaycheck("w2", 862.81) // owes 3.10

	if _, err := NewCorrectionPayment(date(2026, time.February, 20), 6.20, []Paycheck{week1, week2}, ""); err != nil {
		t.Errorf("expected the exact-matching amount 6.20 to be accepted, got %v", err)
	}
	if _, err := NewCorrectionPayment(date(2026, time.February, 20), 6.00, []Paycheck{week1, week2}, ""); !errors.Is(err, ErrPaymentAmountMismatch) {
		t.Errorf("expected ErrPaymentAmountMismatch for a short amount, got %v", err)
	}
}

func TestNewCorrectionPayment_RejectsEmptyOrDuplicatePaychecks(t *testing.T) {
	week1 := backfilledPaycheck("w1", 862.81)

	if _, err := NewCorrectionPayment(date(2026, time.February, 20), 0, nil, ""); !errors.Is(err, ErrNoPaychecksToSettle) {
		t.Errorf("expected ErrNoPaychecksToSettle for no paychecks, got %v", err)
	}
	if _, err := NewCorrectionPayment(date(2026, time.February, 20), 6.20, []Paycheck{week1, week1}, ""); !errors.Is(err, ErrDuplicatePaycheckInPayment) {
		t.Errorf("expected ErrDuplicatePaycheckInPayment for a repeated paycheck, got %v", err)
	}
}

func TestNewCorrectionPayment_RequiresPaidOnDate(t *testing.T) {
	week1 := backfilledPaycheck("w1", 862.81)
	if _, err := NewCorrectionPayment(time.Time{}, 3.10, []Paycheck{week1}, ""); !errors.Is(err, ErrMissingDate) {
		t.Errorf("expected ErrMissingDate for zero paid-on date, got %v", err)
	}
}
