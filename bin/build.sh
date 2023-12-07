#!/bin/bash

echo "  >  Installing precommit."
wget -O ./bin/pre-commit https://github.com/pre-commit/pre-commit/releases/download/v2.20.0/pre-commit-2.20.0.pyz
chmod +x ./bin/pre-commit
echo "  > Installing precommit hooks."

#####################################################
# Running using pre-commit                          #
#####################################################
./bin/pre-commit run --all-files

# shellcheck disable=SC2181
if [ ! $? -eq 0 ]; then echo "Test failed, failing script."; exit 1; fi

# setup Dependency
HOME_DIR=$(pwd)


# installing make
mkdir -p /cmake-3.25 && wget -qO- "https://cmake.org/files/v3.25/cmake-3.25.0-rc1.tar.gz" | tar -xzC /cmake-3.25 --strip-components=1
cd /cmake-3.25/
chmod +x ./configure
./configure && make install

# downloading xgboost from git
mkdir -p /xgboost/src/xgboost && git clone --depth 1 --recursive https://github.com/dmlc/xgboost.git /xgboost/src/xgboost

cd /xgboost/src/xgboost/
mkdir -p /xgboost/src/xgboost/build && \
cd /xgboost/src/xgboost/build && \
cmake -DCMAKE_BUILD_TYPE=Debug  -DCMAKE_SYSTEM_NAME=Linux -DCMAKE_SYSTEM_PROCESSOR=arm .. && make -j$(nproc)

cd $HOME_DIR

cp /xgboost/src/xgboost/lib/libxgboost.so /usr/local/lib
cp -r /xgboost/src/xgboost/include/xgboost /usr/local/include/xgboost
cp -r /xgboost/src/xgboost/rabit/include/rabit /usr/local/include/rabit
cp -r /xgboost/src/xgboost/dmlc-core/include/dmlc /usr/local/include/dmlc
# removing  xgboost source code
rm -rf /xgboost/

# ldconfig -n -v /usr/local/lib
export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

go test -cover -race -coverprofile=./coverage.txt -covermode=atomic -v ./...

# shellcheck disable=SC2181
if [ ! $? -eq 0 ]; then echo "Test failed, failing script."; exit 1; fi
echo "Exiting build script."
exit
