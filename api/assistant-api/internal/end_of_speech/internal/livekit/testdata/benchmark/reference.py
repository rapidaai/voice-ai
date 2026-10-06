"""Execute local LiveKit source with explicit local assets; timing is opt-in."""

import abc
import argparse
import ast
import asyncio
import copy
import ctypes
import hashlib
import importlib.metadata
import json
import logging
import math
import os
from pathlib import Path
import platform
import re
import subprocess
import sys
import time
from types import SimpleNamespace
import unicodedata


def digest(path):
    with Path(path).open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def load_source(checkout):
    source = checkout / "livekit-plugins/livekit-plugins-turn-detector/livekit/plugins/turn_detector"
    ns = dict(json=json, re=re, time=time, unicodedata=unicodedata, asyncio=asyncio,
              ABC=abc.ABC, abstractmethod=abc.abstractmethod, _InferenceRunner=object,
              logger=logging.getLogger("reference"))
    exec(compile((source / "models.py").read_bytes(), str(source / "models.py"), "exec"), ns)
    selected = {
        "base.py": {"_EUORunnerBase", "EOUModelBase"},
        "english.py": {"_EUORunnerEn"},
        "multilingual.py": {"_EUORunnerMultilingual"},
    }
    hashes = {"models.py": digest(source / "models.py")}
    for filename, classes in selected.items():
        path = source / filename
        tree = ast.parse(path.read_bytes(), filename=str(path))
        nodes = []
        found = set()
        for node in tree.body:
            if isinstance(node, ast.ImportFrom) and node.module == "__future__":
                nodes.append(node)
            if isinstance(node, ast.ClassDef) and node.name in classes:
                nodes.append(node)
                found.add(node.name)
            if filename == "base.py" and isinstance(node, ast.Assign):
                for target in node.targets:
                    if isinstance(target, ast.Name) and target.id in {"MAX_HISTORY_TOKENS", "MAX_HISTORY_TURNS"}:
                        nodes.append(node)
        if found != classes:
            raise ValueError(f"reference classes changed in {path}")
        # Compile original class bodies without importing the unrelated agent service.
        exec(compile(ast.Module(body=nodes, type_ignores=[]), str(path), "exec"), ns)
        hashes[filename] = digest(path)
    hashes["uv.lock"] = digest(checkout / "uv.lock")
    return ns, hashes


def read_corpus(path, names=()):
    corpus = json.loads(path.read_text())
    if corpus.get("schema") != 1 or corpus.get("kind") not in {"curated_diagnostic", "independent_dataset"}:
        raise ValueError("corpus requires schema=1 and an explicit kind")
    if corpus["kind"] == "independent_dataset":
        for field in ("source", "revision", "split", "license", "sampling", "label_policy", "holdout_status"):
            if not corpus.get(field):
                raise ValueError(f"independent dataset requires {field}")
    seen = set()
    for case in corpus["cases"]:
        name = case["name"]
        if not re.fullmatch(r"[a-zA-Z0-9_-]+", name) or name in seen:
            raise ValueError(f"invalid or duplicate case name: {name}")
        seen.add(name)
        if not isinstance(case.get("text"), str) or not case.get("language"):
            raise ValueError(f"case {name} requires text and language")
        if "complete" not in case or (case["complete"] is not None and type(case["complete"]) is not bool):
            raise ValueError(f"case {name} requires a boolean or null label")
        for message in case.get("history", []):
            if not isinstance(message.get("role"), str) or not isinstance(message.get("content"), str):
                raise ValueError(f"case {name} has an invalid history message")
    if set(names) - seen:
        raise ValueError(f"unknown selected cases: {sorted(set(names) - seen)}")
    cases = [case for case in corpus["cases"] if not names or case["name"] in names]
    if not cases:
        raise ValueError("corpus selection is empty")
    return corpus, cases


