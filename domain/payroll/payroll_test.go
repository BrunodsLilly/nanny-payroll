package payroll

import "testing"

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
