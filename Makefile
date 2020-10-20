.PHONY: build test clean update fmt docker run

GO=CGO_ENABLED=1 GO111MODULE=on go

MICROSERVICE=rfid-llrp-inventory

.PHONY: build test clean fmt docker run

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

GOFLAGS=-ldflags "-X edgexfoundry-holding/rfid-llrp-inventory-service.Version=$(VERSION)"

build:
	$(GO) build $(GOFLAGS) -o $(MICROSERVICE)

test:
	go test -coverprofile=coverage.out ./...
	go vet ./...
	./bin/test-attribution.sh
	./bin/test-go-mod-tidy.sh

clean:
	rm -f $(MICROSERVICE)

update:
	$(GO) mod download

fmt:
	go fmt ./...

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
	./$(MICROSERVICE) -cp=consul://localhost:8500 -confdir=res
