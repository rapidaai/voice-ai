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


class BridgeException(RapidaException):
    """
    Bridge Error -> all bridge class or function should raise BridgeException or child errors
    """

    def __init__(self, message: str, bridge_name: str, error_code: int = 1000):
        self.bridge_name = bridge_name
        self.status_code = 400
        self.message = f"{bridge_name}: {message}"
        self.error_code = error_code


class BridgeClientException(BridgeException):
    """
    BridgeClientException is wrapper for all client exception raised by aiohttp client.
    """

    #
    # Error code for BridgeException translate to internal service failure
    error_code = 1001

    def __init__(self, message: str, bridge_name: str):
        super().__init__(
            message=message, error_code=self.error_code, bridge_name=bridge_name
        )


class BridgeInternalFailureException(BridgeException):
    """
    BridgeInternalFailureException is wrapper for all internal exception raised by internal service.
    """

    #
    # Error code for BridgeException translate to internal service failure
    error_code = 1002

    def __init__(self, message: str, bridge_name: str):
        super().__init__(
            message=message, error_code=self.error_code, bridge_name=bridge_name
        )
