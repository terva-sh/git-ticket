---
schema: 2
id: TKT-01M3CV8R9J3QXQSN07B9VAGV7V
title: Decide how a store declares where each reference namespace resolves
type: spike
status: review
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
claim:
  actor: agent:codex/reference-namespaces
  branch: feat/reference-namespace-design
  worktree: /home/sothr/.t3/worktrees/git-ticket/file-reference-questions
  commit: abebb684bdbd7256352831d91b136386ba6b0099
  claimed_at: 2026-09-26T00:59:37Z
  expires_at: null
archive: null
created_at: 2026-09-25T17:54:32Z
updated_at: 2026-09-26T01:13:21Z
created_by:
  id: agent:claude-code/opus
  name: ""
updated_by:
  id: agent:codex/reference-design
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

- [x] The existing reference model, refs lookup, path validation, and real pr:/origin-ticket:/origin-store: examples are measured before a new declaration is specified.
- [x] One design specifies portable namespace declarations, identifier grammars, external URL templates and foreign-store bindings, including where machine-local overrides live.
- [x] Validation covers declared identifier syntax and configuration while check remains offline; unknown namespace policy, finding severity, and unavailable foreign stores are settled against existing tickets.
- [x] The design states what show, ui, JSON, import --from-store, and refs expose or preserve, with an end-to-end example for a PR and a foreign ticket.
- [x] Compatibility, schema impact, and the plan sections to change are recorded before implementation tasks are filed.
- [ ] Export carries a versioned lookup table for the namespace declarations used by its tickets, excludes machine-local paths, and keeps the existing git am DIR/*.patch route usable.
- [ ] Import previews the offered mappings and lets the receiver explicitly adopt, alias, or decline them; conflicts never silently overwrite a local namespace, and partial-failure behavior is defined.
- [ ] The mapping is bound to the ticket artifact so a stale or substituted table cannot be accepted as the source declaration for a different export.

## Implementation plan

1. Inventory reference namespaces in this store and exercise the real PR and foreign-ticket examples. Confirm what refs, path checking, and a literal url: reference already provide.
2. Design a portable namespace declaration that pairs typed identifiers with destinations. Compare a per-namespace registry with a direct target on each reference; specify grammar, URL templates, foreign-store bindings, and machine-local overrides.
3. Design a versioned export lookup table for the declarations used by the exported tickets. Compare a sidecar with cover or patch embedding; keep git am DIR/*.patch working without adoption, exclude machine-local paths, and bind the table to its ticket patch.
4. Design receiver preview and explicit adoption of that table: keep, alias, or decline each incoming mapping; never overwrite a conflicting local declaration silently. Specify whether config adoption is a separate write or part of ApplyImport, including lock and partial-failure behavior.
5. Design validation and compatibility together: validate declared syntax and local configuration without network access; decide undeclared-namespace behavior and finding severity after measuring existing stores. Define absent foreign-store behavior and the reader contract for show, ui, JSON, refs, and import --from-store.
6. Walk one PR and one foreign-ticket reference through export, preview, optional mapping adoption, ticket adoption, and later reading. Record the chosen format in plan sections 5.1, 5.5, 11, and 12.8, identify any schema migration, then file bounded implementation work.

## Notes

**agent:codex/reference-groom** at 2026-09-26T00:45:44Z

Groomed. The trigger fired in the reported move between stores. I checked the current tree: Reference holds only Ref and optional repository-relative Path; refs finds tickets by namespace/identifier but does not resolve their destination; check validates a local path and warns on an untyped ref. ImportOptions.FromStore is a plain name, and import records origin-ticket: and origin-store: as unverified references. A literal url: reference can carry one destination today, but it does not bind an independently typed pr: or origin-ticket: identifier to that destination.

The user chose to design full namespace declarations and validation together as the first work. This remains a design spike: the plan must specify portable declarations, local overrides, identifier grammar, resolution, and an offline validation contract before implementation. Check cannot verify an external URL or a missing foreign store by reaching the network; its existing no-network guarantee stays part of the design.

Alternatives considered: display-only resolution would leave malformed declared identifiers unchecked; documentation alone would leave the live references uninterpretable; a direct URL per reference is useful but duplicates destinations and does not answer the store-wide binding question. Treating every undeclared namespace as invalid may break existing stores, so enforcement and finding severity need a measured compatibility decision. The ticket's area label now leads, matching this store's convention. The trigger fired; the implementation choice has not been made.

**agent:codex/reference-groom** at 2026-09-26T00:46:31Z

Groomed. The user added a material requirement after the first grooming note: an export should offer a lookup table so the receiving store can choose whether to adopt the sender's reference mappings. I read the current artifact path: Export returns a cover letter and one ticket patch, CLI writes those as 0000-cover-letter.txt and 0001-tickets.patch, and import reads only the patch. This means the mapping must be a new artifact and API input if the receiver is to preview it mechanically.

The plan uses a versioned sidecar as the leading design, not a ruling yet. It would stay outside *.patch so git am continues to apply the ticket files alone; the receiver could inspect or explicitly adopt the map through git-ticket. The table should carry only declarations relevant to exported references, not every configuration value or a machine-local path. A conflicting namespace may already mean something different in the receiver, so adopting must offer alias or decline and must not silently overwrite. The table also needs an integrity tie to the patch, since the patch's ticket blobs are verified but an unrelated sidecar would not be. Embedding in the cover is harder to parse reliably; putting it in the patch would make git am write configuration the receiver never accepted. The design spike will settle the artifact format, trust boundary, and partial-failure semantics before implementation.

**agent:codex/reference-groom** at 2026-09-26T00:50:57Z

PR #222 review recorded. The targeted Terva review ran on head 3bca1d6eaa0be750be136a6e90f537743876473c against base adbcaa2452744a07efa706df2eebc7f50de96ef0, request groomed-ready-3bca1d6, run ce377a7a-4b8b-41f7-ab4e-de097aef61c6: https://git.local.sothr.com/terva-sh/git-ticket/actions/runs/566. The maintained clean summary is on https://git.local.sothr.com/terva-sh/git-ticket/pulls/222. It reported no findings, so there is no review change to accept, dispute, or defer. It saw only the ticket diff and could not verify runtime behavior; the local scratch-store runs and full just ci result are the evidence for those claims. This note records the result without changing either design scope.

**agent:codex/reference-design** at 2026-09-26T01:07:30Z

Design decision for plan 5.1, 5.5, 11, and 12.8: use a versioned tracked .tickets/references.yml with opt-in namespace declarations, named RE2 capture groups, HTTPS templates, and portable foreign-store bindings. Machine-local checkout paths go in ignored references.local.yml and never affect check. The ticket frontmatter shape stays unchanged; no ticket schema bump is needed for the registry. check reports malformed tracked declarations and invalid identifiers in declared namespaces as errors, while undeclared namespaces stay legal and opaque. No check path reaches the network or requires a foreign checkout.

Evidence: this store has 82 references in ten namespaces, including ticket:report and an abbreviated origin-ticket: ID. In a scratch store, pr:Sothr-Infrastructure/documentation#1, origin-ticket:TKT-01M2NZ88, origin-store:ledger, and a literal url: reference all stored, refs found the typed values by namespace, and check --strict passed. Existing code validates only a repository-relative reference path; refs is a lookup, not a resolver. A previous scratch run put a references key in config.yml, then series add rewrote config and lost it, so adding an unrecognized config key is unsafe for older binaries.

Export offers a versioned references.json sidecar containing only declarations used by the ticket patch, their portable stores, undeclared namespace names, and SHA-256 of the exact 0001-tickets.patch bytes. It stays outside *.patch, preserving git am. Import verifies the digest before preview, displays each mapping, and requires adopt, alias, or decline choices; conflicts cannot take a local namespace silently. A conflicting declined reference must be rewritten to a free opaque namespace when tickets are adopted. The sidecar hash binds files within the artifact but is not an identity signature. Mapping adoption is a separate locked registry write, reported separately from partial ticket filing. Legacy origin-ticket: plus origin-store: remains opaque; a declared foreign-ticket namespace adds an independently resolvable reference for new imports.

Alternatives rejected: config.yml loses unknown keys through old renderers; per-reference URLs duplicate destinations and do not bind a typed PR or foreign ID; embedding the map in the ticket patch would make git am write unapproved config; embedding it in the cover is hard to parse and bind; mandatory declarations would invalidate existing references. No built-in ticket grammar is safe against the measured ticket:report and abbreviated provenance. The plan now gives a PR and a foreign-ticket example through export, preview, adoption or decline, and reading. Implementation is separate work; this spike records its contract, not a shipped feature.

**agent:codex/reference-design** at 2026-09-26T01:08:41Z

Criteria 6-8 remain unticked because they describe shipping the sidecar, receiver mapping choices, and artifact binding. The design contract is in the plan; TKT-01M3DM2SHMH0KMP1Z0SZKPTCR8 (Carry reference lookup mappings through export and import) owns the runtime work. I will not mark a design as an implemented feature.

**agent:codex/reference-design** at 2026-09-26T01:10:53Z

The first full just ci run failed because section 11 listed four future check codes before any implementation fixtures existed. TestCorpusCoversEveryPlanCode intentionally requires the live table and fixture corpus to land together. I moved those codes to a planned-error paragraph in section 11; the implementation tasks must add each to the live table with fixtures. A second env -u NO_COLOR mise exec go@1.27.1 -- just ci run passed.

**agent:codex/reference-design** at 2026-09-26T01:13:21Z

PR #224 targeted review on head bb2dd6f852921344cbe989edd67777b582e503c6 against base abebb684bdbd7256352831d91b136386ba6b0099 ran as c708f22b-cc93-4036-a3be-c5ac6abb964a (Actions run 571, review 752). It found a medium mapping gap: declining an identical offered mapping cannot make an unchanged reference opaque when the receiver already has that same declaration. Accepted. Plan 12.8 now says decline copies nothing, unchanged refs remain opaque only when no local declaration exists, and an identical local declaration continues resolving them; conflicting meanings require an opaque alias. The same review found a separate moved-origin revision gap, accepted on the related spike. A follow-up review will check the new head.

## Summary

The design contract is in docs/plan.md sections 5.1, 5.5, 11, and 12.8. It specifies a separate versioned references.yml, opt-in offline grammar validation, local overrides, safe URL and foreign-ticket resolution, an export lookup sidecar bound to the ticket patch, and explicit receiver adopt/alias/decline choices. Measured 82 existing refs in ten namespaces and tested the real PR/provenance shapes in a scratch store. Four implementation drafts now carry the runtime work: TKT-01M3DM2SGK9X1S1FN7RKPF02A1 (Implement portable reference registry and offline validation), TKT-01M3DM2SHMH0KMP1Z0SZKPTCR8 (Carry reference lookup mappings through export and import), and TKT-01M3DM2SJQSX7T51TX8WHK5ZGY (Show resolved reference targets across readers). Criteria 6-8 remain unticked until those features ship.
