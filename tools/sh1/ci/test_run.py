import copy
import json
import importlib.util
from pathlib import Path
import unittest
import sys
import tempfile

spec = importlib.util.spec_from_file_location("lane", Path(__file__).with_name("run.py"))
lane = importlib.util.module_from_spec(spec)
spec.loader.exec_module(lane)


class BaselineEquivalenceTests(unittest.TestCase):
    def inventory(self):
        return {"packages": [{"package": name, "status": "pass", "tests": [
            {"name": "TestRealEntry", "file": "entry_test.go", "executed": True, "status": "pass"},
            {"name": "TestOptIn", "file": "smoke_test.go", "executed": True,
             "status": "skip", "reason": "qualified provider absent", "opt_in": "PROVIDER=1"},
            {"name": "TestTagged", "file": "mutant_test.go", "executed": False,
             "status": "not_observed", "tags": "mutants"}
        ]} for name in (lane.PACKAGE, lane.PACKAGE + "/musepty")]}

    def test_intentional_skips_and_tagged_absence_are_equivalent(self):
        inv = self.inventory()
        result = lane.compare_inventories(inv, copy.deepcopy(inv), 0, 0)
        self.assertTrue(result["equivalent"])
        self.assertEqual(result["outcomes_compared"], 8)

    def test_changes_in_either_direction_fail_even_with_equal_counts(self):
        for change in ("fail", "skip", "missing", "reason", "tag", "dynamic", "opt_in"):
            with self.subTest(change=change):
                before, after = self.inventory(), self.inventory()
                tests = after["packages"][0]["tests"]
                if change in ("fail", "skip"):
                    tests[0].update(status=change, reason="unexpected skip")
                elif change == "missing":
                    tests.pop(0)
                elif change == "reason":
                    tests[1]["reason"] = "different condition"
                elif change == "tag":
                    tests[2]["tags"] = "other_tag"
                elif change == "opt_in":
                    tests[1]["opt_in"] = "OTHER=1"
                else:
                    tests.append({"name": "TestRealEntry/new", "status": "pass", "executed": True})
                for baseline, projection in ((before, after), (after, before)):
                    result = lane.compare_inventories(baseline, projection, 0, 0)
                    self.assertFalse(result["equivalent"])
                    self.assertTrue(result["differences"])

    def test_reordering_does_not_change_sets(self):
        before, after = self.inventory(), self.inventory()
        after["packages"].reverse()
        for package in after["packages"]:
            package["tests"].reverse()
        self.assertTrue(lane.compare_inventories(before, after, 0, 0)["equivalent"])

    def test_unselected_testdata_declarations_are_retained_and_compared(self):
        inv = self.inventory()
        inv["packages"].append({"package": lane.PACKAGE + "/testdata/fake_msp_server",
                                "status": "not_observed", "tests": [
                                    {"name": "TestFixture", "file": "internal/sessionmanager/testdata/fake_msp_server/main_test.go",
                                     "status": "not_observed", "executed": False}]})
        self.assertEqual(lane.compare_inventories(inv, inv, 0, 0)["outcomes_compared"], 10)
        changed = copy.deepcopy(inv)
        changed["packages"][-1]["tests"][0]["name"] = "TestOtherFixture"
        self.assertFalse(lane.compare_inventories(inv, changed, 0, 0)["equivalent"])
        for suffix in ("/other", "/testdata_not_fixture/fake"):
            invalid = copy.deepcopy(inv)
            invalid["packages"][-1]["package"] = lane.PACKAGE + suffix
            with self.assertRaises(ValueError):
                lane.compare_inventories(invalid, invalid, 0, 0)
        invalid = copy.deepcopy(inv)
        invalid["packages"][-1]["tests"][0].update(status="pass", executed=True)
        with self.assertRaises(ValueError):
            lane.compare_inventories(invalid, invalid, 0, 0)

    def test_identical_known_baseline_failures_are_recorded_and_equivalent(self):
        inv = self.inventory()
        inv["packages"][0]["status"] = "fail"
        inv["packages"][0]["tests"][0]["status"] = "fail"
        self.assertTrue(lane.compare_inventories(inv, inv, 1, 1)["equivalent"])
        self.assertFalse(lane.compare_inventories(inv, inv, 1, 0)["equivalent"])

    def test_unknown_missing_partial_and_duplicate_evidence_refuse(self):
        for change in ("missing", "empty", "unknown_package", "unknown_test", "unexecuted",
                       "untagged_absence", "unknown_reason", "duplicate_package", "duplicate_test"):
            with self.subTest(change=change):
                inv = self.inventory()
                package = inv["packages"][0]
                if change == "missing":
                    inv["packages"].pop()
                elif change == "empty":
                    package["tests"] = []
                elif change == "unknown_package":
                    package["status"] = "not_observed"
                elif change == "unknown_test":
                    package["tests"][0]["status"] = "unknown"
                elif change == "unexecuted":
                    package["tests"][0]["executed"] = False
                elif change == "untagged_absence":
                    package["tests"][2]["tags"] = ""
                elif change == "unknown_reason":
                    package["tests"][1]["reason"] = "unknown"
                elif change == "duplicate_package":
                    inv["packages"].append(copy.deepcopy(package))
                else:
                    package["tests"].append(copy.deepcopy(package["tests"][0]))
                with self.assertRaises(ValueError):
                    lane.compare_inventories(inv, inv, 0, 0)

    def test_counts_include_skips_tags_optins_and_subtests(self):
        inv = self.inventory()
        inv["packages"][0]["tests"].append({"name": "TestRealEntry/sub", "status": "pass", "executed": True})
        self.assertEqual(lane.counts(inv), {"discovered": 6, "dynamic_subtests": 1, "executed": 5,
                                         "skipped": 2, "failed": 0, "tagged": 2, "opt_in": 2, "not_observed": 2})


