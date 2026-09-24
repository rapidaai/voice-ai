// Copyright (c) 2023-2025 RapidaAI
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

#include "ort_bridge.h"

const OrtApi* PctOrtGetApi() { return OrtGetApiBase()->GetApi(ORT_API_VERSION); }
const char* PctOrtApiGetErrorMessage(const OrtApi *api, OrtStatus *status) { return api->GetErrorMessage(status); }
void PctOrtApiReleaseStatus(const OrtApi *api, OrtStatus *status) { if (status) api->ReleaseStatus(status); }

OrtStatus* PctOrtApiCreateEnv(const OrtApi *api, OrtLoggingLevel log_level, const char *log_id, OrtEnv **env) { return api->CreateEnv(log_level, log_id, env); }
void PctOrtApiReleaseEnv(const OrtApi *api, OrtEnv *env) { api->ReleaseEnv(env); }

OrtStatus* PctOrtApiCreateSessionOptions(const OrtApi *api, OrtSessionOptions **opts) { return api->CreateSessionOptions(opts); }
void PctOrtApiReleaseSessionOptions(const OrtApi *api, OrtSessionOptions *opts) { api->ReleaseSessionOptions(opts); }

OrtStatus* PctOrtApiSetIntraOpNumThreads(const OrtApi *api, OrtSessionOptions *opts, int n) { return api->SetIntraOpNumThreads(opts, n); }
OrtStatus* PctOrtApiSetInterOpNumThreads(const OrtApi *api, OrtSessionOptions *opts, int n) { return api->SetInterOpNumThreads(opts, n); }
OrtStatus* PctOrtApiSetSessionGraphOptimizationLevel(const OrtApi *api, OrtSessionOptions *opts, GraphOptimizationLevel l) { return api->SetSessionGraphOptimizationLevel(opts, l); }

OrtStatus* PctOrtApiCreateSession(const OrtApi *api, OrtEnv *env, const char *p, OrtSessionOptions *opts, OrtSession **s) { return api->CreateSession(env, p, opts, s); }
void PctOrtApiReleaseSession(const OrtApi *api, OrtSession *s) { api->ReleaseSession(s); }

OrtStatus* PctOrtApiCreateCpuMemoryInfo(const OrtApi *api, enum OrtAllocatorType a, enum OrtMemType m, OrtMemoryInfo **mi) { return api->CreateCpuMemoryInfo(a, m, mi); }
void PctOrtApiReleaseMemoryInfo(const OrtApi *api, OrtMemoryInfo *mi) { api->ReleaseMemoryInfo(mi); }

OrtStatus* PctOrtApiCreateTensorWithDataAsOrtValue(const OrtApi *api, const OrtMemoryInfo *mi, void *d, size_t dl,
    const int64_t *s, size_t sl, ONNXTensorElementDataType dt, OrtValue **v) { return api->CreateTensorWithDataAsOrtValue(mi, d, dl, s, sl, dt, v); }
void PctOrtApiReleaseValue(const OrtApi *api, OrtValue *v) { api->ReleaseValue(v); }

OrtStatus* PctOrtApiCreateRunOptions(const OrtApi *api, OrtRunOptions **opts) { return api->CreateRunOptions(opts); }
OrtStatus* PctOrtApiRunOptionsSetTerminate(const OrtApi *api, OrtRunOptions *opts) { return api->RunOptionsSetTerminate(opts); }
void PctOrtApiReleaseRunOptions(const OrtApi *api, OrtRunOptions *opts) { api->ReleaseRunOptions(opts); }

OrtStatus* PctOrtApiRun(const OrtApi *api, OrtSession *s, const OrtRunOptions *ro,
    const char *const *in, const OrtValue *const *iv, size_t il,
    const char *const *on, size_t ol, OrtValue **ov) { return api->Run(s, ro, in, iv, il, on, ol, ov); }

OrtStatus* PctOrtApiGetTensorMutableData(const OrtApi *api, OrtValue *v, void **d) { return api->GetTensorMutableData(v, d); }
