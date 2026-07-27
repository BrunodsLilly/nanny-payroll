package payroll

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// Employee-side payroll tax parameters, hard-coded for a single California filer
// paid weekly (ADR-0005, ADR-0015). Federal and CA income tax use the 2026
// withholding schedules (IRS Pub 15-T; CA EDD DE 44 Method B). See
// docs/research/2026-ca-nanny-payroll-withholding.md for sources and worked examples.
const (
	oasdiTax     = 0.062  // Social Security, employee share (Pub 926 2026)
	medicareTax  = 0.0145 // Medicare, employee share (Pub 926 2026)
	sdiTax       = 0.013  // CA SDI 2026, no wage cap (EDD; SB 951)
	overtimeRate = 1.5
)

// The OASDI $184,500 wage base (Pub 926 2026) is not modeled: it is unreachable
// at this income and the domain keeps no year-to-date state (ADR-0015).

// maxHoursPerPeriod caps hours at a full week of hours; weekly pay assumed (ADR-0005).
const maxHoursPerPeriod = 168

// CA Method B (DE 44, 2026), weekly single filer with one DE 4 allowance:
// below the low-income exemption CA withholds nothing; otherwise tax income
// above the standard deduction, then subtract the one-allowance credit.
const (
	caLowIncomeExemptionWeeklySingle    = 363.0 // Table 1
	caStandardDeductionWeeklySingle     = 110.0 // Table 3
	caExemptionCreditWeeklyOneAllowance = 3.24  // Table 4, one allowance
)

// taxBracket is one row of a marginal-rate schedule: on amounts above Over, tax
// is Base plus Rate applied to the excess over Over.
type taxBracket struct {
	Over float64
	Base float64
	Rate float64
}

// federalWeeklySingleStandard is the IRS Pub 15-T (2026) percentage-method weekly
// schedule, single / standard withholding (Step 2 box unchecked). The standard
// deduction is baked into the thresholds, so it is applied directly to gross.
var federalWeeklySingleStandard = []taxBracket{
	{Over: 0, Base: 0, Rate: 0},
	{Over: 310, Base: 0, Rate: 0.10},
	{Over: 548, Base: 23.80, Rate: 0.12},
	{Over: 1279, Base: 111.52, Rate: 0.22},
	{Over: 2342, Base: 345.38, Rate: 0.24},
}

// caWeeklySingle is the CA EDD DE 44 Method B (2026) weekly tax-rate schedule,
// Table 23 (single). It applies to CA taxable income (gross minus the standard
// deduction), before subtracting the exemption-allowance credit.
var caWeeklySingle = []taxBracket{
	{Over: 0, Base: 0, Rate: 0.011},
	{Over: 213, Base: 2.34, Rate: 0.022},
	{Over: 505, Base: 8.76, Rate: 0.044},
	{Over: 797, Base: 21.61, Rate: 0.066},
	{Over: 1107, Base: 42.07, Rate: 0.088},
	{Over: 1399, Base: 67.77, Rate: 0.1023},
	{Over: 7144, Base: 655.48, Rate: 0.1133},
	{Over: 8573, Base: 817.39, Rate: 0.1243},
	{Over: 14288, Base: 1527.76, Rate: 0.1353},
	{Over: 19231, Base: 2196.55, Rate: 0.1463},
}

// Employer-side tax parameters (ADR-0016), 2026 values for a CA new-employer
// household filer. FUTA, CA UI, and CA ETT apply only to the first $7,000 of
// annual cash wages per employee.
const (
	futaRate             = 0.006  // net of the 5.4% state credit (Pub 926 2026)
	caUIRate             = 0.034  // CA new-employer UI rate (EDD 2026)
	caETTRate            = 0.001  // CA ETT (EDD 2026)
	unemploymentWageBase = 7000.0 // shared FUTA/UI/ETT annual base per employee
)

var (
	ErrNonPositiveRate = errors.New("hourly rate must be a positive amount")
	ErrMissingDate     = errors.New("date is required")
	ErrInvalidHours    = fmt.Errorf("hours must be positive and at most %d", maxHoursPerPeriod)
	ErrNoRateInForce   = errors.New("no hourly rate in force")
)

type PaycheckID string

// HourlyRate is effective-dated (ADR-0010): rates are appended, never edited,
// and the rate in force for a pay period is the latest one effective on or
// before the period's end.
type HourlyRate struct {
	Amount        float64
	EffectiveFrom time.Time
}

