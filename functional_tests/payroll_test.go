package functionaltests

import (
	"path/filepath"
	"testing"
	"time"

	"nannypayroll/adapters/sqlite"
	"nannypayroll/app"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// The employer sets an hourly rate, runs payroll for the week, and the
// paycheck is persisted. A later raise changes future paychecks but never
// the stored historical one (ADR-0006, ADR-0010).
func TestWeeklyPayrollRun(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "payroll.db"))
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()
	service := app.NewPayrollService(sqlite.NewRateRepository(db), sqlite.NewPaycheckRepository(db))

	if err := service.SetRate(20.0, date(2026, time.July, 1)); err != nil {
		t.Fatalf("setting rate: %v", err)
	}

	first, err := service.RunPayroll(40, date(2026, time.July, 5))
	if err != nil {
		t.Fatalf("running payroll: %v", err)
	}
	if first.Gross != 800.0 {
		t.Errorf("expected gross 800, got %v", first.Gross)
	}
	if first.NetPay != 587.6 {
		t.Errorf("expected net pay 587.6, got %v", first.NetPay)
	}
	if first.HourlyRate != 20.0 {
		t.Errorf("expected rate snapshot 20, got %v", first.HourlyRate)
	}

	// A raise effective in August...
	if err := service.SetRate(25.0, date(2026, time.August, 1)); err != nil {
		t.Fatalf("setting raised rate: %v", err)
	}

	// ...applies to the next run...
	second, err := service.RunPayroll(40, date(2026, time.August, 7))
	if err != nil {
		t.Fatalf("running payroll after raise: %v", err)
	}
	if second.Gross != 1000.0 {
		t.Errorf("expected gross 1000 after raise, got %v", second.Gross)
	}

	// ...but the stored July paycheck reads back unchanged.
	paychecks, err := service.Paychecks(10, 0)
	if err != nil {
		t.Fatalf("listing paychecks: %v", err)
	}
	if len(paychecks) != 2 {
		t.Fatalf("expected 2 stored paychecks, got %d", len(paychecks))
	}
	july := paychecks[1] // list is most recent first
	if july.ID != first.ID {
		t.Errorf("expected oldest stored paycheck to be the July run %s, got %s", first.ID, july.ID)
	}
	if july.NetPay != 587.6 || july.HourlyRate != 20.0 {
		t.Errorf("stored July paycheck changed after raise: net %v, rate %v", july.NetPay, july.HourlyRate)
	}
}

// Running payroll before any rate exists must fail rather than assume one.
func TestRunPayroll_NoRateSet_Fails(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "payroll.db"))
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()
	service := app.NewPayrollService(sqlite.NewRateRepository(db), sqlite.NewPaycheckRepository(db))

	if _, err := service.RunPayroll(40, date(2026, time.July, 5)); err == nil {
		t.Fatal("expected error running payroll with no rate set, got nil")
	}
}
