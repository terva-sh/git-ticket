package cli

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

// copyGOOS is runtime.GOOS behind a variable, so the native-Windows refusal
// is testable from the platforms the feature supports. The same move as
// runningVersion in selfupdate.go.
var copyGOOS = runtime.GOOS

// clipboardTool is one PATH candidate. The args matter: xclip and xsel write
// the primary selection by default, and a copy that lands in the wrong
// selection reads exactly like a copy that did nothing.
type clipboardTool struct {
	name string
	args []string
}

// clipboardTools is the probe order of plan 12.7. The order is load-bearing
// at the end: WSL2 is Linux with clip.exe on PATH, so the Linux tools must
// win when a display server is present, and clip.exe is the answer only when
// nothing else is.
var clipboardTools = []clipboardTool{
	{"pbcopy", nil},
	{"wl-copy", nil},
	{"xclip", []string{"-selection", "clipboard"}},
	{"xsel", []string{"--input", "--clipboard"}},
	{"clip.exe", nil},
}

// runCopy puts a ticket's body on the system clipboard, per plan 12.7. On
// success it prints one line to stderr and nothing to stdout: quiet pipes,
// loud humans.
func runCopy(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("copy", args, nil)
	if err != nil {
		return err
	}
	if ctx.g.json {
		return usageErr("copy has no --json form; show --json carries the body")
	}
	if len(rest) != 1 {
		return usageErr("copy takes one ticket ID")
	}
	if copyGOOS == "windows" {
		return fmt.Errorf("copy supports WSL2, Linux, and macOS, not native Windows; use `show %s --body` and redirect instead", rest[0])
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	t, err := s.Get(context.Background(), rest[0])
	if err != nil {
		return err
	}
	data, err := os.ReadFile(t.Path)
	if err != nil {
		return err
	}
	body, err := ticket.RawBody(data)
	if err != nil {
		return err
	}
	via, err := writeClipboard(ctx.env, []byte(body))
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.env.Stderr, "copied %s (%d bytes) via %s\n", t.ID, len(body), via)
	return nil
}

// writeClipboard finds a way to the clipboard and reports which one it took:
// the probe order first, OSC 52 second, per plan 12.7. A tool that is found
// but fails ends the command rather than falling through, because pretending
// the clipboard changed is the one wrong answer.
func writeClipboard(env Env, body []byte) (string, error) {
	look := env.LookPath
	if look == nil {
		look = exec.LookPath
	}
	run := env.RunTool
	if run == nil {
		run = runClipboardTool
	}
	for _, tool := range clipboardTools {
		path, err := look(tool.name)
		if err != nil {
			continue
		}
		if err := run(path, tool.args, body); err != nil {
			return "", fmt.Errorf("%s failed: %w", tool.name, err)
		}
		return tool.name, nil
	}
	if err := writeOSC52(env, body); err != nil {
		return "", err
	}
	return "OSC 52", nil
}

// runClipboardTool is the real exec behind the RunTool seam: the body on
// stdin, nothing read back, and the tool's own words in the error when it
// fails, because "exit status 1" alone tells the user nothing.
//
// Every descriptor is a real file rather than a pipe, and that is the whole
// fix for a hang the first real-world run hit. wl-copy, xclip, and xsel fork
// a child that serves the selection until something replaces it, and the
// child holds the inherited descriptors open. A pipe there means waiting for
// EOF that only arrives at the next copy, which read as a two-minute timeout
// against wl-copy. A file holds nothing open, so Run returns when the tool's
// own process exits, which wl-copy does promptly after forking.
func runClipboardTool(path string, args []string, stdin []byte) error {
	in, err := clipTemp("git-ticket-clip-in-*")
	if err != nil {
		return err
	}
	defer os.Remove(in.Name())
	defer in.Close()
	if _, err := in.Write(stdin); err != nil {
		return err
	}
	if _, err := in.Seek(0, io.SeekStart); err != nil {
		return err
	}
	out, err := clipTemp("git-ticket-clip-out-*")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	defer out.Close()

	cmd := exec.Command(path, args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out
	if runErr := cmd.Run(); runErr != nil {
		data, _ := os.ReadFile(out.Name())
		if msg := strings.TrimSpace(string(data)); msg != "" {
			return fmt.Errorf("%w: %s", runErr, msg)
		}
		return runErr
	}
	return nil
}

// clipTemp is os.CreateTemp with the one intent of this file: a scratch file
// that stands in for a pipe.
func clipTemp(pattern string) (*os.File, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return nil, fmt.Errorf("clipboard scratch file: %w", err)
	}
	return f, nil
}

// writeOSC52 puts the body on the clipboard through the controlling
// terminal, which is the path that works over SSH where no tool can. The
// failure message names every tool the probe looked for, because the user's
// repair is installing one of them.
func writeOSC52(env Env, body []byte) error {
	open := env.ClipboardTTY
	if open == nil {
		open = func() (io.WriteCloser, error) { return os.OpenFile("/dev/tty", os.O_WRONLY, 0) }
	}
	tty, err := open()
	if err != nil {
		names := make([]string, 0, len(clipboardTools))
		for _, t := range clipboardTools {
			names = append(names, t.name)
		}
		return fmt.Errorf("no clipboard tool on PATH (looked for %s) and no controlling terminal for OSC 52", strings.Join(names, ", "))
	}
	defer tty.Close()
	_, err = fmt.Fprintf(tty, "\x1b]52;c;%s\a", base64.StdEncoding.EncodeToString(body))
	return err
}
