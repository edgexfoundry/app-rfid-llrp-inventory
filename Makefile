.PHONY: build test unittest lint clean update fmt docker run

GO=CGO_ENABLED=1 GO111MODULE=on go

# see https://shibumi.dev/posts/hardening-executables
CGO_CPPFLAGS="-D_FORTIFY_SOURCE=2"
CGO_CFLAGS="-O2 -pipe -fno-plt"
CGO_CXXFLAGS="-O2 -pipe -fno-plt"
CGO_LDFLAGS="-Wl,-O1,–sort-common,–as-needed,-z,relro,-z,now"

ARCH=$(shell uname -m)

MICROSERVICE=app-rfid-llrp-inventory

.PHONY: build test clean fmt docker run

APPVERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

# This pulls the version of the SDK from the go.mod file. It works by looking for the line
# with the SDK and printing just the version number that comes after it.
SDKVERSION=$(shell sed -En 's|.*github.com/edgexfoundry/app-functions-sdk-go/v2 (v[\.0-9a-zA-Z-]+).*|\1|p' go.mod)

GOFLAGS=-ldflags "-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.SDKVersion=$(SDKVERSION) \
					-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.ApplicationVersion=$(APPVERSION) \
					-X edgexfoundry/app-rfid-llrp-inventory.Version=$(APPVERSION)"

GOFLAGS=-ldflags "-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.SDKVersion=$(SDKVERSION) \
					-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.ApplicationVersion=$(APPVERSION) \
					-X edgexfoundry/app-rfid-llrp-inventory.Version=$(APPVERSION)" -trimpath -mod=readonly
CGOFLAGS=-ldflags "-linkmode=external -X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.SDKVersion=$(SDKVERSION) \
					-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.ApplicationVersion=$(APPVERSION) \
					-X edgexfoundry/app-rfid-llrp-inventory.Version=$(APPVERSION)" -trimpath -mod=readonly -buildmode=pie

build:
	$(GO) build $(CGOFLAGS) -o $(MICROSERVICE)

tidy:
	go mod tidy

t:
	[ -z "$$(gofmt -p -l . || echo 'err')" ]

unittest:
	$(GO) test ./... -coverprofile=coverage.out ./...

lint:
	@which golangci-lint >/dev/null || echo "WARNING: go linter not installed. To install, run\n  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.42.1"
	@if [ "z${ARCH}" = "zx86_64" ] && which golangci-lint >/dev/null ; then golangci-lint run --config .golangci.yml ; else echo "WARNING: Linting skipped (not on x86_64 or linter not installed)"; fi

test: unittest lint
	$(GO) vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]
	./bin/test-attribution.sh

clean:
	rm -f $(MICROSERVICE)

update:
	$(GO) mod download

fmt:
	$(GO) fmt ./...

docker:
	docker build \
		--rm \
		--build-arg http_proxy \
		--build-arg https_proxy \
			--label "git_sha=$(GIT_SHA)" \
			-t edgexfoundry/app-rfid-llrp-inventory:$(GIT_SHA) \
			-t edgexfoundry/app-rfid-llrp-inventory:$(APPVERSION)-dev \
			.

run: build
	./$(MICROSERVICE) -cp=consul.http://localhost:8500 -confdir=res

vendor:
	$(GO) mod vendor