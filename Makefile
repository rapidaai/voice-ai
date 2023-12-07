#! /usr/bin/make -f
# Go related variables.

GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin

.PHONY: migrateup
.PHONY: migratedown


# Go files.
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)

# Common commands.
all: fmt test
development: precommit-install githooks-install

# only to setup development machine
githooks-install:
	@echo "  >  Setting up githooks."
	@chmod +x ./githooks/commit-msg
	@chmod +x ./githooks/gitmessage.txt
	@chmod +x ./scripts/git-commit-hook-setup.sh
	sh ./scripts/git-commit-hook-setup.sh


precommit-install:
	@echo "  >  Installing precommit."
	@wget -O ./bin/pre-commit https://github.com/pre-commit/pre-commit/releases/download/v2.20.0/pre-commit-2.20.0.pyz
	@chmod +x ./bin/pre-commit
	@echo "  > Installing precommit hooks."
	@echo ./bin/pre-commit install
	./bin/pre-commit install

gofmt:
	gofmt -s -w ${GOFMT_FILES}

run:
	go run ./api/main.go

build:
	GOOS=linux GOARCH=amd64 go build -o lexatic-backend ./api/main.go

test:
	@echo "  >  Running unit tests."
	GOBIN=$(GOBIN) go test -cover -race -coverprofile=coverage.txt -covermode=atomic -v ./...

migrateup:
	migrate -path sql/migration -database "postgresql://trifacta:secret@localhost:5432/lexatic?sslmode=disable" -verbose up

migratedown:
	migrate -path sql/migration -database "postgresql://trifacta:secret@localhost:5432/lexatic?sslmode=disable" -verbose down
# protoc -Iprotos --go_opt=module=github.com/lexatic/web-backend/protos/lexatic-backend --go_out=./protos/lexatic-backend/ --go-grpc_opt=module=github.com/lexatic/web-backend/protos/lexatic-backend --go-grpc_out=require_unimplemented_servers=false:./protos/lexatic-backend/ protos/lexatic-backend/*.proto

# --go-grpc_out=require_unimplemented_servers=false:.
