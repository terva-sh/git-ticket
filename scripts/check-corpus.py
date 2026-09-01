#!/usr/bin/env python3
"""Check the Phase 0 fixture corpus against docs/plan.md.

Scaffolding with a known lifespan. Phase 1 builds the Go library that reads
these fixtures, and its tests will assert most of this from the inside, where
they have a real parser instead of the regexes below. Fold these invariants
into those tests and delete this file.

It exists because the corpus landed before any code that reads it, so without
it nothing catches a fixture whose frontmatter drifts from section 5.1, an ID
that is not a valid ULID, or a sidecar naming a code that section 11 does not
define. Those are exactly the mistakes that turn into a Phase 1 test asserting
the wrong thing.

Usage: python3 scripts/check-corpus.py    (from the repository root)
Exits nonzero and prints one line per problem.
"""

import json
import os
import re
import sys

CROCKFORD = set("0123456789ABCDEFGHJKMNPQRSTVWXYZ")

# Section 5.1, in the order the renderer must emit.
ORDER = [
    "schema", "id", "title", "type", "status", "priority", "labels",
    "assignees", "milestone", "parent", "dependencies", "references",
    "claim", "archive", "created_at", "updated_at", "created_by",
    "updated_by", "extensions",
]
TYPES = {"task", "bug", "chore", "spike", "epic"}
STATUSES = {"draft", "ready", "in-progress", "blocked", "review", "done", "archived"}
PRIORITIES = {"low", "normal", "high", "urgent"}

# A fixture whose whole purpose is to carry a bad value, keyed by the field it
# is allowed to break. Anything else with a bad value is a mistake.
ALLOWED_BAD = {"status": "bad-status.md", "type": "bad-type.md", "priority": "bad-priority.md"}

problems = []


def note(msg):
    problems.append(msg)


def split_frontmatter(text):
    if not text.startswith("---\n"):
        return None
    end = text.find("\n---\n", 3)
    return None if end < 0 else text[4:end + 1]


def top_keys(fm):
    return [m.group(1) for m in re.finditer(r"(?m)^([A-Za-z_][A-Za-z0-9_]*):", fm)]


def scalar(fm, key):
    m = re.search(r"(?m)^%s:[ ]*(.*)$" % re.escape(key), fm)
    return m.group(1).strip() if m else None


def markdown_fixtures():
    out = []
    for root, _, files in os.walk("testdata"):
        for n in sorted(files):
            if n.endswith(".md") and n != "README.md":
                out.append(os.path.join(root, n))
    return sorted(out)


def plan_codes():
    plan = open("docs/plan.md").read()
    section = plan[plan.index("## 11. Validation"):plan.index("## 12. Interfaces")]
    return set(re.findall(r"^\| `([a-z_]+)` \|", section, re.M))


def main():
    if not os.path.isdir("testdata") or not os.path.isfile("docs/plan.md"):
        print("run this from the repository root", file=sys.stderr)
        return 2

    fixtures = markdown_fixtures()
    ids = {}

    for path in fixtures:
        text = open(path).read()
        name = os.path.basename(path)
        # A reject fixture is unparseable on purpose, so the field checks below
        # would be reading damage the fixture was written to contain.
        unparseable = "/reject/" in path

        if text != text.replace("\r\n", "\n"):
            note(f"{path}: CRLF line endings, 5.3 requires LF")
        if not text.endswith("\n") or text.endswith("\n\n"):
            note(f"{path}: 5.3 requires exactly one trailing newline")

        for m in re.finditer(r"TKT-([0-9A-Z]+)", text):
            ulid = m.group(1)
            if len(ulid) != 26 or not set(ulid) <= CROCKFORD:
                note(f"{path}: {m.group(0)} is not a 26-character Crockford ULID")

        if unparseable:
            continue

        fm = split_frontmatter(text)
        if fm is None:
            note(f"{path}: no frontmatter")
            continue

        present = [k for k in top_keys(fm) if k in ORDER]
        if present != [k for k in ORDER if k in present]:
            note(f"{path}: frontmatter keys out of 5.1 order: {present}")

        for field, valid in (("status", STATUSES), ("type", TYPES), ("priority", PRIORITIES)):
            value = scalar(fm, field)
            if value not in valid and name != ALLOWED_BAD[field]:
                note(f"{path}: {field} '{value}' is outside the set in the plan")

        tid = scalar(fm, "id")
        ids.setdefault(tid, []).append(path)

        if "/stores/" in path:
            expect_mismatch = "filename-mismatch" in path
            if (tid != name[:-3]) != expect_mismatch:
                note(f"{path}: filename and id field disagree unexpectedly")
            archived = scalar(fm, "status") == "archived"
            expect_location_bug = "archive-location-mismatch" in path
            if (archived != ("/archive/" in path)) != expect_location_bug:
                note(f"{path}: status and directory disagree unexpectedly")

    for tid, paths in ids.items():
        if len(paths) > 1 and not all("duplicate-id" in p for p in paths):
            note(f"id {tid} is reused outside the duplicate-id store: {paths}")

    # Every fixture carries an expectation, and every expectation a fixture.
    for root, _, files in os.walk("testdata/parse"):
        for n in files:
            p = os.path.join(root, n)
            if n.endswith(".md") and not os.path.exists(p[:-3] + ".expected.json"):
                note(f"{p}: no sidecar; an absent expectation is not an assertion")
            if n.endswith(".expected.json") and not os.path.exists(p[:-len(".expected.json")] + ".md"):
                note(f"{p}: sidecar with no fixture")

    for case in sorted(os.listdir("testdata/stores")):
        d = os.path.join("testdata/stores", case)
        for required in ("expected.json", "store/config.yml"):
            if not os.path.exists(os.path.join(d, required)):
                note(f"{d}: missing {required}")

    codes = plan_codes()
    used = set()
    for root, _, files in os.walk("testdata"):
        for n in files:
            if not n.endswith(".json"):
                continue
            p = os.path.join(root, n)
            try:
                doc = json.load(open(p))
            except json.JSONDecodeError as e:
                note(f"{p}: invalid JSON, {e}")
                continue
            for bucket in ("errors", "warnings"):
                for finding in doc.get(bucket, []):
                    used.add(finding["code"])
                    if set(finding) != {"code", "file", "ticket", "field"}:
                        note(f"{p}: a finding must carry exactly code, file, ticket, field")

    unknown = used - codes
    if unknown:
        note(f"sidecars use codes section 11 does not define: {sorted(unknown)}")

    uncovered = codes - used
    # Deferred question 9: nothing says what a references path resolves
    # against, so a fixture would invent the answer.
    expected_uncovered = {"reference_path_unresolved"}
    if uncovered - expected_uncovered:
        note(f"section 11 codes with no fixture: {sorted(uncovered - expected_uncovered)}")

    print(f"{len(fixtures)} fixtures, {len(codes)} codes, {len(used & codes)} covered, "
          f"{len(uncovered)} deferred")
    for p in problems:
        print(f"  {p}")
    print("corpus OK" if not problems else f"{len(problems)} problem(s)")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
