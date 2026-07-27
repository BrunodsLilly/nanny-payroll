package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"nannypayroll/adapters/sqlite"
	"nannypayroll/app"
	"nannypayroll/domain/payroll"
	"nannypayroll/ports"
)

// The wiring point is the only place interface satisfaction is compiler-checked (ADR-0001, ADR-0008).
var _ ports.RateRepository = (*sqlite.RateRepository)(nil)
var _ ports.PayrollRepository = (*sqlite.PaycheckRepository)(nil)
var _ ports.PaymentRepository = (*sqlite.PaymentRepository)(nil)

const dateLayout = "2006-01-02"

const usage = `usage: nannypayroll <command> [flags]

commands:
  set-rate           record a new hourly rate, effective from a date
  run                run payroll for a pay period (persists the paycheck)
  backfill           import historical pay periods from a CSV, tracking what was actually paid
  settle-correction  record a real payment settling one or more paychecks' corrections
  payments           list recorded correction payments
  list               list stored paychecks, most recent first

run 'nannypayroll <command> -h' for command flags
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "set-rate":
		err = setRate(os.Args[2:])
	case "run":
		err = run(os.Args[2:])
	case "backfill":
		err = backfill(os.Args[2:])
	case "settle-correction":
		err = settleCorrection(os.Args[2:])
	case "payments":
		err = payments(os.Args[2:])
	case "list":
		err = list(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newService(dbPath string) (app.PayrollService, func() error, error) {
	db, err := sqlite.Open(dbPath)
	if err != nil {
		return app.PayrollService{}, nil, err
	}
	service := app.NewPayrollService(sqlite.NewRateRepository(db), sqlite.NewPaycheckRepository(db), sqlite.NewPaymentRepository(db))
	return service, db.Close, nil
}

func setRate(args []string) error {
	flags := flag.NewFlagSet("set-rate", flag.ExitOnError)
	dbPath := flags.String("db", "nannypayroll.db", "path to the SQLite database")
	amount := flags.Float64("amount", 0, "hourly rate in dollars (required)")
	from := flags.String("from", time.Now().Format(dateLayout), "effective date (YYYY-MM-DD)")
	flags.Parse(args)

	effectiveFrom, err := time.Parse(dateLayout, *from)
	if err != nil {
		return fmt.Errorf("-from must be YYYY-MM-DD: %w", err)
	}

	service, closeDB, err := newService(*dbPath)
	if err != nil {
		return err
	}
	defer closeDB()

	if err := service.SetRate(*amount, effectiveFrom); err != nil {
		return err
	}
	fmt.Printf("rate set: $%.2f/hour effective %s\n", *amount, effectiveFrom.Format(dateLayout))
	return nil
}

func run(args []string) error {
	flags := flag.NewFlagSet("run", flag.ExitOnError)
	dbPath := flags.String("db", "nannypayroll.db", "path to the SQLite database")
	hours := flags.Float64("hours", 0, "hours worked in the pay period (required)")
	periodEnd := flags.String("period-end", time.Now().Format(dateLayout), "last day of the pay period (YYYY-MM-DD)")
	flags.Parse(args)

	end, err := time.Parse(dateLayout, *periodEnd)
	if err != nil {
		return fmt.Errorf("-period-end must be YYYY-MM-DD: %w", err)
	}

	service, closeDB, err := newService(*dbPath)
	if err != nil {
		return err
	}
	defer closeDB()

	p, err := service.RunPayroll(*hours, end)
	if err != nil {
		return err
	}

	fmt.Printf("paycheck %s — period ending %s\n", p.ID, p.PeriodEnd.Format(dateLayout))
	fmt.Printf("  %.2f hours @ $%.2f/hour\n", p.Hours, p.HourlyRate)
	fmt.Printf("  gross               %10.2f\n", p.Gross)
	fmt.Printf("  OASDI               %10.2f\n", p.OASDI)
	fmt.Printf("  Medicare            %10.2f\n", p.Medicare)
	fmt.Printf("  federal income tax  %10.2f\n", p.FederalIncomeTax)
	fmt.Printf("  SDI                 %10.2f\n", p.StateDisabilityInsurance)
	fmt.Printf("  state income tax    %10.2f\n", p.StateIncomeTax)
	fmt.Printf("  net pay             %10.2f\n", p.NetPay)
	fmt.Println("  employer taxes (not withheld from pay):")
	fmt.Printf("    employer OASDI    %10.2f\n", p.EmployerOASDI)
	fmt.Printf("    employer Medicare %10.2f\n", p.EmployerMedicare)
	fmt.Printf("    FUTA              %10.2f\n", p.FUTA)
	fmt.Printf("    CA UI             %10.2f\n", p.StateUnemploymentInsurance)
	fmt.Printf("    CA ETT            %10.2f\n", p.EmploymentTrainingTax)
	fmt.Printf("  employer taxes      %10.2f\n", p.EmployerTaxes)
	fmt.Printf("  cost of employment  %10.2f\n", p.CostOfEmployment)
	return nil
}

func backfill(args []string) error {
	flags := flag.NewFlagSet("backfill", flag.ExitOnError)
	dbPath := flags.String("db", "nannypayroll.db", "path to the SQLite database")
	csvPath := flags.String("csv", "", "path to a CSV of historical pay periods (required)")
	flags.Parse(args)

	if *csvPath == "" {
		return fmt.Errorf("-csv is required")
	}
	f, err := os.Open(*csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV header: %w", err)
	}
	wantHeader := []string{"period_end", "hours", "hourly_rate", "actual_net_paid"}
	if len(header) != len(wantHeader) {
		return fmt.Errorf("expected CSV header %v, got %v", wantHeader, header)
	}
	for i := range wantHeader {
		if header[i] != wantHeader[i] {
			return fmt.Errorf("expected CSV header %v, got %v", wantHeader, header)
		}
	}

	service, closeDB, err := newService(*dbPath)
	if err != nil {
		return err
	}
	defer closeDB()

	var lastPeriodEnd time.Time
	var rowNum int
	var totalCorrection, totalEmployerTaxes float64
	for {
		rowNum++
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("row %d: %w", rowNum, err)
		}
		if len(record) != len(wantHeader) {
			return fmt.Errorf("row %d: expected %d columns, got %d", rowNum, len(wantHeader), len(record))
		}

		periodEnd, err := time.Parse(dateLayout, record[0])
		if err != nil {
			return fmt.Errorf("row %d: period_end must be YYYY-MM-DD: %w", rowNum, err)
		}
		if !lastPeriodEnd.IsZero() && !periodEnd.After(lastPeriodEnd) {
			return fmt.Errorf("row %d: period_end %s must be strictly after the previous row's %s (CSV must be in ascending order, ADR-0017)",
				rowNum, record[0], lastPeriodEnd.Format(dateLayout))
		}
		hours, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return fmt.Errorf("row %d: hours must be a number: %w", rowNum, err)
		}
		rate, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return fmt.Errorf("row %d: hourly_rate must be a number: %w", rowNum, err)
		}
		actualNetPaid, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			return fmt.Errorf("row %d: actual_net_paid must be a number: %w", rowNum, err)
		}

		p, err := service.BackfillPaycheck(hours, rate, periodEnd, actualNetPaid)
		if err != nil {
			return fmt.Errorf("row %d (period ending %s): %w", rowNum, record[0], err)
		}
		lastPeriodEnd = periodEnd

		correction := p.Correction()
		totalCorrection += correction
		totalEmployerTaxes += p.EmployerTaxes
		fmt.Printf("period ending %s: correct net %10.2f  actual paid %10.2f  correction %+8.2f  employer taxes accrued %8.2f\n",
			p.PeriodEnd.Format(dateLayout), p.NetPay, p.ActualNetPaid, correction, p.EmployerTaxes)
	}

	if lastPeriodEnd.IsZero() {
		fmt.Println("no rows imported")
		return nil
	}

	fmt.Printf("\nimported %d pay period(s)\n", rowNum-1)
	fmt.Printf("total correction owed to employee: %+.2f (positive = you owe her more; negative = she was overpaid)\n", totalCorrection)
	fmt.Printf("total employer taxes accrued but never remitted: %.2f (owed to IRS/CA EDD, not the employee — confirm filing/remittance with a CPA)\n", totalEmployerTaxes)
	return nil
}

func settleCorrection(args []string) error {
	flags := flag.NewFlagSet("settle-correction", flag.ExitOnError)
	dbPath := flags.String("db", "nannypayroll.db", "path to the SQLite database")
	paidOnStr := flags.String("paid-on", time.Now().Format(dateLayout), "date the payment actually happened (YYYY-MM-DD)")
	amount := flags.Float64("amount", 0, "dollar amount actually transferred (required; must equal the total correction owed for -paychecks)")
	paychecksStr := flags.String("paychecks", "", "comma-separated paycheck IDs this payment settles (required)")
	note := flags.String("note", "", "optional note, e.g. \"Venmo transfer\" or \"check #123\"")
	flags.Parse(args)

	paidOn, err := time.Parse(dateLayout, *paidOnStr)
	if err != nil {
		return fmt.Errorf("-paid-on must be YYYY-MM-DD: %w", err)
	}
	if *paychecksStr == "" {
		return fmt.Errorf("-paychecks is required (comma-separated paycheck IDs, as shown by 'list')")
	}
	var ids []payroll.PaycheckID
	for _, id := range strings.Split(*paychecksStr, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		ids = append(ids, payroll.PaycheckID(id))
	}

	service, closeDB, err := newService(*dbPath)
	if err != nil {
		return err
	}
	defer closeDB()

	payment, err := service.RecordCorrectionPayment(paidOn, *amount, ids, *note)
	if err != nil {
		return err
	}

	fmt.Printf("correction payment %s recorded: $%.2f paid %s, settling %d paycheck(s)\n",
		payment.ID, payment.Amount, payment.PaidOn.Format(dateLayout), len(payment.Paychecks))
	for _, id := range payment.Paychecks {
		fmt.Printf("  settled: %s\n", id)
	}
	if *note != "" {
		fmt.Printf("  note: %s\n", *note)
	}
	return nil
}

func payments(args []string) error {
	flags := flag.NewFlagSet("payments", flag.ExitOnError)
	dbPath := flags.String("db", "nannypayroll.db", "path to the SQLite database")
	limit := flags.Int("limit", 20, "maximum payments to show")
	offset := flags.Int("offset", 0, "payments to skip")
	flags.Parse(args)

	service, closeDB, err := newService(*dbPath)
	if err != nil {
		return err
	}
	defer closeDB()

	records, err := service.CorrectionPayments(*limit, *offset)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		fmt.Println("no correction payments recorded")
		return nil
	}

	for _, p := range records {
		fmt.Printf("%-18s %-12s %10.2f  %s\n", p.ID, p.PaidOn.Format(dateLayout), p.Amount, p.Note)
		for _, id := range p.Paychecks {
			fmt.Printf("  settles: %s\n", id)
		}
	}
	return nil
}

func list(args []string) error {
	flags := flag.NewFlagSet("list", flag.ExitOnError)
	dbPath := flags.String("db", "nannypayroll.db", "path to the SQLite database")
	limit := flags.Int("limit", 20, "maximum paychecks to show")
	offset := flags.Int("offset", 0, "paychecks to skip")
	flags.Parse(args)

	service, closeDB, err := newService(*dbPath)
	if err != nil {
		return err
	}
	defer closeDB()

	paychecks, err := service.Paychecks(*limit, *offset)
	if err != nil {
		return err
	}
	if len(paychecks) == 0 {
		fmt.Println("no paychecks recorded")
		return nil
	}

	settled, err := service.SettledPaycheckIDs()
	if err != nil {
		return err
	}

	fmt.Printf("%-18s %-12s %8s %8s %10s %10s %10s %8s\n", "ID", "PERIOD END", "HOURS", "RATE", "GROSS", "NET", "CORRECTION", "SETTLED")
	for _, p := range paychecks {
		settledMark := "-"
		if p.Correction() != 0 {
			if settled[p.ID] {
				settledMark = "yes"
			} else {
				settledMark = "NO"
			}
		}
		fmt.Printf("%-18s %-12s %8.2f %8.2f %10.2f %10.2f %10.2f %8s\n",
			p.ID, p.PeriodEnd.Format(dateLayout), p.Hours, p.HourlyRate, p.Gross, p.NetPay, p.Correction(), settledMark)
	}
	return nil
}
