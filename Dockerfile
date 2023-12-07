FROM golang:1.19.0-buster as builder

# installing required dependencies
RUN dpkg --add-architecture amd64 && \
    apt-get update -y

RUN apt-get install --no-install-recommends -y  build-essential=12.6 libcurl4-openssl-dev=7.64.0-4+deb10u3 libssl-dev=1.1.1n-0+deb10u3 && \
    rm -rf /var/lib/apt/lists/*

# installing cmake to build xgbooster from source
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]

RUN mkdir -p /cmake-3.25 && \
    wget -qO- "https://cmake.org/files/v3.25/cmake-3.25.0-rc1.tar.gz" | tar -xzC /cmake-3.25 --strip-components=1

WORKDIR /cmake-3.25/
RUN ./configure && make install

# installing xgboost
RUN mkdir -p /xgboost/src/xgboost && \
    git clone --depth 1 --recursive https://github.com/dmlc/xgboost.git /xgboost/src/xgboost

WORKDIR /xgboost/src/xgboost/
RUN mkdir -p /xgboost/src/xgboost/build
WORKDIR /xgboost/src/xgboost/build
RUN cmake -DCMAKE_BUILD_TYPE=Debug  -DCMAKE_SYSTEM_NAME=Linux -DCMAKE_SYSTEM_PROCESSOR=arm ..
RUN make -j$(nproc)

RUN cp /xgboost/src/xgboost/lib/libxgboost.so /usr/local/lib && \
    ldconfig -n -v /usr/local/lib && \
    cp -r /xgboost/src/xgboost/include/xgboost /usr/local/include/xgboost && \
    cp -r /xgboost/src/xgboost/rabit/include/rabit /usr/local/include/rabit && \
    cp -r /xgboost/src/xgboost/dmlc-core/include/dmlc /usr/local/include/dmlc && \
    rm -rf /xgboost/

# create the appropriate directories
ENV HOME=/app

# create directory for the app user
RUN mkdir -p $HOME

# create the app user
RUN addgroup --system lomo-app && adduser --system --group lomo-app

# create app dir
ENV APP_HOME=$HOME

# as work dir
WORKDIR $APP_HOME/

RUN chown lomo-app:lomo-app $APP_HOME

USER lomo-app

# copy mod and sum
COPY --chown=lomo-app:lomo-app go.mod go.mod
COPY --chown=lomo-app:lomo-app go.sum go.sum


# donwload dependencies
RUN go mod download

# COPY PROJECT
COPY --chown=lomo-app:lomo-app . .

# checking ld for xgboost
# RUN ld -lxgboost --verbose --entry main
# set lib path
ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

# building
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -s -w" -o internal_recommendation_service ./api/main.go
# RUN CGO_ENABLED=1 GOOS=linux go build -buildmode=c-shared -o internal_recommendation_service ./api/main.go

# https://stackoverflow.com/questions/55200508/docker-cant-run-a-go-output-file-that-already-exist
FROM debian

RUN apt-get update && apt-get install -y --no-install-recommends apt-utils curl libgomp1 && \
    rm -rf /var/lib/apt/lists/*
# RUN apt-get install --no-install-recommends -y  build-essential=12.6 && \
#     rm -rf /var/lib/apt/lists/*

# # copy all .so and c lib needed
COPY --from=builder  /usr/local/lib/libxgboost.so /usr/local/lib/libxgboost.so
COPY --from=builder /usr/local/include/xgboost /usr/local/include/xgboost
COPY --from=builder /usr/local/include/rabit /usr/local/include/rabit
COPY --from=builder /usr/local/include/dmlc /usr/local/include/dmlc

# RUN ldconfig -n -v /usr/local/lib
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt


ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

# # copy from builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
# # RUN addgroup --system lomo-app && adduser --system --group lomo-app

# path for service and artifacts
ENV APP=/app/
RUN mkdir -p $APP
RUN chown -R lomo-app:lomo-app  $APP


USER lomo-app
COPY --chown=lomo-app:lomo-app --from=builder /app/internal_recommendation_service $APP/internal_recommendation_service

WORKDIR $APP/
