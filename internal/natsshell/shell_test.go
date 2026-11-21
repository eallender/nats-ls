// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package natsshell

import (
	"bytes"
	"os"
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
	for _, mode := range []string{"PATH", "bundled", "missing"} {
		t.Run(mode, func(t *testing.T) {
			path := t.TempDir()
			tmp := t.TempDir()
			t.Setenv("PATH", path)
			t.Setenv("TMPDIR", tmp)
			bundledNATS = nil
			switch mode {
			case "PATH":
				if err := os.WriteFile(filepath.Join(path, "nats"), []byte(fakeCLI), 0700); err != nil {
					t.Fatal(err)
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
			files, err := os.ReadDir(tmp)
			if err != nil || len(files) != 0 {
				t.Errorf("expected shell to remove its temporary files, got %v, %v", files, err)
			}
		})
	}
}

func TestSessionInvalidShell(t *testing.T) {
	t.Setenv("SHELL", filepath.Join(t.TempDir(), "missing-shell"))
	if err := (&Session{}).Run(); err == nil {
		t.Fatal("expected a shell startup error")
	}
}
