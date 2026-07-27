# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture decisions are mandatory reading

This repo maintains permanent, dated Architecture Decision Records in `docs/adr/`. **Any new architectural or design decision (new dependency, new domain concept, new API shape, new persistence strategy, changing a rounding/tax rule, etc.) must be captured as a new ADR before or alongside the code change that implements it.** Do not silently make an architectural choice and only reflect it in code/commit messages.

- Read `docs/adr/README.md` and `docs/adr/STYLE_GUIDE.md` (ADR-0000) before writing one.
- Read the existing ADRs before touching related code — they explain *why* the code is shaped the way it is, not just what it does.
- One ADR = one decision. Use the next sequential number (`docs/adr/000N-kebab-case-title.md`).
- Required frontmatter fields: `id`, `title`, `status`, `date`, `supersedes`, `superseded_by`, `code_anchors`. Body sections: `# Context`, optionally `# Alternatives`, `# Decision`, optionally `# References`.
- If a decision changes, write a new ADR that supersedes the old one (update both files' `supersedes`/`superseded_by` fields) — do not edit history in place.
- When adding an ADR, do not just describe the change — do it in the same style as the existing ones (see `docs/adr/0001-hexagonal-architecture.md` for a full example with Alternatives/References).

Current ADRs:
| # | Decision |
|---|---|
| 0001 | Hexagonal architecture (domain/app/ports/adapters, inward-only dependencies) |
| 0002 | Go workspace, one module per layer |
| 0003 | Typed IDs (`PaycheckID`) instead of raw ints |
| 0004 | Round `float64` money values to cents |
| 0005 | ~~Simplified flat-rate tax withholding~~ — superseded by 0015 |
| 0006 | Paychecks are immutable — computed once, persisted, never recalculated |
| 0007 | SQLite (via `modernc.org/sqlite`) for persistence, over Postgres |
| 0008 | CLI-first; web adapter deleted/deferred, will return strictly read-only |
| 0009 | No `Employee` in the domain — single-employment system |
| 0010 | Effective-dated hourly rates stored in persistence, snapshotted into paychecks |
| 0011 | `DATE`-typed columns, bound/scanned as `time.Time` via the driver |
| 0012 | Business logic in domain + app only; rich domain model (constructors, `RateHistory.RateAsOf`) |
| 0013 | ~~DTOs at adapter boundaries~~ — superseded by 0014 |
| 0014 | Persistence models at adapter boundaries — schema knowledge never leaves the adapter |
| 0015 | 2026 withholding schedules (IRS Pub 15-T percentage method + CA DE 44 Method B), still hard-coded for one single CA weekly filer |
| 0016 | Model employer-side taxes (employer SS/Medicare, FUTA, CA UI/ETT) and cost of employment; YTD wages from stored paychecks drive the $7,000 wage base |
| 0017 | Backfill historical paychecks with a tracked `ActualNetPaid`; `Paycheck.Correction()` derives what's owed |
| 0018 | Correction payments are append-only transaction records (date, amount, settled paychecks), not a `Settled` boolean |

## Commands

This is a multi-module Go workspace (`go.work`) on Go 1.26.4. There is no single `./...` build from the repo root — `go.work`'s `use` directives don't make `.` itself a module, so build/test per module or per package path:

```bash
go build ./domain/... ./ports/... ./app/... ./adapters/sqlite/... ./cmd/cli/...
go test ./domain/...                # pure domain unit tests
go test ./functional_tests/...      # end-to-end: app service + real SQLite in a temp dir
go test ./domain/... -run TestCalculateNetPay_40Hours_At20PerHour_DeductsTaxes -v   # single test
```

The CLI (the only driving adapter, ADR-0008) — every subcommand takes `-db` (default `nannypayroll.db`):

```bash
go run ./cmd/cli set-rate -amount 20 -from 2026-07-01   # record a rate, effective-dated (ADR-0010)
go run ./cmd/cli run -hours 40 -period-end 2026-07-05   # run payroll: computes, persists, prints the stub
go run ./cmd/cli backfill -csv history.csv              # import historical periods paid outside this tool (ADR-0017)
go run ./cmd/cli settle-correction -amount 6.20 -paychecks id1,id2 -note "Venmo"  # record a real payment settling corrections (ADR-0018)
go run ./cmd/cli payments                               # list recorded correction payments
go run ./cmd/cli list                                   # stored paychecks, most recent first (CORRECTION + SETTLED columns)
```

## Architecture

Hexagonal (ports & adapters) + DDD, enforced by strict inward-only dependencies across separate Go modules joined by `go.work` (ADR-0001, ADR-0002):

```
domain  (nannypayroll/domain)   - zero external deps, pure business logic
ports   (nannypayroll/ports)    - interfaces expressed only in domain types; depends on domain only
app     (nannypayroll/app)      - orchestrates use cases; depends on domain + ports
adapters/*                      - implement ports; each is its own module and may pull in external deps
  adapters/sqlite                 (persistence; modernc.org/sqlite, pure Go)
cmd/cli (nannypayroll/cmd/cli)   - the driving adapter AND wiring point; the only place
                                   interface satisfaction is compiler-checked
```

- No import flows outward-to-inward. If domain or app needs to import an adapter, that's a dependency-rule violation. `adapters/sqlite` deliberately does not import `ports` — satisfaction is checked by `var _ ports.X = ...` assertions in `cmd/cli/main.go` only.
- Module naming is local-only (`nannypayroll/...`), resolved via `go.work` — no `replace` directives or `require` lines for sibling modules in individual `go.mod` files.
- **Write path is CLI-only** (ADR-0008): the employer/admin records rates and runs payroll from the terminal. There is deliberately no web adapter right now; when one returns it must be read-only over the same repositories and never accept pay inputs from a request.
- Domain model (ADR-0009: no `Employee` — this system models exactly one employment): `HourlyRate` (amount + `EffectiveFrom` date), `RateHistory` (append-only; `RateAsOf(date)` picks the rate in force), `PayPeriod` (end date + hours + rate, has `Calculate()` → `Paycheck`), `Paycheck` (ID, period end, snapshotted hours/rate, gross/OASDI/Medicare/FIT/SDI/state tax/net, all rounded to cents). Tax rates are package-level constants in `domain/payroll/payroll.go`, hard-coded for one CA filer (ADR-0005, superseded by ADR-0015). Federal income tax uses the IRS Pub 15-T (2026) percentage-method weekly single/standard schedule and CA state income tax uses CA EDD DE 44 (2026) Method B (weekly, single, one allowance), both as package-level bracket tables (`federalWeeklySingleStandard`, `caWeeklySingle`) cross-referenced to `docs/research/2026-ca-nanny-payroll-withholding.md`; OASDI/Medicare/SDI stay flat-rate. Employer-side taxes (employer SS/Medicare, FUTA, CA UI/ETT) and cost of employment are modeled too (ADR-0016): `PayPeriod.Calculate(priorYearWages)` takes year-to-date gross so FUTA/UI/ETT can be capped at the shared $7,000 annual wage base, and the app supplies it via `PayrollRepository.SumGrossForYear`. `Paycheck.ActualNetPaid` and the derived `Correction()` (ADR-0017) track what was really paid vs. what should have been, for reconciling historical periods paid outside this tool; `app.PayrollService.BackfillPaycheck` and the `nannypayroll backfill -csv` command import them in chronological order, guarded by `PayrollRepository.ExistsForPeriodEnd` against double-import. A correction is settled by recording an actual `CorrectionPayment` (ADR-0018) — an append-only transaction record with a date, dollar amount, and the specific paycheck(s) it settles, validated by `payroll.NewCorrectionPayment` to equal exactly what's owed and never double-settle a paycheck; `nannypayroll settle-correction` records one, `payments` lists them, and `list` shows a SETTLED column.
- The domain model is deliberately rich, not thin (ADR-0012): invariants live in domain constructors (`NewHourlyRate`, `NewPayPeriod`) that return sentinel domain errors (`ErrNonPositiveRate`, `ErrInvalidHours`, `ErrNoRateInForce`, ...); rate selection is `RateHistory.RateAsOf`, not a SQL query. Adapters must not validate or decide — the CLI only parses flags and dates, then relays domain errors.
- Ports (`ports/repository.go`) are storage-only: `PayrollRepository` (save/find/list paychecks) and `RateRepository` (`Save` + `History()` returning the full `RateHistory` for the domain to pick from). Both implemented by `adapters/sqlite`.
- Paychecks are write-once (ADR-0006): `RunPayroll` loads the rate history, picks the rate as of the period end (ADR-0010), computes, snapshots hours+rate into the record, persists immediately, and never recalculates. Reads always come from storage — the functional test asserts a raise doesn't alter stored history.
- Persistence mapping goes through adapter-owned persistence models, not shared DTOs (ADR-0014): `adapters/sqlite/model.go` has private row structs mirroring the columns with explicit to/from-domain functions; never scan directly into domain types. A future adapter (Postgres, JSON export) defines its own model rather than reusing sqlite's. Date columns are declared `DATE` and bound/scanned as `time.Time` via the `modernc.org/sqlite` driver (ADR-0011); dates are normalized to UTC midnight at the CLI boundary so stored values sort consistently.

`NOTES.md` and `TODO.md` track deferred decisions (export formats, cloud-hosted storage for a future read-only web UI) and phase progress — check both before assuming something is undecided or unbuilt, since ADRs may have since resolved a NOTES.md item.
