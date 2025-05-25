BIN_NAME := memdb
LINTER_VERSION := v1.64.8
MOCKERY_VERSION_v2 := 51.1

GOBIN=${GOPATH}/bin
PROTOC_GEN_GO_VERSION := v1.36.6

.PHONY: all
all: clean lint test build

.PHONY: .install-mockery
.install-mockery:
	go install github.com/vektra/mockery/v2@v2.${MOCKERY_VERSION_v2}

.PHONY: .install-linter
.install-linter:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOPATH}/bin ${LINTER_VERSION}

.PHONY: .install-protoc-gen-go
.install-protoc-gen-go:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}

.PHONY: lint
lint:
	golangci-lint --version
	golangci-lint linters
	golangci-lint run -v

.PHONY: lint.fix
lint.fix:
	golangci-lint run --fix

.PHONY: test
test:
	go test -race -count=1 ./internal/...

test.cover:
	go test -covermode=count -coverprofile=cover.out -count=1 ./internal/...
	go tool cover -func=cover.out
	go tool cover -html=cover.out

.PHONY: build
build:
	go build -o build/${BIN_NAME} cmd/server/main.go && \
		go build -o build/${BIN_NAME}-cli cmd/client/main.go

.PHONY: build.docker
build.docker:
	docker build -t memdb:latest .

.PHONY: generate
generate:
	go generate ./...

.PHONY: clean
clean:
	rm -rf build/ cover.out