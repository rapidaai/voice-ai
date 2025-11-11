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
from abc import abstractmethod
from typing import Dict, Literal

from aiobotocore.credentials import AioCredentials
from aiobotocore.session import AioSession, get_session

from app.configs.auth.aws_auth import AWSAuth
from app.connectors import Connector
from app.observabilities import within_span

_log = logging.getLogger("app.connectors.aws.aws_connector")


class AWSConnector(Connector):
    """
    AWSConnector interface all aws service connector should implement
    """

    _aws_config: AWSAuth

    def __init__(self, aws_config: AWSAuth):
        self._aws_config = aws_config

    @abstractmethod
    def name(self) -> Literal["sqs", "s3", "sts"]:
        raise NotImplementedError

    def get_session(self) -> AioSession:
        """
        Get aiobotocore session
        :return:
        """
        _session: AioSession = get_session()
        if self._aws_config.region:
            _session.set_config_variable("region", self._aws_config.region)
        if self._aws_config.access_key_id and self._aws_config.secret_access_key:
            _session.set_credentials(
                access_key=self._aws_config.access_key_id.get_secret_value(),
                secret_key=self._aws_config.secret_access_key.get_secret_value(),
            )
        return _session

    async def get_credentials(self, session_name: str) -> Dict:
        """
        Getting credentials from current session
        """
        _log.debug(f"Requested aws credentials for session name {session_name}")
        with within_span(
            f"AWS Resolving Credential {session_name}",
            span_type="external",
            span_subtype="aws",
            span_action="get_credentials",
        ):
            _session = self.get_session()
            credential: AioCredentials = await _session.get_credentials()
            _credential = await credential.get_frozen_credentials()
            # .get_frozen_credentials()

            return {
                "access_key": _credential.access_key,
                "secret_key": _credential.secret_key,
                "token": _credential.token,
                "region": _session.get_config_variable("region"),
            }
