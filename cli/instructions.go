package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// instructionsText is the agent workflow block of plan 12.1. It lives in a
// Markdown file rather than a Go string so a wording change reviews as a prose
// diff, and so nobody has to escape a backtick to edit it.
//
//go:embed instructions.md
var instructionsText string

// instructionsFile is where `init --instructions` writes the block. It is the
// file terva and most agent harnesses already read.
const instructionsFile = "AGENTS.md"

// runInstructions prints the block for a person to paste or redirect. It reads
// no store, so it answers outside a repository and before init.
func runInstructions(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("instructions", args, nil)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usageErr("instructions takes no arguments")
	}
	if ctx.g.json {
		writeJSON(ctx.out, instructionsEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "instructions",
			Text:          instructionsText,
		})
		return nil
	}
	_, err = fmt.Fprint(ctx.out, instructionsText)
	return err
}

// instructionsFileConflict refuses when AGENTS.md is already there, per plan
// 12.1: that file is one the user maintains, and appending to it is their edit
// to make. runInit calls this before creating the store, so a refusal leaves
// nothing half-done.
func instructionsFileConflict(root string) error {
	path := filepath.Join(root, instructionsFile)
	if _, err := os.Stat(path); err == nil {
		return usageErr("%s exists; run `git ticket instructions` and append it yourself", instructionsFile)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

// writeInstructionsFile writes the block to AGENTS.md at the repository root
// and returns the path it wrote.
func writeInstructionsFile(root string) (string, error) {
	path := filepath.Join(root, instructionsFile)
	body := instructionsText
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
