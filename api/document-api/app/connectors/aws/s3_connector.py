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
from typing import Literal, Optional

import botocore.exceptions
from aiobotocore.session import AioSession
from types_aiobotocore_s3.client import S3Client
from types_aiobotocore_s3.type_defs import EmptyResponseMetadataTypeDef

from app.configs.auth.aws_auth import AWSAuth
from app.connectors.aws import AWSConnector
from app.exceptions.connector_exception import (
    ConnectorClientFailureException,
    ConnectorException,
)
from app.observabilities import within_span

_log = logging.getLogger("app.connectors.aws.s3_connector")


class S3Connector(AWSConnector):
    """
    Simple storage service (Amazon Simple Storage Service)
    Provide an interface for storing data, or retrieve as objects in a bucket.
    """

    # for the purpose of anyone who wants to use this
    # connector directly and implement their own way to operate
    connection: Optional[AioSession] = None

    def __init__(self, aws_config: AWSAuth):
        super().__init__(aws_config)

    async def connect(self):
        """
        Connect to AWS S3
        """
        if self.connection:
            _log.debug(f"Already connected to {self.name()}")
            return

        self.connection = self.get_session()

    async def disconnect(self):
        """Not needed for s3"""
        self.connection = None

    async def is_connected(self, bucket_name: str) -> bool:
        """check if bucket exists"""
        return await self.bucket_exists(bucket_name) is True

    def name(self) -> Literal["s3"]:
        """
        return s3 literal value
        """
        return "s3"

    async def _operate(self, operation: str, **kwargs):
        """
        Operate on s3 client
        :type operation: str
        :param operation
        """
        try:
            # connect / create session for s3 before any operation
            await self.connect()
            async with self.connection.create_client(self.name()) as client:
                client: S3Client = client
                return await getattr(client, operation)(**kwargs)
        except botocore.exceptions.ConnectionError as boto_error:
            _log.error(f"Failed to connect for {self.name()} . {str(boto_error)}")
            raise ConnectorClientFailureException(
                connector_name=self.name(), message=str(boto_error)
            )
        except botocore.exceptions.ClientError as error:
            _log.error(f"Failed to do {operation} from {self.name()}. {str(error)}")
            error_message = str(error)
            if error.response["Error"]["Code"] == "S3.Client.exceptions.NoSuchBucket":
                error_message = "bucket does not exist."

            raise ConnectorClientFailureException(
                connector_name=self.name(), message=error_message
            )
        except Exception as err:
            _log.error(
                f"Failed to do the operation {operation} from {self.name()}. {str(err)}",
                exc_info=True,
            )
            raise ConnectorClientFailureException(
                connector_name=self.name(), message=str(err)
            )

    async def bucket_exists(self, bucket_name: str):
        """
        Check if bucket exists in current s3
        :param: bucket_name
        :return:
        """
        _log.debug(f"Trying to get info for {bucket_name} from AWS.")
        try:
            response: EmptyResponseMetadataTypeDef = await self._operate(
                "head_bucket", Bucket=bucket_name
            )
            return response["ResponseMetadata"]["HTTPStatusCode"] == 200
        except ConnectorException as err:
            _log.error(f"Failed to check if bucket exists. {str(err)}")
            return False

    async def operate(self, bucket_name: str, operation: str, **kwargs):
        """
        Operating on s3 object
        :type operation: str
        :param bucket_name:
        :param operation:
        :param kwargs:
        """
        with within_span(
            name=f"S3 {bucket_name.lower()}",
            span_type="external",
            span_subtype="aws_s3",
            span_action=operation,
        ):
            return await self._operate(operation, Bucket=bucket_name, **kwargs)
