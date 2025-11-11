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
from dateutil.parser import parse


def is_date(string, fuzzy=False):
    """
    Return whether the string can be interpreted as a date.
    - https://stackoverflow.com/a/25341965/7120095
    :param string: str, string to check for date
    :param fuzzy: bool, ignore unknown tokens in string if True
    """

    try:
        parse(string, fuzzy=fuzzy)
        return True
    except ValueError:
        return False
