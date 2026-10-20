# Stage 1: Build frontend (architecture-independent, built once on the build host)
FROM --platform=$BUILDPLATFORM oven/bun:1 AS frontend
WORKDIR /app/front
COPY front/package.json front/bun.lock ./
RUN bun install --frozen-lockfile
COPY front/ .
RUN bun run build

# Stage 2: Build backend on the build host, cross-compiled for the target platform
FROM --platform=$BUILDPLATFORM golang:1.26 AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=frontend /app/front/dist ./front/dist
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags "-X github.com/synclet-io/synclet/pkg/buildinfo.version=${VERSION}" -o /synclet .

# Stage 3: Runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /synclet /usr/local/bin/synclet
ENTRYPOINT ["synclet"]
