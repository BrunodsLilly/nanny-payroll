---
id: ADR-0008
title: CLI-first; web adapter deferred and read-only
status: accepted
date: 2026-07-05
supersedes: none
superseded_by: none
code_anchors:
    - cmd/cli
---
# Context
The admin and the only user of the write path are the same person: the employer.
The weekly workflow is small — enter the hours worked, run payroll, persist the
paycheck. A browser UI adds a server, templates, and input-handling concerns the
write path does not need. Worse, the existing `adapters/ui` endpoint accepted
hours *and rate* as query parameters, which fights ADR-0006: pay data must come
from and go to the repositories, not from a request.

# Alternatives
**Keep the HTTP adapter as the write path.** Rejected. It is the slowest route
to a working weekly payroll run, and a writable web surface is unnecessary
attack/complexity surface for a single-admin tool.

# Decision
Delete the web driving adapter (`adapters/ui`) and its wiring point (`cmd/web`).
The only driving adapter is a CLI (`cmd/cli`), which becomes the wiring point
where interface satisfaction is compiler-checked. All writes go through the CLI:
recording hourly rates and running payroll for a period.

A web UI returns later as a strictly read-only adapter over the same
repositories — it renders stored records and must never accept pay inputs from
the request. ADR-0006's `POST` endpoint sketch is deferred along with the web
adapter; the `GET`-from-storage shape is what the future web UI implements.

# References
- ADR-0006 (paycheck immutability is unchanged; only its HTTP API sketch is deferred)
