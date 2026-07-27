// Persistence models mirroring the table columns; all schema knowledge stays
// in this adapter and domain types never carry storage concerns (ADR-0014).
package sqlite

import (
	"time"

	"nannypayroll/domain/payroll"
)

type rateRow struct {
	effectiveFrom time.Time
	amount        float64
}

func toRateRow(rate payroll.HourlyRate) rateRow {
	return rateRow{effectiveFrom: rate.EffectiveFrom, amount: rate.Amount}
}

func (r rateRow) toDomain() payroll.HourlyRate {
	return payroll.HourlyRate{Amount: r.amount, EffectiveFrom: r.effectiveFrom}
}

type paycheckRow struct {
	id               string
	periodEnd        time.Time
	hours            float64
	hourlyRate       float64
	gross            float64
	oasdi            float64
	medicare         float64
	federalIncomeTax float64
	sdi              float64
	stateIncomeTax   float64
	netPay           float64
	employerOASDI    float64
	employerMedicare float64
	futa             float64
	stateUI          float64
	ett              float64
	employerTaxes    float64
	costOfEmployment float64
	actualNetPaid    float64
}

func toPaycheckRow(p payroll.Paycheck) paycheckRow {
	return paycheckRow{
		id:               string(p.ID),
		periodEnd:        p.PeriodEnd,
		hours:            p.Hours,
		hourlyRate:       p.HourlyRate,
		gross:            p.Gross,
		oasdi:            p.OASDI,
		medicare:         p.Medicare,
		federalIncomeTax: p.FederalIncomeTax,
		sdi:              p.StateDisabilityInsurance,
		stateIncomeTax:   p.StateIncomeTax,
		netPay:           p.NetPay,
		employerOASDI:    p.EmployerOASDI,
		employerMedicare: p.EmployerMedicare,
		futa:             p.FUTA,
		stateUI:          p.StateUnemploymentInsurance,
		ett:              p.EmploymentTrainingTax,
		employerTaxes:    p.EmployerTaxes,
		costOfEmployment: p.CostOfEmployment,
		actualNetPaid:    p.ActualNetPaid,
	}
}

func (r paycheckRow) toDomain() payroll.Paycheck {
	return payroll.Paycheck{
		ID:                         payroll.PaycheckID(r.id),
		PeriodEnd:                  r.periodEnd,
		Hours:                      r.hours,
		HourlyRate:                 r.hourlyRate,
		Gross:                      r.gross,
		OASDI:                      r.oasdi,
		Medicare:                   r.medicare,
		FederalIncomeTax:           r.federalIncomeTax,
		StateDisabilityInsurance:   r.sdi,
		StateIncomeTax:             r.stateIncomeTax,
		NetPay:                     r.netPay,
		ActualNetPaid:              r.actualNetPaid,
		EmployerOASDI:              r.employerOASDI,
		EmployerMedicare:           r.employerMedicare,
		FUTA:                       r.futa,
		StateUnemploymentInsurance: r.stateUI,
		EmploymentTrainingTax:      r.ett,
		EmployerTaxes:              r.employerTaxes,
		CostOfEmployment:           r.costOfEmployment,
	}
}

type correctionPaymentRow struct {
	id     string
	paidOn time.Time
	amount float64
	note   string
}

func toCorrectionPaymentRow(p payroll.CorrectionPayment) correctionPaymentRow {
	return correctionPaymentRow{
		id:     string(p.ID),
		paidOn: p.PaidOn,
		amount: p.Amount,
		note:   p.Note,
	}
}

// toDomain builds the domain CorrectionPayment; paychecks is fetched
// separately (a join table) and supplied by the caller.
func (r correctionPaymentRow) toDomain(paychecks []payroll.PaycheckID) payroll.CorrectionPayment {
	return payroll.CorrectionPayment{
		ID:        payroll.PaymentID(r.id),
		PaidOn:    r.paidOn,
		Amount:    r.amount,
		Paychecks: paychecks,
		Note:      r.note,
	}
}
