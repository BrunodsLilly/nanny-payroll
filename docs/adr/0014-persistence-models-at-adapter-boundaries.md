---
id: ADR-0014
title: Persistence models at adapter boundaries (renamed from DTO)
status: accepted
date: 2026-07-06
supersedes: ADR-0013
superseded_by: none
code_anchors:
    - adapters/sqlite/model.go
---
# Context
ADR-0013 named the sqlite row structs (`rateRow`, `paycheckRow`) "DTOs." That
term more precisely means data crossing the *outer* boundary of the
application — an API request or response body — not a database row. Calling a
row-mapping struct a DTO is a defensible but nonstandard usage (treating any
adapter's external system, including a database, as "outside" the domain), and
it invites confusion for anyone applying the term in its stricter, more common
sense.

A second question follows from the rename: are these structs meant to be
shared by future adapters, or are they specific to sqlite?

# Decision
Rename the concept from "DTO" to **persistence model**. `adapters/sqlite/dto.go`
becomes `adapters/sqlite/model.go`; the struct names (`rateRow`, `paycheckRow`)
and their to/from-domain mapping functions are unchanged — only the label and
file name change. The rule from ADR-0013 stands: adapters convert between their
own persistence models and domain types at the boundary; domain types never
carry storage or serialization tags.

Persistence models are **owned by their adapter, not shared across adapters**.
A model's shape exists to mirror one storage engine's idioms — sqlite's `DATE`
columns and driver conversions (ADR-0011) are not the same problem a Postgres
adapter (`JSONB`, arrays, `pgtype.Date`) or a JSON export adapter would face.
Forcing a single shared model across adapters would either flatten every
adapter to the lowest common denominator, defeating the swappability hexagonal
architecture (ADR-0001) is for, or produce a redundant layer that just
duplicates the domain model. If a future adapter needs the same shape sqlite
uses, that is coincidence, not a reason to extract a shared type — each adapter
defines its own.

# References
- ADR-0001 (dependency rule; adapter-owned models protect it)
- ADR-0011 (sqlite's own storage-specific concern this model reflects)
- ADR-0013 (superseded — same rule, imprecise name)
