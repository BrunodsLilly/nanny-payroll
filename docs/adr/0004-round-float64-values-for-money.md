---
id: ADR-0004
title: Round float64 values for money
status: accepted
date: 2026-07-02
supersedes: none
superseded_by: none
code_anchors:
- 
    - domain/payroll/employee.go
---
# Context
float64 precision can create numbers with more decimals than money can express.

# Decision
We must round to the thousandths place.
