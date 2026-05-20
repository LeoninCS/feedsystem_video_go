# syntax=docker/dockerfile:1.7
#
# Multi-stage build for backend API and worker.
#
# Build:
#   docker build -f backend/Dockerfile -t feedsystem-backend:api --target api .
#   docker build -f backend/Dockerfile -t feedsystem-backend:worker --target worker .
#

ARG GO_VERSION=1.24.5
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn

FROM golang:${GO_VERSION} AS deps
ARG GOPROXY
ARG GOSUMDB
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB}

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

FROM deps AS source
COPY backend/ ./

ENV CGO_ENABLED=0

FROM source AS api-build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd

FROM source AS worker-build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM alpine:3.21 AS base
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -s /sbin/nologin app
WORKDIR /app
COPY --from=source /src/backend/configs ./configs
RUN mkdir -p ./.run/uploads && chown -R app:app /app
USER app

FROM base AS api
COPY --from=api-build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM base AS worker
COPY --from=worker-build /out/worker /app/worker
ENTRYPOINT ["/app/worker"]
