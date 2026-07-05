# NannyPayroll — NOTES

Design concerns, deferred decisions, and gotchas. Append with `/note <text>`.

---

## Architecture

- Hexagonal (ports & adapters) + DDD
- Go workspace (`go.work`) — one `go.mod` per module to isolate CVE blast radius
- Dependency rule: domain → no deps; ports → domain only; app → domain + ports;
  adapters → ports + their own external dep; cmd/web → all adapters (wiring point)
- UI (templ + htmx) is a **driving** adapter — it instantiates app services at startup
- Interface satisfaction is enforced by the compiler at `cmd/web/main.go` only
- Swapping persistence (e.g. SQLite → Postgres) = add adapter module + change one line in main.go

## Persistence Layer

Decided: SQLite. See ADR-0007.

## Deferred: Domain Model

Domain nouns not yet pinned. Plausible candidates:
- `Employee` — the nanny being paid
- `Employer` — the family paying
- `PayPeriod` — weekly/biweekly window
- `Paycheck` — the output of a pay run (amount, taxes, net)
- `HourlyRate` — value object (amount + currency)
- `TaxWithholding` — federal/state deductions

**Do not define these until first failing test forces it.** Outside-in TDD.

## Go Version

Check current stable before writing go.mod files:
```bash
go version
```

## templ

- CLI required for code generation: `go install github.com/a-h/templ/cmd/templ@latest`
- `.templ` files generate `.go` files — add generated files to `.gitignore` or commit them (team preference TBD)
- `templ generate` must run before `go build`

## Module Naming

Using local-only paths (`nannypayroll/domain`, not `github.com/...`).
`go.work` resolves all local deps — no `replace` directives needed in individual go.mod files.
