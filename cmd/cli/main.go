package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"nannypayroll/adapters/sqlite"
	"nannypayroll/app"
	"nannypayroll/ports"
)

// The wiring point is the only place interface satisfaction is compiler-checked (ADR-0001, ADR-0008).
var _ ports.RateRepository = (*sqlite.RateRepository)(nil)
var _ ports.PayrollRepository = (*sqlite.PaycheckRepository)(nil)

const dateLayout = "2006-01-02"

const usage = `usage: nannypayroll <command> [flags]

commands:
  set-rate   record a new hourly rate, effective from a date
  run        run payroll for a pay period (persists the paycheck)
  list       list stored paychecks, most recent first

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
	service := app.NewPayrollService(sqlite.NewRateRepository(db), sqlite.NewPaycheckRepository(db))
	return service, db.Close, nil
}

func setRate(args []string) error {
	flags := flag.NewFlagSet("set-rate", flag.ExitOnError)
	dbPath := flags.String("db", "nannypayroll.db", "path to the SQLite database")
	amount := flags.Float64("amount", 0, "hourly rate in dollars (required)")
	from := flags.String("from", time.Now().Format(dateLayout), "effective date (YYYY-MM-DD)")
	flags.Parse(args)

	if *amount <= 0 {
		return fmt.Errorf("-amount must be a positive dollar amount")
	}
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

	if *hours <= 0 {
		return fmt.Errorf("-hours must be positive")
	}
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

	fmt.Printf("%-18s %-12s %8s %8s %10s %10s\n", "ID", "PERIOD END", "HOURS", "RATE", "GROSS", "NET")
	for _, p := range paychecks {
		fmt.Printf("%-18s %-12s %8.2f %8.2f %10.2f %10.2f\n",
			p.ID, p.PeriodEnd.Format(dateLayout), p.Hours, p.HourlyRate, p.Gross, p.NetPay)
	}
	return nil
}
