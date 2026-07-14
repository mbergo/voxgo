BINARY := voxgo
PREFIX ?= /usr/local
IMAGE ?= ghcr.io/mbergo/voxgo
TAG ?= latest
ENGINE ?= $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)

.PHONY: build install clean test vet run-web run-daemon container publish

build:
	go build -trimpath -ldflags "-s -w" -o $(BINARY) .

install: build
	install -Dm755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

# Easy-start helpers
run-web: build
	./$(BINARY) web

run-daemon: build
	./$(BINARY) daemon

# OCI image (podman or docker, whichever is installed)
container:
	$(ENGINE) build -t $(IMAGE):$(TAG) .

publish: container
	$(ENGINE) push $(IMAGE):$(TAG)

clean:
	rm -f $(BINARY)
