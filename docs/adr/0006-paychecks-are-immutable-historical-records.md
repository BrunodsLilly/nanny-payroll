---
id: ADR-0006
title: Paychecks are immutable historical records
status: accepted
date: 2026-07-02
supersedes: none
superseded_by: none
code_anchors:
    - domain/payroll/
    - app/payroll_service.go
    - ports/repository.go
---
# Context
NannyPayroll computes net pay from hours worked, hourly rate, and tax rules.
An employee's hourly rate can change over time. If paychecks were recalculated on demand from current data, historical pay stubs would silently change - which is incorrect.

# Decision
A paycheck is computed once at run-time (when payroll is "run" for a pay period), persisted immediately as an immutable record, and never recalculated. Retrieval always reads from storage.

The API reflects this:
- `POST /employees/{id}/paychecks` with `{"hours": N}` - runs payroll, persists and returns the paycheck
- `GET /employees/{id}/paychecks` - returns stored historical records


