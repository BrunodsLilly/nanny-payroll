---
id: ADR-0010
title: Effective-dated hourly rates stored in persistence
status: accepted
date: 2026-07-05
supersedes: none
superseded_by: none
code_anchors:
    - domain/payroll
    - ports/repository.go
    - adapters/sqlite
---
# Context
ADR-0006's premise is that hourly rates change over time while historical
paychecks must not. With `Employee` removed (ADR-0009) the rate no longer has a
home on an entity, and per ADR-0008 pay inputs must come from the repositories,
not from a request or a command-line flag typed fresh every week.

# Alternatives
- **Hard-coded constant**, like the tax rates in ADR-0005. Rejected: a raise is
  a data change, not a code change, and losing the history of past rates makes
  old paychecks unexplainable.
- **Flag on every payroll run** (`run --hours 40 --rate 20`). Rejected:
  repetitive, easy to typo, and the current rate would live nowhere.

# Decision
`HourlyRate` is a domain value: an amount plus an `EffectiveFrom` date. Rates
are appended via the CLI and never edited — the full rate history is retained.
Running payroll for a period fetches the rate effective as of the period end
and snapshots the amount (with the hours) into the immutable paycheck, so a
stored paycheck stays self-explanatory even after later raises.

# References
- ADR-0006 (immutable paychecks; the rate snapshot is what makes them so)
