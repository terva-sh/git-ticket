---
schema: 2
id: TKT-01M2NHGHTZ4PWHBKRHE43XJG8D
title: Add a doctor command for ticket hygiene
type: task
status: done
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
references: []
claim:
  actor: agent:claude/t3code
  branch: t3code/review-new-tickets
  worktree: /home/sothr/.t3/worktrees/git-ticket/t3code-6907238f
  commit: 7396b9e3ba043345142a80f2408575a110fd80d7
  claimed_at: 2026-09-16T18:21:09Z
  expires_at: null
archive: null
created_at: 2026-09-16T16:41:30Z
updated_at: 2026-09-16T18:29:49Z
created_by:
  id: agent:claude/t3code
  name: ""
updated_by:
  id: agent:claude/t3code
  name: ""
extensions: {}
---

## Description

`git ticket check` answers whether the store is *valid*: a ticket under the wrong filename, a ticket in a directory its status does not imply, a stale `epics.md`, an unresolved reference. Those are things that are broken.

Hygiene is a different question and nothing asks it. A ticket can be perfectly valid and still be one nobody can pick up: no acceptance criteria, no description beyond a title, `in-progress` with no claim, a claim that expired weeks ago, an epic with no children, a dependency on something already done, a draft untouched since it was filed. None of that is invalid and all of it costs the next reader time.

Add `git ticket doctor`: hygiene rules, reported by level, highest first, so the first thing printed is the thing most worth fixing.

`check` and `doctor` stay separate. `check` is what CI runs and what `--fix` repairs, and it must keep answering a yes-or-no question about validity. Hygiene is advice, it is going to be opinionated, and an untidy store should not fail a build.

### Hard rules and soft rules

The levels are not only about severity. They are about whether a rule can be measured objectively at all, and that difference decides what the command may claim.

A **hard rule** is checkable. *Every ticket carries at least one label* is either true or false for a given ticket, and the command can say so without qualification.

A **soft rule** is a judgement the command can prompt but cannot settle. *A ticket's labels should be ordered most-descriptive first* is the example: nothing mechanical knows which of `auth` and `ui` describes a ticket better. A soft finding should therefore read as a question rather than a verdict, and must never be what makes the command exit non-zero.

Keeping the two apart matters because a tool that reports a judgement in the same voice as a fact teaches people to skim both.

### Why label order is worth a rule at all

`git-ticket-canvas` shows the first two labels on a card, three when the cards are compact, and collapses the rest behind a `+N` disclosure. Label order therefore decides which labels somebody sees while glancing over a board, and a ticket whose most descriptive label sits fourth is a ticket that reads as something else entirely.

