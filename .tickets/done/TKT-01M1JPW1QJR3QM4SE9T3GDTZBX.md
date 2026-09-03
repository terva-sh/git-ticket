---
schema: 1
id: TKT-01M1JPW1QJR3QM4SE9T3GDTZBX
title: Build the ticket-file merge driver
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - format
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-03T04:01:44Z
updated_at: 2026-09-03T05:36:15Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Plan 7.5 settles the design and this builds it. Read 7.5 first; it holds the field-by-field rules and the reason each one is what it is.

The shape: a `git ticket merge-driver` command taking Git the three stages it passes a driver, resolving what 7.5 says is resolvable, and writing conflict markers for the rest. It exits 0 when it produced a clean file and non-zero when it left a conflict, which is the contract Git expects.

The driver stays inside the 7.4 promise. Git invokes it and it runs no Git command itself, so the command table does not grow. Confirm that with `TestGitCommandsAreReadOnly` rather than by reading.

### What settling the spike established

Two agents editing one ticket conflict, and mostly not over anything either of them typed. Branch A setting `priority` and branch B adding a label conflict on `updated_by` alone, because every mutation rewrites that field and `updated_at` beside it. Git merges the priority and the label without complaint. `Notes` and `Comments` are the second source: two appends land at the same offset and collide as one hunk though the entries do not disagree.

So the first useful driver resolves the provenance stamps and the append-only sections and adjudicates nothing else. That clears the conflicts a real workflow hits without the driver ever choosing between two peoples edits.

### The install half, which is the harder half

A driver needs a `.gitattributes` entry and a `merge.*.driver` config line. The first is a tracked file, so `init` can write it and a clone gets it. The second is not, and Git refuses on purpose to take an executable name from a repository, because a clone would then run a command the person cloning never chose.

That boundary is not a problem to route around. It means the config line is a persons decision every time, so the job is to make it one command they run rather than a paragraph they transcribe. Decide whether that command is `git ticket install-merge-driver` or documentation, and whether `init` writing `.gitattributes` for a driver nobody has configured is help or litter.

### Watch for

A driver runs on a file mid-merge, which is the one state `parse` reports as `merge_conflict` rather than parsing. The driver reads the three stages Git hands it, each of which parses cleanly, so it must not be built on the conflicted working-tree file.

Round-tripping matters more here than anywhere. The driver writes a ticket file, so its output goes through the same renderer as every other write, or a merge silently reformats a file and the next diff is noise. 5.3 renders an absent scalar as `null` rather than omitting it, and a driver that forgets that produces a file that parses and does not match.

## Summary

Built. ticket.Merge does the three-way merge of plan 7.5, git ticket merge-driver is what Git invokes, and git ticket install-merge-driver plus init writing .gitattributes are the two halves of installing it.

The proof is TestMergeDriverUnderRealGit, which reproduces the spike case by running git rather than reasoning about it: branch A sets priority, branch B adds a label, and the merge now resolves. Pointing .gitattributes at merge=text instead of the driver brings back the exact conflict the spike reported, which is what says the driver is doing the work.

Both install questions went to the user. They chose a command over documentation, and init writing the attribute over leaving it out. The reasoning for the second is that the attribute is inert until somebody configures a driver by that name, so a repository that never installs one pays two lines and every clone that does install one is spared knowing the pattern.

One thing in this ticket turned out wrong. It says the driver stays inside the 7.4 promise so the command table does not grow, and that holds for merge-driver, which runs no git at all. install-merge-driver does: it sets two keys with git config, so 7.4 grew its first writing row. The exception is stated narrowly there, because it touches .git/config alone and runs from that one command.
