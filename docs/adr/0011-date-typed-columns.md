---
id: ADR-0011
title: DATE-typed columns, bound as time.Time
status: accepted
date: 2026-07-05
supersedes: none
superseded_by: none
code_anchors:
    - adapters/sqlite
---
# Context
Dates were stored as hand-formatted `YYYY-MM-DD` TEXT columns, with
`time.Format`/`time.Parse` calls scattered through the adapter. That encoding
is invisible to the schema (the column says `TEXT`), accepts junk silently,
and every new query re-implements the conversion.

SQLite has no native date storage class, but a column *declared* `DATE` both
documents intent in the schema and tells the `modernc.org/sqlite` driver to
convert values to and from `time.Time` automatically.

# Alternatives
- **Keep `TEXT` + manual `YYYY-MM-DD` formatting.** Rejected: conversion logic
  duplicated at every query site, and nothing stops a malformed string.
- **INTEGER Unix epoch.** Rejected: compact but unreadable when inspecting the
  database directly, and gains nothing at this scale.

# Decision
Declare date columns as `DATE` and bind/scan Go `time.Time` values directly,
letting the driver own the encoding. Dates are normalized to UTC midnight at
the driving-adapter boundary (the CLI parses `YYYY-MM-DD` into UTC), so stored
values sort and compare consistently.

# References
- ADR-0007 (SQLite via `modernc.org/sqlite`)
