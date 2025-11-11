#!/bin/bash
############################################
# activate and setup other project utils  #
############################################
PIP_ENV=$(which pipenv)
eval "$PIP_ENV" install
echo "Activating pipenv shell."
source "$("$PIP_ENV" --venv)/bin/activate"
# run lock and sync

echo "Updating dependencies."
eval "$PIP_ENV" update --dev

#####################################################
# Run the defined lint using pre-commit             #
#####################################################
eval "$PIP_ENV" run lint

#####################################################
# Generate new requirements
#####################################################
eval "$PIP_ENV" run requirements

#####################################################
# Run the test
#####################################################
eval "$PIP_ENV" run test

# shellcheck disable=SC2181
if [ ! $? -eq 0 ]; then echo "Test failed, failing script."; exit 1; fi
echo "Exiting build script."
exit
