#!/bin/bash
export ENV_PATH="/opt/app/backend-app/artifacts/endpoint-api/env.production"
/opt/app/backend-app/artifacts/endpoint-api/endpoint-api.0.0.1 > /dev/null 2> /dev/null < /dev/null &
