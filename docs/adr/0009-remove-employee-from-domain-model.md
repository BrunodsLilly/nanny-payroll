---
id: ADR-0009
title: Remove Employee from the domain model
status: accepted
date: 2026-07-05
supersedes: none
superseded_by: none
code_anchors:
    - domain/payroll
    - ports/repository.go
---
# Context
ADR-0005 already hard-codes tax treatment for a single CA employee. Carrying
`Employee`, `EmployeeID`, and `EmployeeRepository` for one person is speculative
generality: every query is "the employee's", every lookup returns the same row,
and every API surface grows an `{id}` segment that can only ever hold one value.

# Alternatives
**Keep Employee for future multi-employee support.** Rejected — YAGNI. If a
second employee ever appears, reintroducing the entity is a new ADR plus a data
migration, and by then the real requirements (employer entity? per-employee tax
profiles?) will be known instead of guessed.

# Decision
Delete `Employee`, `EmployeeID`, and `EmployeeRepository`. The system models a
single employment: pay periods, hourly rates, and paychecks. The typed-ID rule
(ADR-0003) still applies to what remains — `PaycheckID` — it just no longer has
an `EmployeeID` sibling.

# References
- ADR-0003 (typed IDs — principle unchanged, `EmployeeID` removed)
- ADR-0005 (single-employee simplification this decision extends)
