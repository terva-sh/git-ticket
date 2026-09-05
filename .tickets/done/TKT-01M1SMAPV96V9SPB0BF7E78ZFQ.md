---
schema: 1
id: TKT-01M1SMAPV96V9SPB0BF7E78ZFQ
title: Warn that just install and install.sh write to different directories
type: chore
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
claim: null
archive: null
created_at: 2026-09-05T20:32:00Z
updated_at: 2026-09-05T23:29:21Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

This repository has two ways to install its own binary, and they
write to different directories. Which one `git ticket` means is
decided by PATH order, silently.

`just install` runs `go install ./cmd/git-ticket`, so the binary
lands in GOBIN, which is ~/go/bin here. `install.sh` takes the first
writable of ~/.local/bin and ~/bin, per its own header, so the README
one-liner lands somewhere else. Run both, as anyone verifying a
release does, and there are two binaries with one name.

### What it looked like

Observed on 2026-09-05, immediately after tagging v0.11.0. `just
install` reported success and put v0.11.0 in ~/go/bin. `git ticket
--version` then answered v0.10.0, because ~/.local/bin came first on
PATH and still held the copy an earlier install.sh run had put there.
Nothing failed and nothing warned. `which -a git-ticket` was what
settled it.

### Why it is worth a line rather than a shrug

AGENTS.md now recommends using the installed binary as a free
before-image for a refactor that must change nothing a person sees:
`diff <(git ticket list --all) <(./git-ticket list --all)`. That
advice assumes `git ticket` is the binary installed before the
current branch. With a shadowing copy it can be any age at all, so
the diff compares the wrong pair and reports empty for the wrong
reason. An empty diff is exactly the answer the technique is looking
for, which is what makes this quiet failure expensive.

The same applies to `git ticket self-update`, which upgrades the
binary that ran it and leaves the other one where it was.

### The likely fix

A short entry in AGENTS.md beside the stale-binary gotcha, naming
both destinations and saying that PATH order picks the winner. It
does not need code. `just install` already prints its destination and
`install.sh` already prints its own, so both halves are visible; what
is missing is the warning that they can differ.

Worth checking at write time whether `just install` should print a
warning when a different git-ticket shadows GOBIN on PATH. That is a
justfile change rather than a Go one, and it may be more machinery
than the problem deserves.

## Acceptance criteria

- [x] AGENTS.md names both install destinations, GOBIN for just install and the first writable of ~/.local/bin and ~/bin for install.sh, and says PATH order decides which one git ticket means.
- [x] The before-image entry beside the stale-binary gotcha tells the reader to confirm with which -a git-ticket before trusting an empty diff.
- [x] A decision is recorded either way on whether just install should warn when another git-ticket shadows GOBIN on PATH.

## Notes

**agent:terva/mieli** at 2026-09-05T23:12:03Z

Groomed on 2026-09-05, the same day it was filed, because the tree
moved under it twice in the hours since. There is no trigger clause on
this ticket: it was startable when filed and still is.

### All three criteria are still unmet, checked against the tree

AGENTS.md names GOBIN once, at line 295, inside the comment on the
`just install` line of the Commands block. It does not name
install.sh's destination anywhere and never says PATH order decides.
Criterion 1 stands.

The before-image entry is at AGENTS.md line 616 and now ends "Run it
before `just install`, which destroys the comparison." It still says
nothing about `which -a`. Criterion 2 stands, and the entry got more
specific since filing, which makes the missing caveat sharper rather
than softer.

No decision is recorded either way about `just install` warning on a
shadow. Criterion 3 stands.

### The description says two ways. There are three now

`just install-release` shipped in PR #139 after this ticket was filed,
and it resolves its destination with the same loop install.sh uses,
the first writable of ~/.local/bin and ~/bin. So the tree now has
`just install` at justfile:50 writing to GOBIN, `just install-release`
at justfile:106 and install.sh at install.sh:121 both writing to
~/.local/bin.

That cuts two ways. Two of the three paths now agree, so the hazard is
narrower than when this was filed: only the developer path diverges.
But the description's "two ways" is stale, and anything written for
criterion 1 has to name three destinations across three entry points.

### Criterion 3's open question is half answered by precedent

The description worried that a shadow warning "may be more machinery
than the problem deserves". `install-release` already implements one,
at justfile:160 to 164. It is five lines, it uses `command -v`, and it
was exercised against a decoy git-ticket placed earlier on PATH, which
named the decoy and printed the version it reported.

