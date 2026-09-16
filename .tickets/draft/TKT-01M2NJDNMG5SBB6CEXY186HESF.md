---
schema: 2
id: TKT-01M2NJDNMG5SBB6CEXY186HESF
title: Decide whether a store can author its own doctor rules
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: TKT-01M2NHGHTZ4PWHBKRHE43XJG8D
dependencies:
  - TKT-01M2NHGHTZ4PWHBKRHE43XJG8D
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-16T16:57:24Z
updated_at: 2026-09-16T16:57:29Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

Whether a store can define its own `doctor` rules, rather than only configuring
the ones the binary ships.

Deferred from TKT-01M2NHGHTZ4PWHBKRHE43XJG8D, which says why: configuring
shipped rules and letting a store write new ones are not the same size.
Configuration is a map in a config file. Authoring is an expression language, or
a plugin mechanism, or shelling out, and each of those carries a trust and a
portability problem the first one does not have.

### The trust problem is the whole question

A ticket store is a git repository somebody clones. A rule that executes is a
rule that executes on somebody else's machine, on a checkout they may not have
read first. That is a different risk from anything `git ticket` does today, and
it is the reason this is its own ticket rather than a later flag on the
framework.

### What the framework must not foreclose

The framework ticket's last criterion is that configuring shipped rules does not
foreclose a store defining its own later. This ticket is the other side of that
sentence, and the thing it protects: whatever shape the config takes for
built-ins should leave room for an entry that names something other than a
shipped identifier.

### Trigger

Somebody with a store-specific hygiene rule they cannot express by configuring a
shipped one, after the framework and its first rules have shipped and the config
format has been exercised.

## Acceptance criteria

- [ ] A shape is chosen for store-authored rules, or the question is recorded as declined with the reason
- [ ] The trust problem of executing a cloned repository's rule is answered, whatever the shape
