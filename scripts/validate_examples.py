#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# ///
"""Check every example is valid on its own, against the provider built from this repository.

    python3 scripts/validate_examples.py

`make examples` runs it. Each resource, data source and provider example, and each scenario,
becomes a module of its own, with nothing added but the provider's source: an example has to
define everything it refers to. One `terraform validate` then checks them all, and the errors are
reported against the example's own file. Needs Go and Terraform; a scenario that uses another
provider, like random, needs `terraform init` to be able to download it.
"""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
EXAMPLES = REPO_ROOT / "examples"

PROVIDER_SOURCE = """terraform {
  required_providers {
    gitmoap = {
      source = "gitmoap/gitmoap"
    }
  }
}
"""


def examples() -> list[tuple[str, list[Path]]]:
    """Each example, as its name and the files it's made of."""
    found = [(str(path.relative_to(REPO_ROOT)), [path]) for path in sorted(EXAMPLES.glob("provider/*.tf"))]
    found += [(str(path.relative_to(REPO_ROOT)), [path]) for path in sorted(EXAMPLES.glob("resources/*/resource*.tf"))]
    found += [(str(path.relative_to(REPO_ROOT)), [path]) for path in sorted(EXAMPLES.glob("data-sources/*/data-source*.tf"))]
    for directory in sorted((EXAMPLES / "scenarios").iterdir()):
        found.append((str(directory.relative_to(REPO_ROOT)), [path for path in sorted(directory.rglob("*")) if path.is_file()]))
    return found


VARIABLE = re.compile(r'^variable "([\w-]+)" \{(.*?)^\}', re.MULTILINE | re.DOTALL)
PLACEHOLDERS = {"string": '"example"', "number": "1", "bool": "true", "list": "[]", "set": "[]", "map": "{}"}


def required_variables(files: list[Path]) -> dict[str, str]:
    """A placeholder for each variable the example declares without a default, which the module
    call has to pass."""
    values = {}
    for file in files:
        if file.suffix != ".tf":
            continue
        for name, body in VARIABLE.findall(file.read_text()):
            if re.search(r"^\s*default\s*=", body, re.MULTILINE):
                continue
            kind = re.search(r"^\s*type\s*=\s*([a-z]+)", body, re.MULTILINE)
            values[name] = PLACEHOLDERS.get(kind.group(1) if kind else "string", '"example"')
    return values


def main() -> int:
    for tool in ("go", "terraform"):
        if not shutil.which(tool):
            sys.exit(f"error: {tool} is not on PATH")
    with tempfile.TemporaryDirectory() as temporary:
        root = Path(temporary)
        binaries = root / "bin"
        subprocess.run(["go", "build", "-o", str(binaries / "terraform-provider-gitmoap"), "."], cwd=REPO_ROOT, check=True)
        cli_config = root / "terraformrc"
        cli_config.write_text(
            'provider_installation {\n  dev_overrides {\n    "registry.terraform.io/gitmoap/gitmoap" = "%s"\n  }\n  direct {}\n}\n' % binaries
        )

        workspace = root / "workspace"
        modules = {}
        lines = []
        for index, (name, files) in enumerate(examples()):
            module = f"example_{index}"
            directory = workspace / module
            base = files[0].parent if len(files) > 1 else None
            for file in files:
                target = directory / (file.relative_to(base) if base else "main.tf")
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(file, target)
            if not any("required_providers" in file.read_text() for file in files if file.suffix == ".tf"):
                (directory / "versions.tf").write_text(PROVIDER_SOURCE)
            modules[module] = name
            arguments = "".join(f"  {variable} = {value}\n" for variable, value in required_variables(files).items())
            lines.append(f'module "{module}" {{\n  source = "./{module}"\n{arguments}}}\n')
        (workspace / "main.tf").write_text("\n".join(lines))

        env = {**os.environ, "TF_CLI_CONFIG_FILE": str(cli_config), "TF_IN_AUTOMATION": "1"}
        init = subprocess.run(["terraform", "init", "-backend=false", "-input=false", "-no-color"], cwd=workspace, env=env, capture_output=True, text=True)
        if init.returncode != 0:
            print(init.stdout + init.stderr, file=sys.stderr)
            return 1
        result = subprocess.run(["terraform", "validate", "-json", "-no-color"], cwd=workspace, env=env, capture_output=True, text=True)
        report = json.loads(result.stdout)

    errors = []
    for diagnostic in report.get("diagnostics", []):
        if diagnostic["severity"] != "error":
            continue
        filename = diagnostic.get("range", {}).get("filename", "")
        module, _, file = filename.partition("/")
        where = modules.get(module, filename)
        if file and file not in ("main.tf", "versions.tf"):
            where += f"/{file}"
        line = diagnostic.get("range", {}).get("start", {}).get("line")
        errors.append(f"{where}{f':{line}' if line else ''}: {diagnostic['summary']}: {diagnostic.get('detail', '')}".strip())
    print(f"validated {len(modules)} examples", file=sys.stderr)
    if errors:
        print(f"{len(errors)} errors:", file=sys.stderr)
        for error in errors:
            print(f"  {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
