package ticket

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"
)

// The wire format of an export, per plan 12.8.
//
// An export is a mail message carrying a patch that adds ticket files, so that
// `git am` lands it in a repository that has never heard of git-ticket. The
// format lives here rather than in the CLI because it is the contract between
// two stores, and a host embedding this library has to be able to write one and
// read one without building argv and parsing prose.
//
// Nothing here runs git. The blob name is computed rather than asked for, which
// is what keeps plan 7.4's command table short, and what lets an export be built
// somewhere there is no repository at all.

// MboxFromLine opens every message. git's mailsplit needs it to find where one
// message ends and the next begins, and the value is conventional: format-patch
// writes the commit it came from, and a synthesised message has none, so this is
// zeroes. The date is the fixed one git itself writes, which is not a timestamp
// and is not read as one.
const MboxFromLine = "From 0000000000000000000000000000000000000000 Mon Sep 17 00:00:00 2001"

// ChangeKind names one thing a receiving store imposes on an incoming ticket.
//
// The rule these describe is the interchange rule: the statement of the work
// travels, and what the receiver never agreed to does not. Every kind here is
// the second half of that, and each exists so the loss is named rather than
// silent. The contribution that brought import arrived dropping half of this in
// silence, with check clean and exit 0, which is why naming them is not
// decoration.
type ChangeKind string

const (
	// ChangeLabelDropped carries the label in Value.
	ChangeLabelDropped ChangeKind = "label_dropped"
	// ChangeMilestoneDropped carries the milestone in Value.
	ChangeMilestoneDropped ChangeKind = "milestone_dropped"
	// ChangeDueOnDropped carries the date in Value.
	ChangeDueOnDropped ChangeKind = "due_on_dropped"
	// ChangeBlocksOnDropped carries the sender's blocks_on value in Value.
	ChangeBlocksOnDropped ChangeKind = "blocks_on_dropped"
	// ChangeReferencePathDropped carries the reference in Value. The reference
	// itself survives, and only its path is dropped.
	ChangeReferencePathDropped ChangeKind = "reference_path_dropped"
	// ChangeAcceptanceCriteriaUnchecked carries the item count in Count.
	ChangeAcceptanceCriteriaUnchecked ChangeKind = "acceptance_criteria_unchecked"
	// ChangeDefinitionOfDoneUnchecked carries the item count in Count.
	ChangeDefinitionOfDoneUnchecked ChangeKind = "definition_of_done_unchecked"
	// ChangeWorkRecordCarried reports that the sender's summary, notes and
	// comments arrived as one note. It carries neither a value nor a count.
	ChangeWorkRecordCarried ChangeKind = "work_record_carried"
)

// ChangeKinds is every kind, in the order a report reads best.
//
// It exists so a caller can prove it renders all of them. A kind a host never
// prints is a loss the reader never hears about, which is the failure this
// vocabulary exists to prevent.
func ChangeKinds() []ChangeKind {
	return []ChangeKind{
		ChangeLabelDropped,
		ChangeMilestoneDropped,
		ChangeDueOnDropped,
		ChangeBlocksOnDropped,
		ChangeReferencePathDropped,
		ChangeAcceptanceCriteriaUnchecked,
		ChangeDefinitionOfDoneUnchecked,
		ChangeWorkRecordCarried,
	}
}

// Change is one thing the receiving store imposed, as a value rather than as a
// sentence.
//
// The wording belongs to whoever is speaking. A CLI has its own voice, a web UI
// has another, and a host in a different language has a third, so a preformatted
// English string is the one form none of them can use.
type Change struct {
	Kind ChangeKind
	// Value is the subject of the change: the label, the milestone, the date,
	// the blocks_on value, or the reference. It is empty for the kinds that have
	// no subject.
	Value string
	// Count is how many items the change covers, for the checklist kinds. It is
	// zero elsewhere.
	Count int
}

// AddedFile is one added file, going out or coming back.
type AddedFile struct {
	// Path is slash-spelled and relative to the repository root, which is the
	// form a patch carries on every platform.
	Path string
	Body string
}

