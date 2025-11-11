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
import random

from sqlalchemy import CHAR, TypeDecorator
from sqlalchemy.dialects.postgresql import UUID
import time


class StringUUID(TypeDecorator):
    impl = CHAR
    cache_ok = True

    def process_bind_param(self, value, dialect):
        if value is None:
            return value
        elif dialect.name == 'postgresql':
            return str(value)
        else:
            return value.hex

    def load_dialect_impl(self, dialect):
        if dialect.name == 'postgresql':
            return dialect.type_descriptor(UUID())
        else:
            return dialect.type_descriptor(CHAR(36))

    def process_result_value(self, value, dialect):
        if value is None:
            return value
        return str(value)


def generate_snowflake_id() -> int:
    timestamp = int(time.time() * 1000)  # Current timestamp in milliseconds
    node_id = random.randint(0, 1023)  # Random node identifier
    sequence = random.randint(0, 4095)  # Random sequence number
    snowflake_id = (timestamp << 22) | (node_id << 12) | sequence
    return snowflake_id
