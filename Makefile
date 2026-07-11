BINARY := voxgo
PREFIX ?= /usr/local

.PHONY: build install clean test vet

build:
	go build -trimpath -ldflags "-s -w" -o $(BINARY) .

install: build
	install -Dm755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
