# syntax=docker/dockerfile:1.19
FROM golang:1.26.5-alpine3.24@sha256:111d79159b2326f7e80c4a4706e1ba166acb0e2611df853955f3621828cd49e8 AS build

WORKDIR /src
COPY discord/padinho/go.mod discord/padinho/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY discord/padinho/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/padinho ./cmd/padinho

FROM gcr.io/distroless/static-debian13:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6

WORKDIR /app
COPY --from=build --chown=nonroot:nonroot /out/padinho /app/padinho

USER nonroot:nonroot
ENTRYPOINT ["/app/padinho"]
