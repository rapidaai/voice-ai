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
if [ -z "$PIP_ENV" ]; then eval "$PIP3" install pipenv=="2022.4.30"; exit; fi
