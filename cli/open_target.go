package cli

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// openReferenceTarget runs the platform's ordinary opener only after the TUI
// user selects a resolved target. The target is one argv value, never shell
// text, so an identifier cannot introduce another command.
func openReferenceTarget(target string) error {
	if strings.HasPrefix(target, "https://") {
		u, err := url.Parse(target)
		if err != nil || u.Hostname() == "" || u.User != nil {
			return fmt.Errorf("reference target is not a safe HTTPS URL")
		}
	} else {
		if !filepath.IsAbs(target) {
			return fmt.Errorf("reference target is not an absolute local path")
		}
		if _, err := os.Stat(target); err != nil {
			return err
		}
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		if _, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command("xdg-open", target)
		} else if _, err := exec.LookPath("gio"); err == nil {
			cmd = exec.Command("gio", "open", target)
		} else {
			return fmt.Errorf("no desktop opener found for reference target")
		}
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
