# syntax=docker/dockerfile:1.19
FROM golang:1.26.0-bookworm@sha256:2a0ba12e116687098780d3ce700f9ce3cb340783779646aafbabed748fa6677c AS build

WORKDIR /src/padinho
COPY padinho/go.mod padinho/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY padinho/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/padinho ./cmd/padinho && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian13:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6

WORKDIR /app
COPY --from=build --chown=nonroot:nonroot /out/padinho /app/padinho
COPY --from=build --chown=nonroot:nonroot /out/migrate /app/migrate
COPY --chown=nonroot:nonroot database/migrations/ /app/migrations/

USER nonroot:nonroot
ENTRYPOINT ["/app/padinho"]
