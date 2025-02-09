BIN_NAME := memdb
LINTER_VERSION := v1.62.2
MOCKERY_VERSION_v2 := 51.1

.PHONY: all
all: clean lint test build

.PHONY: .install-mockery
.install-mockery:
	go install github.com/vektra/mockery/v2@v2.${MOCKERY_VERSION_v2}

.PHONY: .install-linter
.install-linter:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOPATH}/bin ${LINTER_VERSION}

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
	go test -race -count=1 ./...

test.cover:
	go test -covermode=count -coverprofile=cover.out -count=1 ./...
	go tool cover -func=cover.out
	go tool cover -html=cover.out

.PHONY: build
build:
	go build -o build/${BIN_NAME} cmd/server/main.go && \
		go build -o build/${BIN_NAME}-cli cmd/client/main.go

.PHONY: generate
generate:
	go generate ./...

.PHONY: clean
clean:
	rm -rf build/ cover.out