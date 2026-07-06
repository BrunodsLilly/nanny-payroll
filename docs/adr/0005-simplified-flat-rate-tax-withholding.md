---
id: ADR-0005
title: Simplified flat-rate tax withholding
status: accepted
date: 2026-07-02
supersedes: none
superseded_by: none
code_anchors:
    - domain/payroll
---
# Context
Employers must estimate how much tax to withhold based on the predicted annual earnings of their employees.

# Decision
Tax law is complicated and most of it YAGNI; only a subset of tax parameters apply to my use-case. I have one employee so I will hard-code the application in parts for her until it needs to be changed.
