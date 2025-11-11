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
from app.exceptions.rapida_exception import RapidaException


class ConnectorException(RapidaException):
    """
    Connector Exception -> all connector class or connector operation should raise
    ConnectorException or child exceptions
    Attributes:
        connector_name -- connection name which initiate the error
        message -- any str
    """

    def __init__(self, message: str, connector_name: str, error_code: int = 2000):
        self.connector_name = connector_name
        super().__init__(
            status_code=422,
            message=f"{connector_name}: {message}",
            error_code=error_code,
        )


class ConnectorClientFailureException(ConnectorException):
    """
    ConnectorClientFailureException thrown when client connection failed to connect to server
    """

    # Error code for ConnectorException translate to connection failure
    error_code = 2001

    def __init__(self, message: str, connector_name: str):
        super().__init__(
            message=message, error_code=self.error_code, connector_name=connector_name
        )


class ConnectorIllegalNameException(ConnectorException):
    """
    Illegal connection name error will be raised when caller is trying to get connector with unknown name
    """

    error_code = 2002

    def __init__(self, connector_name: str, message: str):
        super().__init__(
            message=message, error_code=self.error_code, connector_name=connector_name
        )


class ConnectorNotThereException(ConnectorException):
    """
    Not There connector when a caller trying to get an identified connector but not enabled in env.
    """

    error_code = 2003

    def __init__(self, connector_name: str, message: str):
        super().__init__(
            message=message, error_code=self.error_code, connector_name=connector_name
        )


class ConnectorClientInternalFailureException(ConnectorException):
    """
    ConnectorClientInternalFailureException thrown when client unable to perform given operation on active connection.
    """

    error_code = 2004

    def __init__(self, connector_name: str, message: str):
        super().__init__(
            message=message, error_code=self.error_code, connector_name=connector_name
        )
