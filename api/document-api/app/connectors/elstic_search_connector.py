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

import datetime
import functools
import hashlib
import hmac
import logging
from typing import Callable, Dict, Optional
from urllib.parse import quote, urlencode, urlparse

import elasticsearch
from elasticsearch import (
    AIOHttpConnection,
    AsyncElasticsearch,
    ElasticsearchException,
    ImproperlyConfigured,
)

from app.configs.auth.aws_auth import AWSAuth
from app.configs.auth.basic_auth import BasicAuth
from app.configs.elastic_search_config import ElasticSearchConfig
from app.connectors import Connector
from app.connectors.aws.sts_connector import STSConnector
from app.exceptions.connector_exception import ConnectorClientFailureException
from app.observabilities import SpanOutcome, within_span

_log = logging.getLogger("app.connector.elasticsearch")


class ElasticSearchConnector(Connector):
    """
    Elastic search connector (Connection wrapper for elastic search)
    Fast fail connection implementer if server will not able to establish the connection it will raise exceptions
    Connection failure is relied on ping
    """

    #
    # elastic search connection instance
    connection: Optional[AsyncElasticsearch] = None

    # version of elastic search server
    # not yet implemented
    version: Optional[str]

    #
    # Elastic search config
    _config: ElasticSearchConfig

    # aws credential resolver
    _aws_credential_resolver: Optional[Callable]

    # when using aws authentication
    _aws_region: Optional[str]

    # name of connection
    _name: str

    def __init__(self, config: ElasticSearchConfig, name: str = "es"):
        self._config = config
        self._name = name

        if isinstance(self._config.auth, AWSAuth):
            # if aws authentication is enabled
            _log.debug(f"Enabled aws authentication for connector name {name}")
            aws_config: AWSAuth = self._config.auth
            self._aws_region = aws_config.region

            # initialize sts connector to resolve credentials
            _sts_connector = STSConnector(aws_config)
            self._aws_credential_resolver = _sts_connector.get_temporary_credentials

    @property
    def name(self) -> str:
        """
        Get name of connection
        """
        return self._name

    async def connect(self) -> AsyncElasticsearch:
        """
        Connect to elastic search server
        creating a connection of elastic search.
        """
        if self.connection:
            _log.debug(f"Connection {self._name} is already established.")
            return self.connection
        #
        with within_span(
            (
                f"ES Connection {self._config.host.lower()}:{self._config.port}"
                if self._config.port is not None
                else f"ES Connection {self._config.host.lower()}"
            ),
            span_type="external",
            span_subtype="elasticsearch",
            span_action="connect",
        ) as span:
            try:
                _log.info(
                    f"Trying to connect with {self._config.scheme}://{self._config.host}"
                )
                if isinstance(self._config.auth, BasicAuth):

                    # if authentication is basic auth (user and password)
                    _log.info(
                        "Basic authentication enabled trying the connection with auth header."
                    )
                    self.connection = AsyncElasticsearch(
                        hosts=[
                            (
                                f"{self._config.scheme}://{self._config.host}"
                                # append only when port is there
                                + f":{self._config.port}"
                                if self._config.port
                                else ""
                            )
                        ],
                        http_auth=(
                            self._config.auth.user,
                            self._config.auth.password.get_secret_value(),
                        ),
                        maxsize=self._config.max_connection,
                    )
                elif isinstance(self._config.auth, AWSAuth):

                    # creating connection url from schema, host and port
                    _connection_url = f"{self._config.scheme}://{self._config.host}"

                    _connection_url += (
                        f":{self._config.port}" if self._config.port else ""
                    )
                    self.connection = AsyncElasticsearch(
                        hosts=[_connection_url],
                        aws_host=self._config.host,
                        aws_region=self._aws_region,
                        credential_resolver=self._aws_credential_resolver,
                        # maximum connection size
                        maxsize=self._config.max_connection,
                        connection_class=AWSAuthAIOHttpConnection,
                        # other
                        use_ssl=True,
                        verify_certs=True,
                    )
                else:
                    # if np authentication is enabled for elastic search
                    self.connection = AsyncElasticsearch(
                        hosts=[
                            f"{self._config.scheme}://{self._config.host}:{self._config.port}"
                        ]
                    )
                if not await self.ping:
                    # not able to ping the connection
                    _log.error("Failed to ping to elastic search.")

                    # setting the connection to none so no need to update the state of connector
                    self.connection = None

                    # raise failure if not able to ping
                    raise ConnectorClientFailureException(
                        connector_name=self.name,
                        message="Unable to ping to elastic search server.",
                    )
                return self.connection
            except ImproperlyConfigured as error:

                # Any configuration error
                self.connection = None
                _log.error(f"Problem in elastic search configurations. {error}")
                span.set_status(SpanOutcome.FAILURE, description=str(error))
                raise ConnectorClientFailureException(
                    connector_name=self.name, message=str(error)
                )
            except Exception as e:
                # if another execution happen
                self.connection = None
                # wrapping issue with connector failure to gracefully handle
                _log.error(f"Failed to connect to elastic search. {str(e)}")
                span.set_status(SpanOutcome.FAILURE, description=str(e))
                raise ConnectorClientFailureException(
                    connector_name=self.name, message=str(e)
                )

    # check if the connection is open and active
    async def is_connected(self) -> bool:
        return bool(self.connection and await self.ping is True)

    # Ping elastic search node
    @property
    async def ping(self) -> bool:
        try:
            return await self.operate("ping")
        except ElasticsearchException as error:
            _log.error(f"Unable to ping to elastic search server. {error}")
            return False

    # Close current instance of connection
    async def disconnect(self):
        try:
            await self.connection.close()
        except ElasticsearchException as error:
            _log.error(f"Unable to close connection. {error}")
        self.connection = None

    async def operate(self, command: str, **kwargs):
        """
        Execute command on elastic search server
        """
        with within_span(
            (
                f"ES Operate {self._config.host.lower()}:{self._config.port}"
                if self._config.port is not None
                else f"ES Operate {self._config.host.lower()}"
            ),
            span_type="external",
            span_subtype="elasticsearch",
            span_action=command,
        ) as span:
            try:

                await self.connect()
                return await getattr(self.connection, command)(**kwargs)
            except ElasticsearchException as error:
                span.set_status(SpanOutcome.FAILURE, description=str(error))
                _log.error(
                    f"Unable to execute command {command} on elastic search server. {error}"
                )
                raise ConnectorClientFailureException(
                    connector_name=self.name, message=str(error)
                )
            except elasticsearch.ConnectionTimeout as timeout_error:
                _log.error(
                    f"Unable to execute command {command} on elastic search server. {timeout_error}"
                )
                span.set_status(SpanOutcome.FAILURE, description=str(timeout_error))
                raise ConnectorClientFailureException(
                    connector_name=self.name, message=str(timeout_error)
                )


