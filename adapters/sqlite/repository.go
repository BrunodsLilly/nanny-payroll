package sqlite

import (
	"database/sql"

	"nannypayroll/domain/payroll"

	_ "modernc.org/sqlite"
)

// SQLite has no native date storage class, but declaring columns DATE tells
// the modernc.org/sqlite driver to convert to/from time.Time (ADR-0011).
// Values arrive normalized to UTC midnight from the driving adapter.
const schema = `
CREATE TABLE IF NOT EXISTS rates (
	effective_from DATE PRIMARY KEY,
	amount         REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS paychecks (
	id                 TEXT PRIMARY KEY,
	period_end         DATE NOT NULL,
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
	row := toRateRow(rate)
	_, err := r.db.Exec(
		`INSERT INTO rates (effective_from, amount) VALUES (?, ?)`,
		row.effectiveFrom, row.amount,
	)
	return err
}

func (r *RateRepository) History() (payroll.RateHistory, error) {
	rows, err := r.db.Query(`SELECT effective_from, amount FROM rates`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history payroll.RateHistory
	for rows.Next() {
		var row rateRow
		if err := rows.Scan(&row.effectiveFrom, &row.amount); err != nil {
			return nil, err
		}
		history = append(history, row.toDomain())
	}
	return history, rows.Err()
}

type PaycheckRepository struct {
	db *sql.DB
}

func NewPaycheckRepository(db *sql.DB) *PaycheckRepository {
	return &PaycheckRepository{db: db}
}

func (r *PaycheckRepository) Save(p payroll.Paycheck) error {
	row := toPaycheckRow(p)
	_, err := r.db.Exec(
		`INSERT INTO paychecks
		 (id, period_end, hours, hourly_rate, gross, oasdi, medicare, federal_income_tax, sdi, state_income_tax, net_pay)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.id, row.periodEnd, row.hours, row.hourlyRate,
		row.gross, row.oasdi, row.medicare, row.federalIncomeTax,
		row.sdi, row.stateIncomeTax, row.netPay,
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
	var row paycheckRow
	if err := s.Scan(
		&row.id, &row.periodEnd, &row.hours, &row.hourlyRate,
		&row.gross, &row.oasdi, &row.medicare, &row.federalIncomeTax,
		&row.sdi, &row.stateIncomeTax, &row.netPay,
	); err != nil {
		return payroll.Paycheck{}, err
	}
	return row.toDomain(), nil
}
