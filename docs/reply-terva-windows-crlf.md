# Reply to terva: pin v0.14.3, and one line only you can add

Supersedes `docs/reply-terva-windows-lock.md`, which is left exactly as it was
sent. If you are holding that document, keep it: nothing in it was wrong when it
was written, and this one says which parts time has moved. Do not diff the two
files expecting one to be a corrected copy of the other. This is a second
document about a second release.

Written 2026-09-07, against git-ticket v0.14.3.

## What moved

Three claims in the earlier document are now stale, and they are the ones that
told you what to do.

- It said to pin **v0.14.2**. Pin **v0.14.3**.
- It said the user-facing CRLF defect was open, with a manual workaround. It is
  fixed in v0.14.3 for stores created from now on.
- It gave a workaround "until `init` writes it". `init` writes it.

One claim in it is still true and is the reason this document exists: the fix
cannot reach a store that already exists, and terva has one.

Everything else in it stands. The lock is fixed, `TestWorktreesShareOneLock`
passed on a `windows-latest` runner, and the `ERROR_IO_PENDING` deviation from
your patch is unchanged and still open to reversal.

## Pin v0.14.3

You are on v0.14.1, so this bump also picks up v0.14.2, which you never took.
One move covers the lock fix and the CRLF fix together.

```
require github.com/terva-sh/git-ticket v0.14.3
```

Four edits, and the first two must land in one commit because
`TestGitTicketRequireMatchesCIPin` binds them:

| File | Line | Now | Becomes |
|---|---|---|---|
| `go.mod` | 17 | `v0.14.1 // store schema 3; keep in sync with GIT_TICKET_VERSION` | `v0.14.3`, comment kept |
| `.forgejo/workflows/ci.yml` | 75 | `GIT_TICKET_VERSION: v0.14.1` | `GIT_TICKET_VERSION: v0.14.3` |
| `.tickets/CONVENTIONS.md` | 229 | "Both are v0.14.1." | "Both are v0.14.3." |
| `.gitattributes` | 70 | the merge line alone | see the next section |

The tag is `1dba8cf72c7887ec0be981179e49faea1e971d4c` on both forges, and the
proxy agrees:

```
$ curl -sS https://proxy.golang.org/github.com/terva-sh/git-ticket/@v/v0.14.3.info
{"Version":"v0.14.3", ... "Hash":"1dba8cf72c7887ec0be981179e49faea1e971d4c", ...}
```

`SchemaVersion` is still 3. No store migrates.

### One sentence in CONVENTIONS.md needs more than its number

Line 231 reads:

> The floor below is v0.14.0 rather than the pin, because schema 3 arrived in
> v0.14.0 and v0.14.1 changed only the text of two refusals. Any v0.14.x reads
> this store.

The conclusion holds. v0.14.2 and v0.14.3 are both patches that moved no format:
v0.14.2 is a Windows lock implementation, v0.14.3 is what `init` writes into
`.gitattributes`. But the justification now skips two releases it does not name,
so the clause wants widening rather than the sentence wanting replacing. The
floor stays v0.14.0.

## The line only you can add

This is the part `init` cannot do for you, and it is the whole reason this
document is not just "bump the number".

`init` writes the attribute when it creates a store. Your store already exists.
Nothing in git-ticket will ever go back and add the line to it, and
`install-merge-driver` deliberately will not, because its name is a promise
about what it touches. So this edit is yours:

```
.tickets/**/*.md text eol=lf
```

Put it beside the merge line at `.gitattributes:70`. Order between the two does
not matter to git; `init` writes eol first only because a file has to be
readable before a merge driver is worth anything.

On your machines this changes nothing today. Your working tree is already LF, so
the line is inert until the next checkout that would have converted. That is the
point: it is insurance against a clone you have not made yet, on a machine you
may not own. If somebody already has a CRLF working tree, they need
`git add --renormalize .` or a fresh checkout as well; adding the attribute
alone does not rewrite files already on disk.

### Why this is worth more than a line in a checklist

Your `.gitattributes` already carries seven `text eol=lf` pins. Every one has a
comment, and the comments are good: the card fixtures, the ctrlproto goldens and
their `.txt` censuses, the connproto and extproto published corpora, the i18n
catalogs, the embedded personas. Several say outright that the failure surfaces
on Windows only, which is the release gate, which is the most expensive place to
learn it. One records that the pin was written for `.json` and did not reach the
`.txt` files beside them. Another records that the persona path moved and the pin
stayed pointing at the old directory.

That is a team that understands this trap better than most.

Line 70 is `.tickets/**/*.md merge=gitticket`, with no eol pin. The one path you
did not protect is the one git-ticket wrote for you. You deferred to the tool,
which was correct, and the tool was wrong. That is our fault and not yours, and
it is why this document leads with the manual edit instead of the version bump.

## Your users are already covered by the bump

Production terva reaches git-ticket through `gtcli.Run` in
`packages/agent/ticketcmd.go`, which is the CLI, which is what writes the
attribute. So once you pin v0.14.3, any store a terva user creates through
terva gets both lines with no further work.

`ticket.Init` appears only in your tests, in `packages/agent/build/ticket_tools_test.go`,
`packages/agent/build/toolclass_test.go` and `packages/agent/tools/ticket_test.go`.
Those stores live in temp directories and are never cloned, so they neither need
the attribute nor suffer from its absence.

## One open question, which is yours to answer

`ticket.Init` in the library writes no `.gitattributes` at all. Only the CLI's
`init` does.

That is fine for you, because you embed `cli`. It is not obviously fine in
general: a host that imports only `ticket` gets a store with no protection and
no signal that anything is missing. The library could write the file, or it
could export the lines for a host to write, or it could keep reaching for
nothing outside the store and leave the job to whoever owns the repository.

We have not filed this, because it changes a library surface and 12.4 covers
those, so it is a decision to take with you rather than to hand you. It is
adjacent to the ergonomics handoff you already sent. If you want it, say which
of the three shapes you want and we will take it through the plan.

## What is still not repaired, and the trigger for repairing it

A store created before v0.14.3 has no automatic repair. The manual line above is
the whole answer today.

`TKT-01M1X70GCG5RHGT0ZHVDC09D62` (Decide how a store predating the eol
attribute gets repaired) holds that as a deferred question in plan section 15,
and its trigger is a real report rather than reasoning: somebody
meeting a store made before the line, cloned on Windows, listing nothing.
`check --fix` is the obvious home and the expensive one, because every finding
in section 11 describes the store's own contents and `.gitattributes` sits
outside the store, so it would be the first finding of its kind. A flag on
`install-merge-driver` is the cheap alternative.

If you hit it on a real store, tell us. That is the trigger, and a report from
you is worth more than the argument we could have with ourselves.

## Provenance

The git-ticket claims come from the tree at `ce54ed7`, the published v0.14.3
assets verified with `sha256sum -c`, the module proxy, and an end-to-end run of
the released binary: `init` in a scratch repository wrote both lines, and a store
it created survived a `git clone --config core.autocrlf=true` and still listed
its ticket. That clone flag is the identical mechanism Windows turns on by
default, so it is the condition itself rather than a stand-in for it.

The terva claims come from reading your tree only: `go.mod`, `.gitattributes`,
`.forgejo/workflows/ci.yml`, `.tickets/CONVENTIONS.md`, and a grep for
`ticket.Init` and `gtcli.Run`. Nothing was written there, per the policy in our
`AGENTS.md`. Line numbers are as of that reading and will drift.
