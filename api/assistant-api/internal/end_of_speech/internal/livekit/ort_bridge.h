// Copyright (c) 2023-2025 RapidaAI
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
//
// Thin C bridge for ONNX Runtime C API (LiveKit turn detector).
// Function names are prefixed with Lkt to avoid duplicate symbols
// when linked alongside the Silero VAD bridge.

#ifndef LKT_ORT_BRIDGE_H
#define LKT_ORT_BRIDGE_H

#include <stdlib.h>
#include <string.h>
#include "onnxruntime_c_api.h"

const OrtApi* LktOrtGetApi();

const char* LktOrtApiGetErrorMessage(const OrtApi *api, OrtStatus *status);

void LktOrtApiReleaseStatus(const OrtApi *api, OrtStatus *status);

OrtStatus* LktOrtApiCreateEnv(const OrtApi *api, OrtLoggingLevel log_level, const char *log_id, OrtEnv **env);
void LktOrtApiReleaseEnv(const OrtApi *api, OrtEnv *env);

OrtStatus* LktOrtApiCreateSessionOptions(const OrtApi *api, OrtSessionOptions **opts);
void LktOrtApiReleaseSessionOptions(const OrtApi *api, OrtSessionOptions *opts);

OrtStatus* LktOrtApiSetIntraOpNumThreads(const OrtApi *api, OrtSessionOptions *opts, int intra_op_num_threads);
OrtStatus* LktOrtApiSetInterOpNumThreads(const OrtApi *api, OrtSessionOptions *opts, int inter_op_num_threads);
OrtStatus* LktOrtApiSetSessionGraphOptimizationLevel(const OrtApi *api, OrtSessionOptions *opts, GraphOptimizationLevel level);

OrtStatus* LktOrtApiCreateSession(const OrtApi *api, OrtEnv *env, const char *model_path, OrtSessionOptions *opts, OrtSession **session);
void LktOrtApiReleaseSession(const OrtApi *api, OrtSession *session);

OrtStatus* LktOrtApiCreateCpuMemoryInfo(const OrtApi *api, enum OrtAllocatorType alloc_type, enum OrtMemType mem_type, OrtMemoryInfo **minfo);
void LktOrtApiReleaseMemoryInfo(const OrtApi *api, OrtMemoryInfo *minfo);

OrtStatus* LktOrtApiCreateTensorWithDataAsOrtValue(const OrtApi *api, const OrtMemoryInfo *minfo, void *data, size_t data_len,
    const int64_t *shape, size_t shape_len, ONNXTensorElementDataType data_type, OrtValue **value);
void LktOrtApiReleaseValue(const OrtApi *api, OrtValue *value);

static inline OrtStatus* LktOrtApiCreateRunOptions(const OrtApi *api, OrtRunOptions **opts) {
  return api->CreateRunOptions(opts);
}

static inline OrtStatus* LktOrtApiRunOptionsSetTerminate(const OrtApi *api, OrtRunOptions *opts) {
  return api->RunOptionsSetTerminate(opts);
}

static inline void LktOrtApiReleaseRunOptions(const OrtApi *api, OrtRunOptions *opts) {
  api->ReleaseRunOptions(opts);
}

OrtStatus* LktOrtApiRun(const OrtApi *api, OrtSession *session, const OrtRunOptions *run_options,
    const char *const *input_names, const OrtValue *const *inputs, size_t inputs_len,
    const char *const *output_names, size_t output_names_len, OrtValue **outputs);

OrtStatus* LktOrtApiGetTensorMutableData(const OrtApi *api, OrtValue *value, void **data);

#endif // LKT_ORT_BRIDGE_H
