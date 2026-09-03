<!-- git-ticket:begin -->

## Tickets

Work is tracked as Markdown tickets in `.tickets/`, managed with `git ticket`.
Run `git ticket help` for the full command list.

### Finding work

`git ticket ready` lists what is open, unblocked, and has every dependency
closed. That is the queue. `git ticket list --status in-progress` shows what is
already underway, and `git ticket search PATTERN` takes a regular expression
over the title and body when you only roughly know what you are after.

`git ticket list` answers with open work: every status except `done` and
`archived`. Naming a status brings it back, so `git ticket list --status done`
works, and `git ticket list --all` drops the exclusion. Search is the exception
and spans everything, because finding what was already decided means reading a
done ticket.

A short queue does not mean there is little to do. Everything filed lands in
`draft` and stays there until a person promotes it, so
`git ticket list --status draft` is the rest of the backlog, and it is usually
the larger half. Read it before you report that there is nothing to pick up.

Do not promote a draft yourself. Name the ones that look startable, say what
makes each one startable, and let the person you are working with choose.
Promotion is where somebody weighs this work against everything you cannot see,
so an agent that promotes its own next ticket has appointed itself.

Read the whole ticket with `git ticket show ID` before you start, including its
acceptance criteria and its dependencies. Anywhere an ID is taken, a unique
prefix works, with or without the `TKT-` part.

Before you change a file, `git ticket files PATH` lists the tickets that
recorded a reference to it. It reports what other agents wrote, so it is only as
complete as they were, and nothing derives it from Git history.

### Doing the work

Work starts from a ticket that is `ready`, and a draft cannot be claimed. If you
were asked to pick up something still in `draft`, that request is the promotion:
run `git ticket status ID ready` first and carry on. One you took off the queue
is already there.

Then `git ticket claim ID`, and `git ticket status ID in-progress`. A claim
records who is working, on which branch, from which commit. It is advisory and
reserves nothing, so it tells another agent what is in flight rather than
locking anything.

Then read the code and write the approach into the ticket with
`git ticket plan ID "..."`. Do that after you claim rather than when you file
the ticket. A plan written before anybody read the code is a guess, and it goes
stale between filing and starting. It replaces rather than appends, so revising
it when the approach changes leaves one plan rather than a stack of them.
Somebody who wants to redirect you reads it before there is code to throw away.

While you work:

- `git ticket note ID "..."` records what the next person will need and does
  not have. A note travels with the ticket, so it reaches whoever picks this up
  after you.
- `git ticket ac ID --check N` ticks an acceptance criterion. N counts checkbox
  lines from one, not array positions.

Finish with `git ticket summary ID "..."` saying where it landed, then
`git ticket status ID done` and `git ticket release ID`.

If you cannot proceed, `git ticket status ID blocked --reason "..."`. The
reason is required, because a blocked ticket that does not say why tells the
next person nothing.

### When the store check fails

CI verifies the store with `git ticket check --fix --dry-run --strict`. That
plans every repair, prints what it would do, writes nothing, and exits 1 when
one is pending. A ticket under the wrong filename, a ticket in the directory its
status does not imply, and a stale `.tickets/epics.md` all land there.

Run `git ticket check --fix` and commit what it changed. CI reports the repair
and never commits it for you, the same way it reports unformatted code rather
than reformatting it behind your back.

### Filing new work

`git ticket create --title "..." --type bug --priority high` files a ticket.
Types are task, bug, chore, spike, and epic. Add `--description`, `--label`,
and `--assignee` as they apply, and `--parent` to file it under an epic.

It lands in `draft` and it stays there. That keeps something you filed on the
way past other work out of the queue, and the decision to promote it belongs to
a person. File it, say that you filed it, and go back to what you were doing.

When work blocks or belongs to other work, record it rather than leaving it in
prose:

- `git ticket link ID --depends-on OTHER` says this ticket waits on that one. A
  dependency is satisfied when the other is done, or archived out of done.
- `git ticket link ID --ref proposal:name --path docs/plan.md` points at a
  document or an external record.
- `git ticket deps ID` walks the dependencies, `--transitive` follows the
  chain, and `--dependents` walks it backwards to what waits on this one.

Split a ticket rather than growing it. If you find work that is not what the
ticket asked for, file it and link it.

### Driving it from a script

Add `--json` to any command for one envelope on stdout with a stable error
`code` to switch on. `git ticket schema` prints the legal statuses, types,
priorities, transitions, and codes, so read those rather than hard-coding them.

Every write takes `--if-revision R` and refuses if the ticket moved since you
read it. Pass it whenever you read, decide, and then write, which is most of
what an agent does.

Text that opens with a dash goes after a bare `--`.

<!-- git-ticket:end -->
