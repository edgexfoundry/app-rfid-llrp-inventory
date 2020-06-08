.PHONY: build test clean prepare update docker

#GO = CGO_ENABLED=0 GO111MODULE=on go
GO = GO111MODULE=on go

MICROSERVICES=rfid-inventory

.PHONY: $(MICROSERVICES)

DOCKERS=docker_rfid_inventory

.PHONY: $(DOCKERS)

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

GOFLAGS=-ldflags "-X github.impcloud.net/RSP-Inventory-Suite/rfid-inventory.Version=$(VERSION)"

BUILD_DIR=build
build: $(MICROSERVICES)
	$(GO) build ./...

rfid-inventory:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$@ ./main.go
	cp -r ./res/ $(BUILD_DIR)/

test:
	$(GO) test ./... -coverprofile=coverage.out

clean:
	rm -rf $(BUILD_DIR) 

docker: $(DOCKERS)

docker_rfid_inventory:
	docker build \
--build-arg http_proxy \
--build-arg https_proxy \
--label "git_sha=$(GIT_SHA)" \
-t edgexfoundry/docker-rfid-inventory:$(GIT_SHA) \
-t edgexfoundry/docker-rfid-inventory:$(VERSION)-dev \
.

run:
	docker-compose -f docker-compose.yml up -d

stop:
	docker-compose -f docker-compose.yml down
