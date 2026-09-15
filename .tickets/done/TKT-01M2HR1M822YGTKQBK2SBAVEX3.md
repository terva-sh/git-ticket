---
schema: 2
id: TKT-01M2HR1M822YGTKQBK2SBAVEX3
title: Document interchange and drop the duplicated store layout
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-15T05:18:43Z
updated_at: 2026-09-15T05:23:26Z
created_by:
  id: agent:terva/docs-interchange
  name: ""
updated_by:
  id: agent:terva/docs-interchange
  name: ""
extensions: {}
---

## Description

Three things fell out of reviewing what today's work left undocumented and
whether an embedder can reach it.

### The library review came back clean, with one exception this session added

Everything built today is in `ticket/` and reachable without the CLI:
`ImportOptions.SameOwner` and `ChecklistTick`, the five new `ChangeKind`
constants, `SubSectionHeadings` and `IsSectionName` with
`CodeSectionHeadingDemoted`, and `Init` adopting a directory that has no
config. Note compaction splits correctly too, with `ticket.Entries()` the
library primitive and `cli/notes.go` presentation on top of it.

The exception is `adoptedTickets` in `cli/commands.go`, added while fixing
TKT-01M29PRW7BGJHTNFQZCWV657A5. It counts ticket files by walking
`draft`, `tickets`, `done` and `archive`, which it spells as string literals
because the library keeps `storeDirs` unexported. That is a closed list in two
places with nothing holding them in agreement, and AGENTS.md names closed lists
as the thing to check before duplicating one.

It is also unnecessary. `reportAdoption` already calls `s.List()`, and a store
that was just created holds tickets only if `Init` adopted them, so the count is
`len(all)` after the call rather than a stat before it. The repair is deletion,
not a library change.

### The README's instructions section is now wrong

It describes one block, which it was until TKT-01M2HMZ308RYY9ZXXMJS738S48 split
it. It does not mention that there are two forms, that `--write` installs the
short one, or that `--core` and `--full` select between them. A reader following
it forms the wrong idea of what lands in their AGENTS.md.

### The README does not document interchange at all

There is no `export` or `import` in the lifecycle block or in the prose. The
same goes for `series` and for reading a note history with `note --list` and
`--show N`. Interchange is what the next release headlines, and somebody reading
the README would not learn the feature exists.

### The ledger handoff goes stale on tag

`docs/handoff-ledger-cross-store-move.md` says `--same-owner` is not in
`v0.17.1` and to hold its procedure until it ships. It also predates
TKT-01M29PRW7BGJHTNFQZCWV657A5, so it does not know the `git am` route reaches a
usable store now.

## Acceptance criteria

- [x] adoptedTickets is gone and the count comes from the list reportAdoption already reads, so the store's directory layout is spelled in one place
- [x] The README's instructions section describes both forms and the flags that select them
- [x] The README documents export and import, including the git am route and --same-owner
- [x] The README's lifecycle block carries series and the note reading forms
- [x] The ledger handoff names the version that carries --same-owner and the git am route that now works
- [x] Every command and flag the README names exists in the binary, checked by running them

## Implementation plan

Delete first, since that is the part that can break. Then the README in two passes: correct the instructions section against the split, then add an interchange section beside the existing check and instructions prose, with export, import, the git am route and --same-owner, plus series and the note reading forms in the lifecycle block. Then the ledger handoff, written against v0.18.0 as the next minor. Verify by running every command and flag the README names rather than by reading the diff, which is how main..fix survived the last documentation pass.

## Notes

**agent:terva/docs-interchange** at 2026-09-15T05:23:13Z

The library half came back clean. Everything from this release is in ticket/ and reachable without the CLI, and the note work already sat on ticket.Entries(). The only duplication was adoptedTickets, added by me yesterday, and it is gone: reportAdoption derives the count from the listing it already reads, because a store that Init just created holds tickets only if Init adopted them. No library change was needed for any of it.

**agent:terva/docs-interchange** at 2026-09-15T05:23:13Z

The instructions block stays CLI-only on purpose rather than moving to the library. Its content is git ticket command lines, so an embedder whose CLI has different command names needs their own text, not ours. Moving it would export a string that is wrong for every caller except this binary.

**agent:terva/docs-interchange** at 2026-09-15T05:23:13Z

I ran instructions --write from the repository root while checking the README's flags and appended 133 lines to this repository's own AGENTS.md. Restored with git checkout; the diff was purely additive so nothing was lost. Verifying a writing command belongs in a scratch directory, and the rest of the pass was done that way.

**agent:terva/docs-interchange** at 2026-09-15T05:23:13Z

Nearly wrote a false sentence about label reconciliation. A scratch run showed a foreign label surviving an adopt with no warning, which looked like a silent drop contradicting the plan. It was neither: the receiver had labels: [], which per 4.1 means the store has expressed no opinion, so the label is carried and check is content. I had also misread the rendered YAML, where labels: puts its list on the following lines. Both branches are now verified and the README names the condition.

**agent:terva/docs-interchange** at 2026-09-15T05:23:13Z

Verified the README and the handoff by running every command and flag they name, not by reading the diff. That includes proving created_at travels, which needed a backdated origin: the first run could not tell a carried instant from a fresh one because the origin was seconds old. Backdated to 2024-03-01 the adopted copy keeps the date and its new ULID opens 01HQWMQ8, so it sorts by real age.

## Summary

adoptedTickets is gone. reportAdoption derives the count from the listing it already read, because a store Init just created holds tickets only if Init adopted them. That removes the copy of draft, tickets, done and archive from cli/commands.go, so the store's directory layout is spelled once, in the library that owns it. Both init paths were re-run and print exactly what they did before.

The library review needed no other change. Everything this release carries is in ticket/ and reachable without the CLI: ImportOptions.SameOwner and ChecklistTick, the five ChangeKind constants, SubSectionHeadings and IsSectionName with CodeSectionHeadingDemoted, and Init adopting a config-less directory. Note compaction already sat on ticket.Entries(), with cli/notes.go doing presentation only. The instructions block stays CLI-only on purpose: its content is git ticket command lines, so an embedder with different command names needs their own text.

The README gains an interchange section covering export, import, the git am route, --same-owner and what it does not carry, plus series and the note reading forms in the lifecycle block. Its instructions section was wrong as of the split and now describes both forms and the flags that select them.

The ledger handoff names v0.18.0 rather than saying to wait for a release, and gains the git am route, which keeps the original ID when both stores declare the same series and is usually what a move between your own stores wants.

Verified by running every command and flag the three documents name. That caught two things reading could not: created_at needed a backdated origin to prove it travels, and the label reconciliation sentence was nearly written wrong until both branches of the advisory allowlist rule were run.
