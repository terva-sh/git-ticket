package ticket

import (
	"regexp"
	"strings"
)

// conflictSections are the body headings Merge can report. Everything else it
// names is a frontmatter key, which is what tells markConflicts where to look.
var conflictSections = map[string]bool{
	"Description":         true,
	"Implementation plan": true,
	"Summary":             true,
	"Acceptance criteria": true,
	"Definition of done":  true,
}

// topLevelKey matches a frontmatter key at column zero. Nested keys are
// indented, so this is what ends one field's block and starts the next.
var topLevelKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*:`)

// markConflicts replaces each disputed field in merged with a Git conflict
// block carrying our version and theirs.
//
// The result does not parse, and that is the point: plan 11 reports a file with
// markers as merge_conflict rather than as a YAML failure, which is what a
// person needs to be told. Everything the merge did settle is still in the
// file, so what a reader resolves is the disagreement and not the whole ticket.
func markConflicts(merged, ours, theirs []byte, conflicts []string) []byte {
	lines := strings.Split(string(merged), "\n")
	ourLines := strings.Split(string(ours), "\n")
	theirLines := strings.Split(string(theirs), "\n")

	for _, name := range conflicts {
		start, end, ok := blockRange(lines, name)
		if !ok {
			// The field is not in the rendered output, which happens for a
			// section only one side has. The conflict is still reported and the
			// command still exits non-zero, so Git does not call the merge
			// clean.
			continue
		}
		var block []string
		block = append(block, "<<<<<<< ours")
		block = append(block, sliceBlock(ourLines, name)...)
		block = append(block, "=======")
		block = append(block, sliceBlock(theirLines, name)...)
		block = append(block, ">>>>>>> theirs")

		out := make([]string, 0, len(lines)-(end-start)+len(block))
		out = append(out, lines[:start]...)
		out = append(out, block...)
		out = append(out, lines[end:]...)
		lines = out
	}
	return []byte(strings.Join(lines, "\n"))
}

// sliceBlock returns one field's lines from a rendered ticket, or nothing when
// that side does not carry the field.
func sliceBlock(lines []string, name string) []string {
	start, end, ok := blockRange(lines, name)
	if !ok {
		return nil
	}
	return lines[start:end]
}

// blockRange locates a field's lines: [start, end).
//
// A frontmatter field runs from its key to the next key at column zero, which
// keeps a nested value like updated_by or claim whole. A body section runs from
// its heading to the next heading.
func blockRange(lines []string, name string) (int, int, bool) {
	if conflictSections[name] {
		return sectionRange(lines, name)
	}
	return frontmatterRange(lines, name)
}

func frontmatterRange(lines []string, key string) (int, int, bool) {
	open, close := frontmatterBounds(lines)
	if open < 0 {
		return 0, 0, false
	}
	for i := open + 1; i < close; i++ {
		if lines[i] != key+":" && !strings.HasPrefix(lines[i], key+": ") {
			continue
		}
		end := close
		for j := i + 1; j < close; j++ {
			if topLevelKey.MatchString(lines[j]) {
				end = j
				break
			}
		}
		return i, end, true
	}
	return 0, 0, false
}

// frontmatterBounds returns the indices of the opening and closing --- lines.
func frontmatterBounds(lines []string) (int, int) {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return -1, -1
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return 0, i
		}
	}
	return -1, -1
}

func sectionRange(lines []string, heading string) (int, int, bool) {
	want := "## " + heading
	for i, line := range lines {
		if line != want {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], "## ") {
				end = j
				break
			}
		}
		return i, end, true
	}
	return 0, 0, false
}
