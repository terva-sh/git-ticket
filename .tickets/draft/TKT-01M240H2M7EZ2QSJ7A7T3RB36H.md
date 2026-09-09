---
schema: 2
id: TKT-01M240H2M7EZ2QSJ7A7T3RB36H
title: Create templates from tickets and discover when to use them
type: task
status: draft
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
references:
  - ref: plan:templates
    path: docs/plan.md
  - ref: code:template-loader
    path: ticket/template.go
  - ref: code:ticket-creation
    path: ticket/apply.go
claim: null
archive: null
created_at: 2026-09-09T21:17:36Z
updated_at: 2026-09-09T21:17:36Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: Mieli
extensions: {}
---

## Description

Provide a supported way for a human or model to turn an existing ticket, including completed work, into a reusable named template, and discover which template fits new work.

### Existing functionality

Plan 4.2 already defines ticket-shaped Markdown templates in .tickets/templates/NAME.md. Copying an existing ticket there works because ticket/template.go reads only seedable fields and ignores identity and lifecycle fields. There is no dedicated conversion command. Named templates seed type, priority, labels, assignees, milestone, description, implementation plan, acceptance criteria, and definition of done.

The current template creation path in ticket/apply.go copies checklist sections as written, including checked boxes. In contrast, create --from ID seeds a new ticket's title, description, and unchecked acceptance criteria and records origin. That existing command creates tickets, not reusable templates, and should not be rebuilt here.

config and config --json publish sorted template names. The TUI create picker offers names, and the library exposes Store.Templates() and Store.Template(name). There is no separate usage description for discovery: a template's Description is content seeded into the new ticket. Templates are identified by filename stems, not unique IDs. An id in a copied ticket is ignored.

### Scope

Add safe template creation from a source ticket without changing that ticket. Clear acceptance criteria and definition-of-done checkmarks, exclude source identity and lifecycle records, and let the author revise ticket-specific content into reusable instructions.

Add descriptive discovery for humans and model consumers, with guidance about when to use each template kept separate from the description seeded into a ticket. Preserve compatibility with existing template files and the existing names-only config contract unless a change is explicitly agreed.

### Open questions

Should template names be the stable documentation contract, or should templates have a separate rename-stable identity? If an identity is introduced, decide its scope, how documentation and create resolve it, and how renames and copies behave. This ticket does not presume unique IDs are required.

Settle the authoring command and flags, usage-metadata spelling, discovery command and JSON shape, and whether the TUI picker should display usage guidance before implementation. Decide how conversion handles source-specific routing fields and prevents accidental overwrites. Decide whether resetting checkmarks applies only to conversion or also changes the existing create --template behavior.

### Trigger

The user requested this draft after confirming that ticket-to-ticket creation already exists but reusable template authoring and descriptive discovery do not. Promotion and interface decisions remain for the user.

## Acceptance criteria

- [ ] A supported operation creates a reusable template from an existing ticket, including a done ticket, without changing the source.
- [ ] A converted template has unchecked acceptance criteria and definition of done and does not retain source identity or lifecycle records.
- [ ] Template usage guidance is separate from the description seeded into new tickets.
- [ ] Humans and model consumers can discover available templates with their usage guidance.
- [ ] Existing template files remain usable, with compatibility of existing discovery output explicitly addressed.
- [ ] Stable template identity is decided with the user and recorded in the plan before implementation; a separate ID is not assumed.
- [ ] The plan records the agreed authoring and discovery interfaces before code changes.

## Implementation plan

1. Confirm the current loader, creation path, config output, and TUI picker behavior against the tree.
2. Settle the open interface and identity questions with the user, then update plan 4.2 and the relevant CLI and JSON sections.
3. Implement source-ticket conversion and usage metadata with tests for completed sources, checked checklists, unchanged source content, and overwrite handling.
4. Implement human and machine-readable discovery on the agreed surfaces and test compatibility with existing templates.
5. Exercise conversion, discovery, and ticket creation from the resulting template in a scratch store.
