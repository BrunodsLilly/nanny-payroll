---
id: ADR-0017
title: Backfill historical paychecks with a tracked actual-paid amount
status: accepted
date: 2026-07-25
supersedes: none
superseded_by: none
code_anchors:
    - domain/payroll
    - app/payroll_service.go
    - ports/repository.go
    - adapters/sqlite
    - cmd/cli/main.go
---
# Context
Payroll for this employment ran on the Excel spreadsheet before this tool existed —
same $/hr, same hours-worked facts, but withholding computed by the spreadsheet's flat-rate
formula (already established as wrong: ADR-0015/0016 replaced it with the real 2026
schedules). Those historical weeks were real transactions — hours were worked and specific
dollar amounts were actually paid — but the *tax withholding* on them was off by the same
mechanism ADR-0015 fixed going forward.

The ask is to (1) enter the real history into the system rather than starting the ledger
mid-stream, and (2) surface — per week and in total — the difference between what was
actually paid and what should have been paid, so a correcting payment can be issued. That
diff is two different things and must not be conflated:
- **Employee-side**: net pay actually deposited vs. the correct net pay. Owed to/from the nanny.
- **Employer-side**: employer taxes that accrued (SS/Medicare/FUTA/UI/ETT) but were never
  computed at all historically, let alone remitted. Owed to the IRS/CA EDD, not the nanny —
  a filing/remittance question (Schedule H, CA DE 9/DE 9C), not a payroll correction. This ADR
  makes the accrued amount visible; actually remitting it is outside this tool's scope.

All backfilled periods fall within tax year 2026 (confirmed), so the *same* domain schedules
(ADR-0015/0016) apply — no year-crossing tax-table problem to solve here.

# Alternatives
**Recompute historical paychecks by editing the stored NetPay to the correct value.**
Rejected: destroys the fact of what was actually paid, which is needed both to compute the
correction and as a historical record in its own right (bank statements will show the old
amount). Also conflicts with ADR-0006 — paychecks are immutable once persisted, not
"corrected in place."

**A separate `Correction`/adjustment aggregate, distinct from `Paycheck`.** More conceptually
pure, but adds a second table/port/CLI surface for what is, per period, a single number
(what was actually paid) tied 1:1 to the paycheck it corrects. Rejected for now as
premature machinery; revisit if corrections ever need their own lifecycle (e.g., "has this
correction been paid out yet?").

**Store the correction (NetPay − ActualNetPaid) instead of the raw actual amount.**
Rejected: storing the actual paid amount is the source fact; the correction is a derived
view and should stay derived (`Paycheck.Correction()`) rather than duplicated and liable to
drift if NetPay's calculation is ever revisited.

# Decision
`Paycheck` gains one field: `ActualNetPaid float64` — what really left the bank account for
that period. `PayPeriod.Calculate` continues to compute the *correct* figures exactly as
before (ADR-0015/0016) and defaults `ActualNetPaid` to the computed `NetPay` — i.e., for a
normal `run`, "correct" and "actual" are the same number, no correction. A derived method,
`Paycheck.Correction() float64` (`NetPay − ActualNetPaid`), reports what's owed: positive
means the employee is owed more, negative means she was overpaid.

A new `app.PayrollService.BackfillPaycheck(hours, rate float64, periodEnd time.Time,
actualNetPaid float64)` computes the correct paycheck the same way `RunPayroll` does —
same domain constructors, same `SumGrossForYear` wage-base lookup so employer-tax tapering
accrues correctly across the backfilled weeks in chronological order — then overrides
`ActualNetPaid` before saving. It takes the hourly rate explicitly per call rather than
consulting `RateRepository`/`RateAsOf`: backfill is establishing historical fact directly,
not looking up "what rate would apply" (that lookup is for *future* runs where the rate is
decided in advance, ADR-0010). Backfilling does not write to `RateRepository`.

`PayrollRepository` gains `ExistsForPeriodEnd(periodEnd) (bool, error)` (storage-only,
ADR-0012) so both `RunPayroll` and `BackfillPaycheck` can refuse to create a second paycheck
for a period end that's already recorded, rather than silently duplicating history or
corrupting the YTD wage-base sum.

The CLI gains a `backfill` command reading a CSV — one historical week per row, columns
`period_end,hours,hourly_rate,actual_net_paid` — and calling `BackfillPaycheck` for each row
**in ascending period-end order** (required, and validated) so YTD wages accumulate
correctly. Each row is saved immediately as it's processed (consistent with ADR-0006's
save-immediately behavior for a normal run); the batch is **not transactional** — a failure
partway through leaves earlier rows imported, and `ExistsForPeriodEnd` makes re-running the
same CSV safe (already-imported periods fail loudly with a clear error instead of double
counting). The CLI aggregates and prints, per row and in total: the correction owed to the
employee, and the accrued employer taxes that were never remitted.

# References
- Builds on ADR-0015 (2026 withholding schedules) and ADR-0016 (employer-side taxes).
- Respects ADR-0006 (paychecks immutable, computed once) and ADR-0012 (logic in domain,
  adapters/CLI don't validate or decide).
