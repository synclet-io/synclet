FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY destination/ destination/
RUN CGO_ENABLED=0 go build -o /connector ./destination/

FROM alpine:3.19
COPY --from=builder /connector /connector
ENV AIRBYTE_ENTRYPOINT=/connector
ENTRYPOINT ["/connector"]
