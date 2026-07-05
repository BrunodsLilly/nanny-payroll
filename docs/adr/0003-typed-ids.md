---
id: ADR-0003
title: Typed IDs
status: accepted
date: 2026-07-02
supersedes: none
superseded_by: none
code_anchors:
    - domain/payroll/employee.go
---
# Context
Employees and paychecks need a unique identifier for fetching associated data.

# Alternatives
- Use an incremental number. Rejected because the team doesn't want to leak DB internals and the team doesn't want to indicate number of employees.

# Decision
Create a struct to hold unique IDs.
