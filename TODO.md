# NannyPayroll — TODO

## Phase 0: Scaffold (structure only, no logic)

- [X] `go.work` — workspace root linking all modules
- [X] `domain/go.mod` — module `nannypayroll/domain`, zero external deps
- [X] `ports/go.mod` — depends on domain only
- [X] `app/go.mod` — depends on domain + ports
- [X] `adapters/sqlite/go.mod` — depends on domain + `modernc.org/sqlite`
- [X] `cmd/cli/go.mod` — depends on all adapters; wiring point (was `cmd/web`, replaced per ADR-0008)

## Phase 1: Domain model (TDD)

- [X] Decide on domain nouns — `PayPeriod`, `Paycheck`, `HourlyRate` (ADR-0006, ADR-0009, ADR-0010)
- [X] Write first failing test in `domain/payroll/`
- [X] Grow domain model test-first

## Phase 2: Persistence adapter (TDD)

- [X] Resolve SQLite vs Postgres — SQLite (ADR-0007)
- [X] Repository integration coverage (`functional_tests/` runs against real SQLite)
- [X] Implement adapter (`adapters/sqlite/repository.go`)

## Phase 3: CLI driving adapter (ADR-0008)

- [X] `set-rate` — record effective-dated hourly rate (ADR-0010)
- [X] `run` — run payroll for a period, persist immutable paycheck (ADR-0006)
- [X] `list` — read stored paychecks
- [ ] Export artifacts (pay stubs) — format undecided, needs an ADR (see NOTES.md)

## Phase 4: Cloud storage + read-only web UI (deferred)

- [ ] Choose cloud-hosted store both CLI (read-write) and web (read-only) can reach — needs an ADR
- [ ] Read-only web adapter over the same repositories — renders stored records, never accepts pay inputs
