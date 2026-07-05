---
id: ADR-0007
title: SQLite for persistence
status: accepted
date: 2026-07-05
supersedes: none
superseded_by: none
code_anchors:
    - adapters/sqlite
---
# Context
NannyPayroll needs to persist employees and paychecks (ADR-0006). The choice of
datastore was deferred in `NOTES.md` pending Phase 2 of the build.

# Alternatives
**Postgres** via `pgx/v5`. Payroll data is relational by nature (employees,
pay periods, payments), and Postgres is the better fit for multi-user or
production-scale scenarios. Rejected for now because it requires a running
server, which adds deployment and local-dev overhead this single-employer
tool doesn't need yet.

# Decision
Use SQLite via `modernc.org/sqlite` (pure Go, no CGO) as the `adapters/sqlite`
implementation of `ports.EmployeeRepository` and `ports.PayrollRepository`.
Zero infra — a single file, no server to run or manage.

The hexagonal boundary (ADR-0001) makes this a non-binding choice: swapping to
Postgres later means adding a new adapter module and changing one line in
`cmd/web/main.go`, without touching `domain`, `ports`, or `app`.

# References
- `NOTES.md`, "Deferred: Persistence Layer"
