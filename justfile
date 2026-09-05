# git-ticket dev tasks. Run `just` to list them.
#
# These recipes are the same steps .forgejo/workflows/ci.yml runs, in the same
# order, so `just ci` failing here is CI failing there. The workflow keeps its
# steps inline rather than calling just, because the runner image is
# golang:1.25-alpine and adding a package to install just would buy nothing.
# When you change one, change the other.

set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# `-buildvcs=false` inside a linked git worktree, and empty everywhere else.
#
# Go finds the VCS root by looking for a `.git` directory, and a linked worktree
# has a `.git` file pointing into the main repository instead, so linking the
# main package there stops with "error obtaining VCS status: exit status 128"
# before it compiles anything. That hits `build` and `install`, and through
# `build` it hits `check`, `fix`, `ready` and `ci`. `go vet` and `go test` are
# fine, because only linking a binary stamps a version, which is what makes the
# failure confusing: the suite is green and the build is not.
#
# Sub-agents work in worktrees, so without this every one of them meets that
# error before it touches anything of its own.
#
# The comparison is the detection: the two agree in the main tree and differ in
# a linked worktree. Both are silenced so that a directory which is no git
# repository at all yields two empty strings, compares equal, and adds no flag.
#
# `.forgejo/workflows/ci.yml` needs no equivalent, because the runner does an
# ordinary checkout rather than a worktree. That is the one place this file and
# the workflow are allowed to differ.
buildvcs := `if [ "$(git rev-parse --git-dir 2>/dev/null)" != "$(git rev-parse --git-common-dir 2>/dev/null)" ]; then echo "-buildvcs=false"; fi`

# List the recipes.
default:
    @just --list

# The binary lands in the repository root, where the README tells a reader to
# find it. It is gitignored, so it can land there on any ordinary day of work.
# Build git-ticket into the repository root.
build:
    go build {{buildvcs}} -o git-ticket ./cmd/git-ticket
    @echo "built ./git-ticket ($(git rev-parse --short HEAD))"
    @[ -z "{{buildvcs}}" ] || echo "note: linked worktree, so this was built with {{buildvcs}} and ./git-ticket reports devel rather than a version" >&2

# Git spells a binary named git-ticket on PATH as `git ticket`, so this is what
# makes the subcommand work outside this tree, which is what dogfooding it in
# another repository needs.
#
# This writes to GOBIN, and install.sh and `install-release` write to the first
# writable of ~/.local/bin and ~/bin. Run both and there are two binaries named
# git-ticket, with PATH order deciding which one `git ticket` means, so the last
# check warns when the copy just installed is not the one that answers. Per
# TKT-01M1SMAP it warns rather than staying quiet, because it fires only when
# the install did not change what `git ticket` means, which is a fact worth
# hearing every time it is true.
# Install git-ticket into GOBIN, else GOPATH/bin.
install:
    go install {{buildvcs}} ./cmd/git-ticket
    @dest="$(go env GOBIN)"; [ -n "$dest" ] || dest="$(go env GOPATH)/bin"; \
      echo "installed git-ticket -> $dest/git-ticket"; \
      case ":$PATH:" in *":$dest:"*) ;; *) echo "warning: $dest is not on PATH, so \`git ticket\` will not resolve" >&2 ;; esac; \
      first="$(command -v git-ticket || true)"; \
      if [ -n "$first" ] && [ "$first" != "$dest/git-ticket" ]; then \
        echo "warning: $first comes first on PATH, so \`git ticket\` still means that one" >&2; \
        echo "  it reports: $("$first" --version 2>/dev/null || echo unknown)" >&2; \
      fi
    @[ -z "{{buildvcs}}" ] || echo "note: linked worktree, so this was built with {{buildvcs}} and the installed binary reports devel rather than a version" >&2