class ProjectionEntryTests(unittest.TestCase):
    inventory = BaselineEquivalenceTests.inventory
    def run_projection(self, root, changed=False, missing_report=False):
        instance = lane.Lane.__new__(lane.Lane)
        instance.host = root / "host"
        instance.checkout = root / "board"
        instance.temp = root / "work"
        instance.reports = root / "reports"
        instance.tool = root / "sh1"
        instance.candidate = root / "candidate"
        instance.summary = {"commands": [], "failures": [], "runner_os": "Linux", "board_commit": "a" * 40}
        for path in (instance.host, instance.checkout, instance.temp, instance.reports):
            path.mkdir()
        (instance.checkout / "internal/sessionmanager").mkdir(parents=True)
        (instance.checkout / "go.mod").write_text("module example.org/board\n")
        calls = []

        def command(args, cwd=None, env=None, required=True, events=None):
            calls.append([str(a) for a in args])
            if args == lane.TEST_COMMAND:
                events.write_text("retained stream\n")
            if "inventory" in args and not missing_report:
                inv = self.inventory()
                inv.update(guard_ratio="0/10", guards=[{"expected": ["file.go"], "reached": []}])
                if changed and str(args[-1]).endswith("/inventory.json"):
                    inv["packages"][0]["tests"][0]["status"] = "fail"
                lane.write(Path(args[-1]), inv)
            from subprocess import CompletedProcess
            return CompletedProcess(args, 1 if "inventory" in args else 0)

        instance.command = command
        instance.projection()
        return instance, calls

    def test_real_projection_entry_runs_baseline_first_and_invokes_gate(self):
        for changed in (False, True):
            with tempfile.TemporaryDirectory() as directory:
                instance, calls = self.run_projection(Path(directory), changed)
                self.assertEqual(calls[0], lane.TEST_COMMAND)
                self.assertEqual(calls[2][1], "project")
                self.assertEqual(calls[3], lane.TEST_COMMAND)
                self.assertEqual(bool(instance.summary["failures"]), changed)
                diff = json.loads((instance.reports / "equivalence.json").read_text())
                self.assertEqual(diff["equivalent"], not changed)
                guards = json.loads((instance.reports / "guards.json").read_text())
                self.assertFalse(guards["gating"])
                self.assertEqual(guards["guard_ratio"], "0/10")
                self.assertNotIn("-run", lane.TEST_COMMAND)
                self.assertNotIn("-short", lane.TEST_COMMAND)

    def test_missing_inventory_does_not_become_informative_guard_gap(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(FileNotFoundError):
                self.run_projection(Path(directory), missing_report=True)


class CommandEvidenceTests(unittest.TestCase):
    def test_nonzero_exit_survives_malformed_json(self):
        with tempfile.TemporaryDirectory() as directory:
            instance = lane.Lane.__new__(lane.Lane)
            instance.host = instance.board = instance.temp = Path(directory)
            instance.summary = {"commands": [], "failures": []}
            events = Path(directory) / "events.jsonl"
            with self.assertRaises(ValueError):
                instance.command([sys.executable, "-c", "print('malformed'); raise SystemExit(7)"],
                                 required=False, events=events)
            self.assertEqual(instance.summary["commands"][0]["exit"], 7)
            self.assertEqual(events.read_text(), "malformed\n")


class InventoryScopeTests(unittest.TestCase):
    def test_scope_preserves_recursive_test_bytes_without_unrelated_declarations(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            checkout = root / "checkout"
            package = checkout / "internal/sessionmanager"
            child = package / "musepty/testdata"
            child.mkdir(parents=True)
            (checkout / "go.mod").write_text("module example.org/board\n\ngo 1.25.5\n")
            (package / "host_test.go").write_bytes(b"package sessionmanager\n// unchanged test bytes\n")
            (child / "golden.txt").write_bytes(b"unchanged fixture\n")
            other = checkout / "internal/unrelated"
            other.mkdir()
            (other / "one_test.go").write_text("package unrelated\nfunc TestMain() {}\n")
            (other / "two_test.go").write_text("package unrelated\nfunc TestMain() {}\n")
            result = lane.inventory_source(checkout, root / "inventory")
            self.assertEqual((result / "internal/sessionmanager/host_test.go").read_bytes(),
                             (package / "host_test.go").read_bytes())
            self.assertEqual((result / "internal/sessionmanager/musepty/testdata/golden.txt").read_bytes(),
                             (child / "golden.txt").read_bytes())
            self.assertEqual((result / "go.mod").read_bytes(), (checkout / "go.mod").read_bytes())
            self.assertFalse((result / "internal/unrelated").exists())
