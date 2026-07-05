---
id: ADR-0012
title: Business logic lives in domain and app layers only
status: accepted
date: 2026-07-05
supersedes: none
superseded_by: none
code_anchors:
    - domain/payroll/payroll.go
    - ports/repository.go
    - app/payroll_service.go
---
# Context
Two business rules had leaked into outer layers, drifting toward an anemic
domain model:

- "The rate in force is the latest rate effective on or before the period end"
  (ADR-0010) was implemented as SQL (`ORDER BY effective_from DESC LIMIT 1`)
  inside the sqlite adapter. Swapping the adapter would mean re-implementing —
  and possibly diverging on — a payroll rule.
- Input invariants (a rate must be a positive amount, hours must be positive)
  were checked by CLI flag validation. Any new driving adapter would have to
  repeat them or silently accept invalid domain objects.

# Alternatives
**Keep rules in the adapters for query efficiency.** Rejected: the rate history
of a single employment is tiny, and correctness of payroll rules matters more
than avoiding a full-table read. Hexagonal architecture (ADR-0001) exists
precisely so business rules never depend on an adapter's query capabilities.

# Decision
Domain and app are the only layers that hold business logic; adapters translate
and transport.

- The domain model is rich, not thin: invariants are enforced by domain
  constructors (`NewHourlyRate`, `NewPayPeriod`) that return domain errors, so
  an invalid domain value cannot be constructed from any adapter.
- Rate selection is a domain behavior: `RateHistory.RateAsOf(date)` implements
  ADR-0010's rule in pure Go. The `RateRepository` port shrinks to storage-only
  operations (`Save`, `History`) — it fetches records, it does not decide.
- The app layer orchestrates use cases (fetch history → pick rate → build
  period → calculate → persist) and holds no calculation or validation of its
  own beyond sequencing.

# References
- ADR-0001 (hexagonal architecture — this makes the dependency rule meaningful)
- ADR-0010 (the rate-selection rule this moves into the domain)
