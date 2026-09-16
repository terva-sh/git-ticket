---
schema: 2
id: TKT-01M2NHGHTZ4PWHBKRHE43XJG8D
title: Add a doctor command for ticket hygiene
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
references: []
claim: null
archive: null
created_at: 2026-09-16T16:41:30Z
updated_at: 2026-09-16T16:49:47Z
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

- [ ] Hygiene rules are reported by level, highest first
- [ ] check keeps answering only whether the store is valid, and doctor never fails a build by default
- [ ] Each finding names the ticket and says what would resolve it
- [ ] Rules are marked hard or soft, and a soft finding reads as a question rather than a verdict
- [ ] A soft rule can never be what makes the command exit non-zero
- [ ] Every ticket carries at least one label ships as the first hard rule
- [ ] Labels are ordered most-descriptive first ships as the first soft rule
- [ ] Rules ship on by default and a store that configures nothing gets them
- [ ] A store turns a rule off, changes its level, or sets its parameters in .tickets/config.yml
- [ ] Every rule has a stable identifier that configuration refers to
- [ ] Configuring shipped rules does not foreclose a store defining its own later

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