# `just install` builds whatever is in your tree and puts it in GOBIN, which is
# what dogfooding wants. This is the other half. It builds a released tag and
# puts it where install.sh puts a downloaded one, so compiling from source and
# running the README's curl one-liner cannot leave two binaries of different
# ages on PATH with `git ticket` quietly picking between them.
#
# The tag is checked before anything is built. `go build` takes the version from
# what the VCS reports, per plan 12.1, so an untagged commit stamps a
# pseudo-version and a shallow checkout stamps `devel`. Neither is a release,
# and neither should reach a directory on PATH.
#
# The build runs in a throwaway local clone at the tag, so your checkout is
# never touched and its state cannot reach the binary. A temporary worktree was
# the obvious way to do that and does not work: Go looks for a `.git` directory
# to find the VCS root, a linked worktree has a `.git` file instead, and
# `go build` there fails with "error obtaining VCS status: exit status 128"
# rather than stamping anything. A clone has a real `.git` directory, and it
# writes nothing into this repository, where `git worktree add` would.
#
# What proves the result is the artifact and not the tree: the built binary has
# to report the tag that was asked for, with modified false. That is read from
# `--version --json`, because plan 12.4 does not cover the human line and a
# recipe grepping it would be parsing an interface this project does not offer.
# Build a release tag from source and install it where install.sh would.
install-release TAG="":
    #!/usr/bin/env bash
    set -euo pipefail

    tag="{{TAG}}"
    if [ -z "$tag" ]; then
        tag="$(git describe --exact-match --tags HEAD 2>/dev/null || true)"
        if [ -z "$tag" ]; then
            echo "install-release: HEAD is not at a tag, so there is no release here to build." >&2
            echo "  name one instead:  just install-release v0.11.0" >&2
            recent="$(git tag -l 'v*' --sort=-v:refname | head -3 | tr '\n' ' ')"
            [ -z "$recent" ] || echo "  recent tags:       $recent" >&2
            exit 1
        fi
    fi

    if ! git rev-parse -q --verify "refs/tags/$tag^{commit}" >/dev/null; then
        echo "install-release: \"$tag\" is not a tag in this repository." >&2
        echo "  a branch or a commit will not do, because the version comes from the tag." >&2
        echo "  if the tag is new here, fetch it first: git fetch --tags" >&2
        exit 1
    fi

    # install.sh's own resolution, and its no-sudo rule with it. A recipe that
    # escalates on its own is how a machine rots.
    dest=""
    for d in "$HOME/.local/bin" "$HOME/bin"; do
        if mkdir -p "$d" 2>/dev/null && [ -w "$d" ]; then
            dest="$d"
            break
        fi
    done
    if [ -z "$dest" ]; then
        echo "install-release: neither ~/.local/bin nor ~/bin is writable." >&2
        echo "  nothing here sudos. Make one of them writable, or use install.sh --prefix." >&2
        exit 1
    fi

    root="$(git rev-parse --show-toplevel)"
    work="$(mktemp -d)"
    src="$work/src"
    trap 'rm -rf "$work"' EXIT

    echo "building $tag in a temporary clone"
    # Not shallow. A shallow clone has no tag to derive a version from, so the
    # binary would answer `devel` and the check below would refuse it.
    git -c advice.detachedHead=false clone --local --quiet --branch "$tag" "$root" "$src"
    ( cd "$src" && go build -o "$work/git-ticket" ./cmd/git-ticket )

    # The version envelope is a covered surface, per plan 10, so these key names
    # are stable in a way the pretty line is not.
    json="$("$work/git-ticket" --version --json)"
    got_version="$(printf '%s\n' "$json" | sed -n 's/.*"version": *"\([^"]*\)".*/\1/p')"
    got_modified="$(printf '%s\n' "$json" | sed -n 's/.*"modified": *\(true\|false\).*/\1/p')"

    if [ "$got_version" != "$tag" ]; then
        echo "install-release: the build reports version \"$got_version\", not $tag." >&2
        echo "  nothing was installed." >&2
        exit 1
    fi
    if [ "$got_modified" != "false" ]; then
        echo "install-release: the build reports modified: $got_modified, so it is not $tag as released." >&2
        echo "  nothing was installed." >&2
        exit 1
    fi

    # Replace through a temporary name in the destination, so the swap is atomic
    # and a running git-ticket cannot make this fail with ETXTBSY.
    cp "$work/git-ticket" "$dest/.git-ticket.new"
    chmod 0755 "$dest/.git-ticket.new"
    mv "$dest/.git-ticket.new" "$dest/git-ticket"

    echo "installed $dest/git-ticket"
    echo "  $("$dest/git-ticket" --version)"

    case ":$PATH:" in
        *":$dest:"*) ;;
        *) echo "warning: $dest is not on PATH, so \`git ticket\` will not resolve" >&2 ;;
    esac

    first="$(command -v git-ticket || true)"
    if [ -n "$first" ] && [ "$first" != "$dest/git-ticket" ]; then
        echo "warning: $first comes first on PATH, so \`git ticket\` still means that one" >&2
        echo "  it reports: $("$first" --version 2>/dev/null || echo unknown)" >&2
    fi

# The fast suite. Seconds, so there is little reason not to run it.
test *ARGS:
    go test ./... {{ARGS}}

# TestLifecycle builds the real binary and drives a ticket end to end, so this
# covers the CLI and not only the library.
# Run the suite with the race detector, as CI does.
test-race *ARGS:
    go test -race ./... {{ARGS}}

# gofmt every Go file in place.
fmt:
    gofmt -w .

# gofmt -l exits zero whether or not it printed anything, so the file list is
# the test. tee puts the names on stderr as well, because a bare failure with no
# filename sends you looking through the whole tree.
# Check formatting and vet, changing nothing.
lint:
    test -z "$(gofmt -l . | tee /dev/stderr)"
    go vet ./...

# Tidy go.mod and go.sum.
tidy:
    go mod tidy

# The verification command of plan section 11. It plans every repair, prints
# what it would do, writes nothing, and exits 1 when one is pending.
#
# --strict is the flag doing the work. epics_index_stale is a warning, so
# without it a stale index exits 0 and this step goes green over the condition
# it is here to catch. It also fails on an expired claim, or a ticket
# in-progress with nobody holding it, which is how a store starts to drift.
# Verify this repository's own ticket store with the binary from this tree.
check: build
    ./git-ticket check --fix --dry-run --strict

# The repair half of `just check`. CI reports what is pending and never commits
# it, per plan section 11, so this is the local command that settles it.
# Repair what check reports: ticket file placement and the generated epics index.
fix: build
    ./git-ticket check --fix

# What is startable right now, from this repository's own store.
ready: build
    ./git-ticket ready

# The full local gate. Same steps as .forgejo/workflows/ci.yml.
ci: lint test-race check

# Build the release archives locally, without a tag and without publishing.
# Needs goreleaser on PATH. Writes to dist/, which is gitignored: an untracked
# dist/ would make go build stamp modified:true into every binary.
release-snapshot:
    goreleaser release --snapshot --clean --skip=validate

# Validate .goreleaser.yaml without building anything.
release-check:
    goreleaser check

# Remove the built binary and the release artifacts.
clean:
    rm -f ./git-ticket
    rm -rf ./dist
