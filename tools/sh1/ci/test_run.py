import copy
import importlib.util
from pathlib import Path
import unittest
import sys
import tempfile

spec = importlib.util.spec_from_file_location("lane", Path(__file__).with_name("run.py"))
lane = importlib.util.module_from_spec(spec)
spec.loader.exec_module(lane)


class InventoryAdmissionTests(unittest.TestCase):
    def inventory(self):
        return {"packages": [{"package": name, "status": "pass", "tests": [
            {"name": "TestRealEntry", "file": "entry_test.go", "executed": True, "status": "pass"}
        ]} for name in (lane.PACKAGE, lane.PACKAGE + "/musepty")]}

    def test_both_packages_pass(self):
        self.assertEqual(lane.admit_inventory(self.inventory()), [])

    def test_missing_or_empty_or_nonpassing_package_refuses(self):
        for change in ("missing", "empty", "failed", "unexecuted"):
            with self.subTest(change=change):
                inv = self.inventory()
                if change == "missing":
                    inv["packages"].pop()
                elif change == "empty":
                    inv["packages"][1]["tests"] = []
                elif change == "failed":
                    inv["packages"][1]["status"] = "fail"
                else:
                    inv["packages"][1]["tests"][0]["executed"] = False
                self.assertTrue(lane.admit_inventory(inv))

    def test_skip_fail_unknown_and_unobserved_refuse(self):
        for status in ("skip", "fail", "unknown", "not_observed"):
            with self.subTest(status=status):
                inv = self.inventory()
                inv["packages"][0]["tests"][0]["status"] = status
                self.assertTrue(lane.admit_inventory(inv))

    def test_tagged_absence_is_recorded_but_tagged_skip_refuses(self):
        inv = self.inventory()
        tagged = copy.deepcopy(inv["packages"][0]["tests"][0])
        tagged.update(name="TestTagged", status="not_observed", executed=False, tags="mutants")
        inv["packages"][0]["tests"].append(tagged)
        self.assertEqual(lane.admit_inventory(inv), [])
        tagged.update(status="skip", executed=True)
        self.assertTrue(lane.admit_inventory(inv))

    def test_counts_include_skips_tags_optins_and_subtests(self):
        inv = self.inventory()
        inv["packages"][0]["tests"].append({"name": "TestRealEntry/skip", "status": "skip", "executed": True,
                                            "tags": "mutants", "opt_in": "qualified provider"})
        self.assertEqual(lane.counts(inv), {"discovered": 2, "dynamic_subtests": 1, "executed": 3,
                                         "skipped": 1, "failed": 0, "tagged": 1, "opt_in": 1, "not_observed": 0})

    def test_real_entry_calls_admission_and_has_no_name_mask(self):
        source = Path(__file__).with_name("run.py").read_text()
        self.assertIn('self.summary["failures"].extend(admit_inventory(inventory))', source)
        self.assertNotIn("-run", lane.TEST_COMMAND)
        self.assertNotIn("-short", lane.TEST_COMMAND)


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
