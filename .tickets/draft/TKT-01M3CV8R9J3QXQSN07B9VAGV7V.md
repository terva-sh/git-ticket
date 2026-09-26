---
schema: 2
id: TKT-01M3CV8R9J3QXQSN07B9VAGV7V
title: Decide how a store declares where each reference namespace resolves
type: spike
status: draft
status_reason: null
priority: normal
due_on: null
labels:
  - area/format
  - question
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: ticket:TKT-01M3CV8RASXJ4PPHD7N78T7XHK
    path: null
  - ref: ticket:TKT-01M3CT0GV02WDW2W7B3EJSC6R1
    path: null
claim: null
archive: null
created_at: 2026-09-25T17:54:32Z
updated_at: 2026-09-26T00:45:44Z
created_by:
  id: agent:claude-code/opus
  name: ""
updated_by:
  id: agent:codex/reference-groom
  name: ""
extensions: {}
---

## Description

A reference is a namespace, a colon, and an identifier (plan 5.5), and the core preserves namespaces it does not interpret (5.1). The only part of a reference git-ticket ever resolves is its `path`. Nothing records where a namespace's identifiers live, so neither a reader nor a tool can get from a reference to the thing it names. This asks whether a store should be able to declare that for each namespace, and how.

### What prompted it

One session on 2026-09-25 moved tickets between two stores on the same machine and filed one into this repository. These references were written:

- `pr:Sothr-Infrastructure/documentation#1`: a pull request, but on which forge? The identifier shape is the writer's invention.
- `origin-ticket:TKT-…` and `origin-store:ledger`, from 12.8: `ledger` is a name that means a path on one machine (`~/workspace/ledger`), and nothing binds it to one.
- `ledger:TKT-…`, `docs:DOC-…`, and `git-ticket:TKT-…`: ad hoc namespaces that invented the binding in the namespace name, three different ways in one afternoon, because nothing offered a better form.

Each is stored and round-trips correctly, and each is useful only to someone who already knows where it points.

### The idea

Let a store declare, per namespace, an upstream location and the identifier shape that goes with it. What a declaration holds depends on the kind of namespace:

- External: a URL template over a declared identifier grammar. For example, `pr` is `{owner}/{repo}#{number}` and renders as `https://git.local.sothr.com/{owner}/{repo}/pulls/{number}`, and `jira` is `{key}` against a base URL.
- Internal, meaning another git-ticket store: a store binding (a repository URL plus a store path, or a local path), so that `ticket:`, `origin-ticket:`, and `origin-store:` identifiers name something `show` could follow and `check` could at least locate.
- Repository-relative: what `path` already is, left as it is.

A declared grammar would let `check` report a malformed identifier the way it reports `unknown_series`, as a finding whose severity is part of the decision.

### Questions to settle

1. Where declarations live. `config.yml` is committed and shared, but a store binding such as `~/workspace/ledger` is machine-specific. That may argue for a committed declaration (kind, grammar, a URL template, a repository URL) plus a local override (Git config, or a gitignored file) for paths.
2. Is resolution display-only (render a link in `show`, `ui`, the canvas, and `--json`), or does `check` verify anything? 12.8 keeps provenance references deliberately unverified, "which is exactly what a foreign ID needs", and a declared store binding would not change that for a store that is not present.
3. Enforced or advisory. Labels and milestones are advisory allowlists, and series is enforced (5.6). An undeclared namespace probably stays legal, since the core already preserves namespaces it does not know.
4. Whether `import --from-store NAME` should bind `NAME` to a declared store, so that `origin-store:` stops being a bare word.
5. Whether a small built-in vocabulary (`commit`, `url`, `file`, `ticket`) gets default declarations, and how `ticket:` with no store qualifier behaves once other stores can be declared.
6. The relationship to `extensions`, which is the other place integration data lives.

Related: TKT-01M3CT0GV02WDW2W7B3EJSC6R1 (Default series stays TKT after a store stops declaring it), found in the same session, and TKT-01M3CV8RASXJ4PPHD7N78T7XHK (Decide what a move does to dependents left in the sending store).

## Acceptance criteria

- [ ] The existing reference model, refs lookup, path validation, and real pr:/origin-ticket:/origin-store: examples are measured before a new declaration is specified.
- [ ] One design specifies portable namespace declarations, identifier grammars, external URL templates and foreign-store bindings, including where machine-local overrides live.
- [ ] Validation covers declared identifier syntax and configuration while check remains offline; unknown namespace policy, finding severity, and unavailable foreign stores are settled against existing tickets.
- [ ] The design states what show, ui, JSON, import --from-store, and refs expose or preserve, with an end-to-end example for a PR and a foreign ticket.
- [ ] Compatibility, schema impact, and the plan sections to change are recorded before implementation tasks are filed.

## Implementation plan

1. Inventory reference namespaces in this store and exercise the real PR and foreign-ticket examples. Confirm what refs, path checking, and a literal url: reference already provide.
2. Design a namespace declaration that pairs a typed identifier with a destination. Compare a per-namespace registry with a direct target on each reference, then specify the committed fields, identifier grammar, URL template or foreign-store binding, and any machine-local override. Keep machine-specific paths out of committed config.
3. Design validation and compatibility together: validate syntax and local configuration without network access; decide undeclared-namespace behavior and finding severity after measuring existing stores. Define behavior when a foreign store is absent.
4. Specify the reader contract for show, ui, JSON, and refs, and the meaning of import --from-store under a declared binding. Walk one PR and one foreign-ticket reference through the design.
5. Record the chosen format and behavior in plan sections 5.1, 5.5, 11, and 12.8, identify any schema migration, then file bounded implementation work.

## Notes

**agent:codex/reference-groom** at 2026-09-26T00:45:44Z

Groomed. The trigger fired in the reported move between stores. I checked the current tree: Reference holds only Ref and optional repository-relative Path; refs finds tickets by namespace/identifier but does not resolve their destination; check validates a local path and warns on an untyped ref. ImportOptions.FromStore is a plain name, and import records origin-ticket: and origin-store: as unverified references. A literal url: reference can carry one destination today, but it does not bind an independently typed pr: or origin-ticket: identifier to that destination.

The user chose to design full namespace declarations and validation together as the first work. This remains a design spike: the plan must specify portable declarations, local overrides, identifier grammar, resolution, and an offline validation contract before implementation. Check cannot verify an external URL or a missing foreign store by reaching the network; its existing no-network guarantee stays part of the design.

Alternatives considered: display-only resolution would leave malformed declared identifiers unchecked; documentation alone would leave the live references uninterpretable; a direct URL per reference is useful but duplicates destinations and does not answer the store-wide binding question. Treating every undeclared namespace as invalid may break existing stores, so enforcement and finding severity need a measured compatibility decision. The ticket's area label now leads, matching this store's convention. The trigger fired; the implementation choice has not been made.
