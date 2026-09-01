# Tickets

This directory is a git-ticket store. Each file under `tickets/` is one ticket:
Markdown with YAML frontmatter, meant to be read, edited, diffed, and merged
like any other file in the repository. Archived tickets move to `archive/`.

A filename is the ticket ID and nothing else, so renaming a title does not break
`git log` on the old path.

You can edit these files by hand. Run `git ticket check` afterwards, which
reports what a hand edit tends to break: a duplicate ID, a dependency on a
ticket that does not exist, a status outside the set, or leftover merge conflict
markers.

`config.yml` sets defaults and the label vocabulary. Nothing here is generated
from anywhere else, and nothing outside this directory has to be in sync with
it.
