FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY source/ source/
RUN CGO_ENABLED=0 go build -mod=vendor -o /connector ./source/

FROM alpine:3.19
COPY --from=builder /connector /connector
ENTRYPOINT ["/connector"]
