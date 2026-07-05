package app

import (
	"nannypayroll/domain/payroll"
	"nannypayroll/ports"
)

type PayrollService struct {
	employeeRepository ports.EmployeeRepository
	payrollRepository  ports.PayrollRepository
}

func NewPayrollService(employeeRepository ports.EmployeeRepository, payrollRepository ports.PayrollRepository) PayrollService {
	return PayrollService{
		employeeRepository,
		payrollRepository,
	}
}

func (s PayrollService) CalculatePaycheck(hours float64, rate float64) payroll.Paycheck {
	return payroll.PayPeriod{
		Hours:    hours,
		Employee: payroll.Employee{HourlyRate: rate},
	}.Calculate()
}
