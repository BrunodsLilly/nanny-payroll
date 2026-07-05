---
id: ADR-0002
title: Go workspace one module per layer
status: accepted
date: 2026-07-02
supersedes: none
superseded_by: none
code_anchors:
    - domain/
    - app/
    - ports/
    - adapters/
---

# Context
To create this project under a single `go.mod` would lead to a large list of dependencies. No module should have to depend on a package it doesn't need.

# Decision
Use a Go workspace to manage multiple modules into one cohesive system.

# Uncertainties
- The Go compiler could sort out which dependencies to use where, but the team wants to be explicit in modularization. 
