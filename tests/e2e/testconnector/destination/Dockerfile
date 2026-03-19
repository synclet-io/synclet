FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY destination/ destination/
RUN CGO_ENABLED=0 go build -mod=vendor -o /connector ./destination/

FROM alpine:3.19
COPY --from=builder /connector /connector
ENTRYPOINT ["/connector"]
