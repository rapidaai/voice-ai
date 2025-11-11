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
from app.utils.date import is_date


def test_is_date():
    """
    Testing is_date
    :return:
    """
    assert is_date("1990-12-1")
    assert is_date("2005/3")
    assert is_date("Jan 19, 1990")
    assert not is_date("today is 2019-03-27")
    assert is_date("today is 2019-03-27", fuzzy=True)
    assert is_date("Monday at 12:01am")
    assert not is_date("xyz_not_a_date")
    assert not is_date("yesterday")
