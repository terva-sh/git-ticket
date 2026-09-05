#!/bin/sh
# install.sh installs the latest stable git-ticket release, per
# TKT-01M1S02QA. The audience is a machine with neither Go nor a
# binary: Go users run `go install`, and an installed binary updates
# itself with `git ticket self-update`.
#
# The source is the GitHub releases of the public mirror, the same
# source and the same `latest` semantics as self-update, so the two
# cannot disagree about what stable means. The sha256 is verified
# against checksums.txt before anything is unpacked. The script never
# sudos on its own: pick a writable --prefix instead.
#
# Usage:
#   sh install.sh [--prefix DIR]
#
# --prefix DIR is the directory the binary lands in. Without it the
# script takes the first of ~/.local/bin and ~/bin that exists or can
# be created.

set -u

REPO="terva-sh/git-ticket"
API="${GIT_TICKET_INSTALL_API:-https://api.github.com}"

fail() {
    echo "install.sh: $*" >&2
    exit 1
}

# --- arguments ---------------------------------------------------------
PREFIX=""
while [ $# -gt 0 ]; do
    case "$1" in
    --prefix)
        [ $# -ge 2 ] || fail "--prefix needs a directory"
        PREFIX="$2"
        shift 2
        ;;
    -h | --help)
        sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'
        exit 0
        ;;
    *)
        fail "unknown argument $1; the flags are --prefix DIR and --help"
        ;;
    esac
done

# --- platform ----------------------------------------------------------
# Linux, macOS, and WSL2, which is Linux to uname. Native Windows takes
# the zip from the releases page, and guessing at PowerShell here would
# help nobody.
case "$(uname -s)" in
Linux) OS="linux" ;;
Darwin) OS="darwin" ;;
*)
    fail "unsupported platform $(uname -s); native Windows takes the zip from https://github.com/$REPO/releases"
    ;;
esac

case "$(uname -m)" in
x86_64 | amd64) ARCH="amd64" ;;
aarch64 | arm64) ARCH="arm64" ;;
*)
    fail "no release is built for $(uname -m)"
    ;;
esac

command -v curl >/dev/null || fail "curl is required"
command -v tar >/dev/null || fail "tar is required"

# sha256sum on Linux, shasum on macOS. One of the two must exist,
# because installing without verifying is not on the menu: a pipe from
# a forge is exactly the delivery a checksum exists for.
if command -v sha256sum >/dev/null; then
    SHA="sha256sum"
elif command -v shasum >/dev/null; then
    SHA="shasum -a 256"
else
    fail "neither sha256sum nor shasum is available, and unverified installs are not offered"
fi

# --- latest release ----------------------------------------------------
TAG=$(curl -fsSL "$API/repos/$REPO/releases/latest" |
    sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
[ -n "$TAG" ] || fail "could not read the latest release tag from $API"

VER="${TAG#v}"
ASSET="git-ticket_${VER}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$TAG"

# --- download and verify -----------------------------------------------
TMP=$(mktemp -d) || fail "mktemp failed"
trap 'rm -rf "$TMP"' EXIT

echo "downloading $ASSET ($TAG)"
curl -fsSL -o "$TMP/$ASSET" "$BASE/$ASSET" || fail "downloading $ASSET failed"
curl -fsSL -o "$TMP/checksums.txt" "$BASE/checksums.txt" || fail "downloading checksums.txt failed"

# Verified before anything is unpacked. The grep narrows the file to
# our asset so a mismatch names the one line that matters.
(
    cd "$TMP" &&
        grep " $ASSET\$" checksums.txt | $SHA -c - >/dev/null 2>&1
) || fail "sha256 verification failed for $ASSET; refusing to install it"
echo "sha256 verified against checksums.txt"

tar -xzf "$TMP/$ASSET" -C "$TMP" || fail "unpacking $ASSET failed"
[ -f "$TMP/git-ticket" ] || fail "the archive did not contain a git-ticket binary"

# --- destination -------------------------------------------------------
# The first writable candidate, and never sudo: if nothing here is
# writable, the caller picks a --prefix, because a script escalating on
# its own is how machines rot.
if [ -n "$PREFIX" ]; then
    DEST="$PREFIX"
    mkdir -p "$DEST" 2>/dev/null || fail "cannot create $DEST"
    [ -w "$DEST" ] || fail "$DEST is not writable; pick another --prefix"
else
    DEST=""
    for d in "$HOME/.local/bin" "$HOME/bin"; do
        if mkdir -p "$d" 2>/dev/null && [ -w "$d" ]; then
            DEST="$d"
            break
        fi
    done
    [ -n "$DEST" ] || fail "neither ~/.local/bin nor ~/bin is writable; pass --prefix DIR"
fi

mv "$TMP/git-ticket" "$DEST/git-ticket" || fail "installing into $DEST failed"
chmod 0755 "$DEST/git-ticket"

# --- report ------------------------------------------------------------
GOT=$("$DEST/git-ticket" --version 2>/dev/null) || fail "the installed binary did not run"
echo "installed $DEST/git-ticket"
echo "  $GOT"

case ":$PATH:" in
*":$DEST:"*) ;;
*)
    # An installed binary the shell cannot find is a support ticket,
    # so this is loud rather than a footnote.
    echo ""
    echo "NOTE: $DEST is not on your PATH. Add it, for example:"
    echo "  export PATH=\"$DEST:\$PATH\""
    ;;
esac

echo ""
echo "later upgrades: git ticket self-update"