(The canvas does not currently colour a card by its first label; the card's colour comes from its status. If it ever does, this rule gets a second reason rather than a different one.)

### The two example rules, written out

- **Hard.** Every ticket carries at least one label.
- **Soft.** A ticket's labels are ordered with the most descriptive first.

A store whose `config.yml` enforces a label allowlist gives the hard rule something to say beyond presence: a label outside the set is already `label_unknown` in `check`, so `doctor` should not repeat it.

## Acceptance criteria

- [x] check keeps answering only whether the store is valid, and doctor never fails a build by default
- [x] Each finding names the ticket and says what would resolve it
- [ ] Rules are marked hard or soft, and a soft finding reads as a question rather than a verdict
- [x] Rules ship on by default and a store that configures nothing gets them
- [x] A store turns a rule off, changes its level, or sets its parameters in .tickets/config.yml
- [x] Every rule has a stable identifier that configuration refers to
- [x] Configuring shipped rules does not foreclose a store defining its own later
- [x] Hard findings are reported before soft ones, and hard or soft is the only axis a rule has
- [x] A doctor finding carries its rule identifier and level, in its own type, leaving check's four-key contract untouched
- [x] A rule identifier named in config that this binary does not know is reported rather than silently unused, by whichever of check or doctor is decided to own it
- [x] Whether doctor has a --json form is decided: either a published envelope kind or a recorded reason there is none
- [x] A soft rule can never be what makes the run fail, though it may put the command in the informational bucket
- [x] Rule identifiers and check's finding codes share one namespace, and schema publishes both
- [x] doctor --strict exits by a graded informational bucket: clean, soft findings only, and hard findings are distinguishable without parsing output
- [x] Plan 10.2 is edited to reserve doctor's numbers, which do not collide with self-update's 10 through 12, and that edit ships with the code

## Implementation plan

`ticket/doctor.go` carries the library half, beside `check.go`, because terva
consumes the library and a hygiene report a host cannot read is half a feature.
`cli/doctor.go` carries the command.

**Types.** `Level` is `hard` or `soft` and is the only axis. `DoctorFinding`
carries `Rule`, `Level`, `Ticket`, `File`, `Message` and `Remedy`: a separate
type from `Finding`, so check's four-key contract and its fixture sidecars are
untouched. `Remedy` is its own field rather than prose inside `Message` because
"says what would resolve it" is a criterion and a field can be asserted on.

**Registry.** A `Rule` is an ID, the level it ships at, a summary, and a func
over a `RuleContext` holding the tickets, the config, the rule's params and a
clock. Store-wide rules and per-ticket rules are then the same shape, which an
`epic with no children` rule needs and a per-ticket one does not mind.
`DefaultRules()` is the shipped set and is what makes a rule on by default.

**Config.** A `doctor.rules` map keyed by rule ID, each entry carrying
`enabled`, `level` and a nested `params`. `params` is explicit rather than
inline so a future key beside it is not ambiguous. The map stays open: an entry
naming something not shipped is reported, not refused, which is what keeps
TKT-01M2NJDNMG5SBB6CEXY186HESF's door open. Refusing would foreclose it.

**Doctor owns the unknown rule ID**, not `check`. `check` answers whether the
store is valid, and if it had to know the rule registry then a store would
become invalid the day a rule is renamed, which makes validity a property of the
binary rather than of the store. `label_unknown` and `unknown_series` compare
two things inside the store; a rule ID compares the store against this binary,
which is a different relation. It reports as `rule_unknown`, hard, since whether
an ID is in the registry is exactly the kind of thing a hard rule settles.

**Exit.** Without `--strict`, always zero. With it, the graded bucket: 0 clean,
20 soft findings only, 21 hard findings. 20 and 21 leave self-update's 10
through 12 alone and leave room between the two buckets. `exitStatusErr` already
exists and `Run` already maps it, so no new mechanism. Plan 10.2 gains the
reservation in the same change.

**`--json` ships**, as a `doctor-report` kind added to `envelopeKinds` and to
plan section 10. The alternative, recording that there is none, was rejected
because terva is the consumer that most needs this and `check-report` already
sets the shape. `schema` publishes the rule identifiers alongside the finding
codes they share a namespace with.

**The framework ships with no rules of its own.** The two that would prove it
belong to TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0, which depends on this. The machinery
is exercised by test-only rules registered through the same registry a shipped
rule uses, so nothing is untested; `rule_unknown` is the one finding a store
gets before that ticket lands.

## Notes

**agent:claude/t3code** at 2026-09-16T16:49:10Z

The ambiguity this was filed with is resolved. The note said "levels" where it meant **labels**, and the example decomposes into two rules rather than one:

- every ticket must carry one or more labels, which is objectively checkable;
- the labels should be ordered by how well each describes the ticket, which is not.

That second one is why rule levels exist at all, and it is a more interesting reason than severity. A soft rule is one nothing mechanical can settle, so the command may raise it and must not rule on it.

One correction to the reasoning, made while filing rather than left to be discovered. The rationale given was that the canvas colours a card by its first label. It does not: labels render as uniform chips and the card's colour comes from its status.

The rule survives on a better fact. `CardView` renders `labels.slice(0, 2)`, or three when cards are compact, and puts the rest behind a `+N` disclosure. Label order decides which labels are visible at a glance at all, which is a stronger argument than colour would have been.

**agent:claude/t3code** at 2026-09-16T16:49:47Z

**This is a rule system with shipped defaults, not a command with a fixed list.** That is a bigger thing than the description was written for, and it changes what has to exist first.

Every store is different. A store where one person works alone does not want to be told its tickets have no assignees; a store with an enforced label allowlist wants more said about labels than one with none; a store that files everything as draft on purpose does not want a finding about untouched drafts. A hygiene tool that cannot be told any of that gets its output ignored, and an ignored report is worse than no report because it trains people to skim.

So:

- **Rules ship with the binary** and are on by default, at the level the tool thinks right. A store that configures nothing gets the house opinion, which is the whole point of shipping an opinion.
- **A store overrides them in `.tickets/config.yml`**, beside the label and milestone allowlists that already live there. Turn a rule off, move it between hard and soft, or set whatever it takes as a parameter.
- **Every rule therefore needs a stable identifier**, because configuration refers to rules by name and a name in somebody's config file cannot be renamed casually. That is the part the current description does not account for and the part that constrains everything after it: identifiers are the compatibility surface of this feature.

### Authoring new rules is a separate question

Configuring shipped rules and letting a store define its own are not the same size. The first is a map in a config file. The second is an expression language, or a plugin mechanism, or shelling out — each with its own trust and portability problems, and a ticket store is a git repository somebody clones, so a rule that executes is a rule that executes on somebody else's machine.

Ship configuration of built-ins. Leave authoring to its own ticket, and do not design the config format in a way that forecloses it.

### What the defaults are for is worth saying

The default set is the tool's opinion about what a well-kept store looks like, which is the only reason to have a hygiene command rather than a linter people write themselves. Defaults that try to offend nobody say nothing. They should be opinionated enough to be worth overriding.

**agent:claude/t3code** at 2026-09-16T16:57:54Z

Groomed. Split before promotion, because the note above turned this from a
command into a rule system and a rule system does not fit one ticket. This
ticket keeps the framework: the command, the hard and soft levels and their exit
semantics, stable rule identifiers, and the `.tickets/config.yml` override
format. Identifiers and the config format are the compatibility surface, so they
are what has to be right first and what the carve-offs wait on.

Two acceptance criteria moved out. Their original wording, verbatim, so a reader
does not have to run `git log` to learn what was asked here:

- [ ] Every ticket carries at least one label ships as the first hard rule
- [ ] Labels are ordered most-descriptive first ships as the first soft rule

Both now live on TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0, which records this ticket as
its origin and depends on it. The "Hard rules and soft rules", "Why label order
is worth a rule at all" and "The two example rules, written out" sections of the
description above are the argument for that ticket rather than for this one, and
were copied there. They stay here because the hard-versus-soft passage is also
the justification for the level system this ticket builds.

This supersedes the "Authoring new rules is a separate question" heading in the
note above, which said to leave authoring to its own ticket. That ticket now
exists: TKT-01M2NJDNMG5SBB6CEXY186HESF, a spike, also with this one as its
origin. The last criterion here, that configuring shipped rules does not
foreclose a store defining its own, is the seam between them.

Still unassigned to any ticket: the hygiene conditions the first paragraph of
the description enumerates, which never became criteria anywhere. No acceptance
criteria, no description beyond a title, `in-progress` with no claim, an expired
claim, an epic with no children, a dependency on something already done, a draft
untouched since it was filed. Those are the opinionated default set the second
note argues for, and they want their own ticket once the framework has shipped
and the config format has been exercised by the two rules that go first.

**agent:claude/t3code** at 2026-09-16T17:30:50Z

Reviewed against the tree before promotion, with the user, and three decisions
were taken. Each closes an ambiguity that would have been discovered in code.

**Hard and soft are the only axis. There is no separate severity.** "Reported by
level, highest first" therefore means every hard finding before every soft one,
and a store that "changes a rule's level" is moving it between the two. The
alternative was a severity attribute ordering the report independently, and it
was declined because `check` already spends that vocabulary: `Report` is
`Errors` and `Warnings` with `--strict` promoting one to the other, so a second
command using "level" and "warning" for a different idea would teach a reader
that the two words mean whatever the command they are in wants.

The first criterion said, verbatim:

- [ ] Hygiene rules are reported by level, highest first

It is reworded below to name the two levels, because with exactly two of them
"highest" was a comparative with nothing to compare.

**Doctor gets its own finding type. `Finding` is not reused.** `ticket/check.go`
marshals exactly four keys, and the comment there is explicit that the fixture
sidecars record those four and a fifth "rewrites the corpus for something no
consumer asked for". A doctor finding has to carry at least a rule identifier
and a level, so reusing that type means either growing a recorded contract or
smuggling the rule ID into `Code`. A second type that resembles the first is the
cheaper of the three, and it keeps `check`'s corpus untouched.

**Rule identifiers follow the `Finding.Code` convention**, which is what
`label_unknown`, `location_mismatch`, `unknown_series` and `origin_missing`
already look like. Whether the two share one namespace is still open, and it is
worth settling in the same change: sharing means a doctor rule can never take a
name `check` might want later, and separating means a reader has to know which
command a code came from.

### Two gaps this review found, now criteria

**An unknown rule ID in config is silently unused today.** `ParseConfig` calls
`yaml.Unmarshal` with no `KnownFields`, so unknown keys are ignored, and a
`rules:` map keyed by rule ID accepts any key at all. A store that configures a
rule this binary does not know gets no rule and no complaint. `check` already
has `label_unknown` and `unknown_series` for exactly this shape, so the pattern
exists; what is undecided is which command reports it, since `check` owns
validity and `doctor` owns hygiene and a misspelled rule name is arguably both.

The good news, measured rather than assumed: the block needs no schema bump. An
older binary ignores the unknown top-level key, and an older binary has no
`doctor` to misreport with.

**`--json` needs a section 10 ruling.** `envelopeKinds` in `cli/commands.go` is a
closed list guarded in both directions by `envelopekinds_test.go`, so a
`doctor-report` kind is a contract addition and not a detail. The other honest
answer is the one `note --list` took: ship with no JSON form and record why.

### Left alone deliberately

The second criterion carries two ideas, that `check` stays a validity question
and that `doctor` does not fail a build by default. They were not split, because
they are one separation stated from both sides. But "by default" implies a
`--strict` of doctor's own that nothing else in this ticket mentions, and
whoever builds it should either add that flag deliberately or drop the words.

**agent:claude/t3code** at 2026-09-16T18:16:39Z

Two more decisions, taken with the user, closing the questions the review above
left open.

**Rule identifiers and check's finding codes share one namespace.** The reason is
that doctor referring to a check code is not hypothetical: the first hard rule
already has to know about `label_unknown`, because
TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0's fourth criterion tells it to report label
presence and not restate what `check` already says about membership. A rule that
must name another command's code to define its own boundary is a rule living in
that command's namespace already. Two namespaces would have made that reference
ambiguous in the one place it matters.

The cost is accepted rather than overlooked: a doctor rule can never take a name
`check` might want later, and `schema` publishes the whole space, so every
identifier on either side is spent once and permanently.

**`doctor --strict` exits by a graded informational bucket, not by 0 and 1.**

Plan 10.2's default is one bit, with detail left to the envelope, and it says so
in as many words. But it already reserves one exception, 10 through 12 for
`self-update --check`, and gives the test a new bucket has to pass: the
precedent is "the reserved informational bucket of zypper and terraform's
`-detailed-exitcode`, not fsck's bitmask", because a grade is one ordered
category and a mask is several independent ones.

Doctor's grade is the worst level that fired. Clean, soft findings only, and
hard findings are three states of one ordered category, which is a grade and not
a mask, so it passes the test 10.2 sets. What CI gets from it is the thing a
single bit cannot give: telling a store with nothing to say from one with only
questions, without parsing anything.

This is a plan change and not a command's own choice. AGENTS.md is explicit that
admitting a row a plan section does not have is a maintainer's decision, per the
format-patch ruling in section 15. The maintainer made it here. It is recorded
as a criterion below so the edit to 10.2 ships with the code rather than after
it, and the reserved numbers must not collide with self-update's 10 through 12.

**The fourth criterion had to be reworded for it.** Verbatim, as it stood:

- [ ] A soft rule can never be what makes the command exit non-zero

That was written when 0 and 1 were the only outcomes, where "exits non-zero" and
"fails" were the same sentence. With a bucket they are not, and the distinction
is one this codebase already draws: `cli/cli.go` comments at the self-update
branch that in the graded bucket "the command answered", which is the opposite
of a failure. A soft finding may put the command in the informational bucket.
It still may not fail the run.

**agent:claude/t3code** at 2026-09-16T18:29:44Z

Built. `ticket/doctor.go` is the library half beside `check.go`, `cli/doctor.go`
the command, and the rule identifiers are published by `schema`.

**The third criterion ships unticked, honestly.** The marking half is done: a
rule declares a level, a store can move it, and the level it runs at reaches the
rule through `RuleContext.Level` so a rule moved to soft can phrase itself
differently. `TestAStoreChangesARulesLevel` proves the value arrives. What
cannot be ticked is "a soft finding reads as a question rather than a verdict",
because no soft rule ships: the first one is
TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0's, and a box ticked on a mechanism nobody has
used would be a false one.

**One nuance on the second criterion, ticked but worth stating.** Every finding
carries a remedy, and a finding about a ticket names it. `rule_unknown` is about
`config.yml` rather than about any ticket, so it names the file instead and the
human output prints a dash in the ticket column. "Names the ticket" is
unsatisfiable for a store-level finding; naming what it is about is the version
that is true of all of them.

**A data-loss hazard turned up that was not in the ticket.** `migrate` and
`series add` both rewrite `config.yml` through `RenderConfig`, which emits
field by field, so a block the emitter does not know is a block those commands
delete. A store that turned a rule off would have found it on again after an
unrelated `series add`, with nothing said. `RenderConfig` now renders the doctor
block back, params included, through a structural renderer rather than a typed
one, because params are the one part of that file this package deliberately has
no type for. Proven twice: `TestRenderConfigKeepsTheDoctorBlock` round-trips it,
and a real `series add` against a scratch store was run and the block survived.

**The framework ships with no rules of its own**, which is what the split left
here. Every level, grade, override and ordering below is exercised by rules
registered in tests through the same `Rule` shape a shipped rule uses, and
`cli/doctor.go` carries a `doctorRules` seam for the same reason. `rule_unknown`
is the one finding a real store can get today. `doctor` on this repository's own
store prints "Nothing to tidy." and exits 0, which is honest and will stay true
until TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0 lands.

**Numbers.** 20 is soft-only, 21 is hard, leaving self-update's 10 through 12
alone with a deliberate gap so neither bucket has to move if either grows. Plan
10.2 now reserves them and says why a grade rather than a mask, and section 10.3
gained a `doctor-report` subsection rather than a new numbered section, so
nothing downstream renumbered.

## Summary

git ticket doctor ships: a rule framework with hard and soft as its only axis, stable identifiers sharing check's namespace and published by schema, per-rule configuration in .tickets/config.yml for enabling, level and params, a doctor-report envelope kind, and --strict exiting by a graded bucket reserved in plan 10.2 at 20 for soft and 21 for hard. check is untouched. No rules ship yet, by the split: the first two are TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0, so the framework is exercised by test-registered rules and the third criterion is honestly unticked. RenderConfig now writes the doctor block back, which stops series add and migrate silently deleting a store's rule configuration.
