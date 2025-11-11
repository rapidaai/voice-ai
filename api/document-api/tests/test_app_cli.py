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
import subprocess
import sys

import pytest

PYTHON_VERSION = float(f"{sys.version_info.major}.{sys.version_info.minor}")


@pytest.mark.skipif(reason="Minor marker differences", condition=PYTHON_VERSION != 3.8)
def test_requirements_txt():
    """Validate that requirements.txt and requirements-dev.txt
    are up2date with Pipefile"""
    temp_output_dir = "tests/temp_output"
    req_test_file_path = f"{temp_output_dir}/test-requirements.txt"
    req_dev_test_file_path = f"{temp_output_dir}/test-requirements-dev.txt"

    subprocess.call(f"mkdir -p {temp_output_dir}", shell=True)
    subprocess.call(
        "pipenv requirements \
                                    > {}".format(
            req_test_file_path
        ),
        shell=True,
    )

    subprocess.call(
        "pipenv requirements --dev-only \
                                    > {}".format(
            req_dev_test_file_path
        ),
        shell=True,
    )

    with open("requirements.txt") as file:
        req_file = file.read()

    with open("requirements-dev.txt") as file:
        req_dev_file = file.read()

    with open(req_test_file_path) as file:
        req_test_file = file.read()

    with open(req_dev_test_file_path) as file:
        req_dev_test_file = file.read()

    subprocess.call(f"rm -rf {temp_output_dir}", shell=True)

    assert req_file == req_test_file

    assert req_dev_file == req_dev_test_file
