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
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 20.0}}.Calculate()

	if p.Gross != 800.0 {
		t.Errorf("Expected gross 800, got %v", p.Gross)
	}
	if p.OASDI != 49.60 {
		t.Errorf("expected OASDI 49.60, got %v", p.OASDI)
	}
	if p.Medicare != 11.60 {
		t.Errorf("expected Medicare 11.60, got %v", p.Medicare)
	}
	if p.FederalIncomeTax != 96.00 {
		t.Errorf("expected FIT 96.00, got %v", p.FederalIncomeTax)
	}
	if p.StateDisabilityInsurance != 7.2 {
		t.Errorf("expected SDI 7.20, got %v", p.StateDisabilityInsurance)
	}
	if p.StateIncomeTax != 48.00 {
		t.Errorf("expected StateIncomeTax 48.00, got %v", p.StateIncomeTax)
	}
	if p.NetPay != 587.6 {
		t.Errorf("Expected NetPay 587.6, got %v", p.NetPay)
	}
}

func TestCalculateNetPay_OvertimeHours_IncreasesGross(t *testing.T) {
	p := PayPeriod{Hours: 45.0, Rate: HourlyRate{Amount: 20.0}}.Calculate()
	if p.Gross != 950.00 {
		t.Errorf("expected total 950.00, got %v with overtime", p.Gross)
	}
}

func TestCalculate_SnapshotsHoursAndRate(t *testing.T) {
	p := PayPeriod{Hours: 40.0, Rate: HourlyRate{Amount: 20.0}}.Calculate()
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
