---
schema: 1
id: TKT-01M1RZ5AKF31JC0MRA6XSDW7QE
title: Seed tickets from named templates in the store
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
references: []
claim:
  actor: agent:terva/mieli
  branch: feat/templates
  worktree: /home/sothr/workspace/git.local.sothr.com/terva-sh/git-ticket
  commit: cf3c91514d39e7c8e54f2a8f5e3e476bc514ac75
  claimed_at: 2026-09-05T17:26:03Z
  expires_at: null
archive: null
created_at: 2026-09-05T14:22:03Z
updated_at: 2026-09-05T17:33:08Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Filed 2026-09-05 from a user ask, raised while a real integration was
migrating an existing system off its custom tooling onto git-ticket.
That is the audience: a team that had forms, moving to a store that
has flags.

Half of a template mechanism already exists, which is worth stating so
nobody builds it twice. `create` takes `--description-file`,
`--plan-file`, and repeatable `--ac` and `--dod`, so a template is
currently a wrapper script beside a Markdown file, and for a system
driving git-ticket programmatically that is honestly fine. The store
also has `defaults` in config, but only type, priority, and
claim_expiry: nothing per-type, nothing about seeded checklists.

Three gaps the wrapper does not cover, in the order they bite:

The TUI form. `n` opens blank. Flags can be scripted; a form cannot,
so a person filing a bug in the TUI gets no template at all. Named
templates earn most of their keep exactly there.

Consistency as policy. The point of a form is that every bug carries
the same definition of done. A wrapper is per-person; a template in
the store travels with the repository and binds agents too.

Discoverability. `create --template bug` is findable in --help; the
wrapper script is findable only if you know it exists.

The proposed shape follows the store's own conventions. A template is
a ticket-shaped Markdown file a person reviewed, living in
`.tickets/templates/NAME.md`, so it versions, diffs, and reviews like
everything else in the store. `create --template NAME` seeds type,
priority, labels, checklists, and body sections, and every explicit
flag wins over the template, because a template is a starting point
and not an argument. The TUI form starts from one the same way.
Templates seed and never bind: `check` holds no ticket to its
template, because a ticket that outgrew its template is the normal
case, not a finding.

### Open questions

Whether an unlisted or missing template name is an error or a warning,
and whether the template list belongs in `config` output beside the
allowlists.

Whether a template may seed a status or a created instant, now that
6.2.1 exists. The conservative answer is no: a backport script passes
those explicitly, and a template that files things as done is a
foot-gun.

What `go:embed`-style discovery the TUI needs, given the gotcha that
embed skips dot-directories. Templates live under `.tickets/`, which
is read at runtime, not embedded, so this should be a non-issue, and
whoever builds it should confirm that sentence rather than trust it.

### Trigger

Arguably fired: the integration this ticket was raised from is live.
Parked as a draft all the same, because promotion is a human call.

## Acceptance criteria

- [x] create --template NAME seeds type, priority, labels, checklists, and body sections, with explicit flags winning
- [x] templates live in .tickets/templates/ and travel with the repository
- [x] the TUI create form can start from a template
- [x] templates seed, never bind: check holds no ticket to its template
- [x] the plan records the design before code

## Notes

**agent:terva/mieli** at 2026-09-05T17:33:08Z

Built and all five criteria are ticked. The three open questions were
settled with the user before the branch: a missing template refuses
the create, config publishes the template names, and nothing
lifecycle-shaped seeds.

The plan moved first: 4.2 is the design section, the 10.6 envelope
gains a bare templates list with the reasoning for why it is not an
allowlist object, and the 12.1 create line gains --template.

The mechanics. A template is loaded leniently, not through the strict
ticket parser: the frontmatter unmarshals a seedable subset (type,
priority, labels, assignees, milestone) and parseBody carries the
description, both checklists, and the plan. Everything else in the
file is ignored, which is what lets a template be made by copying a
real ticket: TestTemplateIgnoresLifecycleFields proves the copied
id, status, and title all stay behind. The merge sits before the
defaults resolve, so the precedence reads option, then template, then
store default, and a template carrying a type the schema does not
know refuses through the same validation as a typed one.

The TUI's n fronts the form with a template picker only when the
store defines templates, so a store without them keeps the
one-keystroke path. The blank row is first for the same reason. A
chosen template prefills the description editor with the skeleton,
names itself in the form header, and rides to the store with the
save, where the remaining fields seed.

Proven live as well as in the suite: a scratch store with a bug
template created from it, config listed it, and check --strict stayed
green with templates/ present, which is the fourth criterion's claim
run rather than asserted. Twelve new tests across ticket/, cli/, and
tui/view/ hold the rest.

This store defines no templates yet, deliberately: which templates
this repository wants is its own decision, not this build's.

## Summary

Shipped, per the new plan 4.2. A template is a ticket-shaped Markdown
file in .tickets/templates/, and `create --template NAME` seeds type,
priority, labels, assignees, milestone, both checklists, the
description, and the plan, with every explicit flag winning and
nothing lifecycle-shaped seeding. A missing name refuses the create,
config publishes the template list on both output forms, and the
TUI's n fronts the form with a template picker when the store defines
any, prefilling the description skeleton. The loader ignores the
non-seedable fields, so copying a real ticket makes a valid template.
Proven live: create from template, config listing, and check --strict
green with templates/ present.
