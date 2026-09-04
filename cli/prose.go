package cli

import (
	"io"
	"os"
	"strings"
)

// This file is the file and stdin half of prose input, per plan 12.1.
//
// Every body section is Markdown, and passing a paragraph as one shell word
// makes the shell part of the authoring surface. An apostrophe ends a
// single-quoted string and a backtick inside a double-quoted one runs a
// command, so writing a description with contractions in it means fighting the
// quoting rather than writing. That is the friction TKT-01M1PCXW recorded, and
// it compounds the heading trap: long prose needs care, and fighting the shell
// is what prevents care.
//
// The shape follows how the command already takes the text. A named flag gets a
// named sibling, so --description is joined by --description-file. The four
// commands that take the text as a positional get one --file, since there is
// only one thing it could fill.
//
// A sigil on the value, such as --description @notes.md, was the other
// candidate and is not used. It reads a legal ticket body that opens with @ as
// a path, so it needs an escape for its own escape.

// proseInput is one body section a command takes, as either an inline value or
// a path in a sibling flag.
//
// The caller registers both flags itself, so the flag descriptions stay next to
// the other flags of that command, and then hands the values here. resolveProse
// writes the answer back into text.
type proseInput struct {
	// textFlag is the inline flag's name without its dashes, or "" when the
	// text arrives as a positional argument.
	textFlag string
	// fileFlag is the sibling flag's name without its dashes.
	fileFlag string

	text string
	path string

	// textGiven and fileGiven report whether each was actually supplied. An
	// empty string is a legal value for both, so emptiness cannot stand in for
	// absence: `update --description ""` clears the section.
	textGiven bool
	fileGiven bool
}

// given reports whether the command was handed this section at all.
func (p *proseInput) given() bool { return p.textGiven || p.fileGiven }

// source names where the text came from, for the heading warning. It is the
// flag a reader would have to go and edit, which is the file flag when that is
// the one that carried the text.
func (p *proseInput) source() string {
	if p.fileGiven {
		return "--" + p.fileFlag
	}
	if p.textFlag == "" {
		// A positional. The command name is what the warning can name, and
		// runTextEntry passes it.
		return ""
	}
	return "--" + p.textFlag
}

// resolveProse fills in every section a command was given, reading each file
// and stdin once.
//
// It takes the whole group rather than one input at a time because the two
// refusals are about the group: a section handed both an inline value and a
// file, and two sections both asking for stdin.
func resolveProse(stdin io.Reader, command string, inputs ...*proseInput) error {
	// Both is a usage error naming both, the way --depends-on with --ref is. A
	// caller who typed two meant one, and resolving by precedence writes
	// something nobody asked for.
	for _, p := range inputs {
		if p.textGiven && p.fileGiven {
			if p.textFlag == "" {
				return usageErr("%s takes the text or --%s, not both", command, p.fileFlag)
			}
			return usageErr("%s takes --%s or --%s, not both", command, p.textFlag, p.fileFlag)
		}
	}

	// Stdin is a stream and reading it twice returns nothing the second time,
	// so a command asking for it twice cannot be given what it meant. Refused
	// before the first read, so nothing is consumed by a call that then fails.
	var fromStdin []string
	for _, p := range inputs {
		if p.fileGiven && p.path == "-" {
			fromStdin = append(fromStdin, "--"+p.fileFlag)
		}
	}
	if len(fromStdin) > 1 {
		return usageErr("%s can read stdin for only one section, and %s both name -",
			command, strings.Join(fromStdin, " and "))
	}

	for _, p := range inputs {
		if !p.fileGiven {
			continue
		}
		text, err := readProse(stdin, p.path, "--"+p.fileFlag)
		if err != nil {
			return err
		}
		p.text = text
	}
	return nil
}

// readProse reads one section from a path, where "-" is stdin. label is the
// flag that named the path, which is the part of the failure the caller can act
// on: the os error already carries the path and the reason.
//
// A path that cannot be read is a usage error rather than a store one. Section
// 10 lists no code for a failed read, and the condition is a flag value the
// caller can correct, which is what `usage` covers. Inventing a code would mean
// changing the plan for an error that names its own cause in the message.
func readProse(stdin io.Reader, path, label string) (string, error) {
	var (
		b   []byte
		err error
	)
	if path == "-" {
		b, err = io.ReadAll(stdin)
		if err != nil {
			return "", usageErr("%s: reading stdin: %v", label, err)
		}
	} else {
		b, err = os.ReadFile(path)
		if err != nil {
			return "", usageErr("%s: %v", label, err)
		}
	}
	// A final newline terminates the last line of a text file rather than being
	// content, and printf and a heredoc disagree about whether to emit one.
	// Trailing whitespace goes with it. Nothing else is touched, so indentation
	// inside the prose survives, which matters for a fenced code block.
	return strings.TrimRight(string(b), " \t\r\n"), nil
}
