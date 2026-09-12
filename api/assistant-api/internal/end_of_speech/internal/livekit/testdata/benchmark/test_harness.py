"""Small deterministic harness tests. No model inference or timing."""

import copy
import json
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

import run as experiment
from reference import load_source, read_corpus
from report import compare, latency, quality


class HarnessTests(unittest.TestCase):
    def setUp(self):
        self.reference = {
            "model_type": "en", "threshold": 0.5, "publisher_model_match": True,
            "publisher_tokenizer_match": True, "corpus": {"kind": "curated_diagnostic"},
            "cases": [{"name": "one", "language": "en", "complete": False, "probability": 0.5, "decision": True}],
            "timing": {"rounds": 2, "samples": []},
        }
        self.native = {"schema": 1, "model_type": "en", "threshold": 0.5,
                       "cases": [{"name": "one", "prompt_match": True, "token_match": True,
                                  "results": {stage: {"probability": 0.5, "decision": True}
                                              for stage in ("inference", "predict", "pipeline")}}]}

    def test_parity_is_not_quality(self):
        result = compare(self.reference, self.native)
        self.assertTrue(result["parity_pass"])
        self.assertEqual(result["quality_metric"], "diagnostic_label_agreement")
        self.assertEqual(result["quality_by_language"]["all"]["python"]["false_completion_rate"], 1)
        self.assertEqual(quality([(None, True)])["label_agreement"], None)

    def test_missing_duplicate_and_nonfinite_predictions(self):
        for edit in (lambda rows: rows.clear(), lambda rows: rows.append(rows[0]),
                     lambda rows: rows[0]["results"]["pipeline"].update(probability=float("nan")),
                     lambda rows: rows[0]["results"]["pipeline"].update(decision=False)):
            native = copy.deepcopy(self.native)
            edit(native["cases"])
            with self.assertRaises(ValueError):
                compare(self.reference, native)

    def test_reports_prompt_drift_without_hiding_scores(self):
        self.native["cases"][0]["prompt_match"] = False
        result = compare(self.reference, self.native)
        self.assertFalse(result["parity_pass"])
        self.assertIn("go_pipeline", result["quality_by_language"]["en"])

    def test_latency_requires_repeated_matching_rounds(self):
        with self.assertRaises(ValueError):
            latency(self.reference, "")
        lines = []
        for stage in ("format", "tokenize", "inference", "predict", "pipeline"):
            for index in range(2):
                self.reference["timing"]["samples"].append({"case": "one", "stage": stage, "round": index, "ns_per_op": 20})
                lines.append(f"BenchmarkLiveKitNative/en/one/{stage}-1 10 10 ns/op 0 B/op 0 allocs/op")
        result = latency(self.reference, "\n".join(lines))
        self.assertEqual(len(result["samples"]), 5)
        self.assertEqual(result["samples"][0]["go_over_python_median"], 0.5)
        self.assertEqual(latency(self.reference, "\n".join(lines).replace("-1 ", " ")), result)
        for index in range(2):
            self.reference["timing"]["samples"].append({"case": "one", "stage": "async_eot", "round": index, "ns_per_op": 30})
            lines.append("BenchmarkLiveKitNative/en/one/production_context 10 20 ns/op 0 B/op 0 allocs/op")
        self.assertEqual(latency(self.reference, "\n".join(lines))["samples"][-1]["python_stage"], "async_eot")

    def test_corpus_validation_and_selection(self):
        corpus, cases = read_corpus(Path(__file__).with_name("corpus.json"), ["repetition_incomplete"])
        self.assertEqual(len(cases), 1)
        self.assertFalse(cases[0]["complete"])
        with self.assertRaises(ValueError):
            read_corpus(Path(__file__).with_name("corpus.json"), ["missing"])
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "corpus.json"
            corpus["kind"] = "independent_dataset"
            path.write_text(json.dumps(corpus))
            with self.assertRaisesRegex(ValueError, "requires revision"):
                read_corpus(path)

    def test_experiment_order_and_three_case_latency_without_processes(self):
        names = ("complete_en", "repetition_incomplete", "long_left_truncation")
        reference = copy.deepcopy(self.reference)
        reference["cases"] = [{**reference["cases"][0], "name": name} for name in names]
        for field in ("model_sha256", "tokenizer_sha256", "ort_library_sha256", "python_ort_binary_sha256",
                      "source_sha256", "harness_sha256", "metadata_sha256", "corpus_sha256"):
            reference[field] = "fixed"
        calls = []

        def execute(argv, env, stdout, stderr):
            if "--output" in argv:
                directory = Path(argv[argv.index("--output") + 1])
                directory.mkdir(exist_ok=True)
                current = copy.deepcopy(reference)
                current["timing"] = {"rounds": 0, "samples": []}
                if "--rounds" in argv:
                    self.assertEqual(argv.count("--case"), 3)
                    current["timing"] = {"rounds": 1, "samples": [
                        {"case": name, "stage": stage, "round": 0, "ns_per_op": 20}
                        for name in names for stage in ("format", "tokenize", "inference", "predict", "pipeline", "async_eot")]}
                (directory / "reference.json").write_text(json.dumps(current))
                calls.append("python")
            elif "-test.run=^TestLiveKitNativeParity$" in argv:
                import hashlib

                native = copy.deepcopy(self.native)
                native["cases"] = [{**copy.deepcopy(native["cases"][0]), "name": name} for name in names]
                native["reference_sha256"] = hashlib.sha256(Path(env["LIVEKIT_BENCHMARK_REFERENCE"]).read_bytes()).hexdigest()
                Path(env["LIVEKIT_BENCHMARK_OUTPUT"]).write_text(json.dumps(native))
                calls.append("go")
            else:
                self.assertIn("-test.count=1", argv)
                for name in names:
                    for stage in ("format", "tokenize", "inference", "predict", "pipeline", "production_context"):
                        stdout.write(f"BenchmarkLiveKitNative/en/{name}/{stage} 10 10 ns/op 0 B/op 0 allocs/op\n")
                calls.append("go")
            return SimpleNamespace(returncode=0)

        with tempfile.TemporaryDirectory() as directory:
            args = SimpleNamespace(approve_experiments=True, rounds=7, iterations=10,
                                   output=Path(directory) / "output", model_type="en", threshold=0.5,
                                   checkout=Path(directory), model=Path(directory), tokenizer=Path(directory),
                                   ort_library=Path(directory), go_binary=Path(directory),
                                   corpus=Path(directory), metadata=Path(directory))
            with patch.object(experiment.subprocess, "run", side_effect=execute), patch.object(experiment, "digest", return_value="fixed"):
                experiment.run(args)
            expected = ["python", "go"]
            for index in range(7):
                expected.extend(("python", "go") if index % 2 == 0 else ("go", "python"))
            self.assertEqual(calls, expected)
            result = json.loads((args.output / "latency.json").read_text())
            self.assertEqual(len(result["round_order"]), 7)
            self.assertEqual(len(result["samples"]), 18)
            args.approve_experiments = False
            with self.assertRaisesRegex(ValueError, "tests must finish first"):
                experiment.run(args)

    @unittest.skipUnless(os.environ.get("LIVEKIT_REFERENCE_CHECKOUT") and os.environ.get("LIVEKIT_REFERENCE_TOKENIZER"), "local source/tokenizer paths not set")
    def test_actual_source_and_native_tokenizer_without_inference(self):
        from transformers import PreTrainedTokenizerFast

        ns, hashes = load_source(Path(os.environ["LIVEKIT_REFERENCE_CHECKOUT"]))
        self.assertEqual(set(hashes), {"base.py", "english.py", "multilingual.py", "models.py", "uv.lock"})
        metadata = json.loads(Path(__file__).with_name("metadata.json").read_text())
        _, cases = read_corpus(Path(__file__).with_name("corpus.json"))
        for model_type, name in (("en", "_EUORunnerEn"), ("multilingual", "_EUORunnerMultilingual")):
            tokenizer = PreTrainedTokenizerFast(tokenizer_file=os.environ["LIVEKIT_REFERENCE_TOKENIZER"],
                                               chat_template=metadata[model_type]["chat_template"], truncation_side="left")
            runner = ns[name]()
            runner._tokenizer = tokenizer
            for case in cases:
                with self.subTest(model=model_type, case=case["name"]):
                    messages = [m for m in case.get("history", []) if m["role"] in {"user", "assistant"} and m["content"]]
                    messages.append({"role": "user", "content": case["text"]})
                    prompt = runner._format_chat_ctx(copy.deepcopy(messages[-ns["MAX_HISTORY_TURNS"]:]))
                    ids = tokenizer(prompt, add_special_tokens=False, truncation=True, max_length=128)["input_ids"]
                    self.assertGreater(len(ids), 0)
                    self.assertLessEqual(len(ids), 128)
                    if case["name"] == "complete_en":
                        role = "<|user|>" if model_type == "en" else "user\n"
                        self.assertTrue(prompt.startswith("<|im_start|>" + role))
                    if case["name"] == "long_left_truncation":
                        self.assertEqual(len(ids), 128)


if __name__ == "__main__":
    unittest.main()
