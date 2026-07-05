# NannyPayroll — TODO

## Phase 0: Scaffold (structure only, no logic)

- [X] `go.work` — workspace root linking all modules
- [X] `domain/go.mod` — module `nannypayroll/domain`, zero external deps
- [ ] `domain/payroll/employee.go` — placeholder structs (Employee, PayPeriod, Paycheck)
- [X] `ports/go.mod` — depends on domain only
- [ ] `ports/repository.go` — empty interfaces (EmployeeRepository, PayrollRepository)
- [X] `app/go.mod` — depends on domain + ports
- [ ] `app/payroll_service.go` — skeleton service, no business logic
- [X] `adapters/sqlite/go.mod` — depends on ports + `modernc.org/sqlite`
- [ ] `adapters/sqlite/employee_repo.go` — implements ports interfaces (skeleton)
- [X] `adapters/ui/go.mod` — depends on app + `a-h/templ`
- [ ] `adapters/ui/server.go` — skeleton HTTP server
- [X] `cmd/web/go.mod` — depends on all adapters
- [ ] `cmd/web/main.go` — wiring point; compiler enforces interface contracts here
- [ ] `AGENTS.md` — Go context file for this project (commands, conventions)
- [ ] Verify `go build ./...` passes across workspace

## Phase 1: Domain model (TDD)

- [ ] Decide on domain nouns (Employee, PayPeriod, Paycheck, HourlyRate, TaxWithholding?)
- [ ] Write first failing test in `domain/payroll/`
- [ ] Grow domain model test-first

## Phase 2: Persistence adapter (TDD)

- [ ] Resolve SQLite vs Postgres (see NOTES.md)
- [ ] Write repository integration test against chosen adapter
- [ ] Implement adapter to pass tests

## Phase 3: UI adapter (templ + htmx)

- [ ] Install `templ` CLI and wire into build
- [ ] Skeleton page: list employees
- [ ] First htmx interaction: add employee
