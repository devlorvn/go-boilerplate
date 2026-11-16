# syntax=docker/dockerfile:1

# One image for every service; the command picks it:
#   docker run IMAGE gserver auth
#   docker run IMAGE migrate auth up

FROM golang:1.26.4-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/server ./cmd/server

# distroless/static: no shell, no package manager, runs as uid 65532.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/server /app/server
COPY configs/*.yaml /app/configs/
COPY migrations /app/migrations

ENV COMMON_ENV=prod \
    COMMON_LOG_LEVEL=info
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
