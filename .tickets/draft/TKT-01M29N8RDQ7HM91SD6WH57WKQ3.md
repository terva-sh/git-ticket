---
schema: 2
id: TKT-01M29N8RDQ7HM91SD6WH57WKQ3
title: Record that init's eol=lf line is what protects an import
type: spike
status: draft
status_reason: null
priority: low
due_on: null
labels:
  - question
  - integration
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references:
  - ref: plan:12.8
    path: docs/plan.md
claim: null
archive: null
created_at: 2026-09-12T01:56:15Z
updated_at: 2026-09-12T02:23:46Z
created_by:
  id: agent:terva/mieli
  name: Mieli
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Measured, and the premise this was filed on turned out to be wrong. What remains is two small things.

`import --adopt` is not affected by a CRLF checkout at all. It never runs git, so `core.autocrlf` cannot reach it, and it re-renders every adopted ticket through `Render` rather than copying the arriving bytes.

The `git am` path is not affected either, as long as the receiver has run `git ticket init`. v0.14.3's `.gitattributes` line `.tickets/**/*.md text eol=lf` governs any file arriving at that path, including one `git am` writes. This ticket was filed believing that line protected only a store git-ticket created. It does not, and the measurement is in the note.

### What is actually left

The `eol=lf` line is load-bearing for interchange and was not written for it. v0.14.3 added it so a store cloned on Windows could be read, and it happens to be the whole reason an imported export survives. Nothing records that, so a later edit could remove it as store-only housekeeping and break interchange on Windows with no test failing. Section 12.8 should say what depends on it.

`import` also reports the wrong cause when handed a patch that was itself converted to CRLF in transit. `ParseAddedFiles` compares `l == "new file mode 100644"` exactly, every line ends `\r`, so no hunk is ever recognised and the command says `no added files found`. It refuses safely and writes nothing, but a patch mangled in transit is indistinguishable from a patch that carries no hunks.

### Not this ticket

The one genuinely broken case, a repository with no store at all, is broken for a reason that has nothing to do with line endings: an export carries no `config.yml`, so reads refuse with `store_not_found` and `init` refuses with `store_exists`. TKT-01M29PRW7BGJHTNFQZCWV657A5 carries that.

### Trigger

Anyone editing `ensureAttributes` or the `init` attribute lines, or a report of `no added files found` on a patch that visibly has hunks.</description>

## Acceptance criteria

- [x] Whether import --adopt is affected is measured rather than assumed, and the result recorded
- [ ] Plan 12.8 records that init's eol=lf attribute is what keeps an imported export parseable
- [ ] import either names line endings when a converted patch yields no hunks, or the message is left and the reason recorded

## Notes

**agent:terva/mieli** at 2026-09-12T02:22:59Z

Measured against the released v0.17.1 binary, per criterion 1. The answer narrows this ticket almost to nothing and turns up a different defect.

### The matrix

Windows was reproduced on Linux with `GIT_CONFIG_GLOBAL` carrying `core.autocrlf=true`, which is the same git mechanism rather than a stand-in.

| Path | Result |
| --- | --- |
| `import --adopt`, LF patch, autocrlf=true | Immune. 0 CR bytes, fence `---\n`, `check --strict` clean |
| `import --adopt`, patch itself converted to CRLF | Refuses, exit 1, nothing written |
| `git am` into a repo that has run `init` | Immune. 0 CR bytes, `check --strict` clean |
| `git am` into a repo with no store | Files land CRLF, 37 CR bytes, fence `---\r\n` |

### import --adopt is not affected

It never runs git, so the config cannot reach it, and it re-renders every adopted ticket through `Render`. The arriving bytes are parsed and thrown away rather than copied, which is why the conversion has nothing to act on.

Handed a patch file that had itself been converted to CRLF, it refuses with `no added files found` and writes nothing. That is safe but the message is wrong about the cause: `ParseAddedFiles` compares `l == "new file mode 100644"` exactly, every line ends `\r`, so `newFileSeen` never becomes true and each hunk is skipped. A patch converted in transit is indistinguishable from a patch with no hunks.

### v0.14.3 already covers the git am path

This is the correction that matters. The ticket assumed the `eol=lf` attribute protects only a store git-ticket created, and that an export lands outside its reach. It does not. A receiver that has run `git ticket init` carries `.tickets/**/*.md text eol=lf`, and that line governs any file arriving at that path, including one `git am` writes. Measured: 0 CR bytes and `check --strict` clean under autocrlf=true.

So the only exposed case is a repository that has never run `init`. And there the CRLF is the lesser problem.

### The real defect, which is not about line endings

A repository with no store cannot use the tickets `git am` just landed, whatever their line endings. An export carries the ticket files and no `config.yml`, so every read refuses with `store_not_found: no config.yml` while `init` refuses with `store_exists`, and `init` has no force flag.

Reproduced with `core.autocrlf=false` and pure LF on Linux, 0 CR bytes, so it is platform-independent and the CRLF framing was a distraction. Filed as TKT-01M29PRW7BGJHTNFQZCWV657A5 (Give git am of an export a path to a usable store), which is now the ticket that matters here.

### What is left of this one

Two questions, both much smaller than the description claims.

Whether `init` should keep writing the `eol=lf` line, which is load-bearing for the `git am` path and was not written for it. It should at least be recorded as such in 12.8, because a later edit could remove it as store-only housekeeping and silently break interchange on Windows.

Whether `import` should name line endings when a converted patch produces no hunks, rather than reporting `no added files found`.

The description is rewritten to the measured state and its original premise is kept verbatim below, since the version above argued that v0.14.3 does not cover this and the measurement says it does.

### The original description, verbatim

> Plan 12.8 promises a receiver can land an export with `git am DIR/*.patch` in a repository that has never heard of git-ticket. On Windows that promise has an unwritten caveat, and the caveat makes the arriving files unreadable.
>
> git for Windows installs with `core.autocrlf=true`. Applying an export's patch under that config writes the ticket files with CRLF line endings. Plan 5.3 requires LF, and `parse` refuses a file whose fence line is `---\r`.
>
> ### Why this is not v0.14.3 again
>
> v0.14.3 made `init` write `.tickets/**/*.md text eol=lf` beside the merge-driver line, which protects a store that git-ticket created. An export does not carry that line. `exportTicketFiles` writes a hunk for each ticket file and nothing else, so a repository receiving its first ticket by `git am` has no `.gitattributes` covering `.tickets/`, and nothing tells git to leave those bytes alone.

That reasoning is correct about a repository with no store and wrong about one that has run `init`, which is the case it treated as the general one.
