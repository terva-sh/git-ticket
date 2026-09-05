---
schema: 1
id: TKT-01M1RPM03QD2D41MX7Z00WQZ36
title: Copy a ticket's body from the command line
type: task
status: draft
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T11:52:47Z
updated_at: 2026-09-05T11:52:47Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from a user ask: get a ticket's body out of the store
and into a paste buffer without hand-selecting terminal output. `show`
prints the frontmatter summary and every section under styled headings,
which is built for reading, not for pasting into a PR body or a chat
message. Today the only path is `show` plus mouse selection, and that
carries the header lines and loses fenced blocks to wrapping.

Two halves, one composable and one convenient.

The composable half is `show ID --body`: print the Markdown body as
stored, below the frontmatter, no summary lines, no ANSI, and exit 0.
That alone makes every platform work by pipe: `| pbcopy` on macOS,
`| wl-copy` on Wayland, `| xclip -selection clipboard` on X11,
`| clip.exe` on WSL2. It is also the testable core, a golden-output
test against a fixture, no platform code involved.

The convenient half writes the clipboard directly, and needs a small
helper that probes for the platform tool in a fixed order: pbcopy,
wl-copy, xclip, xsel, clip.exe, with OSC 52 through the controlling
terminal as the fallback when no tool answers. The probe order is
worth a comment in the code, because WSL2 is Linux with clip.exe on
PATH, so the Linux tools must win when both are present and the user
is under Wayland or X11. The helper takes an exec function through
Env, the way the store takes a clock, so tests inject a fake and no
test shells out.

Platform scope, per the user: WSL2, Linux, and macOS. Native Windows
is not a supported platform for this feature; the direct-copy path
there refuses with a message naming the supported platforms, and
`--body` still works everywhere stdout does.

### Open questions

Whether the direct spelling is `show ID --copy` or a `git ticket copy
ID` command. A flag keeps the surface at 32 commands and reads well;
a command is more discoverable in `help`. Either way plan 12.1 gets
the surface in the same change, and no new JSON kind is needed,
because a clipboard write has nothing to say in JSON.

Whether `--body` means the whole stored body or wants a section
selector later. Whole body first: `--section description` is a
separable follow-up nothing here forecloses.

### Trigger

None. This is startable feature work parked as a draft until the user
promotes it.

## Acceptance criteria

- [ ] show ID --body prints the stored Markdown body alone: no frontmatter summary, no ANSI, faithful to the file below the frontmatter
- [ ] the direct-copy path writes the clipboard on macOS, Wayland, X11, and WSL2 through the probed tools, with the probe order tested against an injected exec
- [ ] native Windows direct-copy refuses with a message naming the supported platforms, and --body still prints there
- [ ] plan 12.1 documents the chosen surface in the same change
