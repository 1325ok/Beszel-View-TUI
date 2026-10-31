//go:build windows

package terminal

import "testing"

func TestSupportsGitBashANSIEnvironment(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("MSYSTEM", "MINGW64")
	if !supportsANSIEnvironment() {
		t.Fatal("Git Bash/mintty environment should support ANSI output")
	}
}

func TestRejectsDumbANSIEnvironment(t *testing.T) {
	t.Setenv("TERM", "dumb")
	t.Setenv("MSYSTEM", "")
	if supportsANSIEnvironment() {
		t.Fatal("TERM=dumb should not be treated as an ANSI terminal")
	}
}
