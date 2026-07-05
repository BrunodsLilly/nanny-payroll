---
id: ADR-0001
title: Hexagonal architecture
status: accepted
date: 2026-07-02
supersedes: none
superseded_by: none
code_anchors:
    - domain
    - app
    - ports
    - adapters
---
# Context
Payroll rules are the core of this system. They must be testable and developable without a running database or HTTP server. Entangling business logic with infrastructure makes both harder to test and harder to change independently.

# Alternatives
**MVC (Rails/Laravel)** Fast to scaffold. Rejected because it couples the domain model persistence by design. Hexagonal architecture with DDD written in Go favored due to simplicity and control.

# Decision
The codebase adopts a Hexagonal Architecture/Ports and Adapters Pattern; uses four layers with a strict indward dependency rule:
- `domain` - no dependencies outside `stdlib`
- `app` - depends on `domain` and `ports` only
- `ports` - interfaces expressed in domain types only
- `adapters` - implements ports; may import external libraries

No import flows outward-to-inward. Violation means business logic has taken and infrastructure dependency, which breaks testability.

# References
- Alistair Cockburn, [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture)
