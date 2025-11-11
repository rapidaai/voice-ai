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
from datetime import datetime
from typing import Any, Dict, List, Optional, Tuple


def serialize_elastic_search_response(
    content: Dict[str, Any], data_key: str = "_source"
) -> Tuple[int, Optional[List]]:
    """
    Serializing elastic search response and convert into len(data), data
    :param: content response from es
    :returns: count of data simplified len(data)
    :rtype: int
    :returns: data, records
    :rtype: List
    """
    count: int = content["hits"]["total"]["value"]
    if count <= 0:
        return count, None

    return count, [objects[data_key] for objects in content["hits"]["hits"]]


def list_to_map_with_key_matcher(
    lists: List[Dict[str, Any]], key_matcher: str
) -> Dict[str, Any]:
    """
    Converting dict to Map<Searchable_Key, content> for fast retrieval by key
    :param: lists, which needs to be converted into map
    :param: key_matcher, a key of dictionary which will get used for lookup
    :returns: Map key-> content
    :rtype: Dict
    """
    dict_map: Dict[str, Any] = {}
    for content in lists:
        dict_map[str(content[key_matcher])] = content

    return dict_map


def to_camel_case(string: str) -> str:
    """
    Convert snake_case string to camelCase string
    :param string:
    :return:
    """
    string_split = string.split("_")
    return string_split[0].lower() + "".join(
        word.capitalize() for word in string_split[1:]
    )


def to_snake_case(string: str) -> str:
    """
    Convert camel case string to snake case string
    :param string:
    :return:
    """
    return "".join(["_" + c.lower() if c.isupper() else c for c in string]).lstrip("_")


def serialize_gfycat_response(gfycat_results: List, gfycat_user: int) -> List:

    serialized_results: List = []
    if len(gfycat_results) == 0:
        return serialized_results
    for gfycat_result in gfycat_results:
        try:
            duration = int(
                gfycat_result["numFrames"] * 1000 / gfycat_result["frameRate"]
            )
        except Exception:
            duration = 0

        mime_type = "image/gif"
        if gfycat_result["mp4Url"].endswith("mp4"):
            mime_type = "video/mp4"

        serialized_results.append(
            {
                "id": gfycat_result["gfyId"],
                "owner_id": gfycat_user,
                "username": "gfycat",
                "privacy": 3,
                "mime_type": mime_type,
                "name": gfycat_result.get("title", ""),
                "file": gfycat_result["mp4Url"],
                "preview": gfycat_result["gifUrl"],
                "thumbnail": gfycat_result["mobilePosterUrl"],
                "created": datetime.utcfromtimestamp(
                    gfycat_result["createDate"]
                ).isoformat(),
                "source": "gfycat",
                "banned": False,
                "duration": duration,
                "is_favorite": False,
                "tags": gfycat_result.get("tags", []),
                "lomotif_count": 0,
                "aspect_ratio": "",
                "width": gfycat_result.get("width", 0),
                "height": gfycat_result.get("height", 0),
            }
        )

    return serialized_results
