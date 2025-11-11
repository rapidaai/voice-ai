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
from semantic_router.encoders import OpenAIEncoder


class OpenaiEncoder(OpenAIEncoder):

    """
    The above function is a constructor that initializes an object with a model name and API key.
    
    :param model_name: The `model_name` parameter in the `__init__` method is a string that represents
    the name of the model. It is used to initialize an instance of the class with the specified model
    name
    :type model_name: str
    :param api_key: The `api_key` parameter is a string that represents an authentication key used to
    access an API. It is typically provided by the API provider to authenticate and authorize requests
    made by the client application
    :type api_key: str
    :return: The `super().__init__(name=model_name, api_key=api_key)` statement is returning the
    initialization of the parent class with the `model_name` and `api_key` parameters passed to it.
    """
    def __init__(self, api_key: str, model_name: str = "text-embedding-3-large"):
        return super().__init__(name=model_name, openai_api_key=api_key)
    