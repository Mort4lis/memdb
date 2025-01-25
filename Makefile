BIN_NAME := memdb
LINTER_VERSION := v1.62.2

.PHONY: lint.install
lint.install:
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
	go test -covermode=count -coverprofile=cover.out -p 2 -count=1 ./...
	go tool cover -func=cover.out
	go tool cover -html=cover.out

.PHONY: build
build:
	go build -o build/${BIN_NAME} cmd/server/main.go

.PHONY: generate
generate:
	go generate ./...

.PHONY: clean
clean:
	rm -rf build/ *.coverprofile coverage.*