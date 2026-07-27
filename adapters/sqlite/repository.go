package sqlite

import (
	"database/sql"
	"time"

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
	net_pay            REAL NOT NULL,
	employer_oasdi     REAL NOT NULL,
	employer_medicare  REAL NOT NULL,
	futa               REAL NOT NULL,
	state_ui           REAL NOT NULL,
	ett                REAL NOT NULL,
	employer_taxes     REAL NOT NULL,
	cost_of_employment REAL NOT NULL,
	actual_net_paid    REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS correction_payments (
	id      TEXT PRIMARY KEY,
	paid_on DATE NOT NULL,
	amount  REAL NOT NULL,
	note    TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS correction_payment_paychecks (
	payment_id  TEXT NOT NULL REFERENCES correction_payments(id),
	paycheck_id TEXT NOT NULL UNIQUE REFERENCES paychecks(id),
	PRIMARY KEY (payment_id, paycheck_id)
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
		 (id, period_end, hours, hourly_rate, gross, oasdi, medicare, federal_income_tax, sdi, state_income_tax, net_pay,
		  employer_oasdi, employer_medicare, futa, state_ui, ett, employer_taxes, cost_of_employment, actual_net_paid)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.id, row.periodEnd, row.hours, row.hourlyRate,
		row.gross, row.oasdi, row.medicare, row.federalIncomeTax,
		row.sdi, row.stateIncomeTax, row.netPay,
		row.employerOASDI, row.employerMedicare, row.futa, row.stateUI, row.ett,
		row.employerTaxes, row.costOfEmployment, row.actualNetPaid,
	)
	return err
}

// ExistsForPeriodEnd reports whether a paycheck is already recorded for periodEnd (ADR-0017).
func (r *PaycheckRepository) ExistsForPeriodEnd(periodEnd time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM paychecks WHERE period_end = ?)`, periodEnd).Scan(&exists)
	return exists, err
}

// SumGrossForYear totals gross across paychecks whose period ends in year.
func (r *PaycheckRepository) SumGrossForYear(year int) (float64, error) {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	var total float64
	err := r.db.QueryRow(
		`SELECT COALESCE(SUM(gross), 0) FROM paychecks WHERE period_end >= ? AND period_end < ?`,
		start, end,
	).Scan(&total)
	return total, err
}

const paycheckColumns = `id, period_end, hours, hourly_rate, gross, oasdi, medicare, federal_income_tax, sdi, state_income_tax, net_pay,
		 employer_oasdi, employer_medicare, futa, state_ui, ett, employer_taxes, cost_of_employment, actual_net_paid`

func (r *PaycheckRepository) FindByID(id payroll.PaycheckID) (payroll.Paycheck, error) {
	row := r.db.QueryRow(
		`SELECT `+paycheckColumns+` FROM paychecks WHERE id = ?`, string(id),
	)
	return scanPaycheck(row)
}

func (r *PaycheckRepository) List(limit int, offset int) ([]payroll.Paycheck, error) {
	rows, err := r.db.Query(
		`SELECT `+paycheckColumns+` FROM paychecks ORDER BY period_end DESC LIMIT ? OFFSET ?`, limit, offset,
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
		&row.employerOASDI, &row.employerMedicare, &row.futa, &row.stateUI, &row.ett,
		&row.employerTaxes, &row.costOfEmployment, &row.actualNetPaid,
	); err != nil {
		return payroll.Paycheck{}, err
	}
	return row.toDomain(), nil
}

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Save persists the payment and its settled-paycheck references in one
// transaction, so a partial write can never leave a payment without its
// paychecks or vice versa.
func (r *PaymentRepository) Save(p payroll.CorrectionPayment) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	row := toCorrectionPaymentRow(p)
	if _, err := tx.Exec(
		`INSERT INTO correction_payments (id, paid_on, amount, note) VALUES (?, ?, ?, ?)`,
		row.id, row.paidOn, row.amount, row.note,
	); err != nil {
		tx.Rollback()
		return err
	}
	for _, paycheckID := range p.Paychecks {
		if _, err := tx.Exec(
			`INSERT INTO correction_payment_paychecks (payment_id, paycheck_id) VALUES (?, ?)`,
			row.id, string(paycheckID),
		); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// SettledPaycheckIDs reports every paycheck already referenced by a recorded payment.
func (r *PaymentRepository) SettledPaycheckIDs() (map[payroll.PaycheckID]bool, error) {
	rows, err := r.db.Query(`SELECT paycheck_id FROM correction_payment_paychecks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settled := make(map[payroll.PaycheckID]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		settled[payroll.PaycheckID(id)] = true
	}
	return settled, rows.Err()
}

func (r *PaymentRepository) List(limit int, offset int) ([]payroll.CorrectionPayment, error) {
	rows, err := r.db.Query(
		`SELECT id, paid_on, amount, note FROM correction_payments ORDER BY paid_on DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []payroll.CorrectionPayment
	for rows.Next() {
		var row correctionPaymentRow
		if err := rows.Scan(&row.id, &row.paidOn, &row.amount, &row.note); err != nil {
			return nil, err
		}
		payments = append(payments, row.toDomain(nil))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range payments {
		paycheckIDs, err := r.paycheckIDsFor(string(payments[i].ID))
		if err != nil {
			return nil, err
		}
		payments[i].Paychecks = paycheckIDs
	}
	return payments, nil
}

func (r *PaymentRepository) paycheckIDsFor(paymentID string) ([]payroll.PaycheckID, error) {
	rows, err := r.db.Query(`SELECT paycheck_id FROM correction_payment_paychecks WHERE payment_id = ?`, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []payroll.PaycheckID
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, payroll.PaycheckID(id))
	}
	return ids, rows.Err()
}
