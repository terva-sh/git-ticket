---
schema: 2
id: TKT-01M2NZDPTQQV6T68B0N4S8BAE7
title: doctor --json emits a file path that does not exist
type: bug
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/doctor
  - area/integration
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-16T20:44:37Z
updated_at: 2026-09-16T20:44:53Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

`cli/doctor.go:165` joins the store path onto a path that already has it, so
every finding's `file` field in the JSON envelope names a file that does not
exist.

```go
out.File = storePath(s, filepath.Join(s.Path(), f.File))
```

`storePath` already joins `s.Path()` itself:

```go
func storePath(s *ticket.Store, rel string) string {
	return displayPath(s, filepath.Join(s.Path(), filepath.FromSlash(rel)))
}
```

The library reports `f.File` store-relative (`ticketPath` returns
`statusDir(status) + "/" + ID + ".md"`), so the caller's `filepath.Join` is the
one that is wrong. Passing an absolute path to `storePath` makes the second join
concatenate rather than resolve.

Observed from terva's store, run from the repository root:

```
.tickets/home/sothr/.t3/worktrees/terva/t3code-506e1c8c/.tickets/draft/TKT-01M1RRT5427P8M1YBWWHNE8PQ0.md
```

`.tickets/` prepended to an already-absolute path. Reproduces with an explicit
`--store` too, so it is not a store-resolution question.

The sibling code gets it right, which is the clearest statement of the fix.
`findings()` in `cli/json.go:636`, which does the same conversion for `check`,
calls `displayPath` rather than `storePath` and joins exactly once:

```go
File: displayPath(s, filepath.Join(s.Path(), f.File)),
```

So `cli/doctor.go:165` should be `storePath(s, f.File)`, or the `findings()`
form spelled out. Either is one line.

Worth a test that asserts the emitted path resolves, not merely that it is
non-empty: the current value is a plausible-looking string, which is why it
survived the release. `cli/json.go`'s comment says the repository-relative form
is "the contract the rest of the JSON uses", and this is the one place that
breaks it, so a consumer cannot treat `file` as usable without special-casing
`doctor`.

## Acceptance criteria

- [ ] doctor --json emits a file path that resolves from the directory the contract says it is relative to
- [ ] A test asserts the emitted path exists on disk, rather than asserting it is non-empty
- [ ] doctor and check agree on the form of the file field, verified against the same store
