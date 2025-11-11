#!/bin/bash
echo "verifying python version, only python3 is supported."


############################################
# Validation phase for python3 and pip  #
############################################
PYTHON_PATH=$(which python3)
PIP3=$(which pip3)
if [ -z "$PYTHON_PATH" ]; then echo "please install python3 to setup the development env."; exit; fi
if [ -z "$PIP3" ]; then echo "please install pip3 to setup the development env."; exit; fi

############################################
# install pipenv using pip3                 #
############################################
PIP_ENV=$(which pipenv)
if [ -z "$PIP_ENV" ]; then eval "$PIP3" install pipenv; exit; fi

############################################
# activate and setup other project utils  #
############################################
eval "$PIP_ENV" install --python "$PYTHON_PATH"
echo "Activating pipenv shell."
source "$("$PIP_ENV" --venv)/bin/activate"
# run lock and sync

echo "Updating dependencies."
eval "$PIP_ENV" update --dev

#####################################################
# Setup process for project defined as pipenv task  #
#####################################################
eval "$PIP_ENV" run setup

echo "Exiting setup script."
exit
