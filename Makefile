.PHONY: build test clean fmt docker

GO=CGO_ENABLED=1 GO111MODULE=on go

MICROSERVICES=cmd/rfid-inventory

.PHONY: $(MICROSERVICES)

DOCKERS=docker_rfid_inventory

.PHONY: $(DOCKERS) run up clean fmt deploy iterate tail stop-container down stop

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
GIT_SHA=$(shell git rev-parse HEAD)

GOFLAGS=-ldflags "-X github.impcloud.net/RSP-Inventory-Suite/rfid-inventory.Version=$(VERSION)"

# default tail lines
n = 100

build: $(MICROSERVICES)

cmd/rfid-inventory:
	$(GO) build $(GOFLAGS) -o $@ ./main.go

test:
	$(GO) test $(args) ./... -coverprofile=coverage.out

clean:
	rm -f $(MICROSERVICES)

fmt:
	go fmt ./...

tail:
	docker logs -f --tail $(n) $(shell docker ps -qf name=rfid-inventory)

kill:
	docker kill $(shell docker ps -qf name=rfid-inventory) || true

stop-container:
	docker stop $(shell docker ps -qf name=rfid-inventory) || true

iterate: fmt
	$(MAKE) -j docker stop-container
	$(MAKE) deploy tail

docker: $(DOCKERS)

docker_rfid_inventory:
	docker build \
		--rm \
		--build-arg http_proxy \
		--build-arg https_proxy \
			--label "git_sha=$(GIT_SHA)" \
			-t edgexfoundry/docker-rfid-inventory:$(GIT_SHA) \
			-t edgexfoundry/docker-rfid-inventory:$(VERSION)-dev \
			.

run: cmd/rfid-inventory
	cd ./cmd && ./rfid-inventory -cp=consul://localhost:8500 -confdir=res

up:
	docker-compose up

deploy:
	docker-compose up -d

stop:
	docker-compose stop

down:
	docker-compose down
