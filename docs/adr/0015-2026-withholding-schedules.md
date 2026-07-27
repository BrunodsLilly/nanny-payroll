---
id: ADR-0015
title: 2026 withholding schedules for a single CA filer (percentage method + DE 44 Method B)
status: accepted
date: 2026-07-24
supersedes: ADR-0005
superseded_by: none
code_anchors:
    - domain/payroll
    - docs/research/2026-ca-nanny-payroll-withholding.md
---
# Context
ADR-0005 withheld tax by applying flat percentages to gross pay: 12% federal income
tax, 6% CA state income tax, and 0.9% CA SDI (plus the correct 6.2% OASDI and 1.45%
Medicare). For our one filer — single, California, paid weekly, ~$1,080/week — this is
wrong in three ways, all confirmed against IRS and CA EDD primary sources for tax year
2026 (see `docs/research/2026-ca-nanny-payroll-withholding.md`):

- **Flat federal % ignores the standard deduction.** Real withholding is graduated with a
  $0 band up to ~$310/week (IRS Pub 15-T percentage method). Flat 12% of gross over-withholds
  by ~$42/week.
- **Flat state % ignores CA's Method B calc** (low-income exemption, standard deduction,
  bracket schedule, exemption-allowance credit — CA EDD DE 44). Flat 6% over-withholds
  by ~$35/week.
- **The SDI rate was stale.** The code carried 0.9%; CA SDI is **1.3%** for 2026 with no
  wage cap (SB 951 removed the ceiling effective 2024).

Net effect: the tool reported $793.26 for the reference week when the correct figure is
~$865.91. "Simplified" is fine (ADR-0005's instinct — one filer, YAGNI on the rest of the
tax code) but "wrong by ~$73/week" is not.

# Alternatives
**Keep flat rates, just fix SDilink to 1.3%.** Rejected: the SDI rate was the smallest of the
three errors; federal and state income tax were the bulk of the discrepancy and stay wrong.

**Generalize now: filing status, pay frequency, W-4/DE-4 allowances, optional-FIT flag as
inputs.** Rejected for now. ADR-0009 (single employment) and ADR-0005's spirit both favor
hard-coding the one real filer. Generalization is a larger domain change (new inputs, new
schedules per status/frequency) and would be premature — we still model exactly one single,
weekly, California employee. When a second situation appears, a new ADR adds the inputs.

**Model employer-side taxes (employer SS/Medicare, FUTA, CA UI/ETT) too.** Deferred. The
`Paycheck` today is an employee pay stub; employer cost of employment is a separate concern
(new fields, arguably a new aggregate) and a separate decision. Captured in NOTES.md.

# Decision
Replace the flat federal/state rates with the actual 2026 withholding schedules for a single
CA filer paid weekly, keeping the calculation entirely in the domain (ADR-0012) and still
hard-coded for the one filer (ADR-0005, ADR-0009):

- **OASDI 6.2%**, **Medicare 1.45%** of gross — unchanged. The $184,500 Social Security wage
  base is not modeled: it is unreachable at this income and the domain has no year-to-date
  state (single-period calculation). Documented as an assumption, not an omission.
- **CA SDI 1.3%** of gross, no wage cap (was 0.9%).
- **Federal income tax:** IRS Pub 15-T (2026) percentage-method **weekly, single, standard
  withholding** schedule (Step 2 box unchecked). The standard deduction is baked into the
  bracket thresholds, so the schedule is applied directly to gross. Federal withholding is
  legally optional for household employees (Pub 926); we withhold it, matching prior behavior.
- **CA state income tax:** CA EDD DE 44 (2026) **Method B**, weekly, single, **one DE 4
  allowance**: exempt below the low-income threshold ($363); otherwise tax income above the
  standard deduction ($110) using the Table 23 schedule, then subtract the one-allowance
  exemption credit ($3.24), floored at zero.

The schedules live as package-level bracket tables in `domain/payroll/payroll.go`, sourced
from and cross-referenced to the research note. This remains a snapshot for one tax year and
one filer; a new ADR supersedes this one when the 2027 tables land or the filer's situation
changes.

# References
- IRS Publication 15-T (2026), Federal Income Tax Withholding Methods
- IRS Publication 926 (2026), Household Employer's Tax Guide
- CA EDD, California Withholding Schedules for 2026 — Method B (Exact Calculation)
- `docs/research/2026-ca-nanny-payroll-withholding.md` (worked examples, full citations)
- Supersedes ADR-0005.
