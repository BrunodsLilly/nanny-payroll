---
id: ADR-0018
title: Correction payments are transaction records, not a paid flag
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
ADR-0017 made `Paycheck.Correction()` a derived number: what's owed to the employee when
`ActualNetPaid` differs from the correct `NetPay`. The next question is how to record that
the correction was actually paid. A bare boolean (`Settled bool`) on `Paycheck` would answer
"has this been dealt with?" but throws away *how* — no date, no amount, no way to tell a
single lump-sum payment covering several backfilled weeks from six separate ones, and no way
to catch a typo (paid the wrong amount, or paid a period twice).

This is the same category of decision as ADR-0017 itself: the ledger should record what
*actually happened* — a real transaction — not a status flag laid on top of history.

# Alternatives
**`Settled bool` (+ optional `SettledOn time.Time`) on `Paycheck`.** Rejected as the user
requested: no amount, no way to represent one payment covering several paychecks (the
realistic case — you don't Venmo her seven times for a $3.10 correction each), and it would
make `Paycheck` mutable after the fact, undermining ADR-0006.

**Store the correction as a field that gets zeroed out once paid.** Rejected: destroys the
historical fact that a correction existed and was later paid; an append-only payment record
is strictly more informative and still lets "outstanding correction" be computed as before
(`Correction() != 0 AND paycheck not referenced by any payment`).

**Allow partial settlement of a single paycheck's correction across multiple payments.**
Rejected for now as unneeded complexity for a personal-scale ledger: a correction is either
unsettled or settled in full by exactly one payment. If a correction is ever partially paid
in practice, record it as its own (smaller) correction via a future paycheck, not by
fragmenting this one.

# Decision
Add a new domain type, `CorrectionPayment` — an **append-only transaction record**, mirroring
how `RateHistory` (ADR-0010) and `Paycheck` (ADR-0006) are themselves append-only:

```go
type CorrectionPayment struct {
    ID        PaymentID
    PaidOn    time.Time
    Amount    float64
    Paychecks []PaycheckID // which corrections this transaction settles
    Note      string       // freeform, e.g. "Venmo transfer", "check #123"
}
```

`NewCorrectionPayment(paidOn, amount, paychecks []Paycheck, note)` enforces the invariants
that make this a real transaction record rather than a wish: at least one paycheck is
referenced, no paycheck is referenced twice, and — the key check — **`amount` must equal
exactly the sum of `Correction()` across the referenced paychecks**. This is the same
"adapters relay, domain decides" discipline as everywhere else (ADR-0012): if the numbers
don't add up, the CLI just relays a domain error back to the user rather than silently
accepting a mismatched transaction.

The app layer (`PayrollService.RecordCorrectionPayment`) additionally refuses to record a
payment against a paycheck that's **already settled** by an earlier payment — sourced from a
new storage-only query, `PaymentRepository.SettledPaycheckIDs()`. "Is this correction
outstanding?" is therefore always a derived question — `Correction() != 0 AND paycheck ID not
in SettledPaycheckIDs()` — never a stored flag, so `Paycheck` itself stays exactly as
immutable as ADR-0006 requires.

Persistence (`adapters/sqlite`) adds two tables: `correction_payments` (the transaction) and
`correction_payment_paychecks` (which paychecks it settles), with a `UNIQUE` constraint on
`paycheck_id` there as a second line of defense alongside the app-level check.

The CLI gains `settle-correction` (`-paid-on`, `-amount`, `-paychecks <comma-separated IDs>`,
`-note`) and a `payments` command to list recorded transactions; `list` gains a SETTLED
column. Selecting paychecks by explicit ID (copy-pasted from `list`'s output) rather than a
date range is deliberate: it keeps the transaction record exactly as auditable as its
inputs — you state precisely what a payment covers, not "everything roughly in this range."

Employer-side tax remittance (Schedule H, CA EDD filings) is **not** covered by this ADR —
that's money owed to a government, not the employee, and tracking *that* it was filed/paid is
a separate future decision if it's ever needed.

# References
- Builds on ADR-0017 (tracked actual-paid amount, derived `Correction()`).
- Mirrors the append-only pattern of ADR-0006 (paychecks) and ADR-0010 (rate history).
- Respects ADR-0012 (domain decides, adapters relay) and ADR-0003 (typed IDs — `PaymentID`).
