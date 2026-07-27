---
id: ADR-0016
title: Model employer-side taxes and cost of employment
status: accepted
date: 2026-07-24
supersedes: none
superseded_by: none
code_anchors:
    - domain/payroll
    - app
    - ports
    - adapters/sqlite
    - docs/research/2026-ca-nanny-payroll-withholding.md
---
# Context
Until now a `Paycheck` recorded only employee-side withholding — what the nanny takes
home. But the "nanny tax" is mostly the *employer's* liability: as a household employer you
owe the employer half of Social Security and Medicare (7.65%), federal unemployment (FUTA),
and California unemployment taxes (UI and ETT). A payroll tool that omits these understates
the true cost of employment by ~$127/week for our reference filer and hides taxes the
employer must actually remit (Schedule H, CA DE 9/DE 88).

The complication is **annual wage-base caps**. Per IRS Pub 926 and CA EDD (2026), FUTA, CA
UI, and CA ETT apply only to the **first $7,000 of cash wages per employee per year**. At
$1,080/week those caps are reached around week 7; after that the employer owes $0 of these
three for the rest of the year. Applying the rates every week — the naive per-period
approach — overstates annual UI alone by roughly 8x ($1,909 vs the $238 cap). So a correct
figure requires **year-to-date wages**, which the domain's single-period `Calculate` did not
have (consistent with ADR-0015 leaving the OASDI wage base unmodeled).

# Alternatives
**Ignore the $7,000 caps; apply rates every period.** Rejected: wrong by design and by a
large margin for a full-year employee — exactly the kind of "confidently wrong number" this
project just removed for income tax (ADR-0015).

**Assume every period is early-year (always under the cap).** Rejected: correct only for the
first ~7 weeks, silently wrong thereafter.

**Track YTD as mutable domain state on an employment aggregate.** Rejected as premature —
there is no employment aggregate (ADR-0009) and paychecks are immutable (ADR-0006). YTD is
naturally derivable from the stored, immutable paychecks.

**A separate `EmploymentCost` aggregate distinct from `Paycheck`.** Reasonable, but the
paycheck already represents one payroll run's computed result; splitting it would duplicate
the period/gross and complicate the write-once flow for no present benefit.

# Decision
Add employer-side taxes to the `Paycheck` and compute them in the domain, sourcing
year-to-date wages from stored paychecks:

- **Employer OASDI 6.2%** and **Employer Medicare 1.45%** of gross — mirroring the employee
  shares; the OASDI wage base stays unmodeled (ADR-0015), unreachable at this income.
- **FUTA 0.6%** (the net rate after the standard 5.4% state credit), **CA UI 3.4%** (the
  new-employer rate), and **CA ETT 0.1%**, each applied to the portion of this period's gross
  that still falls under the **$7,000** annual wage base — i.e. `min(gross, max(0, 7000 −
  priorYearWages))`.
- `Paycheck` gains `EmployerOASDI`, `EmployerMedicare`, `FUTA`, `StateUnemploymentInsurance`,
  `EmploymentTrainingTax`, `EmployerTaxes` (their sum), and `CostOfEmployment` (gross +
  employer taxes). Net pay is unchanged.
- `PayPeriod.Calculate` takes `priorYearWages float64` — the gross already paid this calendar
  year. The **app** supplies it via a new storage-only repository method
  `PayrollRepository.SumGrossForYear(year int)`, called before the immutable paycheck is
  saved (ADR-0006). Summing over immutable records keeps YTD a pure read; it assumes payroll
  is run in chronological order within a year, which the write-once model already implies.

The employer rates are 2026 values for a CA new-employer household filer, hard-coded like the
income-tax schedules (ADR-0015); an experience-rated UI rate or a new tax year is a future
ADR. FUTA credit-reduction status (which would raise 0.6%) is assumed not to apply for 2026 —
flagged in the research note as unverified.

# References
- IRS Publication 926 (2026), Household Employer's Tax Guide (FUTA 6%/0.6% credit, $7,000 base)
- CA EDD, Rates and Withholding (2026): UI new-employer 3.4%, ETT 0.1%, $7,000 base
- `docs/research/2026-ca-nanny-payroll-withholding.md` (worked employer-side example)
- Builds on ADR-0015; respects ADR-0006 (immutable paychecks) and ADR-0012 (logic in domain).