class AWSAuthAIOHttpConnection(AIOHttpConnection):
    """Enable AWS Auth with AIOHttpConnection for AsyncElasticsearch

    The AIOHttpConnection class built into elasticsearch-py is not currently
    compatible with passing AWSAuth as the `http_auth` parameter, as suggested
    in the docs when using AWSAuth for the non-async RequestsHttpConnection class:
    https://docs.aws.amazon.com/opensearch-service/latest/developerguide/request-signing.html#request-signing-python

    This approach was synthesized from
    * https://github.com/DavidMuller/aws-requests-auth
    * https://github.com/jmenga/requests-aws-sign
    * https://github.com/byrro/aws-lambda-signed-aiohttp-requests
    """

    SIGV4_TIMESTAMP = "%Y%m%dT%H%M%SZ"
    SIGV4_DATE = "%Y%m%d"
    PAYLOAD_BUFFER = 1024 * 1024
    EMPTY_SHA256_HASH = (
        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    )
    AWS_SERVICE = "es"

    #  aws region for open search implementation
    # can be changed later with `cloud_region`
    aws_region: str

    # host for open search
    aws_host: str

    # call to resolve credential
    credential_resolver: Callable

    def __init__(
        self, aws_host, aws_region, credential_resolver: Callable, *args, **kwargs
    ):
        """
        The aws_token is optional and is used only if you are using STS
        temporary credentials.
        """
        super().__init__(*args, **kwargs)
        self.aws_region = aws_region
        self.aws_host = aws_host
        self.credential_resolver = credential_resolver

    def sign(self, key, msg):
        """
        Copied from https://docs.aws.amazon.com/general/latest/gr/sigv4-signed-request-examples.html
        """
        return hmac.new(key, msg.encode("utf-8"), hashlib.sha256).digest()

    def get_signature_key(self, key, datestamp):
        """
        Copied from https://docs.aws.amazon.com/general/latest/gr/sigv4-signed-request-examples.html
        """
        k_date = self.sign(("AWS4" + key).encode("utf-8"), datestamp)
        k_region = self.sign(k_date, self.aws_region)
        k_service = self.sign(k_region, self.AWS_SERVICE)
        return self.sign(k_service, "aws4_request")

    def get_aws_headers(
        self, method, url, params, body, aws_access_key, aws_secret_key, aws_token=None
    ):
        """
        Returns a dictionary containing the necessary headers for Amazon's
        signature version 4 signing process. An example return value might
        """

        # Create a date for headers and the credential string
        t = datetime.datetime.utcnow()
        amzdate = t.strftime(self.SIGV4_TIMESTAMP)
        datestamp = t.strftime(self.SIGV4_DATE)  # Date w/o time for credential_scope

        # Create the canonical headers and signed headers. Header names
        # and value must be trimmed and lowercase, and sorted in ASCII order.
        # Note that there is a trailing \n.
        headers = {"host": self.aws_host, "x-amz-date": amzdate}
        signed_headers = "host;x-amz-date"
        if aws_token:
            headers["x-amz-security-token"] = aws_token
            signed_headers += ";x-amz-security-token"

        # Create the list of signed headers. This lists the headers
        # in the canonical_headers list, delimited with ";" and in alpha order.
        # Note: The request can include any headers; canonical_headers and
        # signed_headers lists those that you want to be included in the
        # hash of the request. "Host" and "x-amz-date" are always required.

        canonical_headers = ""
        for key in sorted(headers):
            canonical_headers += f"{key.lower()}:{headers[key]}" + "\n"

        # Combine elements to create create canonical request
        canonical_request = (
            method
            + "\n"
            + self.get_canonical_path(url)
            + "\n"
            + self.get_canonical_querystring(params)
            + "\n"
            + canonical_headers
            + "\n"
            + signed_headers
            + "\n"
            + self.get_body(body)
        )

        _log.debug(f"Canonical request {canonical_request}")
        # Match the algorithm to the hashing algorithm you use, either SHA-1 or
        # SHA-256 (recommended)

        algorithm = "AWS4-HMAC-SHA256"
        credential_scope = (
            datestamp
            + "/"
            + self.aws_region
            + "/"
            + self.AWS_SERVICE
            + "/"
            + "aws4_request"
        )
        string_to_sign = (
            algorithm
            + "\n"
            + amzdate
            + "\n"
            + credential_scope
            + "\n"
            + hashlib.sha256(canonical_request.encode("utf-8")).hexdigest()
        )

        _log.debug(f"String to sign {string_to_sign}")
        # Create the signing key using the function defined above.
        signing_key = self.get_signature_key(aws_secret_key, datestamp)

        # Sign the string_to_sign using the signing_key
        string_to_sign_utf8 = string_to_sign.encode("utf-8")
        signature = hmac.new(
            signing_key, string_to_sign_utf8, hashlib.sha256
        ).hexdigest()

        # The signing information can be either in a query string value or in
        # a header named Authorization. This code shows how to use a header.
        # Create authorization header and add to request headers
        authorization_header = (
            algorithm
            + " "
            + "Credential="
            + aws_access_key
            + "/"
            + credential_scope
            + ", "
            + "SignedHeaders="
            + signed_headers
            + ", "
            + "Signature="
            + signature
        )

        headers = {
            "Authorization": authorization_header,
            "x-amz-date": amzdate,
        }
        if aws_token:
            headers["X-Amz-Security-Token"] = aws_token
        return headers

    def get_body(self, request_body):

        if request_body and hasattr(request_body, "seek"):
            position = request_body.tell()
            read_chunk_size = functools.partial(request_body.read, self.PAYLOAD_BUFFER)
            checksum = hashlib.sha256()
            for chunk in iter(read_chunk_size, b""):
                checksum.update(chunk)
            hex_checksum = checksum.hexdigest()
            request_body.seek(position)
            return hex_checksum
        elif request_body:
            # The request serialization has ensured that
            # request.body is a bytes() type.
            return hashlib.sha256(request_body).hexdigest()
        else:
            # for empty body
            return self.EMPTY_SHA256_HASH

    def get_canonical_path(self, url: str):
        """
        Create canonical URI--the part of the URI from domain to query
        string (use '/' if no path)
        """
        parsed_url = urlparse(url)

        # safe chars adapted from boto's use of urllib.parse.quote
        # https://github.com/boto/boto/blob/d9e5cfe900e1a58717e393c76a6e3580305f217a/boto/auth.py#L393
        return quote(parsed_url.path if parsed_url.path else "/", safe="/-_.~")

    def get_canonical_querystring(self, params: Dict):
        """
        Create the canonical query string. According to AWS, by the
        end of this function our query string values must
        be URL-encoded (space=%20) and the parameters must be sorted
        by name.
        This method assumes that the query params in `r` are *already*
        url encoded.  If they are not url encoded by the time they make
        it to this function, AWS may complain that the signature for your
        request is incorrect.
        It appears elasticsearc-py url encodes query paramaters on its own:
            https://github.com/elastic/elasticsearch-py/blob/5dfd6985e5d32ea353d2b37d01c2521b2089ac2b/elasticsearch/connection/http_requests.py#L64
        If you are using a different client than elasticsearch-py, it
        will be your responsibility to urlencoded your query params before
        this method is called.
        """
        canonical_querystring = ""
        if not params:
            return canonical_querystring

        querystring_sorted = "&".join(sorted(urlencode(params).split("&")))
        #
        for query_param in querystring_sorted.split("&"):
            key_val_split = query_param.split("=", 1)

            key = key_val_split[0]
            if len(key_val_split) > 1:
                val = key_val_split[1]
            else:
                val = ""

            if key:
                if canonical_querystring:
                    canonical_querystring += "&"
                canonical_querystring += "=".join([key, val])

        return canonical_querystring

    async def perform_request(
        self, method, url, params=None, body=None, timeout=None, ignore=(), headers=None
    ):
        # need to be removed depends on permission
        if not (method == "GET" or method == "POST"):
            method = "GET"
        credentials = await self.credential_resolver(session_name=f"ES_{method}")
        aws_headers = self.get_aws_headers(
            method=method,
            url=url,
            params=params,
            body=body,
            aws_access_key=credentials["access_key"],
            aws_secret_key=credentials["secret_key"],
            aws_token=credentials["token"],
        )

        # adding all aws headers
        headers.update(aws_headers)

        _log.debug(
            f"Generated all the headers from aws credentials overall headers {headers}"
        )
        return await super().perform_request(
            method, url, params, body, timeout, ignore, headers
        )
