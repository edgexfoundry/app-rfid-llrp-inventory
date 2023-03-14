.PHONY: build test unittest lint clean update fmt docker run

ARCH=$(shell uname -m)

MICROSERVICE=app-rfid-llrp-inventory

.PHONY: build test clean fmt docker run

APPVERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

# This pulls the version of the SDK from the go.mod file. It works by looking for the line
# with the SDK and printing just the version number that comes after it.
SDKVERSION=$(shell sed -En 's|.*github.com/edgexfoundry/app-functions-sdk-go/v3 (v[\.0-9a-zA-Z-]+).*|\1|p' go.mod)

GOFLAGS=-ldflags "-X github.com/edgexfoundry/app-functions-sdk-go/v3/internal.SDKVersion=$(SDKVERSION) \
					-X github.com/edgexfoundry/app-functions-sdk-go/v3/internal.ApplicationVersion=$(APPVERSION) \
					-X edgexfoundry/app-rfid-llrp-inventory.Version=$(APPVERSION)" -trimpath -mod=readonly
GOTESTFLAGS?=-race

# CGO is enabled by default and causes local docker builds to fail due to no gcc,
# but is required for test with -race, so must disable it for the builds only
build:
	CGO_ENABLED=0 go build -tags "$(ADD_BUILD_TAGS)" $(GOFLAGS) -o $(MICROSERVICE)

build-nats:
	make -e ADD_BUILD_TAGS=include_nats_messaging build

tidy:
	go mod tidy

t:
	[ -z "$$(gofmt -p -l . || echo 'err')" ]

unittest:
	go test $(GOTESTFLAGS) ./... -coverprofile=coverage.out ./...

lint:
	@which golangci-lint >/dev/null || echo "WARNING: go linter not installed. To install, run make install-lint"
	@if [ "z${ARCH}" = "zx86_64" ] && which golangci-lint >/dev/null ; then golangci-lint run --config .golangci.yml ; else echo "WARNING: Linting skipped (not on x86_64 or linter not installed)"; fi

install-lint:
	sudo curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.51.2

test: unittest lint
	go vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]
	./bin/test-attribution.sh

clean:
	rm -f $(MICROSERVICE)

fmt:
	go fmt ./...

docker:
	docker build \
		--rm \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		--build-arg http_proxy \
		--build-arg https_proxy \
			--label "git_sha=$(GIT_SHA)" \
			-t edgexfoundry/app-rfid-llrp-inventory:$(GIT_SHA) \
			-t edgexfoundry/app-rfid-llrp-inventory:$(APPVERSION)-dev \
			.

docker-nats:
	make -C . -e ADD_BUILD_TAGS=include_nats_messaging docker

run: build
	./$(MICROSERVICE) -cp=consul.http://localhost:8500 -confdir=res

vendor:
	go mod vendor