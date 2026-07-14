# Multi-stage build: compile voxgo, ship a tiny runtime image.
#
# Audio (PipeWire) and typing (Wayland) live on the host, so the container is
# mainly useful for the web dashboard or CI. Mount the sockets if you want
# sound:
#
#   podman run --rm -p 7853:7853 \
#     -v $XDG_RUNTIME_DIR/pipewire-0:/run/pipewire-0 \
#     -e PIPEWIRE_RUNTIME_DIR=/run \
#     -e OPENAI_API_KEY=... \
#     ghcr.io/mbergo/voxgo web

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /voxgo .

FROM alpine:3.21
RUN apk add --no-cache pipewire-tools wtype libnotify
COPY --from=build /voxgo /usr/local/bin/voxgo
EXPOSE 7853
ENTRYPOINT ["voxgo"]
CMD ["web"]
