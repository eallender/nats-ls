# SPDX-License-Identifier: Apache-2.0
"""Collect license and attribution files for the packages in the release builds."""

import os
from pathlib import Path
import re
import subprocess
import sys


LICENSE_FILE = re.compile(r"(LICENSE|LICENCE|COPYING)([._-].*)?", re.I)
NOTICE_FILE = re.compile(r"(NOTICE|COPYRIGHT|PATENTS)([._-].*)?", re.I)
PACKAGE_FIELDS = (
    '{{if .Dir}}{{.Dir}}{{"\t"}}{{.ImportPath}}{{"\t"}}'
    '{{with .Module}}{{.Path}}{{"\t"}}{{.Version}}{{"\t"}}{{.Dir}}{{end}}{{end}}'
)


def go_output(*args, cwd, env):
    return subprocess.check_output(["go", *args], cwd=cwd, env=env, text=True)


def collect(notices, source, command, main_version, targets):
    for target in targets:
        goos, goarch = target.split("/")
        env = dict(os.environ, GOOS=goos, GOARCH=goarch, CGO_ENABLED="0")
        goroot = Path(go_output("env", "GOROOT", cwd=source, env=env).strip())
        go_version = go_output("env", "GOVERSION", cwd=source, env=env).strip()
        packages = go_output(
            "list", "-mod=readonly", "-buildvcs=false", "-deps", "-f", PACKAGE_FIELDS, command,
            cwd=source, env=env,
        )
        for line in filter(None, packages.splitlines()):
            directory, package, *module = line.split("\t")
            directory = Path(directory)
            if module and module[0]:
                name, version, root = module
                root = Path(root)
                if not version and main_version is None:
                    continue  # NLS's own license is shipped separately.
                component = f"{name} {version or main_version}"
            else:
                root = goroot
                component = f"Go {go_version}"

            files = [
                file
                for parent in (directory, *directory.parents)
                if parent.is_relative_to(root)
                for file in parent.iterdir()
                if file.is_file()
                and (LICENSE_FILE.fullmatch(file.name) or NOTICE_FILE.fullmatch(file.name))
            ]
            if not any(LICENSE_FILE.fullmatch(file.name) for file in files):
                raise SystemExit(f"No license files found for {package} ({component})")
            for file in files:
                key = f"{component} / {file.relative_to(root)}"
                notices[key] = file.read_bytes().rstrip()


def main():
    upstream, version, output, *targets = sys.argv[1:]
    notices = {}
    collect(notices, Path(upstream), "./nats", version, targets)
    collect(notices, Path.cwd(), "./cmd/nls", None, targets)
    with Path(output).open("wb") as report:
        report.write(b"License and attribution files for NLS and the bundled NATS CLI.\n")
        report.write(f"Bundled NATS CLI {version} is from The NATS Authors.\n".encode())
        for title, content in sorted(notices.items()):
            report.write(f"\n{'=' * 72}\n{title}\n\n".encode())
            report.write(content + b"\n")


if __name__ == "__main__":
    main()
