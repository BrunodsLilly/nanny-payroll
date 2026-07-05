# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture decisions are mandatory reading

This repo maintains permanent, dated Architecture Decision Records in `docs/adr/`. **Any new architectural or design decision (new dependency, new domain concept, new API shape, new persistence strategy, changing a rounding/tax rule, etc.) must be captured as a new ADR before or alongside the code change that implements it.** Do not silently make an architectural choice and only reflect it in code/commit messages.

- Read `docs/adr/README.md` and `docs/adr/STYLE_GUIDE.md` (ADR-0000) before writing one.
- Read the existing ADRs (`0001`-`0006`) before touching related code — they explain *why* the code is shaped the way it is, not just what it does.
- One ADR = one decision. Use the next sequential number (`docs/adr/000N-kebab-case-title.md`).
- Required frontmatter fields: `id`, `title`, `status`, `date`, `supersedes`, `superseded_by`, `code_anchors`. Body sections: `# Context`, optionally `# Alternatives`, `# Decision`, optionally `# References`.
- If a decision changes, write a new ADR that supersedes the old one (update both files' `supersedes`/`superseded_by` fields) — do not edit history in place.
- When adding an ADR, do not just describe the change — do it in the same style as the existing ones (see `docs/adr/0001-hexagonal-architecture.md` for a full example with Alternatives/References).

Current ADRs:
| # | Decision |
|---|---|
| 0001 | Hexagonal architecture (domain/app/ports/adapters, inward-only dependencies) |
| 0002 | Go workspace, one module per layer |
| 0003 | Typed IDs (`EmployeeID`, `PaycheckID`) instead of raw ints |
| 0004 | Round `float64` money values (to cents, despite the ADR text saying "thousandths") |
| 0005 | Simplified flat-rate tax withholding, hard-coded for a single CA employee |
| 0006 | Paychecks are immutable — computed once, persisted, never recalculated |
| 0007 | SQLite (via `modernc.org/sqlite`) for persistence, over Postgres |

## Commands

This is a multi-module Go workspace (`go.work`) on Go 1.26.4. There is no single `./...` build from the repo root — `go.work`'s `use` directives don't make `.` itself a module, so build/test per module or per package path:

```bash
go build ./domain/...
go build ./ports/...
go build ./app/...
go build ./adapters/ui/...
go test ./domain/...
go test ./functional_tests/...
go test ./domain/... -run TestCalculateNetPay_40Hours_At20PerHour_DeductsTaxes -v   # single test
```

`cmd/web` has a `go.mod` but no `main.go` yet, and `adapters/sqlite` has a `go.mod` but no implementation file — both are unfinished per `TODO.md`.

`adapters/ui/server.go`'s `/paycheck` endpoint calls `app.NewPayrollService(nil, nil)` since `CalculatePaycheck` doesn't touch the repositories yet — it's a stateless preview endpoint, not the persist-then-read flow described in ADR-0006. Once a real use case needs the repositories, wire real implementations through instead of `nil`.

## Architecture

Hexagonal (ports & adapters) + DDD, enforced by strict inward-only dependencies across separate Go modules joined by `go.work` (ADR-0001, ADR-0002):

```
domain  (nannypayroll/domain)   - zero external deps, pure business logic
ports   (nannypayroll/ports)    - interfaces expressed only in domain types; depends on domain only
app     (nannypayroll/app)      - orchestrates use cases; depends on domain + ports
adapters/*                      - implement ports; each is its own module and may pull in external deps
  adapters/sqlite                 (persistence, not yet implemented)
  adapters/ui                     (HTTP/JSON, driving adapter)
cmd/web (nannypayroll/cmd/web)   - wiring point; the only place interface satisfaction is compiler-checked
```

- No import flows outward-to-inward. If domain or app needs to import an adapter, that's a dependency-rule violation.
- Module naming is local-only (`nannypayroll/...`), resolved via `go.work` — no `replace` directives in individual `go.mod` files.
- Swapping persistence (SQLite → Postgres) means adding a new adapter module and changing one line in `cmd/web/main.go`; it should never require touching `domain`, `ports`, or `app`.
- Domain model currently: `Employee` (ID + hourly rate), `PayPeriod` (hours worked + employee, has `Calculate()` → `Paycheck`), `Paycheck` (gross/OASDI/Medicare/FIT/SDI/state tax/net, all rounded to cents). Tax rates are package-level constants in `domain/payroll/employee.go`, hard-coded for one CA filer (ADR-0005) — see the `TODO` comment there about annualization once `PayPeriod` gains a pay frequency.
- `PayrollRepository`/`EmployeeRepository` (`ports/repository.go`) are the only two ports today; both are unimplemented (no adapter satisfies them yet).
- Paychecks are write-once: computed during a `POST /employees/{id}/paychecks`-style run, persisted, and always read back from storage afterward rather than recomputed (ADR-0006). The current `adapters/ui/server.go` `/paycheck` endpoint predates this and doesn't yet follow the persist-then-read shape described in ADR-0006.

`NOTES.md` and `TODO.md` track deferred decisions (SQLite vs Postgres, `templ` codegen policy, undecided domain nouns) and scaffold progress — check both before assuming something is undecided or unbuilt, since ADRs may have since resolved a NOTES.md item.
