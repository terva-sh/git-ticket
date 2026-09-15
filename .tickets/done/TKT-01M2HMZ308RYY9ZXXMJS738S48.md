---
schema: 2
id: TKT-01M2HMZ308RYY9ZXXMJS738S48
title: Split the agent block and fix what the audit found
type: task
status: done
status_reason: null
priority: high
due_on: null
labels: []
assignees: []
milestone: null
parent: null
origin: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-15T04:24:54Z
updated_at: 2026-09-15T04:35:44Z
created_by:
  id: agent:terva/instructions-core
  name: ""
updated_by:
  id: agent:terva/instructions-core
  name: ""
extensions: {}
---

## Description

The agent workflow block is 2,345 words, and `instructions --write` puts all of
it in a consuming project's AGENTS.md, where it is loaded in every session. It is
the largest always-on context cost this tool imposes, and most of it is
justification rather than instruction: the `--actor` section spends twenty lines
to deliver one rule.

An audit for usability, leaked assumptions, and prose quality found the
following.

### One factual error

The block says `git ticket search PATTERN` "takes a regular expression over the
title and body". It does not. `search QUERY [--regex]` is substring by default,
so an agent that follows the block writes a regex, gets "Nothing matches" and
exit 0, and concludes the ticket is not there. A wrong answer that looks like a
right one.

TestInstructionsNameRealCommands cannot catch this. It checks that every command
and flag named is real, and `search` is real and the sentence names no flag, so
the test passes while the prose misdescribes the command.

### References that only make sense in this repository

The block ships to other projects and carries three things that do not travel:
the actor example `agent:terva/mieli`, where terva is a sibling project; the
naming example "Build git ticket remove, per plan 9.1", where plan 9.1 is this
repository's own design document; and `main..fix` in the export example.

### Process the block assumes a reader has

"When the store check fails" opens with "CI verifies the store with
`git ticket check --fix --dry-run --strict`", asserting that the consuming
project has CI and runs that command. It then compares the behaviour to a CI
that reports unformatted code. A project adopting git-ticket may have neither.

### A section in the wrong place

The paragraph describing how `show` compacts note history sits under "Filing new
work". It is about reading.

### Prose

Clean. A mechanical pass for AI tells found no em dashes, no AI vocabulary, no
filler, no abstract metaphor nouns, and sentence-case headings throughout. Two
real hits: "One you took off the queue is already there" is compressed past
readability, and three mid-sentence colons would read better as full stops.

### What to build

Split the block. A short operational core goes in AGENTS.md and the full
reasoned version stays behind `git ticket instructions`, which is the same move
`show` just made for note history: what is needed every time stays, what is
needed rarely is one command away. The core names the command that prints the
full version, so the reasoning is discoverable from the summary.

Discovery gets three cheap additions, settled with the user: the help line for
`instructions` describes it as a setup command rather than as the guide to
working a store, `init` says nothing about it at all, and neither block names
itself. Hinting on refusals was considered and declined.

## Acceptance criteria

- [x] instructions --write installs a short form; a bare instructions prints the long one; --core and --full override either default and passing both is refused
- [x] The short form names git ticket instructions, so the long form is reachable from the file without anyone saying it exists
- [x] The search sentence says substring and names --regex, in both forms
- [x] Neither form names terva, plan 9.1, or a branch from this repository
- [x] The CI sentence no longer asserts the consuming project runs the check
- [x] The show/notes paragraph reads under finding work, not under filing it
- [x] init names the command when it was not asked to write the file, and the help line says what --write installs
- [x] Plan 12.1 and 10.5 carry the two forms and the new flags
- [x] just ci is green and every path is verified in a built binary

## Implementation plan

Two embedded files. instructions.md keeps the long form and gets the five audit fixes. A new instructions-core.md carries the short one, and --write installs that instead. Printing defaults to the long form and writing to the short, with --core and --full overriding either and both together refused. Discovery is three hints: init names the command when it was not asked to write the file, the short form names it in its own second paragraph, and the help line says what --write installs.

## Notes

**agent:terva/instructions-core** at 2026-09-15T04:35:28Z

Falsification found a hole in my own first cut. TestWriteInstallsTheShortForm compared the two embedded strings by length and checked both carried the markers, but never ran --write, so changing writeInstructions to install the long form left the suite green. TestInitWritesInstructionsOnlyWhenAsked does assert the written bytes equal instructionsCore, but it exercises init --instructions, which calls writeInstructionsFile directly and never goes through writeInstructions. Two paths write the file and only one was pinned. The test now runs instructions --write and compares the bytes exactly rather than with Contains, since a file holding the long form would contain every sentence of the short one.

**agent:terva/instructions-core** at 2026-09-15T04:35:28Z

Added TestTheSearchSentenceSaysSubstring for the factual error, because the gap it exposed is structural rather than an oversight. TestInstructionsNameRealCommands asks whether every command and flag the prose names exists, so prose that names a real command and describes it wrongly passes by construction. The new test does not try to verify prose against behaviour in general; it requires that any paragraph introducing git ticket search says substring, and that one calling it a regular expression also names --regex. Falsified by reverting the sentence to its original wording.

**agent:terva/instructions-core** at 2026-09-15T04:35:28Z

main..fix survived the first pass of the leaked-reference fix and was caught by running the built binary rather than by reading the diff. It is now main..BRANCH, matching the uppercase placeholder convention the rest of the block uses for ID, QUERY, DIR and N.

## Summary

The block is two embedded files. cli/instructions.md keeps the long form at 2,473 words and cli/instructions-core.md carries a 951-word short one, which is what --write installs. A bare instructions still prints the long form, because somebody typing the command wants the reasoning while a setup step filling a file read every session wants it short. --core and --full override either default, and passing both is a usage error rather than a silent precedence rule.

The short form drops the argument, not a step. It carries every rule and the whole sequence in working order, because a summary that leaves out a command leaves an agent doing that step wrong rather than uninformed.

Discovery is three hints, all of them cheap. init names the command in the sentence it prints when it was not asked to write the file, the short form names it in its own second paragraph, and the help line now says what --write installs instead of describing the command as the guide to working a store. Hinting on refusals was considered and declined.

The audit fixes: search is described as a substring match naming --regex, agent:terva/mieli became agent:yourtool/session-3, the plan 9.1 citation became a plain description, main..fix became main..BRANCH, the CI sentence no longer asserts the consuming project runs the check, the show and notes paragraph moved under finding work, and six mid-sentence colons became full stops.

Two new guards. TestWriteInstallsTheShortForm now runs --write and compares bytes exactly, closing a hole falsification found in my own first cut where neither write path was pinned. TestTheSearchSentenceSaysSubstring pins the factual error, which TestInstructionsNameRealCommands cannot catch because it only asks whether the commands named exist.

Plan 12.1 carries why the two forms exist, what the short one keeps, the two defaults, and the three discovery hints. 10.5 says the envelope holds whichever form the flags selected and why it does not name which.

just ci is green. All six paths were verified in a built binary in scratch repositories: both prints, the refusal, --write, --write --full, the refresh back down to core, and init with and without --instructions.
