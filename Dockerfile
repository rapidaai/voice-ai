FROM golang:1.21.0-buster as builder

# installing required dependencies
RUN dpkg --add-architecture amd64 && \
    apt-get update -y


# create the appropriate directories
ENV HOME=/app

# create directory for the app user
RUN mkdir -p $HOME

# create the app user
RUN addgroup --system rapida-app && adduser --system --group rapida-app

# create app dir
ENV APP_HOME=$HOME

# as work dir
WORKDIR $APP_HOME/

RUN chown rapida-app:rapida-app $APP_HOME

USER rapida-app

# copy mod and sum
COPY --chown=rapida-app:rapida-app go.mod go.mod
COPY --chown=rapida-app:rapida-app go.sum go.sum


# donwload dependencies
RUN go mod download

# COPY PROJECT
COPY --chown=rapida-app:rapida-app . .



# building
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags "-linkmode external -s -w" -o web_api ./api/main.go

# https://stackoverflow.com/questions/55200508/docker-cant-run-a-go-output-file-that-already-exist
FROM debian

RUN apt-get update && apt-get install -y --no-install-recommends apt-utils curl libgomp1 && \
    rm -rf /var/lib/apt/lists/*


# # copy from builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
# # RUN addgroup --system rapida-app && adduser --system --group rapida-app

# path for service and artifacts
ENV APP=/app/
RUN mkdir -p $APP
RUN chown -R rapida-app:rapida-app  $APP


USER rapida-app
COPY --chown=rapida-app:rapida-app --from=builder /app/web-api $APP/web-api

WORKDIR $APP/
