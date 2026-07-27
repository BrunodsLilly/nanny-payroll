package functionaltests

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"nannypayroll/adapters/sqlite"
	"nannypayroll/app"
	"nannypayroll/domain/payroll"
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
	service := app.NewPayrollService(sqlite.NewRateRepository(db), sqlite.NewPaycheckRepository(db), sqlite.NewPaymentRepository(db))

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
	if first.NetPay != 660.70 {
		t.Errorf("expected net pay 660.70, got %v", first.NetPay)
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
	if july.NetPay != 660.70 || july.HourlyRate != 20.0 {
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
	service := app.NewPayrollService(sqlite.NewRateRepository(db), sqlite.NewPaycheckRepository(db), sqlite.NewPaymentRepository(db))

	if _, err := service.RunPayroll(40, date(2026, time.July, 5)); err == nil {
		t.Fatal("expected error running payroll with no rate set, got nil")
	}
}

// Backfilling historical weeks (ADR-0017) computes the correct paycheck and
// tracks what was actually paid, in chronological order so the employer wage
// base tapers correctly across the backfilled history, then refuses to
// double-import a period end already recorded.
func TestBackfillPaycheck_TracksCorrectionAndAccumulatesYTD(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "payroll.db"))
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()
	service := app.NewPayrollService(sqlite.NewRateRepository(db), sqlite.NewPaycheckRepository(db), sqlite.NewPaymentRepository(db))

	// Six historical weeks at $27/hr, 40hrs, each actually paid $862.81 (the old
	// spreadsheet's number) instead of the correct $865.91.
	periods := []time.Time{
		date(2026, time.January, 2),
		date(2026, time.January, 9),
		date(2026, time.January, 16),
		date(2026, time.January, 23),
		date(2026, time.January, 30),
		date(2026, time.February, 6),
		date(2026, time.February, 13),
	}
	var totalCorrection float64
	for i, end := range periods {
		p, err := service.BackfillPaycheck(40, 27, end, 862.81)
		if err != nil {
			t.Fatalf("backfilling week %d: %v", i, err)
		}
		if p.NetPay != 865.91 {
			t.Errorf("week %d: expected correct net 865.91, got %v", i, p.NetPay)
		}
		if p.ActualNetPaid != 862.81 {
			t.Errorf("week %d: expected actual net paid to be tracked as 862.81, got %v", i, p.ActualNetPaid)
		}
		totalCorrection += p.Correction()
	}
	// Each week the employee was underpaid by 3.10; seven weeks owed to her.
	if got, want := math.Round(totalCorrection*100)/100, 21.70; got != want {
		t.Errorf("expected total correction owed 21.70, got %v", got)
	}

	// By the 7th week ($6,480 YTD gross before it), only $520 of the $7,000
	// unemployment wage base remains, so FUTA/UI/ETT are prorated down.
	latest, err := service.Paychecks(1, 0)
	if err != nil {
		t.Fatalf("listing paychecks: %v", err)
	}
	if latest[0].FUTA == 6.48 {
		t.Errorf("expected FUTA to taper by the 7th week, still at the first-week amount")
	}

	// Re-importing an already-backfilled period must fail rather than duplicate it.
	if _, err := service.BackfillPaycheck(40, 27, periods[0], 862.81); err == nil {
		t.Fatal("expected error re-backfilling an already-recorded period, got nil")
	}
}

// A correction payment is a real transaction, not a flag (ADR-0018): it must
// exactly match what the referenced paychecks owe, and once recorded, those
// paychecks can't be settled again by a second payment.
func TestRecordCorrectionPayment_SettlesPaychecksExactlyOnce(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "payroll.db"))
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()
	service := app.NewPayrollService(sqlite.NewRateRepository(db), sqlite.NewPaycheckRepository(db), sqlite.NewPaymentRepository(db))

	week1, err := service.BackfillPaycheck(40, 27, date(2026, time.January, 2), 862.81)
	if err != nil {
		t.Fatalf("backfilling week 1: %v", err)
	}
	week2, err := service.BackfillPaycheck(40, 27, date(2026, time.January, 9), 862.81)
	if err != nil {
		t.Fatalf("backfilling week 2: %v", err)
	}

	// Wrong amount is rejected: it must equal exactly what's owed (3.10 + 3.10).
	if _, err := service.RecordCorrectionPayment(date(2026, time.February, 20), 5.00,
		[]payroll.PaycheckID{week1.ID, week2.ID}, ""); err == nil {
		t.Fatal("expected an error for a mismatched payment amount, got nil")
	}

	payment, err := service.RecordCorrectionPayment(date(2026, time.February, 20), 6.20,
		[]payroll.PaycheckID{week1.ID, week2.ID}, "Venmo transfer")
	if err != nil {
		t.Fatalf("recording correction payment: %v", err)
	}
	if payment.Amount != 6.20 {
		t.Errorf("expected recorded amount 6.20, got %v", payment.Amount)
	}

	settled, err := service.SettledPaycheckIDs()
	if err != nil {
		t.Fatalf("listing settled paychecks: %v", err)
	}
	if !settled[week1.ID] || !settled[week2.ID] {
		t.Errorf("expected both paychecks to be settled, got %v", settled)
	}

	// Settling week1 again (even in a new, correctly-amounted payment) must fail.
	week3, err := service.BackfillPaycheck(40, 27, date(2026, time.January, 16), 862.81)
	if err != nil {
		t.Fatalf("backfilling week 3: %v", err)
	}
	if _, err := service.RecordCorrectionPayment(date(2026, time.February, 21), 6.20,
		[]payroll.PaycheckID{week1.ID, week3.ID}, ""); err == nil {
		t.Fatal("expected error re-settling an already-settled paycheck, got nil")
	}

	payments, err := service.CorrectionPayments(10, 0)
	if err != nil {
		t.Fatalf("listing payments: %v", err)
	}
	if len(payments) != 1 || payments[0].ID != payment.ID {
		t.Errorf("expected exactly the one recorded payment, got %v", payments)
	}
}
