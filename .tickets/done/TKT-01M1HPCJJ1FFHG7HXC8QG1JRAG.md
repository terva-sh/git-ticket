---
schema: 1
id: TKT-01M1HPCJJ1FFHG7HXC8QG1JRAG
title: Decide whether a query reads tickets from other branches
type: spike
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - question
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references:
  - ref: review:backlog-md
    path: docs/review-backlog-md.md
claim: null
archive: null
created_at: 2026-09-02T18:34:03Z
updated_at: 2026-09-04T21:28:20Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

A deferred question raised by reading Backlog.md, and the one place their design is ahead of ours on a problem we actually have. See section D of docs/review-backlog-md.md.

Every query reads the working tree and nothing else, so a ticket claimed and moved to in-progress in another worktree is invisible here until the branch merges. AGENTS.md describes exactly that situation: several agents work this repository at once, in separate worktrees, each holding a main it believes is current.

Backlog.md reads the ticket directory out of every branch touched in the last 30 days, using git ls-tree per branch and comparing last-modified times, and shows the newest state it finds. src/core/cross-branch-tasks.ts is the whole mechanism and it is small.

Everything that read needs is read-only, so plan 7.4 permits the shape, but git ls-tree and the git log invocation behind the modification times are not in the 7.4 table and would have to be added there with their reasons before any code ran them.

Do not copy their conflict rule. Two branches that both touched a ticket are resolved by file mtime, and taskResolutionStrategy is a configuration knob choosing which guess to make, most_recent or most_progressed. A claim block has an actor and a timestamp and answers the question properly, which is an advantage we have and they do not.

The trigger: two agents in this repository claim the same ticket because neither could see the other claim. That has not happened, and the PR rule in AGENTS.md is what has been absorbing the pressure so far.

Open beyond the mechanism: whether cross-branch reads are a flag on list and ready, or a store setting, and whether the result marks which branch a ticket came from so a caller can tell a local ticket from a remote one. Answering with a merged view and no provenance would let an agent claim a ticket it cannot see the file for.

## Implementation plan

Drafted here rather than in plan 7.4, because that table is titled "The Git
commands this code runs" and nothing runs these yet. 7.4 sets the precedent
itself for the merge driver: it runs no Git command, "so the table does not
grow".

The practical reason is the guard. `planAllowedGitCommands` builds its allowlist
by reading 7.4, and `TestGitCommandsAreReadOnly` only checks code against that
list, never the reverse. A row with no caller passes and widens the allowlist,
so landing these rows early would mean the cross-branch change arrives to a
green guard and never gets the review the test exists to force. These move into
7.4 in the same commit as the code that calls them, and not before.

### The rows, as drafted

| Command | What reads it |
|---|---|
| `ls-tree` | the `.tickets/` tree as it stands on another branch, for a cross-branch query. It reads a named tree object, never the working tree and never the index |
| `for-each-ref` | the branches worth reading and the last commit date of each, which is the recency filter a cross-branch query applies before it reads any tree |

### Why for-each-ref and not log

The ticket and section D of `docs/review-backlog-md.md` both name `git ls-tree`
and `git log -1 --format` as the two additions. Checking the mechanism against
this repository says otherwise, on two counts.

Backlog.md "lists branches touched in the last 30 days" and neither document
names the command that does the listing. That is an uncounted third addition.
`rev-parse` is already in the table but answers with SHAs alone, so it cannot
serve. `git for-each-ref --format='%(refname:short) %(committerdate:iso8601)'`
answers with the name and the date together in one call.

Having that date makes `log` redundant for the recency filter. Its other use in
Backlog.md is resolution: comparing per-file modification times to decide which
branch holds the newest copy. This ticket already declines that, because a
`claim` block carries an actor and a timestamp and answers properly where an
mtime guesses. Running `git log -1` here confirmed the shape of the guess: it
returns `2026-09-02T19:20:31-05:00 Drew Short`, a committer identity rather than
an actor, which is the same collapse the actor field exists to avoid.

So `log` earns no row until something needs it. If a use appears, it takes its
own row and its own reason.

### What ls-tree has to read

Not `.tickets/tickets/`. The store partitions by status into `draft/`,
`tickets/`, `done/` and `archive/`, so a file's directory is its status. A
cross-branch read that scans the working set alone would miss a ticket another
agent left in `draft/` and, worse, would read a ticket as unchanged when what
changed was the directory it sits in.

This is not hypothetical here. `git ls-tree -r --name-only main -- .tickets/tickets/`
returns nothing at all on `main`, because every ticket in this store is draft,
done, or archived. A cross-branch query built against the working set would have
answered empty and looked like it worked.

So the read is the whole `.tickets/` subtree per branch, and the status of a
cross-branch ticket comes from the path it was found at.

### Still open, and gating the code

