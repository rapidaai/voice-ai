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


class RapidaException(Exception):
    """
    Lomotif common exceptions class.
    Exception should be controlled by error code.
    """

    def __init__(
        self,
        status_code: int,
        message: str,
        error_code: int,
        error_prefix: str = "RAPIDA",
        service_code: str = "KN_API",
    ):
        """
        :param status_code: http status code
        :param error_prefix: error prefix
        :param service_code: service code <should be unique identifiable>
        """
        super().__init__(self)
        self.status_code = status_code
        self.message = message
        self.error_code = f"{error_prefix}_{service_code}_{error_code}"

    def __str__(self):
        return f"{self.error_code} - {self.message}"
