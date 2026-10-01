package action

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/kryft-dev/cdd/internal/git"
)

// Runner starts the commands of Actions. The Picker and the Resolve flow
// take one, so tests inject a fake and never launch a real program.
type Runner interface {
	// Start launches a's command on the Project at path and returns once it
	// has started, without waiting, with its output discarded. It is how a
	// detached Action runs while the Picker stays open.
	Start(a Action, path string) error

	// Run launches a's command on the Project at path with the terminal for
	// stdin, stdout and stderr, waits, and returns its exit status. It is
	// how any other Action runs, after the Picker has quit. The error is
	// for a command that could not run at all, not one that exited non-zero.
	Run(a Action, path string) (int, error)
}

// ExecRunner is the Runner that runs commands as "sh -c".
type ExecRunner struct {
	// TTY is the terminal file Run connects the command to. The empty value
	// is /dev/tty, never the binary's own stdout, which the Wrapper
	// captures as the Jump target.
	TTY string
}

// Command builds the "sh -c" command for a on the Project at path: "{path}"
// in Run is replaced by the shell-quoted path, "{remote}" by the shell-quoted
// home page URL of the Project's git remote, the working directory is the
// Project, and CDD_PATH holds the path. It fails, without a command, when
// Run uses "{remote}" and the Project has none (git.ErrNoRemote,
// git.ErrNotRepo).
func Command(a Action, path string) (*exec.Cmd, error) {
	script := strings.ReplaceAll(a.Run, "{path}", shellQuote(path))
	if strings.Contains(script, "{remote}") {
		url, err := git.RemoteURL(context.Background(), path)
		if err != nil {
			return nil, err
		}
		script = strings.ReplaceAll(script, "{remote}", shellQuote(url))
	}
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = path
	cmd.Env = append(os.Environ(), "CDD_PATH="+path)
	return cmd, nil
}

// shellQuote wraps s in single quotes for sh, so spaces and metacharacters
// in it stay literal.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Start implements Runner. The command gets a session of its own, so it
// outlives the Picker, and /dev/null for stdio.
func (ExecRunner) Start(a Action, path string) error {
	cmd, err := Command(a, path)
	if err != nil {
		return err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }() // reap it
	return nil
}

// Run implements Runner.
func (r ExecRunner) Run(a Action, path string) (int, error) {
	name := r.TTY
	if name == "" {
		name = "/dev/tty"
	}
	tty, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("open terminal: %w", err)
	}
	defer func() { _ = tty.Close() }()

	cmd, err := Command(a, path)
	if err != nil {
		return 0, err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	err = cmd.Run()

	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, err
}
