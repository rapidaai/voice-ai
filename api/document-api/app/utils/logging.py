"""
Copyright (c) 2024 Prashant Srivastav <prashant@rapida.ai>
All rights reserved.

This code is licensed under the MIT License. You may obtain a copy of the License at
https://opensource.org/licenses/MIT.

Unless required by applicable law or agreed to in writing, software distributed under the
License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
either express or implied. See the License for the specific language governing permissions
and limitations under the License.

"""
import logging
from logging.config import dictConfig
from typing import Dict


def get_logger(name: str, log_config: Dict) -> logging.Logger:
    """
    get logger
    :return: logger
    """
    # set logger config
    dictConfig(log_config)
    return logging.getLogger(name)