def generate(args):
    import numpy as np
    import onnxruntime as ort
    from transformers import PreTrainedTokenizerFast

    for package, expected in {"transformers": "4.57.6", "tokenizers": "0.22.2", "jinja2": "3.1.6", "numpy": "1.26.4"}.items():
        if importlib.metadata.version(package) != expected:
            raise ValueError(f"requires {package}=={expected}")
    if args.rounds < 0 or args.iterations < 1 or args.warmup < 0 or not 0 <= args.threshold <= 1:
        raise ValueError("invalid timing count or threshold")
    args.output.mkdir(parents=True, exist_ok=True)
    ns, source_hashes = load_source(args.checkout)
    metadata = json.loads(args.metadata.read_text())[args.model_type]
    if metadata["requested_revision"] != ns["MODEL_REVISIONS"][args.model_type]:
        raise ValueError("metadata revision differs from the local reference models.py")
    corpus, cases = read_corpus(args.corpus, args.case)
    model_digest, tokenizer_digest = digest(args.model), digest(args.tokenizer)
    raw = args.tokenizer.read_bytes()
    blob = hashlib.sha1(f"blob {len(raw)}\0".encode() + raw).hexdigest()
    library = ctypes.CDLL(str(args.ort_library))
    library.OrtGetApiBase.restype = ctypes.POINTER(ctypes.c_void_p * 2)
    version = ctypes.CFUNCTYPE(ctypes.c_char_p)(library.OrtGetApiBase().contents[1])().decode()
    if version != ort.__version__:
        raise ValueError(f"Go library ORT {version} differs from Python ORT {ort.__version__}")
    options = ort.SessionOptions()
    options.intra_op_num_threads = 1
    options.inter_op_num_threads = 1
    options.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
    options.execution_mode = ort.ExecutionMode.ORT_SEQUENTIAL
    session = ort.InferenceSession(str(args.model), sess_options=options, providers=["CPUExecutionProvider"])
    if session.get_providers() != ["CPUExecutionProvider"]:
        raise ValueError("expected CPU-only inference")
    tokenizer = PreTrainedTokenizerFast(tokenizer_file=str(args.tokenizer),
                                       chat_template=metadata["chat_template"], truncation_side="left")
    runner = ns["_EUORunnerEn" if args.model_type == "en" else "_EUORunnerMultilingual"]()
    runner._session, runner._tokenizer = session, tokenizer
    messages = []

    class Executor:
        async def do_inference(self, method, payload):
            nonlocal messages
            messages = json.loads(payload)["chat_ctx"]
            return runner.run(payload)

    # Invoke the source's history filtering/truncation method with a local executor.
    class LocalModel(ns["EOUModelBase"]):
        def _inference_method(self):
            return runner.INFERENCE_METHOD

    model = LocalModel.__new__(LocalModel)
    model._executor = Executor()
    result = {
        "schema": 1, "model_type": args.model_type, "threshold": args.threshold,
        "max_history_turns": ns["MAX_HISTORY_TURNS"], "max_history_tokens": ns["MAX_HISTORY_TOKENS"],
        "model_path": str(args.model), "model_sha256": model_digest,
        "tokenizer_path": str(args.tokenizer), "tokenizer_sha256": tokenizer_digest,
        "ort_library_path": str(args.ort_library), "ort_library_sha256": digest(args.ort_library),
        "ort_version": version, "metadata": metadata, "metadata_sha256": digest(args.metadata),
        "publisher_model_match": metadata["model_sha256"] == model_digest,
        "publisher_tokenizer_match": metadata["tokenizer_git_blob"] == blob,
        "reference_checkout": str(args.checkout), "source_sha256": source_hashes,
        "harness_sha256": {path.name: digest(path) for path in Path(__file__).parent.iterdir() if path.suffix in {".py", ".json"}},
        "model_io": {"inputs": [{"name": item.name, "shape": item.shape, "type": item.type} for item in session.get_inputs()],
                     "outputs": [{"name": item.name, "shape": item.shape, "type": item.type} for item in session.get_outputs()]},
        "reference_commit": subprocess.check_output(["git", "-C", str(args.checkout), "rev-parse", "HEAD"], text=True).strip(),
        "reference_dirty": subprocess.check_output(["git", "-C", str(args.checkout), "status", "--porcelain"], text=True).strip(),
        "corpus_sha256": digest(args.corpus), "corpus": {k: v for k, v in corpus.items() if k != "cases"},
        "environment": {"python": sys.version, "executable": sys.executable, "platform": platform.platform(),
                        "machine": platform.machine(), "cpu_count": os.cpu_count(),
                        "packages": {name: importlib.metadata.version(name) for name in ("numpy", "onnxruntime", "tokenizers", "transformers", "jinja2", "regex")},
                        "unicode_version": unicodedata.unidata_version,
                        "thread_env": {key: os.environ.get(key) for key in ("OMP_NUM_THREADS", "OPENBLAS_NUM_THREADS", "VECLIB_MAXIMUM_THREADS", "TOKENIZERS_PARALLELISM")}},
        "runtime_policy": "CPU; intra=1; inter=1; sequential; all graph optimizations; matched ORT release, separately hashed Python and Go builds",
        "reference_adaptation": "Original AST class bodies; local executor; explicit local tokenizer and publisher template; initialization replaced for controlled assets and single-thread session",
        "timing": {"rounds": args.rounds, "iterations": args.iterations, "warmup": args.warmup, "unit": "ns/op", "samples": []},
        "cases": [],
    }
    import onnxruntime.capi.onnxruntime_pybind11_state as binding
    result["python_ort_binary_sha256"] = digest(binding.__file__)
    for case in cases:
        history = copy.deepcopy(case.get("history", []))
        if case["text"]:
            history.append({"role": "user", "content": case["text"]})
        chat = SimpleNamespace(messages=lambda: [SimpleNamespace(role=m["role"], text_content=m["content"]) for m in history])
        probability = asyncio.run(model.predict_end_of_turn(chat))
        prompt = runner._format_chat_ctx(copy.deepcopy(messages))
        ids = tokenizer(prompt, add_special_tokens=False, return_tensors="np", max_length=ns["MAX_HISTORY_TOKENS"], truncation=True)["input_ids"].astype("int64")
        if not math.isfinite(probability) or not 0 <= probability <= 1 or ids.size == 0:
            raise ValueError(f"invalid reference result for {case['name']}")
        row = {**case, "prompt": prompt, "input_ids": ids[0].tolist(), "probability": probability,
               "decision": probability >= args.threshold, "selected_messages": copy.deepcopy(messages)}
        result["cases"].append(row)

        def predict(text):
            inputs = tokenizer(text, add_special_tokens=False, return_tensors="np", max_length=ns["MAX_HISTORY_TOKENS"], truncation=True)
            return session.run(None, {"input_ids": inputs["input_ids"].astype("int64")})[0].flatten()[-1]

        async_runner = asyncio.Runner()
        stages = {
            "format": lambda: runner._format_chat_ctx(copy.deepcopy(messages)),
            "tokenize": lambda: tokenizer(prompt, add_special_tokens=False, return_tensors="np", max_length=ns["MAX_HISTORY_TOKENS"], truncation=True),
            "inference": lambda: session.run(None, {"input_ids": ids})[0].flatten()[-1],
            "predict": lambda: predict(prompt),
            "pipeline": lambda: predict(runner._format_chat_ctx(copy.deepcopy(messages))),
            "runner_json": lambda: runner.run(json.dumps({"chat_ctx": messages}).encode()),
            "async_eot": lambda: async_runner.run(model.predict_end_of_turn(chat, timeout=3)),
        }
        if args.rounds:
            for operation in stages.values():
                for _ in range(args.warmup):
                    operation()
            for round_index in range(args.rounds):
                for stage, operation in stages.items():
                    started = time.perf_counter_ns()
                    for _ in range(args.iterations):
                        operation()
                    elapsed = time.perf_counter_ns() - started
                    result["timing"]["samples"].append({"case": case["name"], "stage": stage, "round": round_index, "ns_per_op": elapsed / args.iterations})
        async_runner.close()
    (args.output / "reference.json").write_text(json.dumps(result, indent=2, ensure_ascii=True, allow_nan=False) + "\n")
    print(json.dumps({"model_type": args.model_type, "cases": len(cases), "publisher_model_match": result["publisher_model_match"], "publisher_tokenizer_match": result["publisher_tokenizer_match"], "timing_rounds": args.rounds}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("checkout", "model", "tokenizer", "ort-library", "output"):
        parser.add_argument(f"--{name}", required=True, type=lambda value: Path(value).resolve())
    parser.add_argument("--model-type", required=True, choices=("en", "multilingual"))
    parser.add_argument("--threshold", required=True, type=float)
    parser.add_argument("--metadata", type=Path, default=Path(__file__).with_name("metadata.json"))
    parser.add_argument("--corpus", type=Path, default=Path(__file__).with_name("corpus.json"))
    parser.add_argument("--case", action="append", default=[])
    parser.add_argument("--rounds", type=int, default=0)
    parser.add_argument("--iterations", type=int, default=10)
    parser.add_argument("--warmup", type=int, default=3)
    generate(parser.parse_args())