// BlobSHA is the object name git would give this content.
//
// Computing it here keeps the index line honest without asking git to hash
// anything, and an honest index line is what lets `git apply --3way` fall back
// on the object store.
func BlobSHA(data []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d", len(data))
	h.Write([]byte{0})
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// AddedFileHunk renders one new-file hunk and reports the line count it added.
//
// A file with no trailing newline gets git's own marker, because a patch that
// silently adds one changes the blob and the index line then disagrees with
// what applies.
func AddedFileHunk(path string, data []byte) (string, int) {
	var b strings.Builder
	lines := strings.Split(string(data), "\n")
	trailing := len(lines) > 0 && lines[len(lines)-1] == ""
	if trailing {
		lines = lines[:len(lines)-1]
	}
	fmt.Fprintf(&b, "diff --git a/%s b/%s\n", path, path)
	b.WriteString("new file mode 100644\n")
	fmt.Fprintf(&b, "index 0000000000000000000000000000000000000000..%s\n", BlobSHA(data))
	b.WriteString("--- /dev/null\n")
	fmt.Fprintf(&b, "+++ b/%s\n", path)
	fmt.Fprintf(&b, "@@ -0,0 +1,%d @@\n", len(lines))
	for _, l := range lines {
		b.WriteString("+" + l + "\n")
	}
	if !trailing {
		b.WriteString("\\ No newline at end of file\n")
	}
	return b.String(), len(lines)
}

// MboxMessage wraps a subject, a body and a diff as one mail message.
func MboxMessage(from string, when time.Time, subject, body, diff string) string {
	var b strings.Builder
	b.WriteString(MboxFromLine + "\n")
	fmt.Fprintf(&b, "From: %s\n", from)
	fmt.Fprintf(&b, "Date: %s\n", when.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Subject: [PATCH] %s\n", strings.ReplaceAll(subject, "\n", " "))
	b.WriteString("\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("---\n")
	b.WriteString(diff)
	b.WriteString("-- \ngit-ticket\n\n")
	return b.String()
}

// ParseAddedFiles recovers added files from an export's patch.
//
// Every hunk an export writes adds a whole new file, so there is no context to
// track and no deletion to apply: the body is the plus-prefixed lines with the
// prefix removed. The blob name on the index line is checked against the body
// that comes out, which is what turns a truncated or hand-edited patch into an
// error here rather than a puzzling ticket later.
//
// git's `\ No newline at end of file` marker is honoured rather than skipped,
// because AddedFileHunk writes it and a parser that adds the newline back
// rebuilds a file one byte longer than the one that went out. The blob check
// then fires on an artifact nobody touched and blames the sender for it.
func ParseAddedFiles(patch string) ([]AddedFile, error) {
	var out []AddedFile
	lines := strings.Split(patch, "\n")
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "diff --git ") {
			continue
		}
		var path, want string
		var body strings.Builder
		newFileSeen := false
		// The marker follows the last added line, and every hunk here adds a
		// whole file, so one flag per file is the whole of the bookkeeping.
		noTrailingNewline := false
		for i++; i < len(lines); i++ {
			l := lines[i]
			switch {
			case l == "new file mode 100644":
				newFileSeen = true
			case strings.HasPrefix(l, "index ") && strings.Contains(l, ".."):
				want = strings.TrimSpace(l[strings.Index(l, "..")+2:])
			case strings.HasPrefix(l, "+++ b/"):
				path = strings.TrimPrefix(l, "+++ b/")
			case strings.HasPrefix(l, "@@"):
				// The hunk body runs to the next diff, the signature, or the end.
				for i++; i < len(lines); i++ {
					l := lines[i]
					if strings.HasPrefix(l, "diff --git ") || l == "-- " {
						i--
						break
					}
					if strings.HasPrefix(l, "+") {
						body.WriteString(l[1:] + "\n")
						continue
					}
					if strings.HasPrefix(l, "\\ No newline") {
						noTrailingNewline = true
						continue
					}
					if strings.TrimSpace(l) == "" {
						continue
					}
					i--
					break
				}
			}
			if path != "" && body.Len() > 0 {
				break
			}
		}
		if !newFileSeen || path == "" {
			continue
		}
		// Trim once, before the blob check and the value both read it, so the
		// name that is verified is the name of the bytes that are handed back.
		content := body.String()
		if noTrailingNewline {
			content = strings.TrimSuffix(content, "\n")
		}
		got := BlobSHA([]byte(content))
		if want != "" && got != want {
			return nil, fmt.Errorf("%s does not match its blob name (%s, expected %s); the patch has been altered or truncated", path, got[:12], want[:12])
		}
		out = append(out, AddedFile{Path: path, Body: content})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no added files found")
	}
	return out, nil
}
