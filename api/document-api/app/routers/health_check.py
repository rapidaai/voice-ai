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
from typing import Dict

from fastapi import APIRouter, Depends, Request

from app.connectors import Connector
from app.connectors.connector_factory import get_all_connectors

H1 = APIRouter()


@H1.get("/healthz/")
async def health(request: Request):
    """Health check enabled"""
    return {"healthy": True}


@H1.get("/readiness/")
async def readiness(request: Request, connectors: Dict = Depends(get_all_connectors)):
    """rediness enabled"""
    connections_status: Dict = {}
    for key in connectors:
        connection: Connector = connectors[key]
        connections_status[key] = {"is_connected": await connection.is_connected()}
    return connections_status
