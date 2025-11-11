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
from typing import Any

from fastapi import Request
from pydantic import HttpUrl
from starlette.datastructures import Headers


def get_url_from_request(request: Request) -> HttpUrl:
    """
    generate HttpUrl from fastApi request
    :param: request current request with context
    :return HttpUrl
    """
    return HttpUrl(
        scheme=request.url.scheme,
        url=get_absolute_url(request),
    )


def get_absolute_url(request: Request) -> str:
    return "".join(
        [
            f"{request.url.scheme}://",
            f"{request.url.hostname}",
            f":{request.url.port}" if request.url.port is not None else "",
            f"{request.url.path}" if request.url.path is not None else "",
        ]
    )


def url_path_with_query_param(request: Request):
    """
    generate url path with query param
    :param request:
    :return:
    """
    return str(request.url).replace(str(request.base_url), "/")


def get_header_element(headers: Headers, key: str, default: Any):
    """
    get header element from request headers
    :param headers:
    :param key:
    :param default:
    :return:
    """
    try:
        value = (
                headers.get(key, default)
                or headers.get(key.replace("HTTP_", ""), default)
                or headers.get(key.replace("HTTP_", "").replace("_", "-"), default)
        )
        return value
    except (ValueError, IndexError, TypeError):
        return default


def get_url_path_with_query_param(request: Request):
    """
    generate url path with query param
    :param request:
    :return:
    """
    return str(request.url).replace(str(request.base_url), "/")


def get_relative_url(request: Request) -> str:
    """
    getting relative path for request
    :param request:
    :return:
    """
    return "".join(
        [
            f"{request.url.path}" if request.url.path is not None else "/",
        ]
    )


def is_elastic_search_reserved_character(char: str) -> bool:
    """
    check if character needs escaping all elastic search reserved characters
    @see https://www.elastic.co/guide/en/
    elasticsearch/reference/current/query-dsl-query-string-query.html#_reserved_characters
    :param char:
    :return: bool
    """
    escape_chars = {
        "\\": True,
        "+": True,
        "-": True,
        "!": True,
        "(": True,
        ")": True,
        ":": True,
        "^": True,
        "[": True,
        "]": True,
        '"': True,
        "{": True,
        "}": True,
        "~": True,
        "*": True,
        "?": True,
        "|": True,
        "&": True,
        "/": True,
    }
    return escape_chars.get(char, False)
