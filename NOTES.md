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

## Go Version

Check current stable before writing go.mod files:
```bash
go version
```

## Module Naming

Using local-only paths (`nannypayroll/domain`, not `github.com/...`).
`go.work` resolves all local deps — no `replace` directives needed in individual go.mod files.
