// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package natsshell

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Populated for release builds by scripts/bundle-nats-cli.sh.
var bundledNATS []byte

// Session runs an interactive shell on the terminal supplied by Bubble Tea.
type Session struct {
	Server string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (s *Session) SetStdin(r io.Reader)  { s.stdin = r }
func (s *Session) SetStdout(w io.Writer) { s.stdout = w }
func (s *Session) SetStderr(w io.Writer) { s.stderr = w }

func (s *Session) Run() error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	shellPath, err := exec.LookPath(shell)
	if err != nil {
		return fmt.Errorf("open %s: %w", shell, err)
	}
	cmd := exec.Command(shellPath, "-i")

	path := os.Getenv("PATH")
	if len(bundledNATS) != 0 {
		dir, err := os.MkdirTemp("", "nls-nats-")
		if err != nil {
			return fmt.Errorf("prepare bundled NATS CLI: %w", err)
		}
		defer func() { _ = os.RemoveAll(dir) }()
		if err := os.WriteFile(filepath.Join(dir, "nats"), bundledNATS, 0700); err != nil {
			return fmt.Errorf("prepare bundled NATS CLI: %w", err)
		}
		if path != "" {
			path = dir + string(os.PathListSeparator) + path
		} else {
			path = dir
		}
	} else if _, err := exec.LookPath("nats"); err != nil {
		_, _ = fmt.Fprintln(s.stdout, "NATS CLI unavailable on PATH. Install it from https://github.com/nats-io/natscli/releases.")
	}

	_, _ = fmt.Fprintln(s.stdout, "NLS shell · Run nats --help to see commands · Type exit to return to NLS")
	cmd.Env = append(os.Environ(), "NATS_URL="+s.Server, "PATH="+path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = s.stdin, s.stdout, s.stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// An interactive shell can exit with the status of its last command.
		return nil
	}
	return err
}
