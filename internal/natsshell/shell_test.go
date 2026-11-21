// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package natsshell

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a POSIX shell")
	}
	const server = "nats://127.0.0.1:4223"
	const fakeCLI = "#!/bin/sh\nprintf '%s|%s\\n' \"$NATS_URL\" \"$*\"\n"
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("ENV", "")
	t.Setenv("NATS_URL", "nats://elsewhere:4222")

	oldBundle := bundledNATS
	t.Cleanup(func() { bundledNATS = oldBundle })
	for _, mode := range []string{"PATH", "bundled", "fallback", "missing"} {
		t.Run(mode, func(t *testing.T) {
			path := t.TempDir()
			tmp := t.TempDir()
			t.Setenv("PATH", path)
			t.Setenv("TMPDIR", tmp)
			bundledNATS = nil
			switch mode {
			case "PATH", "fallback":
				if err := os.WriteFile(filepath.Join(path, "nats"), []byte(fakeCLI), 0700); err != nil {
					t.Fatal(err)
				}
				if mode == "fallback" {
					bundledNATS = []byte("#!/nonexistent-interpreter\n")
				}
			case "bundled":
				bundledNATS = []byte(fakeCLI)
				if err := os.WriteFile(filepath.Join(path, "nats"), []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
					t.Fatal(err)
				}
			case "missing":
				t.Setenv("SHELL", "")
			}

			var output bytes.Buffer
			session := Session{Server: server}
			session.SetStdin(strings.NewReader("nats stream add 'orders.>' | while read -r line; do printf '%s\\n' \"$line\"; done\nprintf 'shell is usable\\n'\nexit 7\n"))
			session.SetStdout(&output)
			session.SetStderr(&output)
			if err := session.Run(); err != nil {
				t.Fatal(err)
			}
			want := server + "|stream add orders.>"
			if mode == "missing" {
				want = "NATS CLI unavailable on PATH"
			}
			if !strings.Contains(output.String(), want) || !strings.Contains(output.String(), "shell is usable") {
				t.Errorf("expected %q and a usable shell, got %q", want, output.String())
			}
			if mode == "fallback" && !strings.Contains(output.String(), "Bundled NATS CLI unavailable") {
				t.Errorf("expected a bundled CLI diagnostic, got %q", output.String())
			}
			files, err := os.ReadDir(tmp)
			if err != nil || len(files) != 0 {
				t.Errorf("expected shell to remove its temporary files, got %v, %v", files, err)
			}
		})
	}
}

func TestSessionBashResetsPath(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("requires Bash")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", bash)
	t.Setenv("TMPDIR", t.TempDir())
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export PATH=/nonexistent\n"), 0600); err != nil {
		t.Fatal(err)
	}

	oldBundle := bundledNATS
	t.Cleanup(func() { bundledNATS = oldBundle })
	bundledNATS = []byte("#!/bin/sh\nprintf 'bundled CLI\\n'\n")

	var output bytes.Buffer
	session := Session{}
	session.SetStdin(strings.NewReader("test \"$PATH\" = /nonexistent && \"$NLS_NATS_BIN\"\nexit\n"))
	session.SetStdout(&output)
	if err := session.Run(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "bundled CLI\n") {
		t.Errorf("expected bundled CLI to run after Bash reset PATH, got %q", output.String())
	}
}

func TestSessionDefaultStreams(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a POSIX shell")
	}
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("ENV", "")
	session := Session{}
	session.SetStdin(strings.NewReader("exit\n"))
	if err := session.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionInvalidShell(t *testing.T) {
	t.Setenv("SHELL", filepath.Join(t.TempDir(), "missing-shell"))
	if err := (&Session{}).Run(); err == nil {
		t.Fatal("expected a shell startup error")
	}
}
