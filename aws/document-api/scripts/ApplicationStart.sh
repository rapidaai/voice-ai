#!/bin/bash
echo "Starting document application"
cd /opt/app/backend-app/artifacts/document-api/ || exit 1  # Exit if cd fails

# Activate virtual environment
source venv/bin/activate

# Start the application using Uvicorn in the background
echo "Starting the Python application"
nohup uvicorn app.main:app --host 0.0.0.0 --port 9010 --workers 2 > /opt/app/backend-app/artifacts/document-api/uvicorn.log 2>&1 &

# Ensure the command has started successfully by checking the process list
sleep 2  # Give it a moment to start

# Check if Uvicorn started successfully
if pgrep -f "uvicorn" > /dev/null; then
    echo "Uvicorn started on port 9010"
else
    echo "Failed to start Uvicorn"
    exit 1
fi

deactivate