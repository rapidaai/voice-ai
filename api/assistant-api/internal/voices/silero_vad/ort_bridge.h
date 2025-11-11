#include "onnxruntime_c_api.h"

const OrtApi* OrtGetApi();

const char* OrtApiGetErrorMessage(OrtApi *api, OrtStatus *status);

void OrtApiReleaseStatus(OrtApi *api, OrtStatus *status);

OrtStatus* OrtApiCreateEnv(OrtApi *api, OrtLoggingLevel log_level, const char *log_id, OrtEnv **env);
void OrtApiReleaseEnv(OrtApi *api, OrtEnv *env);

OrtStatus* OrtApiCreateSessionOptions(OrtApi* api, OrtSessionOptions** opts);
void OrtApiReleaseSessionOptions(OrtApi* api, OrtSessionOptions* opts);

OrtStatus* OrtApiSetIntraOpNumThreads(OrtApi* api, OrtSessionOptions* opts, int intra_op_num_threads);
OrtStatus* OrtApiSetInterOpNumThreads(OrtApi* api, OrtSessionOptions* opts, int inter_op_num_threads);
OrtStatus* OrtApiSetSessionGraphOptimizationLevel(OrtApi* api, OrtSessionOptions* opts, GraphOptimizationLevel graph_optimization_level);

OrtStatus* OrtApiCreateSession(OrtApi* api, OrtEnv* env, const char* model_path, OrtSessionOptions* opts, OrtSession** session);
void OrtApiReleaseSession(OrtApi* api, OrtSession* session);

OrtStatus* OrtApiCreateCpuMemoryInfo(OrtApi* api, enum OrtAllocatorType alloc_type, enum OrtMemType mem_type, OrtMemoryInfo** minfo);
void OrtApiReleaseMemoryInfo(OrtApi* api, OrtMemoryInfo *minfo);

OrtStatus* OrtApiCreateTensorWithDataAsOrtValue(OrtApi* api, const OrtMemoryInfo* minfo, void* data, size_t data_len,
    const int64_t* shape, size_t shape_len, ONNXTensorElementDataType data_type, OrtValue** value);
void OrtApiReleaseValue(OrtApi* api, OrtValue *value);

OrtStatus* OrtApiRun(OrtApi* api, OrtSession* session, const OrtRunOptions* run_options,
    const char* const* input_names, const OrtValue* const* inputs, size_t inputs_len,
    const char* const* output_names, size_t output_names_len, OrtValue** outputs);

OrtStatus* OrtApiGetTensorMutableData(OrtApi* api, OrtValue* value, void** data);