So the cost side of that question is settled: five lines, working, with
a copyable precedent. What is still a decision is whether `just
install` should adopt it, and one argument has appeared since filing.
`install-release` warns because it installs a release, where being
shadowed defeats the point. `just install` installs a working-tree
build for dogfooding, and a warning there fires on every install for a
developer who deliberately keeps two binaries.

### It recurred again, in a new shape

After tagging v0.11.1 the two copies disagree while being built from
the same commit. ~/.local/bin/git-ticket reports `v0.11.1
(c15770135cea)` and ~/go/bin/git-ticket reports
`v0.11.1-0.20260905224845-c15770135cea (c15770135cea)`, because
`just install` ran before the tag existed and `install-release` ran
after.

That is worth recording because it is not the staleness this ticket
was filed about. Both are current, both are the same commit, and they
still disagree about what to call themselves. A reader comparing
versions to decide which binary is newer would get no useful answer
here. The shadowing copy currently on PATH is the correct one, so
nothing is broken today.

**agent:terva/mieli** at 2026-09-05T23:29:21Z

Built. This note supersedes the argument in the grooming note above,
which came down against warning. Read this one instead.

### The decision criterion 3 asked for: warn

The grooming note argued that `install-release` warns because being
shadowed defeats the point of installing a release, while `just
install` installs a dogfooding build where a developer may keep two
binaries deliberately and would meet the warning on every install.
That reasoning is wrong on its own terms, and I put it to the user
rather than quietly reversing it.

The warning does not fire on every install. It fires only when the
copy just written is not the one that answers to `git ticket`, which
is exactly the case where the install did not do what the person
running it expects. A developer who deliberately keeps two binaries is
the person most in need of being told which one won, not least.

The user settled it on 2026-09-05: warn. The supporting points were
that `install-release` already does it in five lines, and that `just
install` already warns when GOBIN is not on PATH at all, so this is
the same family of check rather than a new kind of noise.

### The description says "It does not need code"

That is now false. The answer to criterion 3 was to add the warning,
so the change is five lines in the `install` recipe beside the
existing PATH check, sharing the `dest` it already computes.

The existing "is not on PATH" warning moved to stderr in the same
edit. It was going to stdout, `install-release` already sends its
equivalent to stderr, and a warning on stdout is a small bug worth
fixing while the line was open.

### What was run

With a real shadow, which this machine has: `just install` wrote to
/home/sothr/go/bin/git-ticket and warned that
/home/sothr/.local/bin/git-ticket comes first on PATH, naming the
version it reports, `v0.11.1 (c15770135cea, go1.26.2)`. Exit 0.

With no shadow, GOBIN placed first on PATH: the install printed its
destination and nothing else. Exit 0. No decoy was needed for either
case.

### The documentation half

The before-image entry gained a paragraph telling the reader to
confirm with `which -a git-ticket` before trusting an empty diff, and
saying why empty is the dangerous answer rather than the reassuring
one.

A second paragraph names all three installers and where each writes,
which is one more than the description knew about, since
`install-release` shipped after this was filed. It also records that
two copies built from the same commit can disagree, one reporting a
pseudo-version and one the tag, so comparing version strings does not
tell a reader which is newer.

## Summary

Shipped, on both halves it asked for.

`just install` now warns when the copy it wrote is not the one that
answers to `git ticket`, naming the winner and the version it reports.
That was criterion 3's open decision, settled with the user as warn.
The warning fires only when the install did not change what
`git ticket` means, which is the case worth hearing about every time
it is true, and it reuses the five-line shape `install-release`
already proved. The neighbouring "is not on PATH" warning moved to
stderr in the same edit, where it belonged.

AGENTS.md gained two paragraphs beside the stale-binary gotcha. The
first tells the reader to confirm with `which -a git-ticket` before
trusting an empty diff from the before-image technique, because that
technique assumes `git ticket` is the binary installed before the
branch and an empty diff is exactly the answer it hopes for. The
second names all three installers and their destinations, GOBIN for
`just install` and the first writable of ~/.local/bin and ~/bin for
`install.sh` and `install-release`, says PATH order alone decides
between them, and notes that `self-update` upgrades only the binary
that ran it.

It also records something the ticket did not know when it was filed:
two copies built from the same commit can disagree about their
version, one reporting a pseudo-version and one the tag, depending on
whether the tag existed at build time. So comparing version strings
does not tell a reader which binary is newer.

The description's claim that this "does not need code" did not
survive the decision on criterion 3, and a note says so.
