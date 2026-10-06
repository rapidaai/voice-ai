import hashlib
import json
from pathlib import Path
import tempfile
import unittest

from quality_report import classification_metrics, evaluate_quality


class QualityReportTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.paths = [Path(self.directory.name) / f"{name}.json" for name in ("corpus", "reference", "baseline", "fft")]
        self.cases = [{"name": str(index), "source_id": str(index), "source_sha256": str(index), "row_index": index, "audio_sha256": str(index),
                       "condition": "clean", "expected_complete": label, "language": "eng",
                       "synthetic": False, "source_dataset": "test", "midfiller": None, "endfiller": None}
                      for index, label in enumerate((True, True, False, False))]
        self.predictions = [{"name": str(index), "probability": probability, "prediction": int(probability > 0.5), "audio_sha256": str(index),
                             "max_feature_error": 0.0}
                            for index, probability in enumerate((0.8, 0.2, 0.8, 0.2))]
        self.corpus = {"cases": self.cases, "conditions": {"clean": "test"}, "source_count": 4, "selected_rows": [0, 1, 2, 3],
                       "dataset": "test", "revision": "test", "seed": 1}
        self.paths[0].write_text(json.dumps(self.corpus))
        self.reference = {"cases": self.predictions, "corpus_sha256": hashlib.sha256(self.paths[0].read_bytes()).hexdigest(),
                          "reference_commit": "test", "source_sha256": {}, "source_dirty": False,
                          "model_sha256": "test", "onnxruntime": "test", "numpy": "test", "providers": ["CPU"]}
        self.paths[1].write_text(json.dumps(self.reference))
        self.go_report = {"cases": self.predictions, "reference_sha256": hashlib.sha256(self.paths[1].read_bytes()).hexdigest()}
        for path in self.paths[2:]:
            path.write_text(json.dumps(self.go_report))

    def test_confusion_and_denominators(self):
        metrics = classification_metrics(self.cases, {case["name"]: case for case in self.predictions})
        self.assertEqual(metrics["true_complete"], 1)
        self.assertEqual(metrics["true_incomplete"], 1)
        self.assertEqual(metrics["false_complete"], 1)
        self.assertEqual(metrics["false_incomplete"], 1)
        self.assertEqual(metrics["accuracy"]["value"], 0.5)
        self.assertEqual(metrics["premature_completion_rate"]["denominator"], 2)
        self.assertEqual(metrics["missed_completion_rate"]["denominator"], 2)
        self.assertEqual(metrics["balanced_accuracy"], 0.5)
        self.assertAlmostEqual(metrics["brier_score"], 0.34)
        self.assertLess(metrics["accuracy"]["wilson_95"][0], 0.5)
        self.assertGreater(metrics["accuracy"]["wilson_95"][1], 0.5)

    def test_single_class_has_no_incomplete_rate(self):
        metrics = classification_metrics(self.cases[:2], {case["name"]: case for case in self.predictions})
        self.assertIsNone(metrics["premature_completion_rate"]["value"])
        self.assertIsNone(metrics["balanced_accuracy"])
        self.assertIsNone(classification_metrics([], {})["accuracy"]["value"])

    def test_matching_reports_do_not_imply_perfect_accuracy(self):
        report = evaluate_quality(*self.paths)
        self.assertEqual(report["conditions"]["clean"]["fft"]["accuracy"]["value"], 0.5)
        self.assertEqual(report["pairwise"]["rapida_vs_fft"]["decision_disagreements"], [])
        self.assertEqual(len(report["errors"]), 2)
        self.assertTrue(report["pipecat_parity_passed"]["fft"])

    def test_paired_regression_and_exact_threshold(self):
        self.go_report["cases"][0]["probability"] = 0.5
        self.go_report["cases"][0]["prediction"] = 0
        self.paths[3].write_text(json.dumps(self.go_report))
        report = evaluate_quality(*self.paths)
        self.assertEqual(report["pairwise"]["rapida_vs_fft"]["correct_to_wrong"], ["0"])
        self.assertEqual(report["pairwise"]["rapida_vs_fft"]["wrong_to_correct"], [])
        self.assertFalse(report["pipecat_parity_passed"]["fft"])

    def test_probability_drift_fails_parity_without_decision_flip(self):
        self.go_report["cases"][0]["probability"] = 0.7
        self.paths[3].write_text(json.dumps(self.go_report))
        report = evaluate_quality(*self.paths)
        self.assertEqual(report["pairwise"]["pipecat_vs_fft"]["decision_disagreements"], [])
        self.assertEqual(report["probability_tolerance_failures"]["fft"], ["0"])
        self.assertFalse(report["pipecat_parity_passed"]["fft"])

    def test_reject_missing_or_duplicate_predictions(self):
        for cases in (self.predictions[:-1], self.predictions + self.predictions[:1]):
            with self.subTest(count=len(cases)):
                self.go_report["cases"] = cases
                self.paths[3].write_text(json.dumps(self.go_report))
                with self.assertRaises(ValueError):
                    evaluate_quality(*self.paths)

    def test_reject_invalid_probability_or_decision(self):
        for probability, prediction in ((float("nan"), 0), (1.1, 1), (0.2, 1)):
            with self.subTest(probability=probability):
                self.go_report["cases"][0].update(probability=probability, prediction=prediction)
                self.paths[3].write_text(json.dumps(self.go_report))
                with self.assertRaises(ValueError):
                    evaluate_quality(*self.paths)

    def test_reject_reference_checksum_mismatch(self):
        self.go_report["reference_sha256"] = "wrong"
        self.paths[3].write_text(json.dumps(self.go_report))
        with self.assertRaisesRegex(ValueError, "reference checksum"):
            evaluate_quality(*self.paths)

    def test_reject_changed_corpus(self):
        self.paths[0].write_text(json.dumps({**self.corpus, "seed": 2}))
        with self.assertRaisesRegex(ValueError, "corpus checksum"):
            evaluate_quality(*self.paths)

    def test_reject_missing_label(self):
        self.cases[0]["expected_complete"] = None
        with self.assertRaisesRegex(ValueError, "ground truth"):
            classification_metrics(self.cases, {case["name"]: case for case in self.predictions})

    def test_reject_unselected_rows(self):
        self.corpus["selected_rows"] = [40, 41, 42, 43]
        self.paths[0].write_text(json.dumps(self.corpus))
        self.reference["corpus_sha256"] = hashlib.sha256(self.paths[0].read_bytes()).hexdigest()
        self.paths[1].write_text(json.dumps(self.reference))
        self.go_report["reference_sha256"] = hashlib.sha256(self.paths[1].read_bytes()).hexdigest()
        for path in self.paths[2:]:
            path.write_text(json.dumps(self.go_report))
        with self.assertRaisesRegex(ValueError, "declared selection"):
            evaluate_quality(*self.paths)

    def test_reject_swapped_source_identity(self):
        self.corpus["conditions"]["noise"] = "test"
        for case in list(self.corpus["cases"]):
            self.corpus["cases"].append({**case, "name": case["name"] + "_noise", "condition": "noise",
                                         "source_id": str(int(case["source_id"]) ^ 1)})
        for prediction in list(self.predictions):
            self.predictions.append({**prediction, "name": prediction["name"] + "_noise"})
        self.paths[0].write_text(json.dumps(self.corpus))
        self.reference["corpus_sha256"] = hashlib.sha256(self.paths[0].read_bytes()).hexdigest()
        self.paths[1].write_text(json.dumps(self.reference))
        self.go_report["reference_sha256"] = hashlib.sha256(self.paths[1].read_bytes()).hexdigest()
        for path in self.paths[2:]:
            path.write_text(json.dumps(self.go_report))
        with self.assertRaisesRegex(ValueError, "source identity"):
            evaluate_quality(*self.paths)


if __name__ == "__main__":
    unittest.main()
