from enum import Enum
from typing import List, Optional

from app.bridges.artifacts.protos.common_pb2 import Metric


class MetricName(str, Enum):
    TIME_TAKEN = "TIME_TAKEN"
    STATUS = "STATUS"
    INPUT_TOKEN = "INPUT_TOKEN"
    OUTPUT_TOKEN = "OUTPUT_TOKEN"
    TOTAL_TOKEN = "TOTAL_TOKEN"
    COST = "COST"
    INPUT_COST = "INPUT_COST"
    OUTPUT_COST = "OUTPUT_COST"
    LLM_REQUEST_ID = "LLM_REQUEST_ID"
    TOKEN_PER_SECOND = "TOKEN_PER_SECOND"
    TIME_TO_FIRST_TOKEN = "TIME_TO_FIRST_TOKEN"
    PROVIDER_TOTAL_TIME = "PROVIDER_TOTAL_TIME"
    PROVIDER_GENERATE_TIME = "PROVIDER_GENERATE_TIME"


# Function to get value from a list of metrics that matches the token
def get_metric_value_by_name(metrics: List[Metric], metric: str) -> Optional[str]:
    """
    This function searches through a list of metrics and returns the value
    of the first metric that matches the given metric name.

    :param metrics: List of Metric objects
    :param metric: MetricName to search for
    :return: The value of the matching metric or None if not found
    """
    for _mtr in metrics:
        if _mtr.name == metric:
            return _mtr.value
    return None


def get_metric_token_count(metrics: List[Metric]) -> int:
    token = get_metric_value_by_name(metrics=metrics, metric="TOTAL_TOKEN")
    if token:
        return int(token)
    return 0
