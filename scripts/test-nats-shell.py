#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Exercise a bundled nls binary and local nats-server through a real terminal."""

import fcntl
import json
import os
from pathlib import Path
import pty
import select
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time


def wait_for_server(process, directory):
    ports = directory / f"{Path(process.args[0]).name}_{process.pid}.ports"
    deadline = time.monotonic() + 10
    while process.poll() is None and time.monotonic() < deadline:
        try:
            return json.loads(ports.read_text())["nats"][0]
        except (FileNotFoundError, json.JSONDecodeError):
            time.sleep(0.05)
    raise RuntimeError("JetStream server did not become ready")


def wait_for_output(terminal, expected):
    output = bytearray()
    deadline = time.monotonic() + 15
    while time.monotonic() < deadline:
        ready, _, _ = select.select([terminal], [], [], max(0, deadline - time.monotonic()))
        if not ready:
            break
        try:
            output.extend(os.read(terminal, 65536))
        except OSError:
            break
        if expected in output:
            return
    raise AssertionError(f"Expected {expected!r}, received {bytes(output)!r}")


def test_terminal(binary, server, directory):
    env = {
        **os.environ,
        "SHELL": "/bin/sh",
        "ENV": "",
        "TERM": "xterm-256color",
        "PS1": "nls-test> ",
        "TMPDIR": str(directory),
        "XDG_CONFIG_HOME": str(directory),
    }
    child, terminal = pty.fork()
    if child == 0:
        os.chdir(directory)
        fcntl.ioctl(0, termios.TIOCSWINSZ, struct.pack("HHHH", 30, 120, 0, 0))
        os.execve(binary, [binary, "--server", server], env)

    try:
        wait_for_output(terminal, b"<s>")
        os.write(terminal, b"s")
        wait_for_output(terminal, b"nls-test> ")
        assert any(directory.glob("nls-nats-*/nats")), "bundled CLI was not extracted"
        os.write(terminal, b"nats stream add ORDERS --subjects 'orders.>' --defaults"
                 b" && nats pub --jetstream orders.created payload"
                 b" && nats stream info ORDERS --json > stream.json"
                 b" && printf 'JETSTREAM_OK\\n'\n")
        wait_for_output(terminal, b"\r\nJETSTREAM_OK\r\n")
        info = json.loads((directory / "stream.json").read_text())
        assert info["config"]["subjects"] == ["orders.>"], info
        assert info["state"]["messages"] == 1, info

        os.write(terminal, b"sleep 30\n")
        time.sleep(0.25)
        os.write(terminal, b"\x03")
        wait_for_output(terminal, b"nls-test> ")
        os.write(terminal, b"exit\n")
        wait_for_output(terminal, b"orders")
        assert not list(directory.glob("nls-nats-*")), "temporary CLI executable leaked"

        os.write(terminal, b"q")
        wait_for_output(terminal, b"Goodbye!")
        assert os.waitstatus_to_exitcode(os.waitpid(child, 0)[1]) == 0
        child = None
    finally:
        os.close(terminal)
        if child is not None:
            os.kill(child, signal.SIGKILL)
            os.waitpid(child, 0)


def main():
    binary, nats_server = (str(Path(name).resolve()) for name in sys.argv[1:])
    with tempfile.TemporaryDirectory() as workdir:
        directory = Path(workdir)
        with subprocess.Popen([
            nats_server, "-js", "-a", "127.0.0.1", "-p", "-1",
            "-sd", workdir, "--ports_file_dir", workdir,
        ]) as server:
            try:
                test_terminal(binary, wait_for_server(server, directory), directory)
            finally:
                server.terminate()


if __name__ == "__main__":
    main()
