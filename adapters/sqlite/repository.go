package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"nannypayroll/domain/payroll"

	_ "modernc.org/sqlite"
)

// Dates are stored as YYYY-MM-DD strings, which sort lexicographically in
// date order.
const dateLayout = "2006-01-02"

const schema = `
CREATE TABLE IF NOT EXISTS rates (
	effective_from TEXT PRIMARY KEY,
	amount         REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS paychecks (
	id                 TEXT PRIMARY KEY,
	period_end         TEXT NOT NULL,
	hours              REAL NOT NULL,
	hourly_rate        REAL NOT NULL,
	gross              REAL NOT NULL,
	oasdi              REAL NOT NULL,
	medicare           REAL NOT NULL,
	federal_income_tax REAL NOT NULL,
	sdi                REAL NOT NULL,
	state_income_tax   REAL NOT NULL,
	net_pay            REAL NOT NULL
);
`

// Open opens (creating if needed) the SQLite database at path and ensures the
// schema exists.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

type RateRepository struct {
	db *sql.DB
}

func NewRateRepository(db *sql.DB) *RateRepository {
	return &RateRepository{db: db}
}

func (r *RateRepository) Save(rate payroll.HourlyRate) error {
	_, err := r.db.Exec(
		`INSERT INTO rates (effective_from, amount) VALUES (?, ?)`,
		rate.EffectiveFrom.Format(dateLayout), rate.Amount,
	)
	return err
}

func (r *RateRepository) RateAsOf(date time.Time) (payroll.HourlyRate, error) {
	row := r.db.QueryRow(
		`SELECT effective_from, amount FROM rates
		 WHERE effective_from <= ? ORDER BY effective_from DESC LIMIT 1`,
		date.Format(dateLayout),
	)
	var from string
	var rate payroll.HourlyRate
	if err := row.Scan(&from, &rate.Amount); err != nil {
		if err == sql.ErrNoRows {
			return payroll.HourlyRate{}, fmt.Errorf("no hourly rate effective on or before %s", date.Format(dateLayout))
		}
		return payroll.HourlyRate{}, err
	}
	effectiveFrom, err := time.Parse(dateLayout, from)
	if err != nil {
		return payroll.HourlyRate{}, err
	}
	rate.EffectiveFrom = effectiveFrom
	return rate, nil
}

type PaycheckRepository struct {
	db *sql.DB
}

func NewPaycheckRepository(db *sql.DB) *PaycheckRepository {
	return &PaycheckRepository{db: db}
}

func (r *PaycheckRepository) Save(p payroll.Paycheck) error {
	_, err := r.db.Exec(
		`INSERT INTO paychecks
		 (id, period_end, hours, hourly_rate, gross, oasdi, medicare, federal_income_tax, sdi, state_income_tax, net_pay)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(p.ID), p.PeriodEnd.Format(dateLayout), p.Hours, p.HourlyRate,
		p.Gross, p.OASDI, p.Medicare, p.FederalIncomeTax,
		p.StateDisabilityInsurance, p.StateIncomeTax, p.NetPay,
	)
	return err
}

func (r *PaycheckRepository) FindByID(id payroll.PaycheckID) (payroll.Paycheck, error) {
	row := r.db.QueryRow(
		`SELECT id, period_end, hours, hourly_rate, gross, oasdi, medicare, federal_income_tax, sdi, state_income_tax, net_pay
		 FROM paychecks WHERE id = ?`, string(id),
	)
	return scanPaycheck(row)
}

func (r *PaycheckRepository) List(limit int, offset int) ([]payroll.Paycheck, error) {
	rows, err := r.db.Query(
		`SELECT id, period_end, hours, hourly_rate, gross, oasdi, medicare, federal_income_tax, sdi, state_income_tax, net_pay
		 FROM paychecks ORDER BY period_end DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paychecks []payroll.Paycheck
	for rows.Next() {
		p, err := scanPaycheck(rows)
		if err != nil {
			return nil, err
		}
		paychecks = append(paychecks, p)
	}
	return paychecks, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPaycheck(s scanner) (payroll.Paycheck, error) {
	var p payroll.Paycheck
	var id, periodEnd string
	if err := s.Scan(
		&id, &periodEnd, &p.Hours, &p.HourlyRate,
		&p.Gross, &p.OASDI, &p.Medicare, &p.FederalIncomeTax,
		&p.StateDisabilityInsurance, &p.StateIncomeTax, &p.NetPay,
	); err != nil {
		return payroll.Paycheck{}, err
	}
	p.ID = payroll.PaycheckID(id)
	end, err := time.Parse(dateLayout, periodEnd)
	if err != nil {
		return payroll.Paycheck{}, err
	}
	p.PeriodEnd = end
	return p, nil
}
