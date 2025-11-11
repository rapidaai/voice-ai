#!/bin/bash
sleep 100
# this service is not exposing any http 
echo "Validating Uvicorn service..."
curl -f http://0.0.0.0:9010/readiness/ || exit 1  # Assuming you have a health check endpoint
