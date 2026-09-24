// Copyright (c) 2023-2025 RapidaAI
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
//
// Thin C bridge for ONNX Runtime C API (Pipecat Smart Turn detector).
// Function names are prefixed with Pct to avoid duplicate symbols
// when linked alongside the Silero VAD and LiveKit bridges.

#ifndef PCT_ORT_BRIDGE_H
#define PCT_ORT_BRIDGE_H

#include <stdlib.h>
#include <string.h>
#include "onnxruntime_c_api.h"

const OrtApi* PctOrtGetApi();
const char* PctOrtApiGetErrorMessage(const OrtApi *api, OrtStatus *status);
void PctOrtApiReleaseStatus(const OrtApi *api, OrtStatus *status);

OrtStatus* PctOrtApiCreateEnv(const OrtApi *api, OrtLoggingLevel log_level, const char *log_id, OrtEnv **env);
void PctOrtApiReleaseEnv(const OrtApi *api, OrtEnv *env);

OrtStatus* PctOrtApiCreateSessionOptions(const OrtApi *api, OrtSessionOptions **opts);
void PctOrtApiReleaseSessionOptions(const OrtApi *api, OrtSessionOptions *opts);

OrtStatus* PctOrtApiSetIntraOpNumThreads(const OrtApi *api, OrtSessionOptions *opts, int intra_op_num_threads);
OrtStatus* PctOrtApiSetInterOpNumThreads(const OrtApi *api, OrtSessionOptions *opts, int inter_op_num_threads);
OrtStatus* PctOrtApiSetSessionGraphOptimizationLevel(const OrtApi *api, OrtSessionOptions *opts, GraphOptimizationLevel level);

OrtStatus* PctOrtApiCreateSession(const OrtApi *api, OrtEnv *env, const char *model_path, OrtSessionOptions *opts, OrtSession **session);
void PctOrtApiReleaseSession(const OrtApi *api, OrtSession *session);

OrtStatus* PctOrtApiCreateCpuMemoryInfo(const OrtApi *api, enum OrtAllocatorType alloc_type, enum OrtMemType mem_type, OrtMemoryInfo **minfo);
void PctOrtApiReleaseMemoryInfo(const OrtApi *api, OrtMemoryInfo *minfo);

OrtStatus* PctOrtApiCreateTensorWithDataAsOrtValue(const OrtApi *api, const OrtMemoryInfo *minfo, void *data, size_t data_len,
    const int64_t *shape, size_t shape_len, ONNXTensorElementDataType data_type, OrtValue **value);
void PctOrtApiReleaseValue(const OrtApi *api, OrtValue *value);

OrtStatus* PctOrtApiCreateRunOptions(const OrtApi *api, OrtRunOptions **opts);
OrtStatus* PctOrtApiRunOptionsSetTerminate(const OrtApi *api, OrtRunOptions *opts);
void PctOrtApiReleaseRunOptions(const OrtApi *api, OrtRunOptions *opts);

OrtStatus* PctOrtApiRun(const OrtApi *api, OrtSession *session, const OrtRunOptions *run_options,
    const char *const *input_names, const OrtValue *const *inputs, size_t inputs_len,
    const char *const *output_names, size_t output_names_len, OrtValue **outputs);

OrtStatus* PctOrtApiGetTensorMutableData(const OrtApi *api, OrtValue *value, void **data);

#endif // PCT_ORT_BRIDGE_H
