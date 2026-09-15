# Handoff to ledger: moving a ticket between your own stores

For whoever maintains `docs/workspace-layout.md` in the ledger repository.
`TKT-01M2H6TM1T2W0WF99HMDZCM520` asked for a paragraph there describing how to
move a ticket between stores. That file is not this repository's to write, so
the paragraph is here to carry over.

The behaviour it describes ships in git-ticket as `import --same-owner`, in
`v0.18.0`. Check that ledger installs that version or later before pasting, with
the command at the bottom.

## The paragraph, to paste into `docs/workspace-layout.md`

> **Moving a ticket to another store.** Export it from the store it is in, then
> adopt it into the store it belongs in with `--same-owner`, which tells
> git-ticket that both stores are yours and the ticket's evidence is your own:
>
> ```sh
> # in the store the ticket is leaving
> git ticket export TKT-OLD --out /tmp/move
>
> # in the store it is joining
> git ticket import /tmp/move --same-owner --from-store ledger          # preview
> git ticket import /tmp/move --adopt --same-owner --from-store ledger  # write
> ```
>
> The ticket is filed under the receiving store's series, so it gets a new ID.
> Its acceptance-criteria and definition-of-done ticks travel as you left them, a
> ticket that was `done` or `archived` arrives that way, and it keeps the instant
> it was originally filed. Your receiving store's label and milestone allowlists
> still apply, so a label that store does not declare is dropped and named. A
> parent that stayed behind is recorded as an `origin-parent:` reference, because
> a `parent` field naming a ticket the receiving store does not hold is an error.
> Read the output: every one of these is reported, and nothing is dropped in
> silence.
>
> Nothing writes the store the ticket left. The adopt prints the `summary` and
> `status` commands that close the origin and name the new ID, and you run those
> in the origin store yourself. Commit both stores separately.

## Why it does not close the origin for you

Worth knowing before somebody files this as a missing feature. `import` runs in
the receiving repository. Plan 7.3 says a sync helper must never silently push,
merge, switch branches, or rewrite a worktree, and reaching into a second
repository to edit a ticket there is the same category. It is also where two
agents collide: the origin store may be open in somebody else's session.

So the closing half is printed rather than performed. That is the same shape
`import` already uses when it suggests `git am` instead of adopting.

## When `git am` is the better move

`--same-owner` mints a new ID in the receiving store, which is right when the
two stores declare different series and the old ID could not survive. When both
stores declare the same series, `git am` moves the ticket **keeping its
original ID**, and for a move between your own stores that is usually what you
want: every note, commit message and handoff that already names the ticket keeps
pointing at it.

```sh
# in the store the ticket is leaving
git ticket export TKT-01ABCD --out ./handoff

# in the store it is going to
git am ./handoff/*.patch
```

From `v0.18.0` that route also works into a repository with no store at all:
`git am` writes the ticket files, then `git ticket init` adopts the directory
they landed in. If the export carries a prefix the receiving store has never
declared, `init` names it and the `git ticket series add` that declares it.
Before `v0.18.0` this left a `.tickets` directory that no command would open, so
do not describe this route in ledger against an older binary.

The closing half is the same either way: nothing writes the sending store, so
run the `summary` and `status` there yourself.

## What to check before pasting

`--same-owner` landed in `v0.18.0` and is not in `v0.17.1` or earlier. Confirm
the version ledger installs actually has it:

```sh
git ticket import --help | grep same-owner
```

If that prints nothing, the paragraph describes a flag the installed binary does
not have, which is worse in a layout document than saying nothing. Wait for the
release, or pin the version the paragraph names.

## The half this does not cover

Moving a **parent and its children together** needs no `--same-owner` reasoning
about the parent: export them in one command and the edges among them are
rewritten to the new IDs automatically. `--same-owner` still carries the ticks
and the statuses. The `origin-parent:` reference only appears for a parent the
export genuinely left behind.
