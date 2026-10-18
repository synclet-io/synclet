# Stage 1: Build frontend
FROM oven/bun:1 AS frontend
WORKDIR /app/front
COPY front/package.json front/bun.lock ./
RUN bun install --frozen-lockfile
COPY front/ .
RUN bun run build

# Stage 2: Build backend
FROM golang:1.26 AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/front/dist ./front/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X github.com/synclet-io/synclet/pkg/buildinfo.version=${VERSION}" -o /synclet .

# Stage 3: Runtime
FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=backend /synclet /usr/local/bin/synclet
ENTRYPOINT ["synclet"]
