import copy
import json
from pathlib import Path
import tempfile
import unittest

from dataset import DATASET_REVISION, export


class DatasetTests(unittest.TestCase):
    def setUp(self):
        row = {"id": "turn", "language": "en", "audio": [{"src": f"https://example.test/{DATASET_REVISION}/audio"}],
               "messages": [{"role": "assistant", "content": "Go on."}],
               "words": [{"word": " first ", "end": 0.6}, {"word": "future", "end": 0.8}],
               "silence_spans": [{"start": 0.1, "end": 0.4}, {"start": 1.0, "end": 1.3}, {"start": 2.0, "end": 2.3}]}
        self.payload = {"partial": False, "rows": [{"row_idx": index, "truncated_cells": [], "row": copy.deepcopy(row)} for index in range(100)]}

    def test_causal_prefix_and_original_labels(self):
        with tempfile.TemporaryDirectory() as directory:
            source, result = Path(directory) / "rows.json", Path(directory) / "corpus.json"
            source.write_text(json.dumps(self.payload))
            export(source, result)
            corpus = json.loads(result.read_text())
            self.assertEqual(len(corpus["cases"]), 200)
            self.assertEqual(len(corpus["no_text_exclusions"]), 100)
            self.assertEqual(corpus["cases"][0]["text"], "first")
            self.assertFalse(corpus["cases"][0]["complete"])
            self.assertEqual(corpus["cases"][1]["text"], "first future")
            self.assertTrue(corpus["cases"][1]["complete"])

    def test_short_final_span_does_not_relabel_previous_hold(self):
        for entry in self.payload["rows"]:
            entry["row"]["silence_spans"][-1]["end"] = 2.1
        with tempfile.TemporaryDirectory() as directory:
            source, result = Path(directory) / "rows.json", Path(directory) / "corpus.json"
            source.write_text(json.dumps(self.payload))
            export(source, result)
            corpus = json.loads(result.read_text())
            self.assertEqual(len(corpus["cases"]), 100)
            self.assertTrue(all(not case["complete"] for case in corpus["cases"]))

    def test_revision_partial_and_truncation_rejected(self):
        for change in ("revision", "partial", "truncated"):
            with self.subTest(change=change), tempfile.TemporaryDirectory() as directory:
                payload = copy.deepcopy(self.payload)
                if change == "revision":
                    payload["rows"][0]["row"]["audio"][0]["src"] = "https://example.test/other/audio"
                elif change == "partial":
                    payload["partial"] = True
                else:
                    payload["rows"][0]["truncated_cells"] = ["words"]
                source, result = Path(directory) / "rows.json", Path(directory) / "corpus.json"
                source.write_text(json.dumps(payload))
                with self.assertRaises(ValueError):
                    export(source, result)
                self.assertFalse(result.exists())

    def test_duration_and_word_cutoff_boundaries(self):
        for duration, expected_count in ((0.2, 100), (0.2 - 0.5e-6, 100), (0.2 - 2e-6, 0)):
            with self.subTest(duration=duration), tempfile.TemporaryDirectory() as directory:
                payload = copy.deepcopy(self.payload)
                for entry in payload["rows"]:
                    entry["row"]["silence_spans"] = [{"start": 1.0, "end": 1.0 + duration}]
                    entry["row"]["words"] = [{"word": "at", "end": 0.7},
                                               {"word": "within", "end": 0.7 + 0.5e-6},
                                               {"word": "future", "end": 0.7 + 2e-6}]
                source, result = Path(directory) / "rows.json", Path(directory) / "corpus.json"
                source.write_text(json.dumps(payload))
                export(source, result)
                cases = json.loads(result.read_text())["cases"]
                self.assertEqual(len(cases), expected_count)
                self.assertTrue(all(case["text"] == "at within" for case in cases))