The design questions in the description are untouched by this and still decide
the shape: whether cross-branch reads are a flag on `list` and `ready` or a
store setting, and whether a result marks the branch it came from. The second
matters more than it looks. A merged view with no provenance lets an agent claim
a ticket whose file is not in its working tree.

One more surfaced while drafting. Other agents push to `origin`, so their work
is on remote-tracking refs rather than local branches, and `for-each-ref` over
`refs/heads/` alone would not see it. Whether the query reads `refs/heads/`,
`refs/remotes/`, or both is a decision, and it changes what the row's reason has
to say.

## Notes

**agent:terva/mieli** at 2026-09-03T06:01:30Z

Groomed. The trigger has not fired: no two agents in this repository have claimed the same ticket blind. The PR rule in AGENTS.md is still absorbing the pressure.

Worth saying plainly, because the timing invites the wrong conclusion. The merge driver shipped today, TKT-01M1JPW1, and it does not touch this. It resolves two edits to one ticket file once Git is already merging them. This ticket is about a claim being invisible until that merge happens, which is the window before the driver has anything to do. Closing this as covered by 7.5 would be a mistake.

One thing did move in this ticket favour. Plan 7.4 grew its first row today, for `config`, so the table is no longer a closed set and there is a worked example of what adding a row costs: the command, its reason, and the narrowest true statement of what it touches. `ls-tree` and `log` would each need that, and both are honest reads.

**agent:terva/mieli** at 2026-09-04T21:07:57Z

in-progress to blocked: the three design decisions in the Implementation plan are the user's to make, and they gate any code

**agent:terva/mieli** at 2026-09-04T21:07:57Z

Parked rather than left in progress, because the claim would otherwise sit on a
branch that no longer exists after this merges, and a held claim blocks another
agent from a ticket nobody is working.

The first step is done. The two Git command rows are drafted in the
Implementation plan, and drafting them against real output changed what they
are: `for-each-ref` in place of `log`, plus the finding that `ls-tree` has to
read the whole `.tickets/` subtree because status is the directory.

Three decisions gate the code, and all three are the user's:

Whether cross-branch reads are a flag on `list` and `ready` or a store setting.

Whether a result marks the branch it came from. A merged view without
provenance lets an agent claim a ticket whose file is not in its working tree,
which is a worse failure than not seeing the ticket at all.

Whether the query reads `refs/heads/`, `refs/remotes/`, or both. Agents here
push to `origin`, so their work is on remote-tracking refs and a scan of local
heads alone would miss exactly the case this ticket exists for. This one changes
what the `for-each-ref` row's reason has to say, so it is not purely an
implementation detail.

The original trigger still has not fired: no two agents in this repository have
claimed the same ticket blind. The PR rule in AGENTS.md is still absorbing the
pressure.

## Summary

Decided. Plan section 8 carries the query surface under "Reading tickets from
other branches", 10.1 carries the `branch` field, 12.4 carries the compatibility
note, and the section 15 entry is now a pointer at 8 rather than a list of open
questions.

The three answers. `--cross-branch` is a flag on `list` and `ready` rather than
a store setting, because it asks a different question than "what is in my tree"
and the caller is the one who knows which they want. Every row names the ref its
winning copy came from, null for the working tree, so a merged view cannot let
an agent claim a ticket whose file it cannot open. The scan reads `refs/heads/`
and `refs/remotes/` both, because an agent here pushes to `origin` and a scan of
local heads alone would miss the exact case this exists for.

A fourth question fell out of the third and was not in the original list: what
happens when a ticket exists in the working tree and on a scanned ref with
different content. 7.5 already answers it twice, so following it was cheaper
than inventing a rule. The later `updated_at` wins for display, which is what
the merge driver does, so a listing cannot predict the wrong outcome of the
merge it is warning about. A claim is never adjudicated, also 7.5, so a live
claim on any scanned ref makes the ticket not ready.

That also sharpened why Backlog.md's rule is declined. It was taste before, and
it is now a fact: they resolve by file mtime and offer a knob to pick which
guess to make, where `updated_at` is a field every mutation writes. This format
reads where theirs guesses.

One thing stated during the design turned out to be unnecessary and was cut
before it reached the plan. The working tree does not need to win as a special
rule, because an uncommitted edit already rewrites `updated_at` and so already
carries the newest timestamp. It is one source among the refs, with a tie
breaking its way because it is the copy the caller can act on.

Plan 7.4 did not grow. The rows stay drafted on this ticket until the code runs
them, for the reason recorded in the Implementation plan: the guard reads its
allowlist from that table and checks code against it in one direction only, so a
row with no caller widens the allowlist and skips the review the guard exists to
force.

Implementation is `TKT-01M1Q54BRW1NB406RXAKWRF2XP` (Build cross-branch reads for
list and ready), filed as a draft and depending on this one. The trigger still
has not fired: no two agents here have claimed the same ticket blind.
