#!/bin/bash
export ENV_PATH="/opt/app/backend-app/artifacts/web-api/env"
/opt/app/backend-app/artifacts/web-api/web-api.0.0.1 > /dev/null 2> /dev/null < /dev/null &
