package cli

import "errors"

// runUI opens the interactive TUI. The command itself is thin on
// purpose: flag parsing and store resolution happen here, where every
// other command's do, and the terminal work lives behind Env.RunUI so
// this package never imports the terminal stack. Plan 12.1 carries
// the command; the split is the plan 12.2 embedding rule at work.
func runUI(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("ui", args, nil)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("ui takes no arguments")
	}
	if ctx.g.json {
		// An interactive screen has no JSON form, and pretending
		// otherwise would hand a script a stream of escape sequences.
		return usageErr("ui is interactive and has no --json form")
	}
	if ctx.env.RunUI == nil {
		return errors.New("this entrypoint has no terminal UI wired; run `git ticket ui` from the git-ticket binary")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	// The same resolution every writing command gets: --actor through
	// ctx.actor with its default-actor warning, and the claim provenance
	// from the working tree. Resolved here, before the alternate screen
	// takes stderr's usefulness away.
	branch, worktree, commit := gitState(ctx.env.Dir)
	env := ctx.env
	return ctx.env.RunUI(UIParams{
		Store:    s,
		Actor:    ctx.actor(s),
		Branch:   branch,
		Worktree: worktree,
		Commit:   commit,
		// The same write path as `git ticket copy`, per plan 12.7:
		// probe the tools, OSC 52 last, never a silent no-op.
		Clipboard: func(body []byte) (string, error) {
			return writeClipboard(env, body)
		},
	})
}
