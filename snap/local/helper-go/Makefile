
build:
	go build -ldflags="-s -w" -o helper-go

tidy:
	go mod tidy

clean:
	rm -f helper-go

# TODO: fixup unit testing
test:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) vet ./...
	gofmt -l .
	[ "`gofmt -l .`" = "" ]