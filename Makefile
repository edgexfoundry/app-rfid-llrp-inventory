.PHONY: build test clean update fmt docker run

GO=CGO_ENABLED=1 GO111MODULE=on go

MICROSERVICE=rfid-llrp-inventory

.PHONY: build test clean fmt docker run

APPVERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

# This pulls the version of the SDK from the go.mod file. It works by looking for the line
# with the SDK and printing just the version number that comes after it.
SDKVERSION=$(shell sed -En 's|.*github.com/edgexfoundry/app-functions-sdk-go (v[\.0-9a-zA-Z-]+).*|\1|p' go.mod)

GOFLAGS=-ldflags "-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.SDKVersion=$(SDKVERSION) \
					-X github.com/edgexfoundry/app-functions-sdk-go/v2/internal.ApplicationVersion=$(APPVERSION) \
					-X edgexfoundry-holding/rfid-llrp-inventory-service.Version=$(APPVERSION)"

build:
	$(GO) build $(GOFLAGS) -o $(MICROSERVICE)

t:
	[ -z "$$(gofmt -p -l . || echo 'err')" ]

test:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) vet ./...
	./bin/test-attribution.sh
	./bin/test-go-mod-tidy.sh
	output="$$(gofmt -l .)" && [ -z "$$output" ]

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
			-t edgexfoundry/docker-rfid-llrp-inventory:$(GIT_SHA) \
			-t edgexfoundry/docker-rfid-llrp-inventory:$(VERSION)-dev \
			.

run: build
	./$(MICROSERVICE) -cp=consul.http://localhost:8500 -confdir=res
