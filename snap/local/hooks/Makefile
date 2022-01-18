GO=go
HOOKS=cmd/configure/configure cmd/install/install

.PHONY: tidy build test clean

build: tidy $(HOOKS)

tidy:
	go mod tidy

$(HOOKS):
	$(GO) build -o $@ ./$(dir $@)

# TODO: fixup unit testing
test:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) vet ./...
	gofmt -l .
	[ "`gofmt -l .`" = "" ]

clean:
	rm -f $(HOOKS)
