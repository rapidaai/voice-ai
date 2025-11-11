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
import os
import shutil
from collections.abc import Generator

from app.configs.storage_config import AssetStoreConfig
from app.storage.file_storage.base_storage import BaseStorage


class LocalStorage(BaseStorage):
    """Implementation for local storage.
    """

    def __init__(self, config: AssetStoreConfig):
        super().__init__()
        folder = config.storage_path_prefix
        self.folder = folder

    def save(self, filename, data):
        if not self.folder or self.folder.endswith('/'):
            filename = self.folder + filename
        else:
            filename = self.folder + '/' + filename

        folder = os.path.dirname(filename)
        os.makedirs(folder, exist_ok=True)

        with open(os.path.join(os.getcwd(), filename), "wb") as f:
            f.write(data)

    def load_once(self, filename: str) -> bytes:
        if not self.folder or self.folder.endswith('/'):
            filename = self.folder + filename
        else:
            filename = self.folder + '/' + filename

        if not os.path.exists(filename):
            raise FileNotFoundError("File not found")

        with open(filename, "rb") as f:
            data = f.read()

        return data

    def load_stream(self, filename: str) -> Generator:
        def generate(filename: str = filename) -> Generator:
            if not self.folder or self.folder.endswith('/'):
                filename = self.folder + filename
            else:
                filename = self.folder + '/' + filename

            if not os.path.exists(filename):
                raise FileNotFoundError("File not found")

            with open(filename, "rb") as f:
                while chunk := f.read(4096):  # Read in chunks of 4KB
                    yield chunk

        return generate()

    def download(self, filename, target_filepath):
        if not self.folder or self.folder.endswith('/'):
            filename = self.folder + filename
        else:
            filename = self.folder + '/' + filename

        if not os.path.exists(filename):
            raise FileNotFoundError("File not found")

        shutil.copyfile(filename, target_filepath)

    def exists(self, filename):
        if not self.folder or self.folder.endswith('/'):
            filename = self.folder + filename
        else:
            filename = self.folder + '/' + filename

        return os.path.exists(filename)

    def delete(self, filename):
        if not self.folder or self.folder.endswith('/'):
            filename = self.folder + filename
        else:
            filename = self.folder + '/' + filename
        if os.path.exists(filename):
            os.remove(filename)
