#!/bin/bash
echo "Setting up permissions and environment variables"
# Change to app directory
cd /opt/app/backend-app/artifacts/document-api/


# Create a virtual environment
python3 -m venv venv

# Activate the virtual environment
source venv/bin/activate

# Install Python dependencies
pip3 install -r requirements.txt


# Deactivate the virtual environment
deactivate