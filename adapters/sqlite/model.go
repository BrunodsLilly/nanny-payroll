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
	}
}

func (r paycheckRow) toDomain() payroll.Paycheck {
	return payroll.Paycheck{
		ID:                       payroll.PaycheckID(r.id),
		PeriodEnd:                r.periodEnd,
		Hours:                    r.hours,
		HourlyRate:               r.hourlyRate,
		Gross:                    r.gross,
		OASDI:                    r.oasdi,
		Medicare:                 r.medicare,
		FederalIncomeTax:         r.federalIncomeTax,
		StateDisabilityInsurance: r.sdi,
		StateIncomeTax:           r.stateIncomeTax,
		NetPay:                   r.netPay,
	}
}
