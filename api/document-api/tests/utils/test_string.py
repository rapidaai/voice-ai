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
from app.utils.string import list_to_map_with_key_matcher, to_camel_case, to_snake_case


def test_to_camel_case():
    """Testing to_camel_case"""
    assert to_camel_case("test_to_camel_case") == "testToCamelCase"
    assert to_camel_case("test_to_camel_case") == "testToCamelCase"
    assert to_camel_case("user_id") == "userId"
    assert to_camel_case("User_id") == "userId"


def test_to_snake_case():
    """Testing to_snake_case"""
    assert to_snake_case("test_to_snake_case") == "test_to_snake_case"
    assert to_snake_case("testToCamelCase") == "test_to_camel_case"
    assert to_snake_case("userId") == "user_id"
    assert to_snake_case("userId") == "user_id"
    assert to_snake_case("user_id") == "user_id"


def test_list_to_map_with_key_matcher():
    """Testing list_to_map_with_key_matcher"""
    list_data = [
        {"key": "key1", "body": "body1", "id": "id1"},
        {"key": "key2", "body": "body2", "id": "id2"},
        {"key": "key3", "body": "body3", "id": "id3"},
        {"key": "key4", "body": "body4", "id": "id4"},
    ]
    _map_of_key_object = list_to_map_with_key_matcher(list_data, "key")
    assert _map_of_key_object["key1"]["body"] == "body1"
    assert _map_of_key_object["key1"]["id"] == "id1"
    assert _map_of_key_object["key2"]["body"] == "body2"
    assert _map_of_key_object["key2"]["id"] == "id2"
    assert _map_of_key_object["key3"]["body"] == "body3"
    assert _map_of_key_object["key3"]["id"] == "id3"
    assert _map_of_key_object["key4"]["body"] == "body4"
    assert _map_of_key_object["key4"]["id"] == "id4"
