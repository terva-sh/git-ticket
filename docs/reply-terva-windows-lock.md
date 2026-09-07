# Reply to terva: the store lock works on Windows, in v0.14.2

Answers `docs/plans/handoff-git-ticket-windows-lock.md`, written 2026-09-07
against v0.14.1. This is the document to carry back.

Your diagnosis was right in every part I could check, and your patch shipped
close to verbatim. One branch differs, and it is called out below because it
changes what a failure tells your reader rather than whether it fails.

## Pin v0.14.2

```
require github.com/terva-sh/git-ticket v0.14.2
```

`GIT_TICKET_VERSION` at `.forgejo/workflows/ci.yml` line 75 moves with it, since
`TestGitTicketRequireMatchesCIPin` holds the two together. `.tickets/CONVENTIONS.md`
needs no edit: it records the schema floor as v0.14.0 and says any v0.14.x reads
the store, which stays true. `SchemaVersion` is still 3 and no store migrates.

The tag is `9aab12c3bd19a00ee3083d84b778c25b3e637a52`, on both forges, and the
proxy agrees:

```
$ curl -sS https://proxy.golang.org/github.com/terva-sh/git-ticket/@v/v0.14.2.info
{"Version":"v0.14.2", ... "Hash":"9aab12c3bd19a00ee3083d84b778c25b3e637a52", ...}
```

## What shipped

`ticket/lock_windows.go` takes the lock through `LockFileEx` and releases it
through `UnlockFileEx`, one byte at offset zero, as you wrote it.
`lock_other.go` narrows to `!unix && !windows`. No dependency added.
`ERROR_LOCK_VIOLATION` maps to `(false, nil)`, so `lock()` polls to its deadline.

Your eight failing tests should pass. All five write tools work on Windows now,
and so does every `git ticket` mutation.

### The one deviation

You wrote that if `ERROR_IO_PENDING` can arise it belongs in the contended
branch. It is in the fatal branch instead.

`os.OpenFile` never passes `FILE_FLAG_OVERLAPPED`, so the handle is synchronous
and `LockFileEx` cannot queue on it. Reaching that error therefore means an
assumption in the file is wrong. In the contended branch it would spin to the
deadline and then report `another process holds the store lock`, blaming a
holder that does not exist. In the fatal branch it reports `lock_timeout:
Overlapped I/O operation is in progress`, which names the condition. Both fail
the write, so this is about diagnosis and not correctness. If you disagree, the
argument is one line to reverse.

## The runtime evidence you could not get

You marked runtime behaviour as unverified, and it was the right thing to flag.
It is now measured.

`TestWorktreesShareOneLock` passed on a `windows-latest` runner, GitHub Actions
run 34085691784 on `17d0515`. That test holds the lock in one worktree, asserts
the other gets `CodeLockTimeout`, releases, and asserts the write then succeeds,
so both branches of `tryFlock` ran on the platform.

The shipped artifact carries it, not only the source. `strings` on
`git-ticket.exe` from `git-ticket_0.14.2_windows_amd64.zip` finds `LockFileEx`
twice and finds `needs flock, which this platform does not provide` zero times.

## We took the lane, and it earned its place immediately

You wrote that you would rather we took the CI lane than skipped it. We took it.
`.github/workflows/ci.yml` runs on `windows-latest`, guarded by
`github.server_url` so Forgejo, which executes `.github/workflows` too, skips a
job it has no runner for. Your note about the inverted action-path trap was
correct and saved a red first run.

It went red twice before it went green, on two defects nothing else could see.
The second one is your problem too.

### It is scoped, and criterion 4 is unticked because of it

The lane runs `go build ./...` and `go vet ./...` over the whole tree for
Windows, but `go test` only over `./ticket/...`. `cli/copy_test.go` execs
`/bin/sh` in two tests and the tui suite has never run on Windows, so a lane
over everything would have arrived red. Widening it is `TKT-01M1X4QTHPJ0`.

The trigger is a push of `main` plus `workflow_dispatch`, because the mirror
only receives `main` at release time. So it fires once per release, and the
release sequence now reads: push `main`, read the lane, then tag.

## The defect this found, which affects your users

This is the part to read even though the lock is fixed.

git converts text files to CRLF on a Windows checkout by default. Plan 5.3
requires a ticket file to carry LF, and `parse` enforces it: a file whose first
line is `---\r` is rejected with

```
parse_error: file does not start with a --- frontmatter fence
```

So a user who clones a repository containing a `.tickets/` store on Windows gets
a store where every ticket fails to read. `list` answers with nothing rather
than with an error that explains itself. That is worse than the lock bug, which
at least failed loudly and only on writes.

It is not hypothetical. It is what took our lane red on its first run: about 30
fixtures converted at once, and `corpus_test.go` named the cause as "CRLF line
endings, 5.3 requires LF".

git-ticket fixed its own checkout with `* text=auto eol=lf` in `.gitattributes`.
That protects this repository and nothing else.

`TKT-01M1X4QTFY8H` (Have init write an eol=lf .gitattributes line for Windows
stores) carries the decision: `git ticket init` already writes `.gitattributes`
for the merge driver, so a store-scoped `text eol=lf` line joins it there.
Making `parse` tolerate CRLF was considered and declined, so a store converted
before that ships stays unreadable until somebody adds the attribute and
re-checks-out.

What this means for terva today, if a terva user works on Windows:

- A store terva creates at runtime is written with LF and is fine.
- A store committed to a repository and cloned on Windows is not, unless that
  repository carries a `.gitattributes` line for it.
- The workaround until `init` writes it is one line in the user's repository:
  `.tickets/**/*.md text eol=lf`, then re-checkout those files.

If terva documents a `.tickets/` store for its own users, that line is worth
saying now rather than after the next release.

## Provenance

Every claim above came from one of: this repository's tree at `9aab12c`, GitHub
Actions runs 34085691784, 34086324593 and 34086614472, the published v0.14.2
assets verified with `sha256sum -c`, the image at
`ghcr.io/terva-sh/git-ticket:0.14.2` pulled with no credentials, or the module
proxy. Nothing here is inferred from a cross-compile alone, which was the gap
your own handoff named.
