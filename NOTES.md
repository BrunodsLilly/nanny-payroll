# NannyPayroll — NOTES

Design concerns, deferred decisions, and gotchas. Append with `/note <text>`.

---

## Architecture

- Hexagonal (ports & adapters) + DDD
- Go workspace (`go.work`) — one `go.mod` per module to isolate CVE blast radius
- Dependency rule: domain → no deps; ports → domain only; app → domain + ports;
  adapters → domain + their own external dep; cmd/cli → all adapters (wiring point)
- The CLI is the only driving adapter and does all writing (ADR-0008)
- Interface satisfaction is enforced by the compiler at `cmd/cli/main.go` only
- Swapping persistence (e.g. SQLite → cloud DB) = add adapter module + change one line in main.go

## Persistence Layer

Decided: SQLite. See ADR-0007.

## Domain Model

Pinned by ADRs: `PayPeriod`, `Paycheck` (immutable, ADR-0006), `HourlyRate`
(effective-dated, ADR-0010). `Employee`/`Employer` deliberately absent —
single-employment system (ADR-0009).

## Deferred: Export formats

Paychecks/pay stubs should be exportable from the CLI (CSV? JSON? printable
pay stub?). Format undecided — write an ADR when export is actually built.

## Deferred: Cloud-hosted storage + read-only web UI

Plan: swap local SQLite for a cloud-based store that both the CLI (read-write)
and a future web UI (read-only connection) can reach. The web UI renders stored
records only — it never accepts pay inputs (ADR-0008). Provider/shape undecided;
ADR when chosen. The old templ + htmx notes go with this phase:

- CLI required for code generation: `go install github.com/a-h/templ/cmd/templ@latest`
- `.templ` files generate `.go` files — add generated files to `.gitignore` or commit them (team preference TBD)
- `templ generate` must run before `go build`

## Tax withholding

Current: 2026 withholding schedules for one single CA weekly filer (ADR-0015,
supersedes ADR-0005). Federal = Pub 15-T percentage method; CA = DE 44 Method B;
SDI 1.3%. Full sources + worked examples in
`docs/research/2026-ca-nanny-payroll-withholding.md`.

Deferred (new ADR when each is actually built):
- **Employer-side taxes** — done (ADR-0016): employer SS/Medicare, FUTA 0.6% net, CA UI/ETT,
  capped at the $7,000 annual wage base via YTD gross; `CostOfEmployment` on each paycheck.
  Still open: experience-rated UI (uses new-employer 3.4%) and FUTA credit-reduction status.
- **Generalization**: filing status, pay frequency, W-4/DE-4 allowances, optional-FIT
  flag as inputs. Hard-coded for the one filer until a second situation appears.
- **OASDI $184,500 wage base** and any year-to-date caps — unmodeled (no YTD state;
  unreachable at this income).
- **Annual refresh**: bracket tables are a 2026 snapshot; new ADR + tables for 2027.

## Historical corrections

Done (ADR-0017): `Paycheck.ActualNetPaid` + derived `Correction()` track what was really
paid vs. the correct amount. `nannypayroll backfill -csv path.csv` imports historical
periods (CSV columns: `period_end,hours,hourly_rate,actual_net_paid`, ascending order,
no duplicate period ends). Employee-side corrections are summed and reported; employer-side
taxes accrued-but-never-remitted are reported too, but **remitting** them (Schedule H, CA
EDD DE 9/DE 9C) is a filing action outside this tool — confirm with a CPA.

Deferred:
- **Issuing the correction**: done (ADR-0018) — `CorrectionPayment` is an append-only
  transaction record (date, amount, settled paycheck IDs), validated to equal exactly what's
  owed; `settle-correction` records one, `payments`/`list` surface settled status.
- **Cross-tax-year backfill**: `BackfillPaycheck` uses today's (2026) schedules; backfilling
  a period from a different tax year would need that year's tables researched first — not
  needed yet since employment started in 2026.

## Go Version

Check current stable before writing go.mod files:
```bash
go version
```

## Module Naming

Using local-only paths (`nannypayroll/domain`, not `github.com/...`).
`go.work` resolves all local deps — no `replace` directives needed in individual go.mod files.
