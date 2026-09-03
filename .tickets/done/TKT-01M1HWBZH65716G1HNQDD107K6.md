---
schema: 1
id: TKT-01M1HWBZH65716G1HNQDD107K6
title: "Warn when text passed for a body section carries a ## heading"
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: doc:gotcha
    path: AGENTS.md
claim: null
archive: null
created_at: 2026-09-02T20:18:35Z
updated_at: 2026-09-03T01:48:09Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

`create --description`, `create --plan`, `update --description`, `plan`, `summary`, `note`, and `comment` all take text that becomes a body section. A line in that text starting with `## ` becomes a section boundary, because that is what `parseBody` splits on. The section comes back holding only the prose above the first one, and everything after it lands in `body.extra`.

Nothing reports it. `check` passes, `show` prints every section in order, and the file reads correctly. `TKT-01M1HVMQ` was filed that way and had to be filed again, and the `AGENTS.md` gotchas now carry it.

The parsing rule itself is right and is not in scope. `## Risks` written into a ticket by hand is a section somebody wanted, it is preserved through `Body.Extra` with a fixture holding that behaviour, and this format exists to be hand-edited.

Comment fencing the body was considered and declined. Section F of `docs/review-backlog-md.md` records marker-based body parsing as something this format is ahead on by not doing, because a parser that depends on a comment fails when a person removes a comment they had no reason to keep. Markers are used for the instructions block under 12.1 for the opposite reason: that block is regenerable, so losing a marker costs a refresh rather than data.

The defect is the silence, not the rule. The failure happens at the moment the text is passed on the command line, and that is where it should be reported.

### Scope

Warn on stderr when text destined for a body section carries a line that `parseBody` would read as a heading, and name `###` as the fix. No format change, no migration, no new finding code, no schema bump.

The warning has to use the same fence tracking `parseBody` does. A `## ` inside a fenced code block is not a heading there and must not warn here, or the guard cries wolf on every ticket that quotes Markdown.

`ac --add` and `dod --add` are out of scope. An item is written as `- [ ] TEXT`, so the line cannot open with `## `.

### Decision 1: warn or refuse

Warn is the recommendation. Passing several sections in one `--description` works today and somebody may be doing it deliberately, so refusing would break a path that functions in order to prevent a mistake. A warning costs one line of stderr and leaves the choice with the caller.

Refusing behind an escape flag is the alternative. It is the stronger guard, but it needs a flag nobody will remember and it turns a papercut into a blocked command.

### Decision 2: what happens under --json

Plan section 10 puts an error message on stderr in both modes. A warning is not an error and must not touch stdout, or it corrupts the envelope a caller is parsing.

Stderr in both modes is the obvious answer and it matches how the error path already behaves. It is worth stating rather than assuming, because a caller parsing stdout has to be unaffected and a caller reading stderr should not be surprised by a line that is not a failure.

## Acceptance criteria

- [x] Writing a body section with text containing a line parseBody would read as a heading prints a warning on stderr that names ### as the fix.
- [x] The warning never touches stdout and never changes the exit status, so a caller parsing the JSON envelope is unaffected.
- [x] It covers create --description, create --plan, update --description, plan, summary, note, and comment.
- [x] A ## inside a fenced code block does not warn, because parseBody does not read it as a heading either.
- [x] ac --add and dod --add do not warn, because an item is written as a checkbox line and cannot open with ##.

## Summary

Shipped. The CLI warns on stderr when text destined for one body section carries a line the parser reads as the start of another, naming the heading it found and `###` as the fix.

Both parked decisions took the recommendation. It warns rather than refuses, because passing several sections in one string works today and somebody may mean it, so refusing would break a working path to prevent a mistake. The warning goes to stderr in both output modes, never touches stdout, and moves no exit status.

The design point worth keeping is that there is now one implementation of the heading rule rather than two. `fenceScanner.heading` in ticket/parse.go is the single decision, `parseBody` drives it, and the exported `ticket.SectionHeadings` drives the same scanner. A warning that disagreed with the parser would be worse than no warning: it would flag text that is fine and stay silent on text that is not. Refactoring `parseBody` onto the scanner changed no behaviour, which the round-trip corpus confirmed.

Coverage is all seven commands. Four of them (note, comment, plan, summary) share `runTextEntry`, so that is one call site, plus `runCreate` for `--description` and `--plan` and `runUpdate` for `--description`.

Three mutations verified the guard. Making the warning a no-op failed nine assertions across every command. Dropping fence tracking failed both packages. Swapping the raw line for the trimmed one failed the indented case.

The fence mutation taught something about the agreement test. Comparing `SectionHeadings` against what `parseBody` produced passed under that mutation, because both sides broke together by construction. Only the explicit expected list caught it. An agreement assertion between two things sharing an implementation proves they share it, not that either is right, so it needs a named expectation beside it.

Plan 5.2 now states the splitting rule and its two exceptions, and section 10 states the warning contract, which was assumed rather than written. The AGENTS.md gotcha said "Nothing catches that" and no longer does.
