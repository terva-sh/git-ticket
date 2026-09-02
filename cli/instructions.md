## Tickets

Work is tracked as Markdown tickets in `.tickets/`, managed with `git ticket`.
Run `git ticket help` for the full command list.

### Finding work

`git ticket ready` lists what is open, unblocked, and has every dependency
closed. That is the queue. `git ticket list --status in-progress` shows what is
already underway, and `git ticket search PATTERN` takes a regular expression
over the title and body when you only roughly know what you are after.

Read the whole ticket with `git ticket show ID` before you start, including its
acceptance criteria and its dependencies. Anywhere an ID is taken, a unique
prefix works, with or without the `TKT-` part.

Before you change a file, `git ticket files PATH` lists the tickets that
recorded a reference to it. It reports what other agents wrote, so it is only as
complete as they were, and nothing derives it from Git history.

### Doing the work

A ticket you just filed is in `draft`, and a draft cannot be claimed. Move it
with `git ticket status ID ready` first. One you took off the queue is already
there.

Then `git ticket claim ID`, and `git ticket status ID in-progress`. A claim
records who is working, on which branch, from which commit. It is advisory and
reserves nothing, so it tells another agent what is in flight rather than
locking anything.

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

### Filing new work

`git ticket create --title "..." --type bug --priority high` files a ticket.
Types are task, bug, chore, spike, and epic. Add `--description`, `--label`,
and `--assignee` as they apply, and `--parent` to file it under an epic.

It lands in `draft`, which keeps something you filed on the way past other work
out of the queue until somebody decides it belongs there. Move it on with
`git ticket status ID ready` when it does.

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
