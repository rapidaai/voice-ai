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


class AuthenticationException(RapidaException):
    def __init__(
        self,
        message: str,
        auth_type: str,
        status_code: int = 400,
        error_code: int = 1000,
    ):
        super().__init__(
            status_code=status_code,
            message=f"{auth_type}: {message}",
            error_code=error_code,
        )


class MissingAuthorizationKeyException(AuthenticationException):
    error_code: int = 3001
    status_code: int = 400

    def __init__(self, auth_type: str, message: str = "Missing Authorization Key"):
        super().__init__(
            message=message,
            error_code=self.error_code,
            auth_type=auth_type,
            status_code=self.status_code,
        )


class InvalidAuthorizationTokenException(RapidaException):
    error_code: int = 3002
    status_code: int = 401

    def __init__(
        self,
        message: str = "Invalid Authorization Token | "
        "Invalid Signature Error | "
        "Invalid Key Error | "
        "Signature has expired",
    ):
        super().__init__(
            message=message,
            error_code=self.error_code,
            status_code=self.status_code,
        )
