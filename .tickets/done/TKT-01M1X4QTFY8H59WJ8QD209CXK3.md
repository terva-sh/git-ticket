---
schema: 2
id: TKT-01M1X4QTFY8H59WJ8QD209CXK3
title: Have init write an eol=lf .gitattributes line for Windows stores
type: bug
status: done
status_reason: null
priority: high
due_on: null
labels:
  - format
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-07T05:16:30Z
updated_at: 2026-09-07T05:57:42Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

On Windows, git converts text files to CRLF on checkout by default. Plan 5.3 requires a ticket file to carry LF line endings, and `parse` enforces it: a file whose first line is `---\r` is rejected with `parse_error: file does not start with a --- frontmatter fence`. So a user who clones a repository containing a `.tickets/` store on Windows gets a store where every ticket fails to read, and `git ticket list` answers with nothing rather than with an error that explains itself.

This is not hypothetical. It is what took the Windows CI lane red on its first run, GitHub Actions run 34085691784: about 30 fixtures converted, and `corpus_test.go:51` named the cause as "CRLF line endings, 5.3 requires LF". This repository fixed its own checkout with `* text=auto eol=lf` in `.gitattributes`, which protects this repository and no other.

### The decision

`git ticket init` writes an `eol=lf` line into `.gitattributes` beside the merge-driver line, so a store is protected at creation. That is the user's ruling, taken on 2026-09-07 alongside the choice to ship v0.14.2 with the lock fix alone.

`init` already writes that file. `ensureMergeAttribute` in `cli/mergedriver.go` appends `.tickets/**/*.md merge=gitticket` when it is absent and leaves everything else in the file untouched, so a second line follows the same pattern and the same append-never-rewrite rule.

The line should be scoped to the store rather than global, because the file belongs to the repository and not to this tool. `.tickets/**/*.md text eol=lf` matches what `mergeAttrPattern` already builds.

### What this does not solve

`init` cannot help a store that already exists, and it cannot help a clone made before the line was written. The user considered making `parse` tolerate CRLF as well and chose not to, so a converted store stays unreadable until somebody adds the attribute and re-checks-out. If that turns out to bite in practice, reopening the tolerance question is the response, and this ticket is where the reasoning lives.

### Plan

The plan changes with the implementation and not before, per AGENTS.md: the format is authoritative and a code change that needs it moves it in the same commit. Section 7.5 covers the merge driver's `.gitattributes` entry and is the natural home for the second line.

## Acceptance criteria

- [x] git ticket init writes a store-scoped eol=lf line into .gitattributes beside the merge-driver line, appending rather than rewriting
- [ ] A store created by init and cloned on Windows parses, proven on the Windows CI lane rather than inferred
- [x] docs/plan.md section 7.5 describes the second line, changed in the same commit as the code

## Notes

**agent:terva/mieli** at 2026-09-07T05:57:39Z

Criterion 2 ships unticked. It asks that a store created by init and cloned on Windows be proven to parse "on the Windows CI lane rather than inferred", and the lane runs `go test ./ticket/...` only, so a cli test cannot reach it. Widening it is TKT-01M1X4QTHPJ0. The criterion stays as written rather than being reworded to match what shipped.

The substantive goal is met, and by a stronger route than the one the criterion named. `TestAStoreSurvivesACRLFClone` in `cli/installdriver_test.go` creates a store, commits it, and clones with `core.autocrlf=true`, which is not a stand-in for the Windows condition but the identical git mechanism, forced on. It asserts the cloned ticket file carries no CR and that `list --all` in the clone returns tickets.

The second subtest is why this is worth more than a Windows run would be. It removes `.gitattributes`, clones the same way, and requires that the store then fail. Without that control, a green first half could mean the conversion never happened. It fails as expected, so the attribute is doing the work.

What a Windows lane run would still add is confidence about path handling in the same code, which is a different concern from the one this ticket is about. That is TKT-01M1X4QTHPJ0's to deliver, and this box is its to tick.

**agent:terva/mieli** at 2026-09-07T05:57:39Z

Two scope decisions taken during the build, neither of them in the ticket as filed.

`install-merge-driver` writes the merge line alone and never the eol line. It already calls the shared helper, so including the eol line would have been free, and free is the wrong reason. The command's name is a promise about what it touches, and a person running it to repair a merge driver should not find an unrelated rule appended to a tracked file. `TestInstallMergeDriverLeavesTheEOLLineAlone` holds it there.

That leaves a store created before this release with no repair but a manual line, which is a real gap and not one to settle inside a command called install-merge-driver. It is now a deferred question in plan section 15, named by TKT-01M1X70GCG5RHGT0ZHVDC09D62 (Decide how a store predating the eol attribute gets repaired), with a trigger of a real report rather than the reasoning alone. `check --fix` is the obvious home and the expensive one, because every finding in section 11 is about the store's own contents and `.gitattributes` sits outside the store.

The second decision is two lines rather than one combined `text eol=lf merge=gitticket`. The helper matches a whole line, so a repository carrying the merge line from before this gains the eol line and keeps what it has. A combined rule would match neither and append a duplicate beside the original. `TestInitAddsOnlyTheLineThatIsMissing` is that case, and it is the realistic one: every store that exists today is in it.

One thing the build corrected about the ticket's own reasoning. The filed description said `ensureMergeAttribute` "appends when absent", which is true, and I first wrote an idempotence test on the assumption that a person might run `init` twice. `init` refuses a second run over an existing store with `store_exists`, so that route cannot double a line at all. The test was replaced with the case that can happen: a repository that already carries both lines, where init must add nothing and must not report `.gitattributes` as changed.

## Summary

`git ticket init` writes two `.gitattributes` lines now, `.tickets/**/*.md text eol=lf` first and the merge-driver line second. Line endings come first because the merge driver only matters once the file can be read at all.

`ensureMergeAttribute` became `ensureAttributes(s, want []string)`, which appends every line that is absent in one write and reports which ones it added. `storeAttributeLines` is what `init` asks for. `install-merge-driver` asks for the merge line alone, because its name is a promise about what it touches.

The proof is `TestAStoreSurvivesACRLFClone`, not an assertion that a line was written. It creates a store, commits it, and clones with `core.autocrlf=true`, which is the identical git mechanism Windows turns on by default rather than a stand-in for it, then requires the cloned file to carry no CR and `list --all` to return tickets. The second subtest removes `.gitattributes` and requires the same clone to fail, so a green first half cannot mean the conversion never happened.

Plan 7.5 gained the second line, the reason it comes first, why it is two lines rather than one combined rule, and the sentence that `install-merge-driver` never writes it. Section 15 gained the question that refusal leaves open, TKT-01M1X70GCG5RHGT0ZHVDC09D62, with a trigger of a real report rather than reasoning.

Criterion 2 ships unticked. The Windows lane runs `./ticket/...` only, so a cli test cannot reach it, and widening it is TKT-01M1X4QTHPJ0's job rather than a reason to reword the box.
