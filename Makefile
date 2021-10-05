.PHONY: build test clean update fmt docker run

GO=CGO_ENABLED=1 GO111MODULE=on go

MICROSERVICE=app-rfid-llrp-inventory

.PHONY: build test clean fmt docker run

APPVERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

# This pulls the version of the SDK from the go.mod file. It works by looking for the line
# with the SDK and printing just the version number that comes after it.
SDKVERSION=$(shell sed -En 's|.*github.com/edgexfoundry/app-functions-sdk-go (v[\.0-9a-zA-Z-]+).*|\1|p' go.mod)

GOFLAGS=-ldflags "-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.SDKVersion=$(SDKVERSION) \
					-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.ApplicationVersion=$(APPVERSION) \
					-X edgexfoundry/app-rfid-llrp-inventory.Version=$(APPVERSION)"

build:
	$(GO) build $(GOFLAGS) -o $(MICROSERVICE)

tidy:
	go mod tidy

t:
	[ -z "$$(gofmt -p -l . || echo 'err')" ]

test:
	$(GO) test -coverprofile=coverage.out ./...
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