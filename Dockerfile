# syntax=docker/dockerfile:1.7

########################
# Stage 1: build
########################
FROM golang:1.23-alpine AS build

# Build args are injected by CI so the binary carries traceable metadata.
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

WORKDIR /src

# Copy module files first for better layer caching. This project has no
# external dependencies, so this stays tiny.
COPY go.mod ./
RUN go mod download

# Copy source and build a fully static binary.
COPY . .

ENV CGO_ENABLED=0 GOOS=linux
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath \
    -ldflags "-s -w \
    -X devops-showcase/internal/version.Version=${VERSION} \
    -X devops-showcase/internal/version.Commit=${COMMIT} \
    -X devops-showcase/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/server ./cmd/server

########################
# Stage 2: runtime
########################
# Distroless static: no shell, no package manager, minimal attack surface.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /

# Run as the built-in non-root user provided by the distroless image.
USER nonroot:nonroot

COPY --from=build /out/server /server

EXPOSE 8080

# The distroless "nonroot" image has no shell, so probes are handled by
# Kubernetes via HTTP GET rather than a container HEALTHCHECK using curl.
ENV PORT=8080 APP_ENV=production LOG_LEVEL=info

ENTRYPOINT ["/server"]
