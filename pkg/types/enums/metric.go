package type_enums

type MetricName string

var (
	TIME_TAKEN   MetricName = "TIME_TAKEN"
	STATUS       MetricName = "STATUS"
	INPUT_TOKEN  MetricName = "INPUT_TOKEN"
	OUTPUT_TOKEN MetricName = "OUTPUT_TOKEN"
	TOTAL_TOKEN  MetricName = "TOTAL_TOKEN"
	COST         MetricName = "COST"
	INPUT_COST   MetricName = "INPUT_COST"
	OUTPUT_COST  MetricName = "OUTPUT_COST"
	//
	LLM_REQUEST_ID MetricName = "LLM_REQUEST_ID"
	//
	TOKEN_PRE_SECOND       MetricName = "TOKEN_PRE_SECOND"
	TIME_TO_FIRST_TOKEN    MetricName = "TIME_TO_FIRST_TOKEN"
	PROVIDER_TOTAL_TIME    MetricName = "PROVIDER_TOTAL_TIME"
	PROVIDER_GENERATE_TIME MetricName = "PROVIDER_GENERATE_TIME"
)

func (m *MetricName) String() string {
	return string(*m)
}