// NewHourlyRate enforces the rate invariants (ADR-0012): adapters cannot
// construct an invalid rate.
func NewHourlyRate(amount float64, effectiveFrom time.Time) (HourlyRate, error) {
	if amount <= 0 {
		return HourlyRate{}, ErrNonPositiveRate
	}
	if effectiveFrom.IsZero() {
		return HourlyRate{}, fmt.Errorf("effective date: %w", ErrMissingDate)
	}
	return HourlyRate{Amount: amount, EffectiveFrom: effectiveFrom}, nil
}

// RateHistory is the append-only record of every rate ever set.
type RateHistory []HourlyRate

// RateAsOf returns the rate in force on date: the latest rate whose
// EffectiveFrom is on or before date (ADR-0010).
func (h RateHistory) RateAsOf(date time.Time) (HourlyRate, error) {
	var inForce HourlyRate
	found := false
	for _, rate := range h {
		if rate.EffectiveFrom.After(date) {
			continue
		}
		if !found || rate.EffectiveFrom.After(inForce.EffectiveFrom) {
			inForce = rate
			found = true
		}
	}
	if !found {
		return HourlyRate{}, fmt.Errorf("%w on or before %s", ErrNoRateInForce, date.Format("2006-01-02"))
	}
	return inForce, nil
}

type PayPeriod struct {
	End   time.Time
	Hours float64
	Rate  HourlyRate
}

// NewPayPeriod enforces the period invariants (ADR-0012).
func NewPayPeriod(end time.Time, hours float64, rate HourlyRate) (PayPeriod, error) {
	if end.IsZero() {
		return PayPeriod{}, fmt.Errorf("period end: %w", ErrMissingDate)
	}
	if hours <= 0 || hours > maxHoursPerPeriod {
		return PayPeriod{}, ErrInvalidHours
	}
	return PayPeriod{End: end, Hours: hours, Rate: rate}, nil
}

// Paycheck snapshots its inputs (hours, rate) so the stored record stays
// self-explanatory after later raises (ADR-0006, ADR-0010).
type Paycheck struct {
	ID                       PaycheckID
	PeriodEnd                time.Time
	Hours                    float64
	HourlyRate               float64
	Gross                    float64
	OASDI                    float64
	Medicare                 float64
	FederalIncomeTax         float64
	StateDisabilityInsurance float64
	StateIncomeTax           float64
	NetPay                   float64
	// Employer-side taxes (ADR-0016): what the employer owes on top of gross;
	// they do not reduce net pay. CostOfEmployment is gross + EmployerTaxes.
	EmployerOASDI              float64
	EmployerMedicare           float64
	FUTA                       float64
	StateUnemploymentInsurance float64
	EmploymentTrainingTax      float64
	EmployerTaxes              float64
	CostOfEmployment           float64
	// ActualNetPaid is what was really paid for this period (ADR-0017). Defaults
	// to NetPay (no correction); backfilled historical periods override it with
	// what the old spreadsheet actually paid.
	ActualNetPaid float64
}

// Correction reports what's owed given what was actually paid: positive means
// the employee is owed more, negative means she was overpaid (ADR-0017).
func (p Paycheck) Correction() float64 {
	return round(p.NetPay - p.ActualNetPaid)
}

// PaymentID identifies a CorrectionPayment (ADR-0003, ADR-0018).
type PaymentID string

var (
	ErrNoPaychecksToSettle        = errors.New("a correction payment must settle at least one paycheck")
	ErrDuplicatePaycheckInPayment = errors.New("a correction payment cannot reference the same paycheck twice")
	ErrPaymentAmountMismatch      = errors.New("payment amount does not match the total correction owed for the referenced paychecks")
)

// CorrectionPayment is an append-only record of a real transaction settling one
// or more paychecks' corrections (ADR-0018) — never a bare "paid" flag.
type CorrectionPayment struct {
	ID        PaymentID
	PaidOn    time.Time
	Amount    float64
	Paychecks []PaycheckID
	Note      string
}

