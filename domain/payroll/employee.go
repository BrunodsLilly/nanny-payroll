package payroll

import "math"

const oasdiTax = 0.062
const medicareTax = 0.0145
const federalIncomeTax = 0.12
const sdi = 0.009
const overtimeRate = 1.5

// simplified flat rate for a single CA filer (~41,600/year, weekly pay assumed)
// TODO: replace with annualization method when pay frequency is added to PayPeriod
const stateIncomeTax = 0.06

type EmployeeID string
type PaycheckID string

type Employee struct {
	ID         EmployeeID
	HourlyRate float64
}

type PayPeriod struct {
	ID       PaycheckID
	Hours    float64
	Employee Employee
}

type Paycheck struct {
	Gross                    float64
	OASDI                    float64
	Medicare                 float64
	FederalIncomeTax         float64
	StateDisabilityInsurance float64
	StateIncomeTax           float64
	NetPay                   float64
}

// Rounds to cents
func round(num float64) float64 {
	return math.Round(num*100) / 100
}

func (p PayPeriod) Calculate() Paycheck {
	overtimeHours := 0.0
	if p.Hours > 40 {
		overtimeHours = p.Hours - 40
	}
	total := round(p.Employee.HourlyRate * (p.Hours - overtimeHours + overtimeRate*overtimeHours))
	oasdi := round(total * oasdiTax)
	medicare := round(total * medicareTax)
	federalIncomeTax := round(total * federalIncomeTax)
	stateDisabilityInsurance := round(total * sdi)
	stateIncomeTax := round(total * stateIncomeTax)
	netPay := round(total - oasdi - medicare - federalIncomeTax - stateDisabilityInsurance - stateIncomeTax)
	return Paycheck{
		Gross:                    total,
		OASDI:                    oasdi,
		Medicare:                 medicare,
		FederalIncomeTax:         federalIncomeTax,
		StateDisabilityInsurance: stateDisabilityInsurance,
		StateIncomeTax:           stateIncomeTax,
		NetPay:                   netPay,
	}
}
