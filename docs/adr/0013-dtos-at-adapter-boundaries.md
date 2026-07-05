---
id: ADR-0013
title: DTOs at adapter boundaries
status: accepted
date: 2026-07-05
supersedes: none
superseded_by: none
code_anchors:
    - adapters/sqlite/repository.go
---
# Context
The sqlite adapter scanned query results straight into domain struct fields.
That couples the table layout to the domain type's shape: reordering a domain
field, or a schema change, silently threatens the other side, and any future
storage concern (nullability, denormalized columns, serialization tags) would
pressure the domain type to accommodate it.

# Alternatives
**Scan directly into domain types.** Rejected: it works while the shapes happen
to match, but the coupling is invisible until it breaks, and domain types must
stay free of persistence concerns (ADR-0001).

# Decision
Each adapter converts between its own private data transfer objects and domain
types at its boundary. For sqlite: unexported row structs (`rateRow`,
`paycheckRow`) mirror the table columns, with explicit mapping functions to and
from the domain. Domain types never carry storage or serialization tags; schema
knowledge never leaves the adapter.

# References
- ADR-0001 (dependency rule this mapping protects)
