# git-ticket dev tasks. Run `just` to list them.
#
# These recipes are the same steps .forgejo/workflows/ci.yml runs, in the same
# order, so `just ci` failing here is CI failing there. The workflow keeps its
# steps inline rather than calling just, because the runner image is
# golang:1.25-alpine and adding a package to install just would buy nothing.
# When you change one, change the other.

set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# List the recipes.
default:
    @just --list

# The binary lands in the repository root, where the README tells a reader to
# find it. It is gitignored, so it can land there on any ordinary day of work.
# Build git-ticket into the repository root.
build:
    go build -o git-ticket ./cmd/git-ticket
    @echo "built ./git-ticket ($(git rev-parse --short HEAD))"

# Git spells a binary named git-ticket on PATH as `git ticket`, so this is what
# makes the subcommand work outside this tree, which is what dogfooding it in
# another repository needs.
# Install git-ticket into GOBIN, else GOPATH/bin.
install:
    go install ./cmd/git-ticket
    @dest="$(go env GOBIN)"; [ -n "$dest" ] || dest="$(go env GOPATH)/bin"; \
      echo "installed git-ticket -> $dest/git-ticket"; \
      case ":$PATH:" in *":$dest:"*) ;; *) echo "warning: $dest is not on PATH, so \`git ticket\` will not resolve" ;; esac

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

# --strict fails on warnings too: an expired claim, or a ticket in-progress with
# nobody holding it, is how a store starts to drift.
# Validate this repository's own ticket store with the binary from this tree.
check: build
    ./git-ticket check --strict

# What is startable right now, from this repository's own store.
ready: build
    ./git-ticket ready

# The full local gate. Same steps as .forgejo/workflows/ci.yml.
ci: lint test-race check

# Remove the built binary.
clean:
    rm -f ./git-ticket