// NewCorrectionPayment enforces that a correction payment is tied to what
// actually happened: it must reference at least one paycheck, never the same
// paycheck twice, and Amount must equal exactly the sum of Correction() across
// the referenced paychecks (ADR-0018).
func NewCorrectionPayment(paidOn time.Time, amount float64, paychecks []Paycheck, note string) (CorrectionPayment, error) {
	if paidOn.IsZero() {
		return CorrectionPayment{}, fmt.Errorf("paid on: %w", ErrMissingDate)
	}
	if len(paychecks) == 0 {
		return CorrectionPayment{}, ErrNoPaychecksToSettle
	}
	seen := make(map[PaycheckID]bool, len(paychecks))
	ids := make([]PaycheckID, len(paychecks))
	var owed float64
	for i, p := range paychecks {
		if seen[p.ID] {
			return CorrectionPayment{}, fmt.Errorf("%w: %s", ErrDuplicatePaycheckInPayment, p.ID)
		}
		seen[p.ID] = true
		ids[i] = p.ID
		owed = round(owed + p.Correction())
	}
	if round(amount) != owed {
		return CorrectionPayment{}, fmt.Errorf("%w: paychecks owe %.2f total, got amount %.2f", ErrPaymentAmountMismatch, owed, amount)
	}
	return CorrectionPayment{PaidOn: paidOn, Amount: round(amount), Paychecks: ids, Note: note}, nil
}

// Rounds to cents
func round(num float64) float64 {
	return math.Round(num*100) / 100
}

// tax applies a marginal-rate schedule to amount, returning the tax on the
// highest bracket whose threshold the amount exceeds.
func tax(amount float64, schedule []taxBracket) float64 {
	bracket := schedule[0]
	for _, b := range schedule {
		if amount > b.Over {
			bracket = b
		}
	}
	return bracket.Base + bracket.Rate*(amount-bracket.Over)
}

// californiaIncomeTax computes weekly CA PIT withholding for a single filer with
// one DE 4 allowance via Method B (2026): exempt below the low-income threshold;
// otherwise tax income above the standard deduction, less the allowance credit,
// floored at zero.
func californiaIncomeTax(gross float64) float64 {
	if gross <= caLowIncomeExemptionWeeklySingle {
		return 0
	}
	computed := tax(gross-caStandardDeductionWeeklySingle, caWeeklySingle) - caExemptionCreditWeeklyOneAllowance
	if computed < 0 {
		return 0
	}
	return computed
}

// Calculate computes one payroll run. priorYearWages is the gross already paid
// this calendar year, used to apply the $7,000 FUTA/UI/ETT wage base (ADR-0016).
func (p PayPeriod) Calculate(priorYearWages float64) Paycheck {
	overtimeHours := 0.0
	if p.Hours > 40 {
		overtimeHours = p.Hours - 40
	}
	gross := round(p.Rate.Amount * (p.Hours - overtimeHours + overtimeRate*overtimeHours))

	// Employee-side withholding (reduces net pay).
	oasdi := round(gross * oasdiTax)
	medicare := round(gross * medicareTax)
	stateDisabilityInsurance := round(gross * sdiTax)
	federalIncomeTax := round(tax(gross, federalWeeklySingleStandard))
	stateIncomeTax := round(californiaIncomeTax(gross))
	netPay := round(gross - oasdi - medicare - federalIncomeTax - stateDisabilityInsurance - stateIncomeTax)

	// Employer-side taxes (do not reduce net pay). FUTA/UI/ETT only apply to the
	// portion of gross still under the annual wage base.
	unemploymentTaxable := unemploymentWageBase - priorYearWages
	if unemploymentTaxable < 0 {
		unemploymentTaxable = 0
	}
	if unemploymentTaxable > gross {
		unemploymentTaxable = gross
	}
	employerOASDI := round(gross * oasdiTax)
	employerMedicare := round(gross * medicareTax)
	futa := round(unemploymentTaxable * futaRate)
	stateUI := round(unemploymentTaxable * caUIRate)
	ett := round(unemploymentTaxable * caETTRate)
	employerTaxes := round(employerOASDI + employerMedicare + futa + stateUI + ett)
	costOfEmployment := round(gross + employerTaxes)

	return Paycheck{
		PeriodEnd:                  p.End,
		Hours:                      p.Hours,
		HourlyRate:                 p.Rate.Amount,
		Gross:                      gross,
		OASDI:                      oasdi,
		Medicare:                   medicare,
		FederalIncomeTax:           federalIncomeTax,
		StateDisabilityInsurance:   stateDisabilityInsurance,
		StateIncomeTax:             stateIncomeTax,
		NetPay:                     netPay,
		ActualNetPaid:              netPay,
		EmployerOASDI:              employerOASDI,
		EmployerMedicare:           employerMedicare,
		FUTA:                       futa,
		StateUnemploymentInsurance: stateUI,
		EmploymentTrainingTax:      ett,
		EmployerTaxes:              employerTaxes,
		CostOfEmployment:           costOfEmployment,
	}
}
