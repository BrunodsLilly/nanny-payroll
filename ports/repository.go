package ports

import "nannypayroll/domain/payroll"

type EmployeeRepository interface {
	Save(employee payroll.Employee) error
	FindByID(id payroll.EmployeeID) (payroll.Employee, error)
}

type PayrollRepository interface {
	Save(paycheck payroll.Paycheck) error
	FindByID(id payroll.PaycheckID) (payroll.Paycheck, error)
	FindByEmployeeID(employeeID payroll.EmployeeID, limit int, offset int) ([]payroll.Paycheck, error)
}
