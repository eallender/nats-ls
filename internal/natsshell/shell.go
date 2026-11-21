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

	stdin, stdout, stderr := s.stdin, s.stdout, s.stderr
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	path := os.Getenv("PATH")
	var binary string
	if len(bundledNATS) != 0 {
		dir, err := os.MkdirTemp("", "nls-nats-")
		if err == nil {
			defer func() { _ = os.RemoveAll(dir) }()
			binary, err = prepareCLI(dir)
		}
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Bundled NATS CLI unavailable: %v. Use an installed NATS CLI or set TMPDIR to a writable directory that allows execution.\n", err)
		} else {
			if path != "" {
				path = filepath.Dir(binary) + string(os.PathListSeparator) + path
			} else {
				path = filepath.Dir(binary)
			}
		}
	}
	if binary == "" {
		if _, err := exec.LookPath("nats"); err != nil {
			_, _ = fmt.Fprintln(stdout, "NATS CLI unavailable on PATH. Install it from https://github.com/nats-io/natscli/releases.")
		}
	}

	_, _ = fmt.Fprintln(stdout, "NLS shell · Run nats --help to see commands · Type exit to return to NLS")
	cmd.Env = append(os.Environ(), "NATS_URL="+s.Server, "PATH="+path, "NLS_NATS_BIN="+binary)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// An interactive shell can exit with the status of its last command.
		return nil
	}
	return err
}

func prepareCLI(dir string) (string, error) {
	binary, err := filepath.Abs(filepath.Join(dir, "nats"))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(binary, bundledNATS, 0700); err != nil {
		return "", err
	}
	// Starting the binary detects noexec mounts even when its mode allows execution.
	if err := exec.Command(binary, "--version").Run(); err != nil {
		return "", err
	}
	return binary, nil
}
