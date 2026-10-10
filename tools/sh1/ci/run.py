"""Hosted-only SH1 lanes. Upload only the explicitly listed JSON reports."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time
import zipfile

MODULE = "github.com/relux-works/swarma-session-host"
PACKAGE = "github.com/aagrigore/task-board/internal/sessionmanager"
TEST_COMMAND = ["go", "test", "-json", "-count=1", "-timeout=12m",
                "./internal/sessionmanager", "./internal/sessionmanager/musepty"]
REPORTS = {"summary.json", "manifest.json", "project.json", "inventory.json",
           "guards.json", "mutations.json", "consumer.json"}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def admit_inventory(inventory):
    """The measured runtime set cannot launder a skip or an absent package."""
    packages = {p["package"]: p for p in inventory["packages"]}
    violations = []
    for name in (PACKAGE, PACKAGE + "/musepty"):
        row = packages.get(name)
        if row is None or row["status"] != "pass":
            violations.append({"package": name, "status": "package_not_passed"})
            continue
        tests = row["tests"]
        if not tests or not any(t.get("executed") for t in tests):
            violations.append({"package": name, "status": "no_tests_executed"})
        for test in tests:
            # Inactive build-tagged declarations remain visible, with tags.
            if test["status"] == "not_observed" and test.get("tags"):
                continue
            if test["status"] != "pass" or not test.get("executed"):
                violations.append({"package": name, "test": test["name"],
                                   "status": test["status"],
                                   "reason": test.get("reason", "unknown")})
    return violations


def counts(inventory):
    tests = [t for p in inventory["packages"]
             if p["package"] in (PACKAGE, PACKAGE + "/musepty") for t in p["tests"]]
    return {"discovered": sum(bool(t.get("file")) for t in tests),
            "dynamic_subtests": sum(not bool(t.get("file")) for t in tests),
            "executed": sum(bool(t.get("executed")) for t in tests),
            "skipped": sum(t["status"] == "skip" for t in tests),
            "failed": sum(t["status"] == "fail" for t in tests),
            "tagged": sum(bool(t.get("tags")) for t in tests),
            "opt_in": sum(bool(t.get("opt_in")) for t in tests),
            "not_observed": sum(t["status"] == "not_observed" for t in tests)}


def inventory_source(checkout, destination):
    """Inventory the two tested packages at their original module-relative paths."""
    destination.mkdir()
    shutil.copyfile(checkout / "go.mod", destination / "go.mod")
    shutil.copytree(checkout / "internal/sessionmanager", destination / "internal/sessionmanager")
    return destination


def publish_local_module(root, proxy):
    """Declared retained board libraries, not a replacement of the candidate."""
    module = root.joinpath("go.mod").read_text().splitlines()[0].split()[1]
    version = "v0.0.0"
    dest = proxy / module / "@v"
    dest.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(root / "go.mod", dest / (version + ".mod"))
    write(dest / (version + ".info"), {"Version": version, "Time": "2026-10-10T00:00:00Z"})
    (dest / "list").write_text(version + "\n")
    with zipfile.ZipFile(dest / (version + ".zip"), "w", zipfile.ZIP_DEFLATED) as archive:
        for path in sorted(root.rglob("*")):
            if ".git" in path.parts:
                continue
            if path.is_symlink():
                raise RuntimeError("local module contains a symlink")
            if path.is_file():
                archive.write(path, module + "@" + version + "/" + path.relative_to(root).as_posix())


class Lane:
    def __init__(self):
        self.host = Path("host").resolve()
        self.board = Path("board").resolve()
        self.checkout = self.board / "tools/board-cli"
        self.temp = Path(os.environ["RUNNER_TEMP"]) / "sh1-work"
        self.temp.mkdir()
        self.reports = self.temp.parent / "sh1-reports"
        self.reports.mkdir()
        self.tool = self.temp / "sh1"
        self.candidate = self.temp / "candidate"
        self.summary = {"lane": os.environ["SH1_LANE"], "board_commit": os.environ["SH1_BOARD_COMMIT"],
                        "commands": [], "failures": [], "run_id": os.environ["GITHUB_RUN_ID"],
                        "run_attempt": os.environ["GITHUB_RUN_ATTEMPT"],
                        "runner_os": os.environ["RUNNER_OS"], "build_tags": [],
                        "opt_in_environment": {}, "adapters": "empty version-1 table"}
        self.started = time.monotonic()

    def command(self, args, cwd=None, env=None, required=True, events=None):
        args = [str(a) for a in args]
        started = time.monotonic()
        print("RUN", self.safe(" ".join(args)), flush=True)
        result = subprocess.run(args, cwd=cwd or self.host, env=env, capture_output=True)
        # Raw output can contain board code; it stays in job logs, never artifacts.
        out = result.stdout.decode(errors="replace")
        err = result.stderr.decode(errors="replace")
        row = {"command": [self.safe(a) for a in args], "exit": result.returncode,
               "duration_seconds": round(time.monotonic() - started, 3)}
        self.summary["commands"].append(row)
        print("EXIT", result.returncode, flush=True)
        if events:
            events.write_bytes(result.stdout)
            print(self.safe(out), end="")
            failures = []
            for line in out.splitlines():
                ev = json.loads(line)
                if ev.get("Action") in ("fail", "skip"):
                    failures.append({k: ev[k] for k in ("Action", "Package", "Test") if k in ev})
            self.summary["test_terminals"] = failures
        else:
            print(self.safe(out), end="")
        print(self.safe(err), end="", file=sys.stderr)
        if result.returncode and required:
            self.summary["failures"].append(row)
            raise RuntimeError("command refused: " + " ".join(row["command"]))
        return result

    def safe(self, text):
        for path, label in ((self.host, "$HOST"), (self.board, "$BOARD"),
                            (self.temp, "$WORK"), (self.temp.parent, "$RUNNER_TEMP")):
            text = text.replace(str(path), label)
        return text

    def prepare(self):
        actual = self.command(["git", "rev-parse", "HEAD"], self.board).stdout.decode().strip()
        if actual != self.summary["board_commit"]:
            raise RuntimeError("board checkout identity differs from resolved commit")
        self.summary["host_commit"] = self.command(["git", "rev-parse", "HEAD"]).stdout.decode().strip()
        self.summary["host_tree"] = self.command(["git", "rev-parse", "HEAD^{tree}"]).stdout.decode().strip()
        self.summary["go_version"] = self.command(["go", "version"]).stdout.decode().strip()
        self.summary["go_environment"] = self.command(["go", "env", "GOOS", "GOARCH", "GOVERSION", "GOTOOLCHAIN"]).stdout.decode().splitlines()
        self.summary["helper_hashes"] = {p.name: sha(p) for p in sorted((self.host / "tools/sh1/ci").glob("*")) if p.is_file()}
        self.summary["workflow_hash"] = sha(self.host / ".github/workflows/sh1-lanes.yml")
        self.command(["go", "build", "-o", self.tool, "./tools/sh1"])
        mod = self.temp / "candidate-go.mod"
        shutil.copyfile(self.checkout / "go.mod", mod)
        parsed = json.loads(self.command(["go", "mod", "edit", "-json"], self.checkout).stdout)
        args = ["go", "mod", "edit", "-module=" + MODULE]
        for replace in parsed.get("Replace") or []:
            old = replace["Old"]
            args.append("-dropreplace=" + old["Path"] + ("@" + old["Version"] if old.get("Version") else ""))
        self.command(args + [mod])
        self.command([self.tool, "generate", "-source", self.checkout,
                      "-package", "internal/sessionmanager", "-old", PACKAGE,
                      "-new", MODULE, "-mod", mod, "-out", self.candidate])
        shutil.copyfile(self.candidate / "sh1-manifest.json", self.reports / "manifest.json")
        manifest = json.loads((self.reports / "manifest.json").read_text())
        self.summary.update({"generator_digest": manifest["generator"],
                             "transform_digest": manifest["transform"],
                             "manifest_hash": sha(self.reports / "manifest.json"),
                             "source_go_mod_hash": manifest["source_mod"],
                             "candidate_go_mod_hash": sha(mod)})

    def projection(self):
        self.command([self.tool, "project", "-candidate", self.candidate,
                      "-checkout", self.checkout, "-package", "internal/sessionmanager",
                      "-commit", self.summary["board_commit"], "-adapters",
                      self.host / "tools/sh1/ci/adapters.json", "-report", self.reports / "project.json"])
        # The guard expectation is derived from the unchanged guard's source scope.
        targets = sorted(p.name for p in (self.checkout / "internal/sessionmanager").glob("claude*.go")
                         if not p.name.endswith("_test.go"))
        declaration = {"version": 1, "guards": [
            {"package": PACKAGE, "test": "TestClaudeContextProductionHasNoFilesystemRemoval", "targets": targets},
            {"package": PACKAGE, "test": "TestTheMutantInventoryCoversEveryPredicateOnTheRefusalPath",
             "targets": ["hosted_takeover_guard.go"]}],
            "opt_in": [
                {"package": PACKAGE, "test": name, "condition": condition}
                for name, condition in (
                    ("TestClaudeHostLiveSmoke", "TASK_BOARD_CLAUDE_LIVE_SMOKE=1 and trusted project and Claude binary"),
                    ("TestClaudeAdoptionLiveSmoke", "TASK_BOARD_CLAUDE_LIVE_SMOKE=1 and trusted project and Claude binary"),
                    ("TestMuseRealProviderBudgetPause", "TASK_BOARD_MUSE_REAL_PROVIDER=1 and qualified TASK_BOARD_MUSE_SMOKE_BIN"),
                )]}
        guards = self.temp / "guard-declaration.json"
        write(guards, declaration)
        self.summary["guard_declaration_hash"] = sha(guards)
        events = self.temp / "tests.jsonl"
        self.command(TEST_COMMAND, self.checkout, required=False, events=events)
        source = inventory_source(self.checkout, self.temp / "inventory-source")
        result = self.command([self.tool, "inventory", "-source", source,
                               "-events", events, "-guards", guards,
                               "-report", self.reports / "inventory.json"], required=False)
        if result.returncode:
            self.summary["failures"].append({"gate": "inventory/guard coverage", "exit": result.returncode})
        if (self.reports / "inventory.json").exists():
            inventory = json.loads((self.reports / "inventory.json").read_text())
            write(self.reports / "guards.json", {"guard_ratio": inventory["guard_ratio"],
                                               "guards": inventory["guards"],
                                               "bound": "unchanged tests emit no SH1_GUARD read markers; reached is unproven"})
            self.summary["inventory_counts"] = counts(inventory)
            self.summary["guard_ratio"] = inventory["guard_ratio"]
            self.summary["failures"].extend(admit_inventory(inventory))
        else:
            self.summary["failures"].append({"gate": "inventory unavailable", "status": "unknown"})
        result = self.command([self.tool, "mutate", "-checkout", self.checkout,
                               "-mutations", self.host / "tools/sh1/ci/mutations.json",
                               "-report", self.reports / "mutations.json", "--"] + TEST_COMMAND, required=False)
        if result.returncode:
            self.summary["failures"].append({"gate": "mutations", "exit": result.returncode, "status": "unproven unless report records kills"})
        if (self.reports / "mutations.json").exists():
            self.summary["mutation_ratio"] = json.loads((self.reports / "mutations.json").read_text())["kill_ratio"]

    def consumer(self):
        # Stage all upstream dependency bytes first, then forbid network fallback.
        cache = self.temp / "upstream-cache"
        env = dict(os.environ, GOMODCACHE=str(cache), GOPROXY="https://proxy.golang.org", GOSUMDB="sum.golang.org")
        self.command(["go", "mod", "download", "all"], self.checkout, env)
        proxy = self.temp / "proxy"
        shutil.copytree(cache / "cache/download", proxy)
        for lib in ("board", "remoteconfig"):
            publish_local_module(self.board / "pkg" / lib, proxy)
        self.summary["retained_library_proxy"] = {str(p.relative_to(proxy)): sha(p) for p in sorted(proxy.rglob("*"))
                                                  if p.is_file() and "skill-project-management/pkg" in str(p.relative_to(proxy))}
        # Dependency exit: these tests need board-internal access and stay in lane 1.
        shutil.rmtree(self.checkout / "internal/sessionmanager")
        stamp = self.command(["git", "show", "-s", "--format=%ct", "HEAD"]).stdout.decode().strip()
        import datetime
        timestamp = datetime.datetime.fromtimestamp(int(stamp), datetime.timezone.utc).strftime("%Y%m%d%H%M%S")
        version = "v0.0.0-" + timestamp + "-" + self.summary["host_commit"][:12]
        result = self.command([self.tool, "consumer", "-candidate", self.candidate,
                               "-checkout", self.checkout, "-version", version,
                               "-proxy", proxy, "-report", self.reports / "consumer.json"], required=False)
        if result.returncode:
            self.summary["failures"].append({"gate": "consumer proxy selection", "exit": result.returncode})
        env = dict(os.environ, GOPROXY=proxy.as_uri(), GOSUMDB="off", GOPRIVATE="", GONOPROXY="none",
                   GONOSUMDB="none", GOMODCACHE=str(self.temp / "build-cache"), GOFLAGS="-mod=mod")
        selection = json.loads(self.command(["go", "mod", "edit", "-json"], self.checkout).stdout)
        if any(r["Old"]["Path"] == MODULE for r in selection.get("Replace") or []):
            raise RuntimeError("candidate replace found before consumer build")
        for args in (["go", "build", "./..."], ["go", "vet", "./..."]):
            result = self.command(args, self.checkout, env, required=False)
            if result.returncode:
                self.summary["failures"].append({"gate": " ".join(args), "exit": result.returncode})

    def finish(self):
        self.summary["duration_seconds"] = round(time.monotonic() - self.started, 3)
        self.summary["report_hashes"] = {p.name: sha(p) for p in sorted(self.reports.iterdir()) if p.is_file()}
        write(self.reports / "summary.json", self.summary)
        if set(p.name for p in self.reports.iterdir()) - REPORTS:
            raise RuntimeError("unexpected artifact file")


def main():
    lane = Lane()
    try:
        lane.prepare()
        if lane.summary["lane"] == "projection":
            lane.projection()
        elif lane.summary["lane"] == "consumer":
            lane.consumer()
        else:
            raise RuntimeError("unknown lane")
    except Exception as error:
        lane.summary["failures"].append({"error": lane.safe(str(error))})
        print(lane.safe(str(error)), file=sys.stderr)
    finally:
        lane.finish()
    return 1 if lane.summary["failures"] else 0


if __name__ == "__main__":
    sys.exit(main())
